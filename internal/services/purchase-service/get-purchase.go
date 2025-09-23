package purchaseService

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"prime-erp-core/internal/models"
	purchaseRepository "prime-erp-core/internal/repositories/purchase"

	"github.com/gin-gonic/gin"
)

func GetPOBigLot(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := models.GetPOBigLotListRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	prePurchaseList, total, page, pageSize, totalPage, err := purchaseRepository.GetPOBigLotList(req.CompanyCode, req.SiteCode, req.Page, req.PageSize)
	if err != nil {
		return nil, errors.New("failed to get big lot list: " + err.Error())
	}

	supplierReq := models.GetSupplierListRequest{}
	for _, prePurchase := range prePurchaseList {
		supplierReq.SupplierCodes = append(supplierReq.SupplierCodes, prePurchase.SupplierCode)
	}

	jsonData, err := json.Marshal(supplierReq)
	if err != nil {
		return nil, errors.New("failed to marshal supplier data to JSON: " + err.Error())
	}

	getSuppliers, err := http.NewRequest("POST", os.Getenv("base_url_supplier")+"/get-suppliers", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.New("failed to create HTTP request: " + err.Error())
	}

	getSuppliers.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(getSuppliers)
	if err != nil {
		return nil, errors.New("failed to execute HTTP request: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("received non-OK HTTP status: " + resp.Status)
	}

	supplierBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response body: " + err.Error())
	}

	supplierResponse := models.GetSupplierListResponse{}
	if err := json.Unmarshal(supplierBody, &supplierResponse); err != nil {
		return nil, errors.New("failed to decode JSON response: " + err.Error())
	}

	mapSupplier := map[string]models.Supplier{}
	for _, suppliers := range supplierResponse.Supplier {
		mapSupplier[suppliers.SupplierCode] = suppliers
	}

	result := models.GetPOBigLotListResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPage,
	}

	for _, prePurchase := range prePurchaseList {
		bigLotResponse := MapPrePurchasesModelToBigLotsResponse(prePurchase)

		bigLotResponse.SupplierName = mapSupplier[prePurchase.SupplierCode].SupplierName

		result.BigLotList = append(result.BigLotList, bigLotResponse)
	}

	return result, nil
}
