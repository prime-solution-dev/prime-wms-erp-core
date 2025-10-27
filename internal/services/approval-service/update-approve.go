package approvalService

import (
	"encoding/json"
	"errors"
	models "prime-erp-core/internal/models"
	repositoryApproval "prime-erp-core/internal/repositories/approval"

	"github.com/gin-gonic/gin"
)

func UpdateApproval(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Approval

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	approvalValue := []models.Approval{}
	approvalItemValue := []models.ApprovalItem{}
	approvalItemPermissionValue := []models.ApprovalItemPermission{}

	for i := range req {

		/* 	for o := range approval.ApprovalItem {
			req[i].ApprovalItem[o].StepSeq = 1
			req[i].ApprovalItem[o].IsCondition = false
			jsonDataApprovalItem, _ := json.Marshal(req[i].ApprovalItem[o])
			req[i].ApprovalItem[o].Condition = jsonDataApprovalItem
			req[i].ApprovalItem[o].Status = "PENDING"
			req[i].ApprovalItem[o].ActionDate = time.Now()
			req[i].ApprovalItem[o].UpdateBy = req[i].UpdateBy
			req[i].ApprovalItem[o].ApprovalItemPermission = []models.ApprovalItemPermission{}
			approvalItemValue = append(approvalItemValue, req[i].ApprovalItem[o])

		} */

		/* if req[i].ApproveCode != "" {
			req[i].ApproveCode = approval.ApproveCode
		} else {
			req[i].ApproveCode = uuid.New().String()
		} */
		/* jsonDataApproval, _ := json.Marshal(req[i])
		req[i].DocumentData = jsonDataApproval */
		req[i].ApprovalItem = []models.ApprovalItem{}

		approvalValue = append(approvalValue, req[i])
	}

	rowsAffected, errCreateApproval := repositoryApproval.UpdateApproval(approvalValue, approvalItemValue, approvalItemPermissionValue)
	if errCreateApproval != nil {
		return nil, errCreateApproval
	}
	if rowsAffected > 0 {
		return map[string]interface{}{
			"status":  "success",
			"message": "Approval updated successfully",
		}, nil
	} else {
		return map[string]interface{}{
			"status":  "success",
			"message": "Approval Not Have Rows Affected",
		}, nil
	}
}
