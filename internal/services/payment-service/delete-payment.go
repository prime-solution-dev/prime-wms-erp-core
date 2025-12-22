package paymentService

import (
	"encoding/json"
	"errors"
	repositorypayment "prime-erp-core/internal/repositories/payment"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DeletePaymentReq struct {
	InvoiceCode []string `json:"invoice_code"`
}

func DeletePayment(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req DeletePaymentReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	requestDataGetPayment := map[string]interface{}{
		"invoice_code": req.InvoiceCode,
	}

	jsonBytesPayment, err := json.Marshal(requestDataGetPayment)
	if err != nil {
		return nil, err
	}

	paymentle, errGetPayment := GetPayment(ctx, string(jsonBytesPayment))
	if errGetPayment != nil {
		return nil, errGetPayment
	}
	resultPayment := paymentle.(ResultPayment).Payment
	paymentID := []uuid.UUID{}

	for _, paymentValue := range resultPayment {
		paymentID = append(paymentID, paymentValue.ID)
	}

	errCreateApproval := repositorypayment.DeletePayment(paymentID, req.InvoiceCode)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Delete Payment Successfully",
	}, nil
}
