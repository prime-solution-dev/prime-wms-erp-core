package approvalService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryApproval "prime-erp-core/internal/repositories/approval"
	authenticationService "prime-erp-core/internal/services/authentication-service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateApproval(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Approval

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	approvalValue := []models.Approval{}
	approvalItemValue := []models.ApprovalItem{}
	approvalItemPermissionValue := []models.ApprovalItemPermission{}
	approvalIDForReturn := []uuid.UUID{}
	mdiItemCode := []string{}

	for _, approval := range req {
		mdiItemCode = append(mdiItemCode, approval.MDItemCode)
	}

	requestData := map[string]interface{}{
		"md_item_code": mdiItemCode,
		"action_code":  []string{"APPROVE"},
	}
	requester, errGetRequester := authenticationService.GetRequester(requestData)
	if errGetRequester != nil {
		return nil, errGetRequester
	}

	for i, approval := range req {

		approvalID := uuid.New()
		req[i].ID = approvalID
		approvalIDForReturn = append(approvalIDForReturn, approvalID)

		approvalItemID := uuid.New()
		newApprovalItem := models.ApprovalItem{
			ID:                     approvalItemID,
			ApprovalID:             approvalID,
			StepSeq:                1,
			IsCondition:            false,
			Condition:              nil,
			Status:                 "PENDING",
			ActionBy:               req[i].CreateBy,
			ActionDate:             time.Now(),
			CreateBy:               req[i].CreateBy,
			UpdateBy:               req[i].CreateBy,
			ApprovalItemPermission: []models.ApprovalItemPermission{},
		}

		for _, requesterValue := range requester {
			approvalItemPermissionID := uuid.New()
			newApprovalItemPermission := models.ApprovalItemPermission{
				ID:             approvalItemPermissionID,
				ApprovalItemID: approvalItemID,
				UserCode:       requesterValue.RequesterID,
			}
			approvalItemPermissionValue = append(approvalItemPermissionValue, newApprovalItemPermission)
		}
		approvalItemValue = append(approvalItemValue, newApprovalItem)

		if req[i].ApproveCode != "" {
			req[i].ApproveCode = approval.ApproveCode
		} else {
			req[i].ApproveCode = uuid.New().String()
		}
		/* 	jsonDataApproval, _ := json.Marshal(req[i])
		req[i].DocumentData = jsonDataApproval */
		req[i].ApprovalItem = []models.ApprovalItem{}

		approvalValue = append(approvalValue, req[i])
	}

	errCreateApproval := repositoryApproval.CreateApproval(approvalValue, approvalItemValue, approvalItemPermissionValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}

	return map[string]interface{}{
		"id":      approvalIDForReturn,
		"status":  "success",
		"message": "Approval create successfully",
	}, nil
}
