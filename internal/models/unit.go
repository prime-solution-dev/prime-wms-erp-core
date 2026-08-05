package models

import "github.com/google/uuid"

type Unit struct {
	ID              uuid.UUID    `json:"id"`
	Topic           string       `json:"topic"`     // PO, SO
	UnitCode        string       `json:"unit_code"` // PC, KG
	UnitName        string       `json:"unit_name"`
	UnitMethodItems []UnitMethod `gorm:"foreignKey:UnitID;references:ID" json:"unit_method_items"`
}

func (Unit) TableName() string {
	return "unit"
}

type UnitMethod struct {
	ID           uuid.UUID `json:"id"`
	MethodCode   string    `json:"method_code"` // PC, KG, KG-Spec
	UnitID       uuid.UUID `json:"unit_id"`
	MethodName   string    `json:"method_name"`
	UnitUomItems []UnitUom `gorm:"foreignKey:MethodID;reference:ID" json:"unit_uom_items"`
}

func (UnitMethod) TableName() string {
	return "unit_method"
}

type UnitUom struct {
	ID       uuid.UUID `json:"id"`
	UomCode  string    `json:"uom_code"` // PC, KG
	MethodID uuid.UUID `json:"method_id"`
	UomName  string    `json:"uom_name"`
}

func (UnitUom) TableName() string {
	return "unit_uom"
}

// DTOs
type GetUnitUomResponse struct {
	ID      uuid.UUID `json:"id"`
	UomCode string    `json:"uom_code"` // PC, KG
	UomName string    `json:"uom_name"`
}

type GetUnitMethodResponse struct {
	ID           uuid.UUID            `json:"id"`
	MethodCode   string               `json:"method_code"` // PC, KG, KG-Spec
	MethodName   string               `json:"method_name"`
	UnitUomItems []GetUnitUomResponse `json:"unit_uom_items"`
}

type GetAllUnitResponse struct {
	ID              string                  `json:"id"`
	Topic           string                  `json:"topic"`     // PO, SO
	UnitCode        string                  `json:"unit_code"` // PC, KG
	UnitName        string                  `json:"unit_name"`
	UnitMethodItems []GetUnitMethodResponse `json:"unit_method_items"`
}
