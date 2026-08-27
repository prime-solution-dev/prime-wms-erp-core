package externalService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		return ResultCustomerResponse{}, errors.New("Error marshaling struct to JSON: " + err.Error())
	}

	req, err := http.NewRequest("POST", config.GET_CUSTOMER_MASTER_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error creating request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// timeout กันปลายทางค้างแล้วลาก request ของเราค้างตาม (default ของ http.Client คือไม่มี timeout)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error sending request: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResultCustomerResponse{}, errors.New("Error reading response body: " + err.Error())
	}

	// เดิมข้ามการเช็ค status ทำให้ปลายทางตอบ 500 แล้วเราคืน struct เปล่ากับ error = nil
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return ResultCustomerResponse{}, errors.New(errResp.Error)
		}
		return ResultCustomerResponse{}, fmt.Errorf("API error status %d: %s", resp.StatusCode, string(body))
	}

	var dataRes ResultCustomerResponse
	if err = json.Unmarshal(body, &dataRes); err != nil {
		return ResultCustomerResponse{}, errors.New("Error parsing response: " + err.Error())
	}

	return dataRes, nil
}
