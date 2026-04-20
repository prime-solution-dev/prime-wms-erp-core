package creditService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateCredit(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Credit

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	conUserID, _ := ctx.Get("user")
	userID := ""
	if conUserID != nil {
		userID = conUserID.(string)
	}
	creditValue := []models.Credit{}
	creditExtraValue := []models.CreditExtra{}
	approvalIDForReturn := []uuid.UUID{}
	customerCode := []string{}
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
	mapCreditExtra := map[uuid.UUID]models.CreditExtra{}
	mapCreditDocref := map[string]models.Credit{}

	for _, creditValue := range GetCreditRes.(ResultCredit).Credit {
		mapCredit[creditValue.CustomerCode] = creditValue
		mapCreditDocref[creditValue.DocRef] = creditValue
		for _, creditExtraValue := range creditValue.CreditExtra {
			mapCreditExtra[creditValue.ID] = creditExtraValue
		}

	}
	creditIDForDelete := []uuid.UUID{}
	creditExtraIDForDelete := []uuid.UUID{}
	creditTransaction := []models.CreditTransaction{}

	for i, credit := range req {
		_, existMapCreditDocref := mapCreditDocref[req[i].DocRef]
		if !existMapCreditDocref {
			creditID := uuid.New()
			for o := range credit.CreditExtra {
				creditExtraID := uuid.New()
				req[i].CreditExtra[o].ID = creditExtraID
				if req[i].CreditExtra[o].CreditID == uuid.Nil {
					req[i].CreditExtra[o].CreditID = creditID
				}
				creditExtraValue = append(creditExtraValue, req[i].CreditExtra[o])

				creditmapCreditExtra, exist := mapCreditExtra[req[i].CreditExtra[o].CreditID]
				if exist {
					creditExtraIDForDelete = append(creditExtraIDForDelete, req[i].CreditExtra[o].CreditID)
					creditTransaction = append(creditTransaction, models.CreditTransaction{
						TransactionCode: creditmapCreditExtra.CreditID.String(),
						TransactionType: "EXTRA",
						Amount:          creditmapCreditExtra.Amount,
						AdjustAmount:    0,
						EffectiveDtm:    creditmapCreditExtra.EffectiveDtm,
						ExpireDtm:       creditmapCreditExtra.EffectiveDtm,
						IsApprove:       false,
						Status:          "INACTIVE",
						Reason:          "",
					})
				}

			}

			req[i].ID = creditID
			req[i].UpdateBy = userID
			req[i].CreateBy = userID
			approvalIDForReturn = append(approvalIDForReturn, creditID)
			req[i].CreditExtra = []models.CreditExtra{}

			if req[i].DocRef != "" {

				creditMapValue, exist := mapCredit[req[i].CustomerCode]
				if exist {
					req[i].ID = creditMapValue.ID
					creditIDForDelete = append(creditIDForDelete, creditMapValue.ID)
					creditTransaction = append(creditTransaction, models.CreditTransaction{
						TransactionCode: creditMapValue.CustomerCode,
						TransactionType: "BASE",
						Amount:          creditMapValue.Amount,
						AdjustAmount:    0,
						EffectiveDtm:    creditMapValue.EffectiveDtm,
						ExpireDtm:       creditMapValue.EffectiveDtm,
						IsApprove:       false,
						Status:          "INACTIVE",
						Reason:          "",
					})
				}
				creditValue = append(creditValue, req[i])
			}

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
	errCreateApproval := repositoryCredit.CreateCredit(creditValue, creditExtraValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"id":      approvalIDForReturn,
		"status":  "success",
		"message": "Approval create successfully",
	}, nil
}
