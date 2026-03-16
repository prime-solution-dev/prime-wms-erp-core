package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	approvalService "prime-erp-core/internal/services/approval-service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateCreditRequest(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.CreditRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	creditRequestValue := []models.CreditRequest{}
	approvalValue := []models.Approval{}
	approvalIDForReturn := []uuid.UUID{}
	//createdAt := time.Now()
	conUserID, _ := ctx.Get("user")
	userID := ""
	if conUserID != nil {
		userID = conUserID.(string)
	}
	for i := range req {
		creditID := uuid.New()
		req[i].ID = creditID
		//req[i].ActionDate = &createdAt

		approvalIDForReturn = append(approvalIDForReturn, creditID)

		if req[i].RequestCode == "" {
			req[i].RequestCode = uuid.New().String()
		}

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

		requestDataCheckAutoApprovalRest := map[string]interface{}{
			"request_user_code": userID,
			"md_item_code":      "CTM-CTM7",
			"cond_range_min":    req[0].Amount,
		}

		jsonDataCheckAutoApprovalRest, err := json.Marshal(requestDataCheckAutoApprovalRest)
		if err != nil {
			errors.New("Error marshalling data :")
		}

		checkAutoApprovalRest, errCheckAutoApprovalRest := approvalService.CheckAutoApprovalRest(ctx, string(jsonDataCheckAutoApprovalRest))
		if errCheckAutoApprovalRest != nil {
			return nil, errCheckAutoApprovalRest
		}
		resultCheckAutoApprovalRest := checkAutoApprovalRest.(approvalService.CheckAutoApprovalResponse)
		if resultCheckAutoApprovalRest.IsAutoApproved {

			creditRequest := []models.CreditRequest{}
			for i := range req {
				req[i].Status = "COMPLETED"
				req[i].ApprovalID = resultCreateApproval.([]map[string]interface{})[i]["id"].(uuid.UUID)
				creditRequest = append(creditRequest, req[i])
			}
			jsonDataUpdateCreditRequest, err := json.Marshal(creditRequest)
			if err != nil {
				errors.New("Error marshalling data :")
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
