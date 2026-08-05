package creditService

import (
	"encoding/json"
	"errors"
	"fmt"
	"prime-erp-core/internal/db"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type GetCreditRequest struct {
	CustomerCodes []string `json:"customer_codes"`
}

type GetCreditResponse struct {
	CreditCustomers []CreditCustomer `json:"credit_customers"`
}

type CreditCustomer struct {
	CustomerCode  string  `json:"customer_code"`
	IsActive      bool    `json:"is_active"`
	Credit        float64 `json:"credit"`
	Extra         float64 `json:"extra"`
	RemainDeposit float64 `json:"remain_deposit"`
	Used          float64 `json:"used"`
	Balance       float64 `json:"balance"`
}

func GetCreditCurrentAPI(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := GetCreditRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	if len(req.CustomerCodes) == 0 {
		return nil, fmt.Errorf("require at least one customer code")
	}

	return GetCreditCurrent(sqlx, req)
}

func GetCreditCurrent(sqlx *sqlx.DB, req GetCreditRequest) (*GetCreditResponse, error) {
	res := GetCreditResponse{}
	customerCheck := map[string]bool{}
	customerStrs := []string{}

	for _, customer := range req.CustomerCodes {
		if !customerCheck[customer] {
			res.CreditCustomers = append(res.CreditCustomers, CreditCustomer{
				CustomerCode:  customer,
				IsActive:      false,
				Credit:        0,
				Extra:         0,
				RemainDeposit: 0,
				Used:          0,
				Balance:       0,
			})

			customerStrs = append(customerStrs, customer)
			customerCheck[customer] = true
		}
	}

	if len(customerStrs) == 0 {
		return nil, fmt.Errorf("no customer to check credit")
	}

	res, err := getCreditByCustomer(sqlx, res, customerStrs)
	if err != nil {
		return nil, err
	}

	res, err = getDepositByCustomer(sqlx, res, customerStrs)
	if err != nil {
		return nil, err
	}

	res, err = getUsedByCustomer(sqlx, res, customerStrs)
	if err != nil {
		return nil, err
	}

	for i, customer := range res.CreditCustomers {
		customer.Balance = customer.Credit + customer.Extra + customer.RemainDeposit - customer.Used
		res.CreditCustomers[i] = customer
	}

	return &res, nil
}

func getDepositByCustomer(sqlx *sqlx.DB, res GetCreditResponse, customerStrs []string) (GetCreditResponse, error) {
	query := fmt.Sprintf(`
		select customer_code , sum(coalesce (amount_total ,0)) amount_total , sum(coalesce(amount_used,0))  amount_used,  sum(coalesce (amount_remain, 0)) amount_remain
		from deposit d 
		where customer_code in ('%s')
		group by customer_code
	`, strings.Join(customerStrs, `','`))
	rows, err := db.ExecuteQuery(sqlx, query)
	if err != nil {
		return res, err
	}

	for _, row := range rows {
		customerCode := row["customer_code"].(string)
		amountRemain := row["amount_remain"].(float64)

		for i, customer := range res.CreditCustomers {
			if customer.CustomerCode == customerCode {
				customer.RemainDeposit += amountRemain
				res.CreditCustomers[i] = customer
				break
			}
		}
	}

	return res, nil
}

func getCreditByCustomer(sqlx *sqlx.DB, res GetCreditResponse, customerStrs []string) (GetCreditResponse, error) {
	query := fmt.Sprintf(`
		select c.customer_code , coalesce(c.amount,0) credit_amount, coalesce (c.is_active,false) credit_is_active
			, coalesce(ce.amount, 0) extra_amount
		from credit c 
		left join credit_extra ce ON c.id = ce.credit_id  and ce.expire_dtm >= now() and ce.effective_dtm <= now()
		where 1=1
		and customer_code in ('%s')
	`, strings.Join(customerStrs, `','`))
	rows, err := db.ExecuteQuery(sqlx, query)
	if err != nil {
		return res, err
	}

	for _, row := range rows {
		customerCode := row["customer_code"].(string)
		creditAmount := row["credit_amount"].(float64)
		creditIsActive := row["credit_is_active"].(bool)
		extraAmount := row["extra_amount"].(float64)

		for i, customer := range res.CreditCustomers {
			if customer.CustomerCode == customerCode {
				customer.IsActive = creditIsActive
				customer.Credit = creditAmount
				customer.Extra = extraAmount

				res.CreditCustomers[i] = customer
			}
		}
	}

	return res, nil
}

func getUsedByCustomer(sqlx *sqlx.DB, res GetCreditResponse, customerStrs []string) (GetCreditResponse, error) {
	type customer struct {
		CustomerCode       string
		TotalPrice         float64
		TotalTransportCost float64
		Paid               float64
	}

	//Sale Order
	querySale := fmt.Sprintf(`
		select s.sale_code ,s.customer_code, coalesce(s.total_amount, 0) total_amount , coalesce(s.total_transport_cost, 0)  total_transport_cost
			, coalesce(s.transport_cost_type, '') transport_cost_type
		from sale s 
		where s.status = 'PENDING' and s.payment_method <> 'CASH' and (s.is_approved = true or s.status_approve = 'COMPLETED')  
			and s.customer_code in ('%s')
	`, strings.Join(customerStrs, `','`))
	rowsSale, err := db.ExecuteQuery(sqlx, querySale)
	if err != nil {
		return res, err
	}

	if len(rowsSale) == 0 {
		return res, nil
	}

	saleCodes := []string{}
	saleCodesMap := map[string]bool{}
	custMap := map[string]customer{}
	for _, row := range rowsSale {
		saleCode := row["sale_code"].(string)
		customerCode := row["customer_code"].(string)
		totalAmount := row["total_amount"].(float64)
		totalTransportCost := row["total_transport_cost"].(float64)
		transportCostType := row["transport_cost_type"].(string)
		transportCost := 0.0

		if transportCostType == "EXCL" {
			transportCost = totalTransportCost
		}

		if _, ok := saleCodesMap[saleCode]; !ok {
			saleCodes = append(saleCodes, saleCode)
			saleCodesMap[saleCode] = true
		}

		cust, existsCust := custMap[customerCode]
		if !existsCust {
			cust = customer{
				CustomerCode:       customerCode,
				TotalPrice:         0,
				TotalTransportCost: 0,
				Paid:               0,
			}
		}

		cust.TotalPrice += totalAmount
		cust.TotalTransportCost += transportCost
		custMap[customerCode] = cust
	}

	//Invoice
	queryInv := fmt.Sprintf(`
		select i.invoice_code, coalesce(i.party_code, '') customer_code, i.invoice_type 
 			, ii.invoice_item 
 			, coalesce(ii.document_ref, '') as sale_code, coalesce(ii.document_ref_item, '') as sale_item
			, i.party_code as customer_code
 		from invoice i 
 		left join invoice_item ii on i.id = ii.invoice_id 
 		where i.status in ('PENDING', 'COMPLETED') and i.invoice_type = 'AR'
			and i.party_code in ('%s')
			and ii.document_ref in ('%s')
	`, strings.Join(customerStrs, `','`), strings.Join(saleCodes, `','`))
	rowsInv, err := db.ExecuteQuery(sqlx, queryInv)
	if err != nil {
		return res, err
	}

	if len(rowsInv) != 0 {
		invoiceCodeMap := map[string]string{}
		invoiceCodeItem := map[string]string{}
		saleCodeItem := map[string]string{}
		invoiceCustomerMap := map[string]string{}

		for _, row := range rowsInv {
			customerCode := row["customer_code"].(string)
			invoiceCode := row["invoice_code"].(string)
			invoiceItem := row["invoice_item"].(string)
			saleCode := row["sale_code"].(string)
			saleItem := row["sale_item"].(string)

			if _, ok := invoiceCodeMap[invoiceCode]; !ok {
				invoiceCodeMap[invoiceCode] = invoiceCode
			}

			invoiceCodeItemKey := invoiceCode + "|" + invoiceItem
			if _, ok := invoiceCodeItem[invoiceCodeItemKey]; !ok {
				invoiceCodeItem[invoiceCodeItemKey] = fmt.Sprintf(`('%s','%s')`, invoiceCode, invoiceItem)
			}

			saleCodeItemKey := saleCode + "|" + saleItem
			if _, ok := saleCodeItem[saleCodeItemKey]; !ok {
				saleCodeItem[saleCodeItemKey] = fmt.Sprintf(`('%s','%s')`, saleCode, saleItem)
			}

			if _, ok := invoiceCustomerMap[invoiceCode]; !ok {
				invoiceCustomerMap[invoiceCode] = customerCode
			}
		}

		invoiceCodeString := mapToString(invoiceCodeMap)

		//AR Payment
		queryPayment := fmt.Sprintf(`
			select t.invoice_code , coalesce(t.amount, 0) amount
			from payment_invoice t 
			where t.invoice_code in ('%s')
		`, strings.Join(invoiceCodeString, `','`))
		fmt.Println(queryPayment)
		rowsPayment, err := db.ExecuteQuery(sqlx, queryPayment)
		if err != nil {
			return res, err
		}

		for _, row := range rowsPayment {
			invoiceCode := row["invoice_code"].(string)
			amount := row["amount"].(float64)

			customerCode := invoiceCustomerMap[invoiceCode]
			cust := custMap[customerCode]
			cust.Paid += amount
			custMap[customerCode] = cust
		}

		invoiceCodeItemString := mapToString(invoiceCodeItem)

		//DN & CN
		queryDN := fmt.Sprintf(`
			select i.invoice_code, i.invoice_type
				, ii.document_ref as invoice_ref, ii.document_ref_item as invoice_item_ref, coalesce(ii.total_amount, 0) as amount
			from invoice i 
			left join invoice_item ii on i.id = ii.invoice_id 
			where i.status in ('PENDING', 'COMPLETED') and i.invoice_type in ('CN', 'DN')
				and ii.document_ref <> '' and ii.document_ref_item != ''
				and (ii.document_ref, ii.document_ref_item ) in (%s) 
		`, strings.Join(invoiceCodeItemString, `,`))
		fmt.Println(queryDN)
		rowsDN, err := db.ExecuteQuery(sqlx, queryDN)
		if err != nil {
			return res, err
		}

		dnInvoiceCodeMap := map[string]string{}
		for _, row := range rowsDN {
			invoiceCode := row["invoice_code"].(string)
			invoiceType := row["invoice_type"].(string)
			invoiceRef := row["invoice_ref"].(string)
			amount := row["amount"].(float64)

			dnInvoiceCodeMap[invoiceCode] = invoiceCode

			customerCode := invoiceCustomerMap[invoiceRef]
			cust := custMap[customerCode]

			//Adjust Total Price for CN
			if invoiceType == "CN" {
				cust.TotalPrice -= amount
			}

			//Adjust Total Price for DN && add for find payment
			if invoiceType == "DN" {
				cust.TotalPrice += amount
				dnInvoiceCodeMap[invoiceCode] = invoiceCode
			}

			custMap[customerCode] = cust
		}

		//DN Payment
		if len(dnInvoiceCodeMap) > 0 {
			queryDNPayment := fmt.Sprintf(`
				select t.invoice_code , coalesce(t.amount, 0) amount
				from payment_invoice t 
				where t.invoice_code in ('%s')
			`, strings.Join(mapKeys(dnInvoiceCodeMap), `','`))
			rowsDNPayment, err := db.ExecuteQuery(sqlx, queryDNPayment)
			if err != nil {
				return res, err
			}

			for _, row := range rowsDNPayment {
				invoiceCode := row["invoice_code"].(string)
				amount := row["amount"].(float64)

				customerCode := invoiceCustomerMap[invoiceCode]
				cust := custMap[customerCode]
				cust.Paid += amount
				custMap[customerCode] = cust
			}

		}
	}

	//Summarize Used
	for _, cust := range custMap {
		used := cust.TotalPrice + cust.TotalTransportCost - cust.Paid

		for i, customer := range res.CreditCustomers {
			if customer.CustomerCode == cust.CustomerCode {
				customer.Used += used
				res.CreditCustomers[i] = customer

				break
			}
		}
	}

	return res, nil
}

func mapToString(m map[string]string) []string {
	var sb []string
	for _, v := range m {
		sb = append(sb, v)
	}
	return sb
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
