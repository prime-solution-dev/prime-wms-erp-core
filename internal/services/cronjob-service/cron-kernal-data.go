package CronjobService

import (
	"prime-erp-core/internal/cronjob"
	"sync"

	"github.com/gin-gonic/gin"
)

func init() {
	cronjob.RegisterJob("wms-kernal", GetKernal, "*/1 * * * *")
}

func GetKernalManual(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	GetKernal()

	return nil, nil
}

func GetKernal() {
	ttt := "start kernal service"
	println(ttt)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		CreditExtra()
	}()

	go func() {
		defer wg.Done()
		CreditRequestEffectiveDtm()
	}()
}
