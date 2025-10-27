package priceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"prime-erp-core/internal/db"

	"github.com/gin-gonic/gin"
)

type GetPaymentTermRequest struct {
	TermCode []string `json:"term_code"`
	TermType []string `json:"term_type"`
}

type GetPaymentTermResponse struct {
	TermCode string `json:"term_code"`
	TermType string `json:"term_type"`
	TermName string `json:"term_name"`
}

func GetPaymentTerm(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	req := GetPaymentTermRequest{}
	res := []GetPaymentTermResponse{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, err
	}
	defer sqlx.Close()

	cond := ``
	if len(req.TermCode) > 0 {
		cond += fmt.Sprintf(` AND term_code IN ('%s') `, strings.Join(req.TermCode, `','`))
	}

	if len(req.TermType) > 0 {
		cond += fmt.Sprintf(` AND term_type IN ('%s') `, strings.Join(req.TermType, `','`))
	}

	query := fmt.Sprintf(`
		SELECT term_code, coalesce(term_type,'') term_type, coalesce (term_name ,'') term_name
		FROM payment_term 
		WHERE 1=1
		%s
		`, cond)
	//println(query)
	rows, err := db.ExecuteQuery(sqlx, query)
	if err != nil {
		return nil, err
	}

	for _, item := range rows {
		termCode := item["term_code"].(string)
		termType := item["term_type"].(string)
		termName := item["term_name"].(string)

		res = append(res, GetPaymentTermResponse{
			TermCode: termCode,
			TermType: termType,
			TermName: termName,
		})
	}

	return res, nil
}
