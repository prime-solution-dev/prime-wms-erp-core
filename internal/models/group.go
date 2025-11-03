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
	ValueInt   int         `json:"value_int"`
	Seq        int         `json:"seq"`
	CreateDtm  string      `json:"create_dtm"`
	UpdateBy   time.Time   `json:"update_by"`
	UpdateDtm  string      `json:"update_dtm"`
	CreateBy   time.Time   `json:"create_by"`
	GroupItems []GroupItem `gorm:"foreignKey:GroupID;references:ID" json:"group_items"`
}

func (Group) TableName() string {
	return "group"
}

type GroupItem struct {
	ID        uuid.UUID `json:"id"`
	ItemCode  string    `json:"item_code"`
	GroupID   uuid.UUID `json:"group_id"`
	ItemName  string    `json:"item_name"`
	Value     string    `json:"value"`
	ValueInt  int       `json:"value_int"`
	CreateDtm string    `json:"create_dtm"`
	UpdateBy  time.Time `json:"update_by"`
	UpdateDtm string    `json:"update_dtm"`
	CreateBy  time.Time `json:"create_by"`
}

func (GroupItem) TableName() string {
	return "group_item"
}

type GetGroupRequest struct {
	GroupCodes []string `json:"group_codes"`
	ItemCodes  []string `json:"item_codes"`
}

type GetGroupResponse struct {
	Group
	Items []GroupItem `json:"items"`
}
