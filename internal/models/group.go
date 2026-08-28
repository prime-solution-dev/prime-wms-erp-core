package models

import (
	"time"

	"github.com/google/uuid"
)

type Group struct {
	ID         uuid.UUID   `json:"id"`
	GroupCode  string      `json:"group_code"`
	GroupName  string      `json:"group_name"`
	Value      string      `json:"value"`
	ValueInt   float64     `json:"value_int"`
	Seq        int         `json:"seq"`
	CreateDtm  time.Time   `json:"create_dtm"`
	UpdateBy   string      `json:"update_by"`
	UpdateDtm  time.Time   `json:"update_dtm"`
	CreateBy   string      `json:"create_by"`
	GroupItems []GroupItem `gorm:"foreignKey:GroupID;references:ID" json:"group_items"`
}

func (Group) TableName() string {
	return "group"
}

type GroupItem struct {
	ID       uuid.UUID `json:"id"`
	ItemCode string    `json:"item_code"`
	GroupID  uuid.UUID `json:"group_id"`
	ItemName string    `json:"item_name"`
	Value    string    `json:"value"`
	// ParentGroupCode / ParentGroupItemCode มิเรอร์สาย parent ข้ามกลุ่มจากฝั่ง WMS
	// เก็บเป็น code ไม่ใช่ uuid เพราะ WMS ลบ-สร้าง item ใหม่ทุกครั้งที่ save
	ParentGroupCode     *string   `json:"parent_group_code"`
	ParentGroupItemCode *string   `json:"parent_group_item_code"`
	ValueInt            float64   `json:"value_int"`
	CreateDtm           time.Time `json:"create_dtm"`
	UpdateBy            string    `json:"update_by"`
	UpdateDtm           time.Time `json:"update_dtm"`
	CreateBy            string    `json:"create_by"`
}

func (GroupItem) TableName() string {
	return "group_item"
}

type GetGroupRequest struct {
	GroupCodes []string `json:"group_codes"`
	ItemCodes  []string `json:"item_codes"`
}

type GetGroupItemResponse struct {
	ID        string  `json:"id"`
	ItemCode  string  `json:"item_code"`
	GroupID   string  `json:"group_id"`
	ItemName  string  `json:"item_name"`
	Value     string  `json:"value"`
	ValueInt  float64 `json:"value_int"`
	CreateDtm string  `json:"create_dtm"`
	UpdateBy  string  `json:"update_by"`
	UpdateDtm string  `json:"update_dtm"`
	CreateBy  string  `json:"create_by"`
}

type GetGroupResponse struct {
	ID        string                 `json:"id"`
	GroupCode string                 `json:"group_code"`
	GroupName string                 `json:"group_name"`
	Value     string                 `json:"value"`
	ValueInt  float64                `json:"value_int"`
	Seq       int                    `json:"seq"`
	CreateDtm string                 `json:"create_dtm"`
	UpdateBy  string                 `json:"update_by"`
	UpdateDtm string                 `json:"update_dtm"`
	CreateBy  string                 `json:"create_by"`
	Items     []GetGroupItemResponse `json:"items"`
}
