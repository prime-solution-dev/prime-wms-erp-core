package creditService

import (
	"context"
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	"prime-erp-core/internal/requestcontext"
)

func UpdateCredit(ctx context.Context, jsonPayload string) (interface{}, error) {

	var req []models.Credit

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	userID := requestcontext.GetUserOrDefault(ctx)
	creditValue := []models.Credit{}
	creditExtraValue := []models.CreditExtra{}

	for i, credit := range req {
		for o := range credit.CreditExtra {
			creditExtraValue = append(creditExtraValue, req[i].CreditExtra[o])
		}
		req[i].CreateBy = userID
		req[i].UpdateBy = userID
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
