package models

import (
	"time"

	"github.com/google/uuid"
)

type Deposit struct {
	ID           uuid.UUID  `json:"id"`
	DepositCode  string     `json:"deposit_code"`
	DocRefType   string     `json:"doc_ref_type"`
	DocRef       string     `json:"doc_ref"`
	CustomerCode string     `json:"customer_code"`
	DepositDate  *time.Time `json:"deposit_date"`
	AmountTotal  float64    `json:"amount_total"`
	AmountUsed   float64    `json:"amount_used"`
	AmountRemain float64    `json:"amount_remain"`
	Status       string     `json:"status"`
	Remark       string     `json:"remark"`
	CreateBy     string     `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm    time.Time  `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
	UpdateBy     string     `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDate   time.Time  `gorm:"autoUpdateTime;<-" json:"update_date"`
}

func (Deposit) TableName() string { return "deposit" }
