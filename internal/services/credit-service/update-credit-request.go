package creditService

import (
	"encoding/json"
	"errors"
	"fmt"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	approvalService "prime-erp-core/internal/services/approval-service"
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
	customerCode := []string{}
	now := time.Now()
	for _, credit := range req {
		customerCode = append(customerCode, credit.CustomerCode)
	}

	requestDataGetCredit := map[string][]string{
		"customer_code": customerCode,
	}
	jsonBytesGetCredit, err := json.Marshal(requestDataGetCredit)
	if err != nil {
		return nil, err
	}

	GetCreditRes, errGetCredit := GetCredit(ctx, string(jsonBytesGetCredit))
	if errGetCredit != nil {
		return nil, errGetCredit
	}

	mapCredit := map[string]models.Credit{}
	mapCreditExtra := map[string]models.CreditExtra{}

	for _, creditValue := range GetCreditRes.(ResultCredit).Credit {
		mapCredit[creditValue.DocRef] = creditValue
		for _, creditExtraValue := range creditValue.CreditExtra {
			mapCreditExtra[creditExtraValue.DocRef] = creditExtraValue
		}

	}
	creditIDForDelete := []uuid.UUID{}
	creditExtraIDForDelete := []uuid.UUID{}
	updateApproval := []models.Approval{}

	for i := range req {
		if req[i].Status == "REJECT" {
			if len(GetCreditRes.(ResultCredit).Credit) > 0 {
				if req[i].RequestType == "EXTRA" {
					/* if len(GetCreditRes.(ResultCredit).Credit[0].CreditExtra) > 0 {
					} */
					if creditExtraValue, ok := mapCreditExtra[req[i].RequestCode]; ok {
						creditTransaction = append(creditTransaction, models.CreditTransaction{
							TransactionCode: creditExtraValue.DocRef,
							TransactionType: "EXTRA",
							Amount:          creditExtraValue.Amount,
							AdjustAmount:    0,
							EffectiveDtm:    creditExtraValue.EffectiveDtm,
							ExpireDtm:       creditExtraValue.ExpireDtm,
							IsApprove:       false,
							Status:          "REJECT",
							Reason:          "",
						})
						req[0].RequestCode = creditExtraValue.DocRef
					}

					creditExtraIDForDelete = append(creditExtraIDForDelete, req[i].ID)

				} else {
					if creditValue, ok := mapCredit[req[i].RequestCode]; ok {
						creditTransaction = append(creditTransaction, models.CreditTransaction{
							TransactionCode: creditValue.DocRef,
							TransactionType: "BASE",
							Amount:          creditValue.Amount,
							AdjustAmount:    0,
							IsApprove:       false,
							Status:          "REJECT",
							Reason:          "",
						})
						req[0].RequestCode = creditValue.DocRef
					}

					creditIDForDelete = append(creditIDForDelete, req[i].ID)
				}
			}
			req[i].ActionDate = &now
		}
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
				})
			} else {

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
			updateApproval = append(updateApproval, models.Approval{
				ID:     req[i].ApprovalID,
				Status: "COMPLETED",
			})
		}
		if req[i].Status == "COMPLETED" {
			req[i].IsApprove = true
			req[i].ApproveDate = &now
		}

		creditRequestValue = append(creditRequestValue, req[i])
		if req[i].Status != "REJECT" {
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
	if len(creditIDForDelete) > 0 || len(creditExtraIDForDelete) > 0 {
		errDeleteCredit := repositoryCredit.DeleteCredit(creditIDForDelete, creditExtraIDForDelete)
		if errDeleteCredit != nil {
			return nil, errDeleteCredit
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
	if len(updateApproval) > 0 {
		updateApprovalPayload, _ := json.Marshal(updateApproval)
		_, err := approvalService.UpdateApproval(ctx, string(updateApprovalPayload))
		if err != nil {
			return nil, fmt.Errorf("failed to update approval: %v", err)
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
