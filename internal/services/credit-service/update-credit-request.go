package creditService

import (
	"encoding/json"
	"errors"
	"fmt"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		/* if req[i].Status == "REJECT" {
			creditTransaction = append(creditTransaction, models.CreditTransaction{
				TransactionCode: req[i].RequestCode,
				TransactionType: req[i].RequestType,
				Amount:          req[i].Amount,
				AdjustAmount:    0,
				EffectiveDtm:    req[i].EffectiveDtm,
				ExpireDtm:       req[i].ExpireDtm,
				//ForceExpireDtm:  req[i].e,
				//ApproveDate:     "",
				IsApprove: false,
				Status:    "REJECT",
				Reason:    "",
			})

		} */
		if req[i].Status == "COMPLETED" {
			creditExtra := []models.CreditExtra{}
			CreditID := uuid.New()
			if req[i].RequestType == "EXTRA" {
				creditExtra = append(creditExtra, models.CreditExtra{
					ID:       uuid.New(),
					CreditID: CreditID,
					//ExtraType:    "",
					Amount:       req[i].Amount,
					EffectiveDtm: req[i].EffectiveDtm,
					ExpireDtm:    req[i].ExpireDtm,
					DocRef:       req[i].RequestCode,
					//ApproveDate:  "",
				})
			} else {
				now := time.Now()
				credit = append(credit, models.Credit{
					ID:                 CreditID,
					CustomerCode:       req[i].CustomerCode,
					Amount:             req[i].Amount,
					EffectiveDtm:       req[i].EffectiveDtm,
					IsActive:           true,
					DocRef:             req[i].RequestCode,
					ApproveDate:        &now,
					AlertBalanceCredit: false,
					CreditExtra:        creditExtra,
				})

				req[i].IsAction = true

			}

		}

		creditRequestValue = append(creditRequestValue, req[i])

		creditTransaction = append(creditTransaction, models.CreditTransaction{
			TransactionCode: req[i].RequestCode,
			TransactionType: req[i].RequestType,
			Amount:          req[i].Amount,
			AdjustAmount:    0,
			EffectiveDtm:    req[i].EffectiveDtm,
			ExpireDtm:       req[i].ExpireDtm,
			IsApprove:       false,
			Status:          req[i].Status,
			Reason:          "",
		})

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
		fmt.Println(string(jsonByteserrCredit))
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
			"message": "Approval Not Have Rows Affected ",
		}, nil
	}
}
