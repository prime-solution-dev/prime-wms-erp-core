package creditService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	"prime-erp-core/internal/requestcontext"
	approvalService "prime-erp-core/internal/services/approval-service"
	"time"

	"github.com/google/uuid"
)

func CreateCreditRequest(ctx context.Context, jsonPayload string) (interface{}, error) {

	var req []models.CreditRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	if err := validateExtraCreditRequests(ctx, req, GetHistory); err != nil {
		return nil, err
	}

	creditRequestValue := []models.CreditRequest{}
	approvalValue := []models.Approval{}
	approvalIDForReturn := []uuid.UUID{}
	//createdAt := time.Now()
	userID := requestcontext.GetUserOrDefault(ctx)
	for i := range req {
		creditID := uuid.New()
		req[i].ID = creditID
		//req[i].ActionDate = &createdAt

		approvalIDForReturn = append(approvalIDForReturn, creditID)

		if req[i].RequestCode == "" {
			req[i].RequestCode = uuid.New().String()
		}
		req[i].CreateBy = userID
		req[i].UpdateBy = userID
		creditRequestValue = append(creditRequestValue, req[i])

		approval := models.Approval{
			ID:            uuid.New(),
			ApproveTopic:  "CL",
			DocumentType:  "CR",
			DocumentCode:  req[i].RequestCode,
			DocumentData:  nil,
			ActionDate:    time.Now(),
			Status:        "PENDING",
			Remark:        "",
			CurentStepSeq: 1,
			MDItemCode:    "CTM-CTM7",
			CreateBy:      userID,
		}
		approvalValue = append(approvalValue, approval)
	}

	jsonBytesCreateApproval, err := json.Marshal(approvalValue)
	if err != nil {
		return nil, err
	}
	resultCreateApproval, errApproval := approvalService.CreateApproval(ctx, string(jsonBytesCreateApproval))
	if errApproval != nil {
		return nil, errApproval
	}

	errCreateApproval := repositoryCredit.CreateCreditRequest(creditRequestValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}
	if len(approvalValue) > 0 {

		// approvalService.CheckAutoApprovalRest เป็นแค่เปลือก JSON ห่อ
		// CheckAutoApproval(ctx, gormx, req, user) ที่ไม่แตะ gormx เลย (_ = gormx) — เรียก
		// CheckAutoApproval ตรงๆ แบบเดียวกับ sale-service/create-sale.go ตัดรอบ JSON
		// marshal/unmarshal ทิ้งไปด้วย ผล/ประเภทคืนค่าเดิมทุกอย่าง
		checkAutoApprovalReq := approvalService.CheckAutoApprovalRequest{
			RequestUserCode: userID,
			ModuleCode:      "CUSTOMIZE",
			TopicCode:       "CUSTOMIZE",
			MdItemCode:      "CTM-CTM7",
			CondRangeMin:    req[0].Amount,
		}

		resultCheckAutoApprovalRest, errCheckAutoApprovalRest := approvalService.CheckAutoApproval(ctx, nil, checkAutoApprovalReq, userID)
		if errCheckAutoApprovalRest != nil {
			return nil, errCheckAutoApprovalRest
		}
		//mapResultCreateApproval := resultCreateApproval.(map[string]interface{})
		//ids := mapResultCreateApproval["id"].([]uuid.UUID)
		if resultCheckAutoApprovalRest.IsAutoApproved {

			creditRequest := []models.CreditRequest{}
			data := resultCreateApproval.(map[string]interface{})
			ids := data["id"].([]uuid.UUID)
			for i := range req {
				req[i].Status = "COMPLETED"
				req[i].ApprovalID = ids[i]
				req[i].UpdateBy = userID
				creditRequest = append(creditRequest, req[i])
			}
			jsonDataUpdateCreditRequest, err := json.Marshal(creditRequest)
			if err != nil {
				return nil, errors.New("Error marshalling data : " + err.Error())
			}

			_, errUpdateCreditRequest := UpdateCreditRequest(ctx, string(jsonDataUpdateCreditRequest))
			if errUpdateCreditRequest != nil {
				return nil, errUpdateCreditRequest
			}

		}

	}

	return map[string]interface{}{
		"id":      approvalIDForReturn,
		"status":  "success",
		"message": "Approval create request successfully",
	}, nil
}

type getCreditHistoryFunc func(context.Context, string) (interface{}, error)

func validateExtraCreditRequests(ctx context.Context, requests []models.CreditRequest, getHistory getCreditHistoryFunc) error {
	checkedCustomers := make(map[string]struct{})

	for _, request := range requests {
		if request.RequestType != "EXTRA" {
			continue
		}
		if request.CustomerCode == "" {
			return fmt.Errorf("customer_code is required for EXTRA credit request")
		}
		if _, checked := checkedCustomers[request.CustomerCode]; checked {
			continue
		}

		payload, err := json.Marshal(GetHistoryReq{
			CustomerCode: []string{request.CustomerCode},
		})
		if err != nil {
			return err
		}

		result, err := getHistory(ctx, string(payload))
		if err != nil {
			return err
		}
		history, ok := result.(ResultHistory)
		if !ok {
			return fmt.Errorf("unexpected GetHistory response type %T", result)
		}

		hasActiveBase := false
		for _, item := range history.HistoryRes {
			if item.RequestType == "BASE" && item.Status == "ACTIVE" {
				hasActiveBase = true
				break
			}
		}
		if !hasActiveBase {
			return fmt.Errorf("customer %s must have an ACTIVE BASE credit before creating an EXTRA credit request", request.CustomerCode)
		}

		checkedCustomers[request.CustomerCode] = struct{}{}
	}

	return nil
}

// RequestType EXTRA requires an ACTIVE BASE credit for the same customer.
