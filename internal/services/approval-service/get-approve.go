package approvalService

import (
	"context"
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryApproval "prime-erp-core/internal/repositories/approval"
	authenticationService "prime-erp-core/internal/services/authentication-service"
	"sort"

	"github.com/google/uuid"
)

type GetApprovalRequest struct {
	ID           []uuid.UUID `json:"id"`
	ApproveCode  []string    `json:"approval_code"`
	Status       []string    `json:"status"`
	DocumentCode []string    `json:"document_code"`
	CondRangeMin float64     `json:"cond_range_min"`
	Page         int         `json:"page"`
	PageSize     int         `json:"page_size"`
}
type ResultApproval struct {
	Total       int               `json:"total"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
	TotalPages  int               `json:"total_pages"`
	ApprovalRes []models.Approval `json:"approval"`
}

// GetApproval รับ context.Context เพื่อให้ caller ที่แปลงแล้วเรียกได้ตรงๆ และส่งต่อให้
// authenticationService.GetRequester/GetUserApproval เพื่อแนบ token ไปกับ request ที่ยิงต่อ —
// แต่ห้ามส่ง gin's Context ดิบเข้ามาที่พารามิเตอร์นี้ ต่อให้มันขึ้นรูปตรงกับ interface
// context.Context ได้ก็ตาม เพราะ repo นี้ไม่ได้เปิด gin's ContextWithFallback ไว้ ตัว Value() ของมัน
// จะมองไม่เห็นค่าที่ requestcontext เก็บ (Task 5-6 Finding A) ผู้ที่ยังไม่แปลงต้องส่ง
// ctx.Request.Context() เท่านั้น
func GetApproval(ctx context.Context, jsonPayload string) (interface{}, error) {

	var req GetApprovalRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	approval, totalPages, totalRecords, errApproval := repositoryApproval.GetApprovalPreload(req.ID, req.ApproveCode, req.Status, req.DocumentCode, req.Page, req.PageSize)
	if errApproval != nil {
		return nil, errApproval
	}
	mdiItemCode := []string{}
	userCodeApprovalValue := []string{}
	for _, approvalValue := range approval {
		mdiItemCode = append(mdiItemCode, approvalValue.MDItemCode)
		userCodeApprovalValue = append(userCodeApprovalValue, approvalValue.CreateBy)
	}
	requestData := map[string]interface{}{
		"md_item_code": mdiItemCode,
		"action_code":  []string{"APPROVE"},
	}
	requester, errGetRequester := authenticationService.GetRequester(ctx, requestData)
	if errGetRequester != nil {
		return nil, errGetRequester
	}
	userCode := map[string]string{}
	for _, requesterValue := range requester {
		userCode[requesterValue.RequesterCode] = requesterValue.RequesterCode
	}
	requestDataGetUserApproval := map[string]interface{}{
		"user_code":        userCodeApprovalValue,
		"module_item_code": mdiItemCode,
		"active":           true,
		"is_not_expired":   true,
	}
	userApproval, errGetUserApproval := authenticationService.GetUserApproval(ctx, requestDataGetUserApproval)
	if errGetUserApproval != nil {
		return nil, errGetUserApproval
	}
	sort.Slice(userApproval.Data, func(i, j int) bool {
		if userApproval.Data[i].CondRangeMin == nil {
			return false
		}
		if userApproval.Data[j].CondRangeMin == nil {
			return true
		}
		return *userApproval.Data[i].CondRangeMin > *userApproval.Data[j].CondRangeMin
	})
	approvalItemPermission := map[string][]models.ApprovalItemPermission{}
	for _, userApprovalValue := range userApproval.Data {

		if userApprovalValue.CondRangeMin == nil {
			continue
		}
		if req.CondRangeMin >= float64(*userApprovalValue.CondRangeMin) {
			_, ok := userCode[userApprovalValue.ApproveCode]
			if ok {
				approvalItemPermission[userApprovalValue.ModuleItemCode] = append(approvalItemPermission[userApprovalValue.ModuleItemCode], models.ApprovalItemPermission{
					UserCode:    userApprovalValue.ApproveCode,
					AutoApprove: userApprovalValue.AutoApprove,
				})
			}
			for _, secondarysValue := range userApprovalValue.Secondarys {
				_, ok := userCode[secondarysValue.ApproveCode]
				if ok {
					approvalItemPermission[userApprovalValue.ModuleItemCode] = append(approvalItemPermission[userApprovalValue.ModuleItemCode], models.ApprovalItemPermission{
						UserCode:    secondarysValue.ApproveCode,
						AutoApprove: userApprovalValue.AutoApprove,
					})
				}
			}
			break
		}
	}

	for a := range approval {
		approvalItemPermissionResult, existsapprovalItemPermission := approvalItemPermission[approval[a].MDItemCode]
		if existsapprovalItemPermission {
			for ai := range approval[a].ApprovalItem {
				approval[a].ApprovalItem[ai].ApprovalItemPermission = approvalItemPermissionResult
			}
		}
	}

	resultApproval := ResultApproval{
		Total:       totalRecords,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  totalPages,
		ApprovalRes: approval,
	}

	return resultApproval, nil
}
