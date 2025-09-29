package creditService

import (
	"encoding/json"
	"errors"
	repositoryCredit "prime-erp-core/internal/repositories/credit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DeleteCreditReq struct {
	ID []uuid.UUID `json:"id"`
}

func DeleteCreditExtra(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req DeleteCreditReq

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	errDeleteCredit := repositoryCredit.DeleteCreditExtra(req.ID)
	if errDeleteCredit != nil {
		return nil, errDeleteCredit
	}

	return nil, nil
}
