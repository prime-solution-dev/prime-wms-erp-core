package invoiceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	models "prime-erp-core/internal/models"
	repositoryInvoice "prime-erp-core/internal/repositories/invoice"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UpdateInvoice(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	if len(req) == 0 {
		return nil, errors.New("invoice is required")
	}
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	conUserID, _ := ctx.Get("user")
	userID := ""
	if conUserID != nil {
		userID = conUserID.(string)
	}
	invoiceValue := []models.Invoice{}
	invoiceItemValue := []models.InvoiceItem{}
	invoiceId := make([]uuid.UUID, 0, len(req))
	seenInvoices := map[uuid.UUID]bool{}
	for i, invoice := range req {
		if invoice.ID == uuid.Nil || seenInvoices[invoice.ID] {
			return nil, errors.New("invoice IDs must be non-empty and unique")
		}
		seenInvoices[invoice.ID] = true
		var stored models.Invoice
		if err := gormx.Where("id = ?", invoice.ID).Take(&stored).Error; err != nil {
			return nil, err
		}
		var oldItems []models.InvoiceItem
		if err := gormx.Where("invoice_id = ?", invoice.ID).Find(&oldItems).Error; err != nil {
			return nil, err
		}
		numberedItems, err := assignInvoiceItemNumbers(oldItems, invoice.InvoiceItem)
		if err != nil {
			return nil, err
		}
		req[i].InvoiceItem = numberedItems
		invoiceId = append(invoiceId, invoice.ID)
		req[i].UpdateBy = userID
		for o := range invoice.InvoiceItem {
			invoiceItemID := uuid.New()
			req[i].InvoiceItem[o].ID = invoiceItemID
			req[i].InvoiceItem[o].InvoiceID = invoice.ID
			invoiceItemValue = append(invoiceItemValue, req[i].InvoiceItem[o])
		}
		req[i].InvoiceItem = []models.InvoiceItem{}
		req[i].InvoiceDeposit = []models.InvoiceDeposit{}
		invoiceValue = append(invoiceValue, req[i])
	}
	errDeleteInvoiceItem := repositoryInvoice.DeleteInvoiceItem(invoiceId)
	if errDeleteInvoiceItem != nil {
		return nil, errDeleteInvoiceItem
	}
	errCreateApproval := repositoryInvoice.CreateInvoice([]models.Invoice{}, invoiceItemValue, []models.InvoiceDeposit{})
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	rowsAffected, errCreateApproval := repositoryInvoice.UpdateInvoice(invoiceValue, []models.InvoiceItem{})
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

// Allocate new numbers above the maximum before any requested deletions.
func assignInvoiceItemNumbers(oldItems, requested []models.InvoiceItem) ([]models.InvoiceItem, error) {
	oldByID := make(map[uuid.UUID]models.InvoiceItem, len(oldItems))
	maxNumber := 0
	for _, item := range oldItems {
		oldByID[item.ID] = item
		n, err := strconv.Atoi(item.InvoiceItem)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid stored invoice_item: %q", item.InvoiceItem)
		}
		if n > maxNumber {
			maxNumber = n
		}
	}
	result := make([]models.InvoiceItem, 0, len(requested))
	seen := map[uuid.UUID]bool{}
	for _, item := range requested {
		if old, exists := oldByID[item.ID]; exists {
			if seen[item.ID] {
				return nil, fmt.Errorf("duplicate invoice item ID: %s", item.ID)
			}
			seen[item.ID] = true
			item.InvoiceItem = old.InvoiceItem
		} else if item.ID == uuid.Nil {
			if maxNumber == int(^uint(0)>>1) {
				return nil, errors.New("invoice item number overflow")
			}
			maxNumber++
			item.InvoiceItem = strconv.Itoa(maxNumber)
		} else {
			return nil, fmt.Errorf("invoice item not found in this invoice: %s", item.ID)
		}
		result = append(result, item)
	}
	return result, nil
}
