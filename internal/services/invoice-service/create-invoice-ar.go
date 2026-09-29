package invoiceService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	models "prime-erp-core/internal/models"
	customerService "prime-erp-core/internal/services/customer-service"
	interfaceService "prime-erp-core/internal/services/interface-service"
	purchaseService "prime-erp-core/internal/services/purchase-service"
	systemConfigService "prime-erp-core/internal/services/system-config"
	"slices"
)

func CreateInvoiceAR(ctx context.Context, jsonPayload string) (interface{}, error) {

	var req []models.Invoice

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	customerCode := []string{}

	for _, reqValue := range req {
		customerCode = append(customerCode, reqValue.PartyCode)
	}

	requestDataGetCustomers := map[string]interface{}{
		"customer_code": customerCode,
	}

	customers, err := customerService.GetCustomers(ctx, requestDataGetCustomers)
	if err != nil {
		return nil, err
	}

	convertCustomerMap := map[string]customerService.GetCustomerResponse{}
	for _, customer := range customers.Customers {
		convertCustomerMap[customer.CustomerCode] = customer
	}
	prefix := "IV"
	if req[0].PaymentMethod == "CASH" {
		prefix = "CS"
	}
	if req[0].PaymentMethod == "CREDIT" {
		prefix = "IV"
	}
	configCodeValue := "RUNNING_AR"
	count := 0
	for i := range req {
		if req[i].InvoiceCode == "" {
			count++
		}
	}
	var purchaseCodes []string
	if count > 0 {
		purchaseCodes, err = GenerateInvoiceCodes(ctx, count, prefix, configCodeValue)
		if err != nil {
			return nil, errors.New("failed to generate invoice codes: " + err.Error())
		}
	}

	//depositCut := []models.Deposit{}
	productCodes := []string{}
	codeIndex := 0
	for i := range req {
		conMapCustomer, exist := convertCustomerMap[req[i].PartyCode]
		if exist {
			req[i].PartyName = conMapCustomer.CustomerName
			for _, soldValue := range conMapCustomer.Billing {
				req[i].PartyBranch = soldValue.BranchID
				req[i].PartyAddress = conMapCustomer.Address
			}
			req[i].PartyEmail = conMapCustomer.Email
			req[i].PartyTel = conMapCustomer.Phone
			req[i].PartyTaxID = conMapCustomer.TaxID
			req[i].PartyExternalID = conMapCustomer.ExternalID
		}
		if req[i].InvoiceCode == "" {
			req[i].InvoiceCode = purchaseCodes[codeIndex]
			codeIndex++
		}
		/* for it := range req[i].InvoiceItem {
			if req[i].InvoiceItem[it].ArticleType == "DEPOSIT" {
				depositCut = append(depositCut, models.Deposit{
					DepositCode: req[i].InvoiceItem[it].DocumentRef,
					AmountUsed:  req[i].InvoiceItem[it].SubtotalExclVat,
				})
			}

		} */
		for it := range req[i].InvoiceItem {
			productCodes = append(productCodes, req[i].InvoiceItem[it].ProductCode)
		}
	}

	requestData := map[string]interface{}{
		"module":    []string{"INVOICE"},
		"topic":     []string{"AR"},
		"sub_topic": []string{"CREATE"},
	}

	hookConfig, err := interfaceService.GetHookConfig(ctx, requestData)
	if err != nil {
		return nil, err
	}
	if len(hookConfig) > 0 && req[0].Status != "TEMP" {
		urlHook := ""
		for _, hookConfigValue := range hookConfig {
			urlHook = hookConfigValue.HookUrl
		}

		productReq := models.GetProductRequest{
			ProductCode: productCodes,
			SiteCode:    []string{req[0].SiteCode},
			CompanyCode: []string{req[0].CompanyCode},
		}
		mapProductInterface, errGetProductInterface := purchaseService.GetProductInterface(ctx, productReq)
		if errGetProductInterface != nil {
			return nil, errors.New("failed to get product interface: " + errGetProductInterface.Error())
		}
		reqHook := slices.Clone(req)
		for i := range reqHook {
			reqHook[i].InvoiceItem = slices.Clone(req[i].InvoiceItem)
			for it := range reqHook[i].InvoiceItem {
				mapProductInterface, exists := mapProductInterface[reqHook[i].InvoiceItem[it].ProductCode]
				if exists {
					priceUnit, _ := calculateAPPriceUnit(
						reqHook[i].InvoiceItem[it].UnitUom, mapProductInterface.UnitInterface,
						reqHook[i].InvoiceItem[it].PriceUnit, reqHook[i].InvoiceItem[it].Qty, reqHook[i].InvoiceItem[it].TotalWeight,
					)
					reqHook[i].InvoiceItem[it].PriceUnit = math.Round(priceUnit*100) / 100
					reqHook[i].InvoiceItem[it].UnitUom = mapProductInterface.UnitInterface
				}
			}
		}

		requestDataCreateHook := interfaceService.HookInterfaceRequest{
			RequestData: reqHook,
			UrlHook:     urlHook,
		}
		HookInterfaceValue, err := interfaceService.HookInterface(ctx, requestDataCreateHook)
		if err != nil {
			if req[0].Status == "COMPLETED" {
				req[0].Status = "TEMP"
				jsonBytesCreateInvoice, err := json.Marshal(req)
				if err != nil {
					return nil, err
				}
				_, errCreateInvoice := CreateInvoice(ctx, string(jsonBytesCreateInvoice))
				if errCreateInvoice != nil {
					return nil, errCreateInvoice
				}
			}
			return nil, err
		}
		if HookInterfaceValue != nil {
			externalID := HookInterfaceValue.(map[string]interface{})
			str, _ := externalID["id"].(string)
			req[0].ExternalID = str
			jsonBytesCreateInvoice, err := json.Marshal(req)
			if err != nil {
				return nil, err
			}

			createInvoiceReturn, errCreateInvoice := CreateInvoice(ctx, string(jsonBytesCreateInvoice))
			if errCreateInvoice != nil {
				return nil, errCreateInvoice
			}

			return createInvoiceReturn, nil
		}
	} else {
		jsonBytesCreateInvoice, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		createInvoiceReturn, errCreateInvoice := CreateInvoice(ctx, string(jsonBytesCreateInvoice))
		if errCreateInvoice != nil {
			return nil, errCreateInvoice
		}
		return createInvoiceReturn, nil
	}

	return nil, nil
}
// GenerateInvoiceCodes จองเลขที่เอกสารแบบ atomic ให้ invoice (AR/AP/CN/DN)
//
// เดิมเรียก systemConfigService.GetRunningSystemConfigInvoice (SELECT เฉยๆ ไม่มี lock)
// แล้วค่อยเรียก UpdateRunningSystemConfigInvoice ทีหลัง คนละ transaction — สองคนกดพร้อมกัน
// ได้เลขซ้ำ ทั้งสองฟังก์ชันนั้นยังรับ gin's *Context (ไม่ได้แปลงและอยู่นอก scope งานนี้)
// ตอนนี้ลบทั้งคู่ทิ้งแล้ว (ไม่มี caller/route เหลือ) — ย้ายมาใช้ ReserveRunningCodes +
// InvoiceRunningPeriod ที่ระบบมีอยู่แล้ว (system-config/reserve-running-code.go) ซึ่ง
// sale/delivery/quotation-service ใช้แบบเดียวกันนี้มาก่อนแล้วสำหรับ RUNNING_SO/RUNNING_DBS/
// RUNNING_QU — ล็อกแถว config ด้วย SELECT ... FOR UPDATE จนกว่าจะเขียน current_running เสร็จ
// ปิดช่องเลขซ้ำไปในตัว InvoiceRunningPeriod คำนวณปี พ.ศ. 2 หลัก (ยกเว้น RUNNING_AP ที่ใช้
// ค.ศ.) ตรงกับ GetRunningSystemConfigInvoice/UpdateRunningSystemConfigInvoice เดิมทุกประการ
//
// prefix ที่ส่งเข้ามา (เช่น "IV"/"CS" สลับกันตาม payment_method) ใช้ประกอบเลขของรอบนี้
// เท่านั้น — ReserveRunningCodes ไม่เขียน prefix นี้ทับค่าที่เก็บอยู่ใน system_config row
func GenerateInvoiceCodes(ctx context.Context, count int, prefix string, configCodeValue string) ([]string, error) {
	if count <= 0 {
		return []string{}, nil // No purchases to generate codes for
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		// ข้อความเดิมของ GetRunningSystemConfigInvoice ตอน ConnectGORM ล้มเหลว คือสตริงตายตัว
		// "failed to connect to database" (เขียนผ่าน ctx.JSON ตรงๆ) ไม่ใช่ err ดิบ — คง
		// ข้อความเดิมไว้ ไม่ต่อท้าย driver error กันข้อมูลภายในหลุดออกไปหา client
		return nil, errors.New("failed to connect to database")
	}
	defer db.CloseGORM(gormx)

	codes, err := systemConfigService.ReserveRunningCodes(
		gormx, configCodeValue, count, prefix, systemConfigService.InvoiceRunningPeriod(configCodeValue))
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice codes: %v", err)
	}

	if len(codes) != count {
		return nil, errors.New("failed to get correct number of purchase order codes from system config")
	}

	return codes, nil
}
