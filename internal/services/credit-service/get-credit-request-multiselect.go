package creditService

import (
	"context"
	"sort"
	"strings"

	models "prime-erp-core/internal/models"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	customerService "prime-erp-core/internal/services/customer-service"
	"prime-erp-core/internal/utils"
)

// *_in / *_not ของทั้งสองคอลัมน์เป็น customer_code เสมอ (คอลัมน์ชื่อแสดงชื่อแต่ส่งรหัส)
type CustomerCodeLikeTerm struct {
	Like string   `json:"customer_code_like"`
	In   []string `json:"customer_code_in"`
	Not  []string `json:"customer_code_not"`
}

type CustomerNameLikeTerm struct {
	Like string   `json:"customer_name_like"`
	In   []string `json:"customer_code_in"`
	Not  []string `json:"customer_code_not"`
}

// customerCodeTerms แปลง customer_code_likes เป็น term ของ helper (ค้นที่ customer_code)
func (r GetCreditReq) customerCodeTerms() []utils.MultiSelectTerm {
	terms := make([]utils.MultiSelectTerm, 0, len(r.CustomerCodeLikes))
	for _, t := range r.CustomerCodeLikes {
		terms = append(terms, utils.MultiSelectTerm{Column: "customer_code", Like: t.Like, In: t.In, Not: t.Not})
	}
	return terms
}

// resolveCustomerNameCodes แปลง customer_name_likes เป็นรหัสลูกค้า เพราะชื่ออยู่ใน customer master
// กติกาเดียวกับ helper: มี In → ใช้ In, ไม่มี In → ชื่อที่ตรง like ตัด Not ออก, ทุก entry รวมกัน (OR)
// filtered=false คือไม่ต้องกรองด้วยชื่อ (ไม่ส่ง หรือมี entry "ทุกรายการ" ที่ไม่ตัดอะไร)
// filtered=true และ codes ว่าง = ไม่มีลูกค้าที่ตรง ผลต้องว่าง
func resolveCustomerNameCodes(
	terms []CustomerNameLikeTerm,
	fetchByNameLike func(like string) ([]string, error),
) (codes []string, filtered bool, err error) {
	if len(terms) == 0 {
		return nil, false, nil
	}
	seen := map[string]bool{}
	for _, term := range terms {
		if len(term.In) > 0 {
			for _, code := range term.In {
				seen[code] = true
			}
			continue
		}
		if term.Like == "" && len(term.Not) == 0 {
			return nil, false, nil
		}
		matched, err := fetchByNameLike(term.Like)
		if err != nil {
			return nil, false, err
		}
		excluded := make(map[string]bool, len(term.Not))
		for _, code := range term.Not {
			excluded[code] = true
		}
		for _, code := range matched {
			if !excluded[code] {
				seen[code] = true
			}
		}
	}
	codes = make([]string, 0, len(seen))
	for code := range seen {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes, true, nil
}

// fetchCustomerCodesByNameLike ใช้ customer_name_like (string) ที่ master รองรับอยู่แล้ว
func fetchCustomerCodesByNameLike(ctx context.Context) func(string) ([]string, error) {
	return func(like string) ([]string, error) {
		requestData := map[string]interface{}{}
		if like != "" {
			requestData["customer_name_like"] = like
		}
		customers, err := customerService.GetCustomers(ctx, requestData)
		if err != nil {
			return nil, err
		}
		codes := make([]string, 0, len(customers.Customers))
		for _, customer := range customers.Customers {
			codes = append(codes, customer.CustomerCode)
		}
		return codes, nil
	}
}

// sortCreditRequestsByName เรียงตามชื่อลูกค้า (ไม่มีชื่อไปท้าย) ใช้ customer_code เป็นตัวเรียงสำรอง
func sortCreditRequestsByName(credit []models.CreditRequest, names map[string]string, desc bool) {
	sort.SliceStable(credit, func(i, j int) bool {
		nameI, okI := names[credit[i].CustomerCode]
		nameJ, okJ := names[credit[j].CustomerCode]
		if okI != okJ {
			return okI
		}
		nameI, nameJ = strings.ToLower(nameI), strings.ToLower(nameJ)
		if nameI != nameJ {
			if desc {
				return nameI > nameJ
			}
			return nameI < nameJ
		}
		return credit[i].CustomerCode < credit[j].CustomerCode
	})
}

// paginateCreditRequests ตัดหน้าหลังเรียงแล้ว (page / page_size = 0 เหมือน repository: ทั้งหมดในหน้าเดียว)
func paginateCreditRequests(credit []models.CreditRequest, page int, pageSize int) ([]models.CreditRequest, int) {
	total := len(credit)
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = total
		if pageSize == 0 {
			pageSize = 1
		}
	}
	totalPages := (total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	if start >= total || start < 0 {
		return []models.CreditRequest{}, totalPages
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return credit[start:end], totalPages
}

func (r GetCreditReq) listOptions(nameCodes []string) repositoryCredit.CreditRequestListOptions {
	return repositoryCredit.CreditRequestListOptions{
		CustomerCode:      r.customerCodeTerms(),
		CustomerNameCodes: nameCodes,
		SortBy:            r.SortBy,
		SortDir:           r.SortDir,
	}
}
