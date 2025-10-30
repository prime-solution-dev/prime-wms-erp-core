package depositService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryDeposit "prime-erp-core/internal/repositories/deposit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateDepost(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Deposit

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	depositValue := []models.Deposit{}
	depositIDForReturn := []uuid.UUID{}
	depositCode := []string{}
	for i := range req {
		depositID := uuid.New()
		req[i].ID = depositID

		depositIDForReturn = append(depositIDForReturn, depositID)

		if req[i].DepositCode == "" {
			req[i].DepositCode = uuid.New().String()
		}

		depositValue = append(depositValue, req[i])
	}
	for _, deposit := range req {
		depositCode = append(depositCode, deposit.DepositCode)
	}

	requestGetDeposit := map[string][]string{
		"deposit_code": depositCode,
	}
	jsonBytesGetDeposit, err := json.Marshal(requestGetDeposit)
	if err != nil {
		return nil, err
	}
	getDeposit, errWarehouse := GetDeposit(ctx, string(jsonBytesGetDeposit))
	if errWarehouse != nil {
		return nil, errWarehouse
	}
	resultDeposit := getDeposit.(ResultDeposit).Deposit
	if len(resultDeposit) > 0 {
		depositID := []uuid.UUID{}
		for _, resultDepositValue := range resultDeposit {
			depositID = append(depositID, resultDepositValue.ID)
		}
		errDeleteDeposit := repositoryDeposit.DeleteDeposit(depositID)
		if errDeleteDeposit != nil {
			return nil, errDeleteDeposit
		}
	}

	errCreateApproval := repositoryDeposit.CreateDeposit(depositValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"id":      depositIDForReturn,
		"status":  "success",
		"message": "Create Deposit Successfully",
	}, nil
}
