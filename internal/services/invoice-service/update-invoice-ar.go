package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	depositService "prime-erp-core/internal/services/deposit-service"
	interfaceService "prime-erp-core/internal/services/interface-service"
	"strconv"

	"github.com/gin-gonic/gin"
)

func UpdateInvoiceAR(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	jsonBytesCreateInvoice, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	createInvoiceReturn, errCreateInvoice := UpdateInvoice(ctx, string(jsonBytesCreateInvoice))
	if errCreateInvoice != nil {
		return nil, errCreateInvoice
	}

	requestData := map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"AR"},
		"sub_topic": []string{"UPDATE"},
	}

	hookConfig, err := interfaceService.GetHookConfig(requestData)
	if err != nil {
		return nil, err
	}
	if len(hookConfig) > 0 {
		urlHook := ""
		for _, hookConfigValue := range hookConfig {
			urlHook = hookConfigValue.HookUrl
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: req,
			UrlHook:     urlHook,
		}
		_, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			return nil, err
		}
	}
	depositMapResult, err := interfaceService.GetDeposit(req[0].ExternalID)
	if err != nil {
		return nil, err
	}
	if len(depositMapResult) > 0 {
		var deposit []models.Deposit

		for _, v := range depositMapResult {
			depMap, _ := v.(map[string]interface{})

			totalFloat, err := strconv.ParseFloat(depMap["total"].(string), 64)
			if err != nil {
				totalFloat = 0
			}
			drFloat, err := strconv.ParseFloat(depMap["dr"].(string), 64)
			if err != nil {
				totalFloat = 0
			}
			crFloat, err := strconv.ParseFloat(depMap["cr"].(string), 64)
			if err != nil {
				totalFloat = 0
			}

			deposit = append(deposit, models.Deposit{
				DepositCode:  depMap["anchor"].(string),
				CustomerCode: req[0].PartyCode,
				AmountTotal:  totalFloat,
				AmountUsed:   drFloat,
				AmountRemain: crFloat,
				Status:       "PENDING",
			})
		}
		if len(deposit) > 0 {
			jsonBytesCreateDeposit, err := json.Marshal(deposit)
			if err != nil {
				return nil, err
			}

			_, errDeposit := depositService.CreateDepost(ctx, string(jsonBytesCreateDeposit))
			if errDeposit != nil {
				return nil, errDeposit
			}
		}

	}
	return createInvoiceReturn, nil

}
