package priceService

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime-erp-core/internal/models"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// payload ที่ครบถ้วน ใช้เป็นฐานแล้วทำให้แต่ละ field เสียทีละตัว
func validExtraPayload() map[string]interface{} {
	return map[string]interface{}{
		"price_list_group_id": uuid.New().String(),
		"extra_key":           "PG06_1",
		"condition_code":      "PG06",
		"operator":            "<=",
		"value_int":           1.0,
		"length_extra_key":    1,
		"cond_range_min":      0.0,
		"cond_range_max":      45.0,
		"price_list_group_extra_keys": []map[string]interface{}{
			{"code": "PG06", "value": "PG06_1", "seq": 1},
		},
	}
}

func TestUpdateExtras_RejectsEmptyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		mutate func(m map[string]interface{})
	}{
		{"price_list_group_id ว่าง", func(m map[string]interface{}) {
			m["price_list_group_id"] = uuid.Nil.String()
		}},
		{"extra_key ว่าง", func(m map[string]interface{}) { m["extra_key"] = "" }},
		{"extra_key มีแต่ช่องว่าง", func(m map[string]interface{}) { m["extra_key"] = "   " }},
		{"condition_code ว่าง", func(m map[string]interface{}) { m["condition_code"] = "" }},
		{"operator ว่าง", func(m map[string]interface{}) { m["operator"] = "" }},
		{"operator ไม่รู้จัก", func(m map[string]interface{}) { m["operator"] = "~~" }},
		{"cond_range_min มากกว่า max ตอน operator เป็น <>", func(m map[string]interface{}) {
			m["operator"] = "<>"
			m["cond_range_min"] = 50.0
			m["cond_range_max"] = 10.0
		}},
		{"extra_keys เป็น slice ว่าง", func(m map[string]interface{}) {
			m["price_list_group_extra_keys"] = []map[string]interface{}{}
		}},
		{"extra_keys มี code ว่าง", func(m map[string]interface{}) {
			m["price_list_group_extra_keys"] = []map[string]interface{}{
				{"code": "", "value": "PG06_1", "seq": 1},
			}
		}},
		{"extra_keys มี value ว่าง", func(m map[string]interface{}) {
			m["price_list_group_extra_keys"] = []map[string]interface{}{
				{"code": "PG06", "value": "", "seq": 1},
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validExtraPayload()
			tt.mutate(payload)
			body, _ := json.Marshal([]map[string]interface{}{payload})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListExtra", bytes.NewReader(body))

			_, err := UpdateExtras(c, string(body))
			if err == nil {
				t.Fatal("ต้องได้ error แต่ผ่าน validation ไปได้")
			}
			if _, ok := err.(*utils.BindingError); !ok {
				t.Fatalf("ต้องเป็น *utils.BindingError เพื่อให้ได้ HTTP 400 แต่ได้ %T: %v", err, err)
			}
		})
	}
}

// operator ที่ extraConditionMatched รองรับต้องผ่าน validation ทุกตัว
// test ชุดเดิมเป็นเคส reject ทั้งหมด จึงไม่จับกรณี false rejection
// ซึ่งเป็นบั๊กที่ whitelist ชุดแรกทำไว้ (ตัด < และ > ออกทั้งที่ UI ให้เลือกและระบบคิดราคารองรับ)
func TestValidateExtras_AcceptsEveryOperatorTheCalculatorSupports(t *testing.T) {
	operators := []string{"=", ">=", "<=", "<", ">", "<>"}

	for _, op := range operators {
		t.Run(op, func(t *testing.T) {
			extras := []models.UpdatePriceListExtraRequest{{
				PriceListGroupID: uuid.New(),
				ExtraKey:         "PG06_1",
				ConditionCode:    "PG06",
				Operator:         op,
				CondRangeMin:     0,
				CondRangeMax:     45,
				PriceListGroupExtraKeys: []models.UpdatePriceListGroupExtraKeyRequest{
					{Code: "PG06", Value: "PG06_1", Seq: 1},
				},
			}}

			if err := validateExtras(extras); err != nil {
				t.Fatalf("operator %q ต้องผ่าน validation แต่ถูกปฏิเสธ: %v", op, err)
			}
		})
	}
}

// operator ที่ใช้ขอบเดียว (>=, <=, <, >, =) ไม่ควรถูกตรวจ min > max เพราะอีกขอบไม่มี
// ความหมาย ข้อมูลเก่าใน DB ที่มี operator=">=" กับ min=100,max=0 เป็นรูปแบบที่ถูกต้อง
// ตาม extraConditionMatched (">=" ใช้แค่ min) ต้องไม่ถูกปฏิเสธ
func TestValidateExtras_MinGreaterThanMaxOnlyRejectedForBetween(t *testing.T) {
	baseExtra := func(operator string, min, max float64) []models.UpdatePriceListExtraRequest {
		return []models.UpdatePriceListExtraRequest{{
			PriceListGroupID: uuid.New(),
			ExtraKey:         "PG06_1",
			ConditionCode:    "PG06",
			Operator:         operator,
			CondRangeMin:     min,
			CondRangeMax:     max,
			PriceListGroupExtraKeys: []models.UpdatePriceListGroupExtraKeyRequest{
				{Code: "PG06", Value: "PG06_1", Seq: 1},
			},
		}}
	}

	if err := validateExtras(baseExtra(">=", 100, 0)); err != nil {
		t.Fatalf(">= ที่ min=100, max=0 ต้องผ่าน (ใช้แค่ min) แต่ถูกปฏิเสธ: %v", err)
	}

	if err := validateExtras(baseExtra("<>", 50, 10)); err == nil {
		t.Fatal("<> ที่ min=50, max=10 ต้องไม่ผ่าน (ใช้ทั้ง min และ max)")
	}
}

// payload ที่ไม่ใช่ JSON ต้องได้ BindingError ไม่ใช่ 500
func TestUpdateExtras_RejectsMalformedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListExtra", bytes.NewBufferString("{oops"))

	_, err := UpdateExtras(c, "{oops")
	if err == nil {
		t.Fatal("ต้องได้ error")
	}
	if _, ok := err.(*utils.BindingError); !ok {
		t.Fatalf("ต้องเป็น *utils.BindingError แต่ได้ %T", err)
	}
}

// ไม่ส่งรายการมาเลยต้องถูกปฏิเสธ
func TestUpdateExtras_RejectsEmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/price/UpdatePriceListExtra", bytes.NewBufferString("[]"))

	_, err := UpdateExtras(c, "[]")
	if err == nil {
		t.Fatal("ต้องได้ error")
	}
	if _, ok := err.(*utils.BindingError); !ok {
		t.Fatalf("ต้องเป็น *utils.BindingError แต่ได้ %T", err)
	}
}
