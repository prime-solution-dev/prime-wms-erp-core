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

	//Sale
	querySale := fmt.Sprintf(`
		select s.sale_code ,s.customer_code, s.total_amount , s.total_transport_cost , s.transport_cost_type 
		from sale s 
		where s.status = 'PENDDING' and s.is_approved = true 
			and s.customer_code in ('%s')
	`, strings.Join(customerStrs, `','`))
	rowsSale, err := db.ExecuteQuery(sqlx, querySale)
	if err != nil {
		return res, err
	}

	if len(rowsSale) == 0 {
		return res, nil
	}

	saleCodeStrs := []string{}
	saleCodeStrsCheck := map[string]bool{}
	custMap := map[string]customer{}
	for _, row := range rowsSale {
		saleCode := row["sale_code"].(string)
		customerCode := row["customer_code"].(string)
		totalPrice := row["total_amount"].(float64)
		totalTransCost := row["total_transport_cost"].(float64)
		transType := row["transport_cost_type"].(string)

		if _, ok := saleCodeStrsCheck[saleCode]; !ok {
			saleCodeStrs = append(saleCodeStrs, saleCode)
			saleCodeStrsCheck[saleCode] = true
		}

		cust, existCust := custMap[customerCode]
		if !existCust {
			newCust := customer{
				CustomerCode:       customerCode,
				TotalPrice:         0,
				TotalTransportCost: 0,
				Paid:               0,
			}

			cust = newCust
		}

		cust.TotalPrice += totalPrice
		if transType == `EXC` { //TODO: recheck logic from BA
			cust.TotalTransportCost = totalTransCost
		}

		custMap[saleCode] = cust
	}

	if len(saleCodeStrs) == 0 {
		return res, nil
	}

	//Paid
	queryPaid := fmt.Sprintf(`
		select i.customer_code, i.doc_ref as sale_code, pi2.amount paid_amount
		from invoice i 
		left join payment_invoice pi2 on i.invoice_code = pi2.invoice_code 
		where i.customer_code in ('%s') and i.doc_ref in ('%s')
	`, strings.Join(customerStrs, `','`), strings.Join(saleCodeStrs, `','`))
	rowsPaid, err := db.ExecuteQuery(sqlx, queryPaid)
	if err != nil {
		return res, nil
	}

	for _, row := range rowsPaid {
		customerCode := row["customer_code"].(string)
		paid := row["paid_amount"].(float64)

		cust, existCust := custMap[customerCode]

		if !existCust {
			continue
		}

		cust.Paid += paid
		custMap[customerCode] = cust
	}

	//Summary
	for i, customer := range res.CreditCustomers {
		if cust, existCust := custMap[customer.CustomerCode]; existCust {
			customer.Used += cust.TotalPrice + cust.TotalTransportCost - cust.Paid
			res.CreditCustomers[i] = customer
		}
	}

	return res, nil
}
