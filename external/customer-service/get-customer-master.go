package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"prime-erp-core/config"
	"time"

	"github.com/google/uuid"
)

type GetCustomerRequest struct {
	Customers        []string `json:"customers"`
	CustomerCodeLike string   `json:"customer_code_like"`
	CustomerNameLike string   `json:"customer_name_like"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
}

type GetCustomerResponse struct {
	ID           uuid.UUID                    `gorm:"type:uuid;primary_key" json:"id"`
	CustomerCode string                       `gorm:"type:varchar(50)" json:"customer_code"`
	CustomerType string                       `gorm:"type:varchar(50)" json:"customer_type"`
	CustomerName string                       `gorm:"type:varchar(50)" json:"customer_name"`
	CreateBy     string                       `gorm:"type:varchar(50)" json:"create_by"`
	CreateDate   time.Time                    `gorm:"type:timestamp" json:"create_date"`
	UpdateBy     string                       `gorm:"type:varchar(50)" json:"update_by"`
	UpdateDate   time.Time                    `gorm:"type:timestamp" json:"update_date"`
	Address      []GetCustomerAddressResponse `gorm:"foreignKey:CustomerCode;references:CustomerCode" json:"address"`
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

type ResultCustomerResponse struct {
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
	Customers  []GetCustomerResponse `json:"customers"`
}

func GetCustomer(jsonPayload GetCustomerRequest) (ResultCustomerResponse, error) {

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error marshaling struct to JSON:")
	}
	reqProduct, err := http.NewRequest("POST", config.GET_CUSTOMER_MASTER_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}

	reqProduct.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(reqProduct)
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error parsing DateTo: " + err.Error())
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
	return customers, nil
}
