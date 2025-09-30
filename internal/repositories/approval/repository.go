package repositoryApproval

import (
	"errors"
	"fmt"
	"math"
	"prime-erp-core/internal/db"
	models "prime-erp-core/internal/models"
	"strings"

	"github.com/google/uuid"
)

func GetApprovalPreload(id []uuid.UUID, approveCode []string, status []string, documentCode []string, page int, pageSize int) ([]models.Approval, int, int, error) {
	aproval := []models.Approval{}

	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return nil, 0, 0, err
	}
	searchID := ""
	if len(id) > 0 {
		quotedStrings := make([]string, len(id))
		for i, s := range id {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchID = fmt.Sprintf(` and approval.id IN (%s)`, whereInClause)
	}

	searchApproveCode := ""
	if len(approveCode) > 0 {
		quotedStrings := make([]string, len(approveCode))
		for i, s := range approveCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchApproveCode = fmt.Sprintf(` and approval.approve_code IN (%s)`, whereInClause)
	}
	searchStatus := ""
	if len(status) > 0 {
		quotedStrings := make([]string, len(status))
		for i, s := range status {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchStatus = fmt.Sprintf(` and approval.status IN (%s)`, whereInClause)
	}
	searchDocumentCode := ""
	if len(documentCode) > 0 {
		quotedStrings := make([]string, len(documentCode))
		for i, s := range documentCode {
			quotedStrings[i] = fmt.Sprintf("'%s'", s)
		}
		whereInClause := strings.Join(quotedStrings, ", ")
		searchDocumentCode = fmt.Sprintf(` and approval.document_code IN (%s)`, whereInClause)
	}
	var approvalID []uuid.UUID
	gormx.Table("approval").Select("approval.id").
		Joins("inner join approval_item on approval.id = approval_item.approval_id").
		Joins("inner join approval_item_permission on approval_item.id = approval_item_permission.approval_item_id").
		Where("1=1 " + searchID + "" + searchApproveCode + "" + searchStatus + "" + searchDocumentCode + "").
		Group("approval.id").Scan(&approvalID)

	if len(approvalID) > 0 {

		var count = len(approvalID)

		query := gormx.Preload("ApprovalItem.ApprovalItemPermission")

		query = query.Where("id in (?)", approvalID)

		totalRecords := count
		totalPages := 0
		offset := (page - 1) * pageSize
		if totalRecords > 0 {

			if pageSize > 0 && page > 0 {
				query = query.Limit(pageSize).Offset(offset)
				totalPages = int(math.Ceil(float64(totalRecords) / float64(pageSize)))
			} else {
				query = query.Limit(int(totalRecords)).Offset(offset)
				totalPages = (int(totalRecords) / 1)
			}

		}

		err = query.Order("update_date desc").Find(&aproval).Error
		sqlDB, err1 := gormx.DB()
		if err1 != nil {
			return nil, 0, 0, err1
		}

		// Close the connection
		if err2 := sqlDB.Close(); err2 != nil {
			return nil, 0, 0, err2
		}
		return aproval, totalPages, int(totalRecords), err
	} else {
		return nil, 0, 0, err
	}

}

func CreateApproval(aproval []models.Approval, aprovalItem []models.ApprovalItem, approvalItemPermission []models.ApprovalItemPermission) (err error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return err
	}
	tx := gormx.Begin()
	defer func() {
		if rc := recover(); rc != nil {
			tx.Rollback()
			err = errors.New("panic error cant't save approval")
		}
	}()
	if err = tx.Error; err != nil {
		return err
	}
	if len(aproval) > 0 {
		result := tx.Create(&aproval)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	if len(aprovalItem) > 0 {
		result := tx.Create(&aprovalItem)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}
	if len(approvalItemPermission) > 0 {
		result := tx.Create(&approvalItemPermission)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
	}

	err = tx.Commit().Error
	return err
}
func UpdateApproval(aproval []models.Approval, aprovalItem []models.ApprovalItem, approvalItemPermission []models.ApprovalItemPermission) (int, error) {
	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return 0, err
	}
	rowsAffected := 0
	for _, aprovalValue := range aproval {
		result := gormx.Table("approval").Where("id = ?", aprovalValue.ID).Updates(&aprovalValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}
	for _, aprovalItemValue := range aprovalItem {
		result := gormx.Table("approval_item").Where("id = ?", aprovalItemValue.ID).Updates(&aprovalItemValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}
	for _, approvalItemPermissionValue := range approvalItemPermission {
		result := gormx.Table("approval_item_permission").Where("id = ?", approvalItemPermissionValue.ID).Updates(&approvalItemPermissionValue)

		if result.Error != nil {
			gormx.Rollback()
			return 0, result.Error
		}
		rowsAffected = int(result.RowsAffected)
	}

	return rowsAffected, nil
}
