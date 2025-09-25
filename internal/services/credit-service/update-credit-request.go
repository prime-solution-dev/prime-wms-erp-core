package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
)

func UpdateCreditRequest(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.CreditRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	creditRequestValue := []models.CreditRequest{}
	creditTransaction := []models.CreditTransaction{}
	credit := []models.Credit{}

	for i := range req {
		if req[i].Status == "REJECT" {
			creditTransaction = append(creditTransaction, models.CreditTransaction{
				TransactionCode: req[i].RequestCode,
				TransactionType: "BASE",
				Amount:          req[i].Amount,
				AdjustAmount:    0,
				/* EffectiveDtm:    time.Now(),
				ExpireDtm:       time.Now(),
				ForceExpireDtm:  time.Now(),
				ApproveDate:     "", */
				IsApprove: false,
				Status:    "REJECT",
				Reason:    "",
			})

		}
		if req[i].Status == "COMPLETED" {
			creditExtra := []models.CreditExtra{}
			if req[i].RequestType == "EXTRA" {
				creditExtra = append(creditExtra, models.CreditExtra{
					//ExtraType:    "",
					Amount: req[i].Amount,
					//EffectiveDtm: "",
					//ExpireDtm:    "",
					DocRef: req[i].RequestCode,
					//ApproveDate:  "",
				})
			}

			credit = append(credit, models.Credit{
				CustomerCode: req[i].CustomerCode,
				Amount:       req[i].Amount,
				//EffectiveDtm:       "",
				IsActive: true,
				DocRef:   req[i].RequestCode,
				//ApproveDate:        "",
				//AlertBalanceCredit: "",
				CreditExtra: creditExtra,
			})

		}
		req[i].IsAction = true
		creditRequestValue = append(creditRequestValue, req[i])
	}
	if len(creditTransaction) > 0 {
		jsonByteserrCreditTransaction, err := json.Marshal(creditTransaction)
		if err != nil {
			return nil, err
		}
		_, errCreditTransaction := CreateCreditTransaction(ctx, string(jsonByteserrCreditTransaction))
		if errCreditTransaction != nil {
			return nil, errCreditTransaction
		}
	}

	if len(credit) > 0 {
		jsonByteserrCredit, err := json.Marshal(credit)
		if err != nil {
			return nil, err
		}
		_, errCreateCredit := CreateCredit(ctx, string(jsonByteserrCredit))
		if errCreateCredit != nil {
			return nil, errCreateCredit
		}
	}

	rowsAffected, errCreateApproval := repositoryCredit.UpdateCreditRequest(creditRequestValue)
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
