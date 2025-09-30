package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID           uuid.UUID `json:"id"`
	PaymentCode  string    `json:"payment_code"`
	CustomerCode string    `json:"customer_code"`
	PaymentDate  time.Time `json:"payment_date"`
	Amount       float64   `json:"amount"`
	Method       string    `json:"method"` // CASH, BANK_TRANSFER, CHEQUE
	Status       string    `json:"status"` // PENDING, COMPLETED, CANCELLED
	Remark       string    `json:"remark"`

	CreateBy       string           `json:"create_by"`
	CreateDtm      time.Time        `json:"create_dtm"`
	UpdateBy       string           `json:"update_by"`
	UpdateDate     time.Time        `json:"update_date"`
	PaymentInvoice []PaymentInvoice `json:"payment_invoice"`
}

func (Payment) TableName() string {
	return "payment"
}

type PaymentInvoice struct {
	ID          uuid.UUID `json:"id"`
	PaymentID   uuid.UUID `json:"payment_id"`
	InvoiceCode string    `json:"invoice_code"`
	Amount      float64   `json:"amount"`
	ApplyDate   time.Time `json:"apply_date"`

	CreateBy   string    `json:"create_by"`
	CreateDtm  time.Time `json:"create_dtm"`
	UpdateBy   string    `json:"update_by"`
	UpdateDate time.Time `json:"update_date"`
}

func (PaymentInvoice) TableName() string {
	return "payment_invoice"
}
