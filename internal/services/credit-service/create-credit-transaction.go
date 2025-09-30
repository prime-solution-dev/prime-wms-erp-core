package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateCreditTransaction(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.CreditTransaction

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	creditTransactionValue := []models.CreditTransaction{}
	approvalIDForReturn := []uuid.UUID{}
	for i := range req {
		creditID := uuid.New()
		req[i].ID = creditID

		approvalIDForReturn = append(approvalIDForReturn, creditID)

		if req[i].TransactionCode == "" {
			req[i].TransactionCode = uuid.New().String()
		}

		creditTransactionValue = append(creditTransactionValue, req[i])

	}

	errCreateApproval := repositoryCredit.CreateCreditTransaction(creditTransactionValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"id":      approvalIDForReturn,
		"status":  "success",
		"message": "Approval create Transaction successfully",
	}, nil
}
