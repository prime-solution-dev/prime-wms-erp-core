package creditService

import (
	"context"
	"encoding/json"
	"errors"
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

		// approvalService.CheckAutoApprovalRest ยังไม่แปลง (นอก scope) และรับ gin's *Context
		// แต่มันเป็นแค่เปลือก JSON ห่อ CheckAutoApproval(gormx, req, user) ที่ไม่แตะ gormx เลย
		// (_ = gormx) — เรียก CheckAutoApproval ตรงๆ แบบเดียวกับ sale-service/create-sale.go
		// ตัดรอบ JSON marshal/unmarshal ทิ้งไปด้วย ผล/ประเภทคืนค่าเดิมทุกอย่าง
		checkAutoApprovalReq := approvalService.CheckAutoApprovalRequest{
			RequestUserCode: userID,
			ModuleCode:      "CUSTOMIZE",
			TopicCode:       "CUSTOMIZE",
			MdItemCode:      "CTM-CTM7",
			CondRangeMin:    req[0].Amount,
		}

		resultCheckAutoApprovalRest, errCheckAutoApprovalRest := approvalService.CheckAutoApproval(nil, checkAutoApprovalReq, userID)
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

// RequestType Base  ให้เอา customer ไป where credit_request status = pedding type base ว่ามีไหม ถ้า มี return กลับและสร้างไม่ได้
// RequestType extra ไม่ต้องเช็ค
