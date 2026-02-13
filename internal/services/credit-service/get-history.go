package creditService

import (
	"encoding/json"
	"errors"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
	"slices"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetHistoryReq struct {
	ID           []uuid.UUID `json:"id"`
	CustomerCode []string    `json:"customer_code"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type GetHistoryRes struct {
	ID                  uuid.UUID  `json:"id"`
	CustomerCode        string     `json:"customer_code"`
	RequestCode         string     `json:"request_code"`
	RequestType         string     `json:"request_type"`
	CreditLimit         float64    `json:"credit_limit"`
	IncreaseCreditLimit float64    `json:"increase_credit_limit"`
	StartDateTime       *time.Time `json:"start_date_time"`
	EndDateTime         *time.Time `json:"end_date_time"`
	SubmitDateTime      *time.Time `json:"submit_date_time"`
	ApproveDateTime     *time.Time `json:"approve_date_time"`
	Status              string     `json:"status"`
	EffectiveDtm        *time.Time `json:"effective_dtm"`
	ExpireDtm           *time.Time `json:"expire_dtm"`
	RequestDate         *time.Time `json:"request_date"`
}
type ResultHistory struct {
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
	HistoryRes []GetHistoryRes `json:"credit_request"`
}

func GetHistory(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req GetCreditReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	credit, totalPages, totalRecords, errApproval := repositoryCredit.GetCreditRequest(req.ID, req.CustomerCode, nil, nil, nil, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}
	historyRes := []GetHistoryRes{}
	reqCode := []string{}
	reqCodeAmount := map[string]GetHistoryRes{}
	for _, creditValue := range credit {
		if !creditValue.IsAction {
			if creditValue.RequestType == "EXTRA" {
				historyRes = append(historyRes, GetHistoryRes{
					ID:                  creditValue.ID,
					CustomerCode:        creditValue.CustomerCode,
					RequestCode:         creditValue.RequestCode,
					RequestType:         creditValue.RequestType,
					CreditLimit:         0,
					IncreaseCreditLimit: creditValue.Amount,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ApproveDate,
					Status:              creditValue.Status,
					EffectiveDtm:        creditValue.EffectiveDtm,
					ExpireDtm:           creditValue.ExpireDtm,
					RequestDate:         creditValue.RequestDate,
				})
			}
			if creditValue.RequestType == "BASE" {
				historyRes = append(historyRes, GetHistoryRes{
					ID:                  creditValue.ID,
					CustomerCode:        creditValue.CustomerCode,
					RequestCode:         creditValue.RequestCode,
					RequestType:         creditValue.RequestType,
					CreditLimit:         creditValue.Amount,
					IncreaseCreditLimit: 0,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ApproveDate,
					Status:              creditValue.Status,
					EffectiveDtm:        creditValue.EffectiveDtm,
					ExpireDtm:           creditValue.ExpireDtm,
					RequestDate:         creditValue.RequestDate,
				})
			}
		} else {
			if creditValue.RequestType == "EXTRA" {
				reqCodeAmount[creditValue.RequestCode] = GetHistoryRes{
					ID:                  creditValue.ID,
					CustomerCode:        creditValue.CustomerCode,
					RequestCode:         creditValue.RequestCode,
					RequestType:         creditValue.RequestType,
					CreditLimit:         0,
					IncreaseCreditLimit: creditValue.Amount,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ActionDate,
					Status:              creditValue.Status,
					EffectiveDtm:        creditValue.EffectiveDtm,
					ExpireDtm:           creditValue.ExpireDtm,
					RequestDate:         creditValue.RequestDate,
				}
			}
			if creditValue.RequestType == "BASE" {
				reqCodeAmount[creditValue.RequestCode] = GetHistoryRes{
					ID:                  creditValue.ID,
					CustomerCode:        creditValue.CustomerCode,
					RequestCode:         creditValue.RequestCode,
					RequestType:         creditValue.RequestType,
					CreditLimit:         creditValue.Amount,
					IncreaseCreditLimit: 0,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ActionDate,
					Status:              creditValue.Status,
					EffectiveDtm:        creditValue.EffectiveDtm,
					ExpireDtm:           creditValue.ExpireDtm,
					RequestDate:         creditValue.RequestDate,
				}
				reqCodeAmount[creditValue.CustomerCode] = GetHistoryRes{
					ID:                  creditValue.ID,
					CustomerCode:        creditValue.CustomerCode,
					RequestCode:         creditValue.RequestCode,
					RequestType:         creditValue.RequestType,
					CreditLimit:         creditValue.Amount,
					IncreaseCreditLimit: 0,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ActionDate,
					Status:              creditValue.Status,
					EffectiveDtm:        creditValue.EffectiveDtm,
					ExpireDtm:           creditValue.ExpireDtm,
					RequestDate:         creditValue.RequestDate,
				}
				reqCode = append(reqCode, creditValue.CustomerCode)
			}

			reqCode = append(reqCode, creditValue.RequestCode)
			if creditValue.Status == "REJECT" {
				reqCode = append(reqCode, creditValue.RequestCode)
			}
		}

	}

	requestData := map[string]interface{}{
		"customer_code": req.CustomerCode,
	}
	jsonBytesGetCredit, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	GetCreditRes, errGetCredit := GetCredit(ctx, string(jsonBytesGetCredit))
	if errGetCredit != nil {
		return nil, errGetCredit
	}
	codeInCredit := []string{}
	for _, creditValue := range GetCreditRes.(ResultCredit).Credit {
		isActive := "INACTIVE"
		if creditValue.IsActive {
			isActive = "ACTIVE"
		}
		codeInCredit = append(codeInCredit, creditValue.DocRef)
		for _, creditExtraValue := range creditValue.CreditExtra {

			historyRes = append(historyRes, GetHistoryRes{
				ID:                  creditExtraValue.ID,
				CreditLimit:         0,
				RequestType:         "EXTRA",
				IncreaseCreditLimit: creditExtraValue.Amount,
				StartDateTime:       creditExtraValue.EffectiveDtm,
				EndDateTime:         creditExtraValue.ExpireDtm,
				SubmitDateTime:      creditExtraValue.CreateDtm,
				ApproveDateTime:     creditExtraValue.ApproveDate,
				Status:              isActive,
			})

			reqCodeAmount[creditExtraValue.CreditID.String()] = GetHistoryRes{
				ID:                  creditExtraValue.ID,
				CreditLimit:         0,
				IncreaseCreditLimit: creditExtraValue.Amount,
				StartDateTime:       creditExtraValue.EffectiveDtm,
				EndDateTime:         creditExtraValue.ExpireDtm,
				SubmitDateTime:      creditExtraValue.CreateDtm,
				ApproveDateTime:     creditExtraValue.ApproveDate,
				Status:              isActive,
			}
			reqCode = append(reqCode, creditExtraValue.CreditID.String())

		}
		historyRes = append(historyRes, GetHistoryRes{
			ID:                  creditValue.ID,
			CreditLimit:         creditValue.Amount,
			RequestType:         "BASE",
			IncreaseCreditLimit: 0,
			StartDateTime:       creditValue.EffectiveDtm,
			SubmitDateTime:      creditValue.CreateDtm,
			ApproveDateTime:     creditValue.ApproveDate,
			Status:              isActive,
		})
	}
	reqCode = slices.DeleteFunc(reqCode, func(v string) bool {
		return slices.Contains(codeInCredit, v)
	})

	requestApprovalData := map[string]interface{}{
		"transaction_code": reqCode,
	}
	jsonBytesGetApproval, err := json.Marshal(requestApprovalData)
	if err != nil {
		return nil, err
	}

	approvalRes, errGetApproval := GetTransaction(ctx, string(jsonBytesGetApproval))
	if errGetApproval != nil {
		return nil, errGetApproval
	}

	for _, approvalValue := range approvalRes.(ResultCreditTransaction).CreditTransaction {
		if approvalValue.Status == "INACTIVE" || approvalValue.Status == "CANCELED" || approvalValue.Status == "REJECT" {
			reqCodeAmountMap, exist := reqCodeAmount[approvalValue.TransactionCode]
			if exist {
				creditLimit := 0.0
				increaseCreditLimit := 0.0
				if approvalValue.TransactionType == "EXTRA" {
					increaseCreditLimit = approvalValue.Amount
				} else {
					creditLimit = approvalValue.Amount
				}
				historyRes = append(historyRes, GetHistoryRes{
					ID:                  approvalValue.ID,
					CreditLimit:         creditLimit,
					IncreaseCreditLimit: increaseCreditLimit,
					StartDateTime:       reqCodeAmountMap.StartDateTime,
					EndDateTime:         reqCodeAmountMap.EndDateTime,
					SubmitDateTime:      reqCodeAmountMap.SubmitDateTime,
					ApproveDateTime:     reqCodeAmountMap.ApproveDateTime,
					Status:              approvalValue.Status,
				})
			}
		}

	}
	order := map[string]int{
		"PENDING":   1,
		"COMPLETED": 2,
		"ACTIVE":    3,
		"CANCELED":  4,
		"CANCELLED": 5,
		"REJECT":    6,
		"INACTIVE":  7,
	}
	sort.Slice(historyRes, func(o, j int) bool {
		return order[historyRes[o].Status] < order[historyRes[j].Status]
	})
	resultApproval := ResultHistory{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		HistoryRes: historyRes,
	}

	return resultApproval, nil
}
