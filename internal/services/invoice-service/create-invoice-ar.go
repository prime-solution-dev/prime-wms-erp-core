package invoiceService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	interfaceService "prime-erp-core/internal/services/interface-service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateInvoiceAR(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	createInvoiceReturn, errCreateInvoice := CreateInvoice(ctx, jsonPayload)
	if errCreateInvoice != nil {
		return nil, errCreateInvoice
	}

	invoiceMap, _ := createInvoiceReturn.(map[string]interface{})
	idInvoice := invoiceMap["id"].([]uuid.UUID)
	requestData := map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"AP"},
		"sub_topic": []string{"CREATE"},
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
		HookInterfaceValue, err := interfaceService.HookInterface(requestDataCreateHook)
		if err != nil {
			return nil, err
		}
		if HookInterfaceValue != nil {
			str, _ := HookInterfaceValue.(string)

			invoiceValue := []models.Invoice{}
			invoiceValue = append(invoiceValue, models.Invoice{
				ID:         idInvoice[0],
				ExternalID: str,
			})

			_, errCreateApproval := repositoryInvoice.UpdateInvoice(invoiceValue, []models.InvoiceItem{})
			if errCreateApproval != nil {
				return nil, errCreateApproval
			}
		}
	}

	return createInvoiceReturn, nil

}
