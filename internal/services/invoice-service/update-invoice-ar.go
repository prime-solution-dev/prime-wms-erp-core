package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	interfaceService "prime-erp-core/internal/services/interface-service"

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
	return createInvoiceReturn, nil

}
