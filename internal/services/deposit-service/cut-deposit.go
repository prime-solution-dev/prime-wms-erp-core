package depositService

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/db"
	models "prime-erp-core/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CutDepost(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.Deposit

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(`prime_erp`)
	defer db.CloseGORM(gormx)
	if err != nil {
		return nil, err
	}

	for _, deposit := range req {
		amountUsed := gorm.Expr("amount_used + ?", deposit.AmountUsed)

		result := gormx.Table("deposit").Where("deposit_code = ?", deposit.DepositCode).Updates(map[string]interface{}{
			"amount_used":   amountUsed,
			"amount_remain": gorm.Expr("amount_total - amount_used"),
		})

		if result.Error != nil {
			gormx.Rollback()
			return nil, result.Error
		}

	}

	return nil, nil

}
