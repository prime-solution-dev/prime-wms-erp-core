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
	mapCreditDocref := map[string]models.Credit{}

	for _, creditValue := range GetCreditRes.(ResultCredit).Credit {
		mapCredit[creditValue.CustomerCode] = creditValue
		mapCreditDocref[creditValue.DocRef] = creditValue
	}
	creditIDForDelete := []uuid.UUID{}
	creditTransaction := []models.CreditTransaction{}

	for i, credit := range req {
		creditID := uuid.New()
		for o := range credit.CreditExtra {
			creditExtraID := uuid.New()
			req[i].CreditExtra[o].ID = creditExtraID
			req[i].CreditExtra[o].CreditID = creditID
			creditExtraValue = append(creditExtraValue, req[i].CreditExtra[o])
		}

		req[i].ID = creditID
		approvalIDForReturn = append(approvalIDForReturn, creditID)
		req[i].CreditExtra = []models.CreditExtra{}

		_, existMapCreditDocref := mapCreditDocref[req[i].DocRef]
		if !existMapCreditDocref {
			creditValue = append(creditValue, req[i])
			creditMapValue, exist := mapCredit[req[i].CustomerCode]
			if exist {
				creditIDForDelete = append(creditIDForDelete, creditMapValue.ID)
				creditTransaction = append(creditTransaction, models.CreditTransaction{
					TransactionCode: creditMapValue.CustomerCode,
					TransactionType: "BASE",
					Amount:          req[i].Amount,
					AdjustAmount:    0,
					EffectiveDtm:    req[i].EffectiveDtm,
					ExpireDtm:       req[i].EffectiveDtm,
					/* ForceExpireDtm:  time.Now(),
					ApproveDate:     "",  */
					IsApprove: false,
					Status:    "INACTIVE",
					Reason:    "",
				})
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

	if len(creditIDForDelete) > 0 {
		errDeleteCredit := repositoryCredit.DeleteCredit(creditIDForDelete)
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
