package paymentService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositorypayment "prime-erp-core/internal/repositories/payment"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreatePayment(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Payment

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}
	paymentValue := []models.Payment{}
	paymentInvoiceValue := []models.PaymentInvoice{}
	paymentIDForReturn := []uuid.UUID{}
	for i, payment := range req {
		paymentID := uuid.New()
		req[i].ID = paymentID

		paymentIDForReturn = append(paymentIDForReturn, paymentID)

		if req[i].PaymentCode == "" {
			req[i].PaymentCode = uuid.New().String()
		}

		for o := range payment.PaymentInvoice {
			paymentItemID := uuid.New()
			req[i].PaymentInvoice[o].ID = paymentItemID
			req[i].PaymentInvoice[o].PaymentID = paymentID
			paymentInvoiceValue = append(paymentInvoiceValue, req[i].PaymentInvoice[o])
		}

		req[i].PaymentInvoice = []models.PaymentInvoice{}
		paymentValue = append(paymentValue, req[i])
	}

	errCreateApproval := repositorypayment.CreatePayment(paymentValue, paymentInvoiceValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"id":      paymentIDForReturn,
		"status":  "success",
		"message": "Create payment Successfully",
	}, nil
}
