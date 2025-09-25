package models

import (
	"time"

	"github.com/google/uuid"
)

type CreditRequest struct {
	ID                           uuid.UUID  `json:"id"`
	RequestCode                  string     `json:"request_code"`
	CustomerCode                 string     `json:"customer_code"`
	CustomerName                 string     `gorm:"-" json:"customer_name"`
	TemporaryIncreaseCreditLimit float64    `json:"temporary_increase_credit_limit"`
	ConsumedCredit               float64    `gorm:"-" json:"consumed_credit"`
	BalanceCreditLimit           float64    `gorm:"-" json:"balance_credit_limit"`
	CustomeStatus                bool       `gorm:"-" json:"customer_status"`
	Amount                       float64    `json:"amount"`
	RequestType                  string     `json:"request_type"`
	Status                       string     `json:"status"`
	IsApprove                    bool       `json:"is_approve"`
	Reason                       string     `json:"reason"`
	EffectiveDtm                 *time.Time `json:"effective_dtm"`
	ExpireDtm                    *time.Time `json:"expire_dtm"`
	RequestDate                  *time.Time `json:"request_date"`
	ActionDate                   *time.Time `json:"action_date"`
	IsAction                     bool       `json:"is_action"`
	CreateBy                     string     `json:"create_by"`
	CreateDtm                    *time.Time `json:"create_dtm"`
	UpdateBy                     string     `json:"update_by"`
	UpdateDate                   *time.Time `json:"update_date"`
}

func (CreditRequest) TableName() string { return "credit_request" }

type Credit struct {
	ID                 uuid.UUID     `json:"id"`
	CustomerCode       string        `json:"customer_code"`
	Amount             float64       `json:"amount"`
	EffectiveDtm       *time.Time    `json:"effective_dtm"`
	IsActive           bool          `json:"is_active"`
	DocRef             string        `json:"doc_ref"`
	ApproveDate        *time.Time    `json:"approve_date"`
	AlertBalanceCredit bool          `json:"alert_balance_credit"`
	CreateBy           string        `json:"create_by"`
	CreateDtm          *time.Time    `json:"create_dtm"`
	UpdateBy           string        `json:"update_by"`
	UpdateDate         *time.Time    `json:"update_date"`
	CreditExtra        []CreditExtra `json:"credit_extra" gorm:"foreignKey:CreditID;references:ID"`
}

func (Credit) TableName() string { return "credit" }

type CreditExtra struct {
	ID           uuid.UUID  `json:"id"`
	CreditID     uuid.UUID  `json:"credit_id"`
	ExtraType    string     `json:"extra_type"`
	Amount       float64    `json:"amount"`
	EffectiveDtm *time.Time `json:"effective_dtm"`
	ExpireDtm    *time.Time `json:"expire_dtm"`
	DocRef       string     `json:"doc_ref"`
	ApproveDate  *time.Time `json:"approve_date"`
	CreateBy     string     `json:"create_by"`
	CreateDtm    *time.Time `json:"create_dtm"`
	UpdateBy     string     `json:"update_by"`
	UpdateDate   *time.Time `json:"update_date"`
}

func (CreditExtra) TableName() string { return "credit_extra" }

type CreditTransaction struct {
	ID              uuid.UUID  `json:"id"`
	TransactionCode string     `json:"transaction_code"`
	TransactionType string     `json:"transaction_type"`
	Amount          float64    `json:"amount"`
	AdjustAmount    float64    `json:"adjust_amount"`
	EffectiveDtm    *time.Time `json:"effective_dtm"`
	ExpireDtm       *time.Time `json:"expire_dtm"`
	ForceExpireDtm  *time.Time `json:"force_expire_dtm"`
	IsApprove       bool       `json:"is_approve"`
	Status          string     `json:"status"`
	Reason          string     `json:"reason"`
	ApproveDate     *time.Time `json:"approve_date"`
	CreateBy        string     `json:"create_by"`
	CreateDtm       *time.Time `json:"create_dtm"`
	UpdateBy        string     `json:"update_by"`
	UpdateDate      *time.Time `json:"update_date"`
}

func (CreditTransaction) TableName() string { return "credit_transaction" }
