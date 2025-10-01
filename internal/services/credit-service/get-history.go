package creditService

import (
	"encoding/json"
	"errors"
	repositoryCredit "prime-erp-core/internal/repositories/credit"
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
	RequestCode         string     `json:"request_code"`
	CreditLimit         float64    `json:"credit_limit"`
	IncreaseCreditLimit float64    `json:"increase_credit_limit"`
	StartDateTime       *time.Time `json:"start_date_time"`
	EndDateTime         *time.Time `json:"end_date_time"`
	SubmitDateTime      *time.Time `json:"submit_date_time"`
	ApproveDateTime     *time.Time `json:"approve_date_time"`
	Status              string     `json:"status"`
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

	credit, totalPages, totalRecords, errApproval := repositoryCredit.GetCreditRequest(req.ID, req.CustomerCode, req.Page, req.PageSize)
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
					RequestCode:         creditValue.RequestCode,
					CreditLimit:         0,
					IncreaseCreditLimit: creditValue.Amount,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ActionDate,
					Status:              creditValue.Status,
				})
			}
			if creditValue.RequestType == "BASE" {
				historyRes = append(historyRes, GetHistoryRes{
					ID:                  creditValue.ID,
					RequestCode:         creditValue.RequestCode,
					CreditLimit:         creditValue.Amount,
					IncreaseCreditLimit: 0,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ActionDate,
					Status:              creditValue.Status,
				})
			}
		} else {
			if creditValue.RequestType == "EXTRA" {
				reqCodeAmount[creditValue.RequestCode] = GetHistoryRes{
					ID:                  creditValue.ID,
					RequestCode:         creditValue.RequestCode,
					CreditLimit:         0,
					IncreaseCreditLimit: creditValue.Amount,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ActionDate,
					Status:              creditValue.Status,
				}
			}
			if creditValue.RequestType == "BASE" {
				reqCodeAmount[creditValue.RequestCode] = GetHistoryRes{
					ID:                  creditValue.ID,
					RequestCode:         creditValue.RequestCode,
					CreditLimit:         creditValue.Amount,
					IncreaseCreditLimit: 0,
					StartDateTime:       creditValue.EffectiveDtm,
					EndDateTime:         creditValue.ExpireDtm,
					SubmitDateTime:      creditValue.CreateDtm,
					ApproveDateTime:     creditValue.ActionDate,
					Status:              creditValue.Status,
				}
			}
			reqCode = append(reqCode, creditValue.RequestCode)
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

	for _, creditValue := range GetCreditRes.(ResultCredit).Credit {
		isActive := "Inactive"
		if creditValue.IsActive {
			isActive = "Active"
		}

		for _, creditExtraValue := range creditValue.CreditExtra {

			historyRes = append(historyRes, GetHistoryRes{
				ID:                  creditExtraValue.ID,
				CreditLimit:         0,
				IncreaseCreditLimit: creditExtraValue.Amount,
				StartDateTime:       creditExtraValue.EffectiveDtm,
				EndDateTime:         creditExtraValue.ExpireDtm,
				SubmitDateTime:      creditExtraValue.CreateDtm,
				ApproveDateTime:     creditExtraValue.ApproveDate,
				Status:              isActive,
			})
		}
		historyRes = append(historyRes, GetHistoryRes{
			ID:                  creditValue.ID,
			CreditLimit:         creditValue.Amount,
			IncreaseCreditLimit: 0,
			StartDateTime:       creditValue.EffectiveDtm,
			SubmitDateTime:      creditValue.CreateDtm,
			ApproveDateTime:     creditValue.ApproveDate,
			Status:              isActive,
		})
	}

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

		reqCodeAmountMap, exist := reqCodeAmount[approvalValue.TransactionCode]
		if exist {
			historyRes = append(historyRes, GetHistoryRes{
				ID:                  approvalValue.ID,
				CreditLimit:         reqCodeAmountMap.CreditLimit,
				IncreaseCreditLimit: reqCodeAmountMap.IncreaseCreditLimit,
				StartDateTime:       reqCodeAmountMap.StartDateTime,
				EndDateTime:         reqCodeAmountMap.EndDateTime,
				SubmitDateTime:      reqCodeAmountMap.SubmitDateTime,
				ApproveDateTime:     reqCodeAmountMap.ApproveDateTime,
				Status:              approvalValue.Status,
			})
		}
	}

	resultApproval := ResultHistory{
		Total:      totalRecords,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		HistoryRes: historyRes,
	}

	return resultApproval, nil
}
