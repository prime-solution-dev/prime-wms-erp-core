package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
)

func UpdateCredit(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Credit

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	creditValue := []models.Credit{}
	creditExtraValue := []models.CreditExtra{}

	for i, credit := range req {
		for o := range credit.CreditExtra {
			creditExtraValue = append(creditExtraValue, req[i].CreditExtra[o])
		}
		creditValue = append(creditValue, req[i])
	}

	rowsAffected, errCreateApproval := repositoryCredit.UpdateCredit(creditValue, creditExtraValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	if rowsAffected > 0 {
		return map[string]interface{}{
			"status":  "success",
			"message": "Approval updated successfully",
		}, nil
	} else {
		return map[string]interface{}{
			"status":  "success",
			"message": "Approval Not Have Rows Affected",
		}, nil
	}
}
