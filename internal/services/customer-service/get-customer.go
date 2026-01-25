package customerService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type CustomerGroup struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	GroupCode  string    `json:"group_code"`
	GroupValue string    `json:"group_value"`
	ActiveFlg  bool      `json:"active_flg"`
	CreateDtm  time.Time `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
}
type GetCustomerAddressResponse struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	AddressCode  string    `gorm:"type:varchar(50)" json:"address_code"`
	CustomerCode string    `gorm:"type:varchar(50)" json:"customer_code"`
	Address      string    `gorm:"type:varchar(50)" json:"address"`
	Province     string    `gorm:"type:varchar(50)" json:"province"`
	District     string    `gorm:"type:varchar(50)" json:"district"`
	SubDistrict  string    `gorm:"type:varchar(50)" json:"sub_district"`
	PostCode     string    `gorm:"type:varchar(50)" json:"post_code"`
	Latitude     string    `gorm:"type:varchar(50)" json:"latitude"`
	Longitude    string    `gorm:"type:varchar(50)" json:"longitude"`
	Remark       string    `gorm:"type:varchar(50)" json:"remark"`
	CreateBy     string    `gorm:"type:varchar(50)" json:"create_by"`
	CreateDate   time.Time `gorm:"type:timestamp" json:"create_date"`
	UpdateBy     string    `gorm:"type:varchar(50)" json:"update_by"`
	UpdateDate   time.Time `gorm:"type:timestamp" json:"update_date"`
}
type GetCustomerResponse struct {
	ID                                uuid.UUID            `gorm:"type:uuid;primary_key" json:"id"`
	CustomerCode                      string               `gorm:"type:varchar(50)" json:"customer_code"`
	CustomerType                      string               `gorm:"type:varchar(50)" json:"customer_type"`
	CustomerName                      string               `gorm:"type:varchar(50)" json:"customer_name"`
	CreateBy                          string               `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm                         time.Time            `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
	UpdateBy                          string               `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDTM                         time.Time            `gorm:"autoUpdateTime;<-" json:"update_dtm"`
	IsInternalCustomer                bool                 `json:"is_internal_customer"`
	ConsignmentLogicalWarehouse       string               `json:"consignment_logical_warehouse"`
	DefaultWarehousePickingAllocation string               `json:"default_warehouse_picking_allocation"`
	MinimumShelfLife                  int                  `json:"minimum_shelf_life"`
	MinimumShelfLifeFoc               int                  `json:"minimum_shelf_life_foc"`
	IsNeedNextExp                     bool                 `json:"is_need_next_exp"`
	BatchCannotReverse                bool                 `json:"batch_cannot_reverse"`
	IsCreditHold                      bool                 `json:"is_credit_hold"`
	CreditTerm                        int                  `json:"credit_term"`
	SalesContactName                  string               `json:"sales_contact_name"`
	Phone                             string               `json:"phone"`
	Email                             string               `json:"email"`
	ActiveFlg                         bool                 `json:"active_flg"`
	ExternalID                        string               `gorm:"<-:create" json:"external_id"`
	BranchName                        string               `json:"branch_name"`
	CustomerGroup                     []CustomerGroup      `gorm:"foreignKey:CustomerID;references:ID" json:"customer_group"`
	Address                           string               `gorm:"-" json:"address"`
	TaxID                             string               `gorm:"-" json:"tax_id"`
	PayerTerm                         string               `gorm:"type:varchar(50)" json:"payer_term"`
	Sold                              []GetSoldResponse    `gorm:"foreignKey:CustomerID;references:ID" json:"sold"`
	Billing                           []GetBillingResponse `gorm:"foreignKey:CustomerID;references:ID" json:"billing"`
}
type GetBillingResponse struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	CustomerCode string    `gorm:"type:varchar(50);<-:select" json:"customer_code"`
	CustomerName string    `gorm:"type:varchar(50);<-:select" json:"customer_name"`
	BillingCode  string    `gorm:"type:varchar(50)" json:"billing_code"`
	FirstName    string    `gorm:"type:varchar(50)" json:"first_name"`
	LastName     string    `gorm:"type:varchar(50)" json:"last_name"`
	IDCard       string    `gorm:"type:varchar(50)" json:"id_card"`
	Address      string    `gorm:"type:varchar(50)" json:"address"`
	Province     string    `gorm:"type:varchar(50)" json:"province"`
	District     string    `gorm:"type:varchar(50)" json:"district"`
	SubDistrict  string    `gorm:"type:varchar(50)" json:"sub_district"`
	PostCode     string    `json:"post_code"`
	Name         string    `json:"name"`
	TaxID        string    `json:"tax_id"`
	Country      string    `json:"country"`
	BranchID     string    `json:"branch_id"`
	Latitude     string    `gorm:"type:varchar(MAX)" json:"latitude"`
	Longtitude   string    `gorm:"type:varchar(MAX)" json:"longtitude"`
	Remark       string    `gorm:"type:varchar(50)" json:"remark"`
	Contact      string    `gorm:"type:varchar(50)" json:"contact"`
	Phone        string    `gorm:"type:varchar(50)" json:"phone"`
	Email        string    `gorm:"type:varchar(50)" json:"email"`
	CreateBy     string    `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm    time.Time `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
	UpdateBy     string    `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDTM    time.Time `gorm:"autoUpdateTime;<-" json:"update_dtm"`
	ActiveFlg    bool      `json:"active_flg"`
	SameAsSoldTo bool      `json:"same_as_sold_to"`
}
type GetSoldResponse struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	CustomerCode string    `gorm:"type:varchar(50);<-:select" json:"customer_code"`
	CustomerName string    `gorm:"type:varchar(50);<-:select" json:"customer_name"`
	SoldCode     string    `gorm:"type:varchar(50)" json:"sold_code"`
	Name         string    `json:"name"`
	FirstName    string    `gorm:"type:varchar(50)" json:"first_name"`
	LastName     string    `gorm:"type:varchar(50)" json:"last_name"`
	IDCard       string    `gorm:"type:varchar(50)" json:"id_card"`
	TaxID        string    `json:"tax_id"`
	Address      string    `gorm:"type:varchar(50)" json:"address"`
	Province     string    `gorm:"type:varchar(50)" json:"province"`
	District     string    `gorm:"type:varchar(50)" json:"district"`
	SubDistrict  string    `gorm:"type:varchar(50)" json:"sub_district"`
	PostCode     string    `gorm:"type:varchar(50)" json:"post_code"`
	Latitude     string    `gorm:"type:varchar(MAX)" json:"latitude"`
	Longtitude   string    `gorm:"type:varchar(MAX)" json:"longtitude"`
	Remark       string    `gorm:"type:varchar(50)" json:"remark"`
	Contact      string    `gorm:"type:varchar(50)" json:"contact"`
	Phone        string    `gorm:"type:varchar(50)" json:"phone"`
	Email        string    `gorm:"type:varchar(50)" json:"email"`
	Country      string    `json:"country"`
	BranchID     string    `json:"branch_id"`
	CreateBy     string    `gorm:"type:varchar(100)" json:"create_by"`
	CreateDtm    time.Time `gorm:"autoCreateTime;<-:create" json:"create_dtm"`
	UpdateBy     string    `gorm:"type:varchar(100)" json:"update_by"`
	UpdateDTM    time.Time `gorm:"autoUpdateTime;<-" json:"update_dtm"`
	ActiveFlg    bool      `json:"active_flg"`
}
type ResultCustomerResponse struct {
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
	Customers  []GetCustomerResponse `json:"customers"`
}

func GetCustomers(requestData map[string]interface{}) (ResultCustomerResponse, error) {

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		errors.New("Error marshalling data :")
	}

	reqHttp, err := http.NewRequest("POST", os.Getenv("base_url_customer")+"/Customer/GetCustomers", bytes.NewBuffer(jsonData))
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}

	reqHttp.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(reqHttp)
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error parsing DateTo : " + err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}
	var customers ResultCustomerResponse
	err = json.Unmarshal(body, &customers)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	fmt.Println("Response Status:", resp.Status)

	return customers, nil

}
