package timeService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"prime-erp-core/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GetTimeRequest struct {
	Topic []string `json:"topic"`
	Code  []string `json:"code"`
	Name  []string `json:"name"`
}

func (GetTimeResponse) TableName() string { return "time" }

type GetTimeResponse struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Topic     string    `gorm:"type:varchar(50)" json:"topic"`
	Code      string    `gorm:"type:varchar(50)" json:"code"`
	Name      string    `gorm:"type:varchar(100)" json:"name"`
	StartTime string    `gorm:"type:varchar(20)" json:"start_time"`
	EndTime   string    `gorm:"type:varchar(20)" json:"end_time"`
}

func GetTime(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var res []GetTimeResponse
	var req GetTimeRequest

	if jsonPayload != "" {
		if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
			return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
		}
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return nil, err
	}
	defer db.CloseGORM(gormx)

	query := gormx.Order("code ASC")

	if len(req.Topic) > 0 {
		query = query.Where("topic IN ?", req.Topic)
	}

	if len(req.Code) > 0 {
		query = query.Where("code IN ?", req.Code)
	}

	if len(req.Name) > 0 {
		query = query.Where("name IN ?", req.Name)
	}

	if err := query.Find(&res).Error; err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve data"})
		return nil, err
	}

	return res, nil
}
