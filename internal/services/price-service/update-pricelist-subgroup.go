package priceService

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	externalService "prime-erp-core/external/warehouse-service"
	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"
	priceDomain "prime-erp-core/internal/services/price-service/domain"
	"prime-erp-core/internal/utils"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// seam for unit testing: allow stubbing repository function
var updateSubGroupFunc = priceListRepository.UpdatePriceListSubGroups

// jsonValidator ใช้แทน gin's default binding validator ที่ ctx.ShouldBindJSON เคยเรียกให้
// gin ตั้งชื่อ tag เป็น "binding" (ไม่ใช่ "validate" ที่เป็น default ของ go-playground/validator)
// ดู github.com/gin-gonic/gin/binding/default_validator.go: lazyinit() -> SetTagName("binding")
// ต้องตั้งเหมือนกันทุกประการ ไม่งั้น struct tag `binding:"required"` ในโมเดลจะไม่ถูกอ่าน
var jsonValidator = func() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}()

// bindJSONRequest แทน ctx.ShouldBindJSON เดิม: decode payload แล้ว validate ด้วย struct
// tag "binding" ตัวเดียวกับที่ gin ใช้ ผล error (message/status ปลายทางที่ writeError คืน)
// ต้องเหมือนเดิมทุกประการ เพื่อไม่ให้ response shape ของ 3 endpoint นี้เปลี่ยน
//
// ต้องใช้ json.NewDecoder(...).Decode เหมือน gin/binding/json.go:decodeJSON ทุกประการ
// ไม่ใช่ json.Unmarshal — สอง decoder นี้ให้ error message ต่างกันตอน body ว่าง (EOF vs
// "unexpected end of JSON input") และ Decoder ยอมรับ body ที่มีขยะต่อท้าย JSON object
// ตัวแรกส่วน Unmarshal จะ reject ถ้าสลับไปใช้ Unmarshal validation behaviour จะเปลี่ยนจริง
func bindJSONRequest(jsonPayload string, req interface{}) error {
	if err := json.NewDecoder(strings.NewReader(jsonPayload)).Decode(req); err != nil {
		return &utils.BindingError{
			Message: fmt.Sprintf("Invalid request: %v", err.Error()),
		}
	}

	if err := jsonValidator.Struct(req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errorMessages []string
			for _, fieldError := range validationErrors {
				errorMessages = append(errorMessages, getValidationErrorMessage(fieldError))
			}
			return &utils.BindingError{
				Message: fmt.Sprintf("Validation failed: %v", errorMessages),
			}
		}
		return &utils.BindingError{
			Message: fmt.Sprintf("Invalid request: %v", err.Error()),
		}
	}

	return nil
}

// UpdatePriceListSubGroup updates a price_list_sub_group record using JSON binding and validation
func UpdatePriceListSubGroup(ctx context.Context, jsonPayload string) (interface{}, error) {
	var req models.UpdatePriceListSubGroupRequest

	if err := bindJSONRequest(jsonPayload, &req); err != nil {
		return nil, err
	}

	// save จากหน้า detail = ใช้ราคาที่คำนวณใหม่แล้ว จึงล้าง need_recalculate (repo ล้างเฉพาะ subgroup ที่ส่ง field ราคามา แม้ค่าเท่าเดิม)
	req.ClearRecalculate = true

	// ราคาที่ขึ้นกับค่าที่ผู้ใช้กรอก (เช่น pcs = kg*weight_spec) คำนวณก่อนเขียน แล้วรวมเข้า
	// change เดียวกัน: บันทึกครั้งเดียว history แถวเดียว ถ้าคำนวณไม่ได้ก็ยังไม่มีอะไรถูกเขียน
	if err := fillDependentPrices(ctx, req.Changes); err != nil {
		return nil, fmt.Errorf("failed to calculate dependent prices: %w", err)
	}

	// Call repository function (batch)
	if err := updateSubGroupFunc(req); err != nil {
		return nil, fmt.Errorf("failed to update price list sub group: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Price list sub group updated successfully",
	}, nil
}

// fillDependentPrices เติม total_net_price ที่ขึ้นกับราคาที่ผู้ใช้กรอกเองลงใน changes
//
// รันเฉพาะสูตรที่ params.required อ้าง uom ที่กรอกเอง หรืออ้างผลของสูตรที่เพิ่งรัน
// (ตามลำดับ sortFormulasByDependency) สูตรอื่น เช่น (base_price+extra)*weight_spec ไม่เกี่ยวกับ
// ค่าที่กรอกจึงไม่รันตอน save · สูตรของ uom ที่กรอกเองไม่รัน ค่าที่กรอกชนะเสมอ
//
// ต่างจาก update-latest ตรงที่ห้ามบันทึกค่าที่คิดจาก fallback: ดึงสต็อกไม่ได้ = error
// weight_spec / avg_kg_stock ไม่มีค่า = ข้ามสูตรนั้น (ไม่ใช้ 1.0 แทน) · extra ใช้ค่าที่เก็บไว้
func fillDependentPrices(ctx context.Context, changes []models.UpdatePriceListSubGroupItem) error {
	ids := make([]uuid.UUID, 0, len(changes))
	for _, c := range changes {
		if c.TotalNetPriceWeight != nil || c.TotalNetPriceUnit != nil {
			ids = append(ids, c.SubGroupID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	subGroups, err := getPriceListSubGroupsByIDsFunc(ids)
	if err != nil {
		return err
	}
	subGroupMap := make(map[uuid.UUID]*models.PriceListSubGroup, len(subGroups))
	codes := make([]string, 0, len(subGroups))
	for i := range subGroups {
		subGroupMap[subGroups[i].ID] = &subGroups[i]
		codes = append(codes, subGroups[i].SubGroupCode)
	}
	formulasMap, err := getPriceListSubGroupFormulasMapBySubGroupCodesFunc(codes)
	if err != nil {
		return err
	}

	// วางแผนก่อนว่าสูตรไหนจะรัน เพื่อเรียก warehouse เฉพาะเมื่อมีสูตรที่ต้องใช้สต็อกจริง
	plans := make([][]models.PriceListFormulas, len(changes))
	needStock := map[uuid.UUID]bool{}
	for i, c := range changes {
		subGroup, ok := subGroupMap[c.SubGroupID]
		if !ok || (c.TotalNetPriceWeight == nil && c.TotalNetPriceUnit == nil) {
			continue // id ที่ไม่มีอยู่จริง repository จะคืน not found เอง
		}
		plans[i] = dependentFormulas(sortFormulasByDependency(subGroup.SubGroupCode, formulasMap[subGroup.SubGroupCode]), manualVars(c))
		for _, f := range plans[i] {
			req := formulaRequiredVars(f)
			if slices.Contains(req, "weight_spec") || slices.Contains(req, "avg_kg_stock") {
				needStock[c.SubGroupID] = true
			}
		}
	}

	stock, err := loadStockForSave(ctx, subGroupMap, needStock)
	if err != nil {
		return err
	}

	for i := range changes {
		c := &changes[i]
		if len(plans[i]) == 0 {
			continue
		}
		subGroup := subGroupMap[c.SubGroupID]
		inv := stock[c.SubGroupID.String()]
		avgKgStock := 0.0
		if len(inv.InventoryWeight) > 0 {
			avgKgStock = inv.InventoryWeight[0].AvgProduct
		}

		data := priceDomain.PriceData{
			AvgKgStock: avgKgStock,
			WeightSpec: inv.WeightSpec,
			Kg:         subGroup.TotalNetPriceWeight,
			Pcs:        subGroup.TotalNetPriceUnit,
		}
		if c.TotalNetPriceWeight != nil {
			data.Kg = *c.TotalNetPriceWeight
		}
		if c.TotalNetPriceUnit != nil {
			data.Pcs = *c.TotalNetPriceUnit
		}

		dirty := manualVars(*c)
		for _, f := range plans[i] {
			req := formulaRequiredVars(f)
			// สูตรต้นทางถูกข้าม สูตรนี้ก็ไม่ได้รับค่าใหม่ ต้องไม่รัน
			if !slices.ContainsFunc(req, func(v string) bool { return dirty[v] }) {
				continue
			}
			if (slices.Contains(req, "weight_spec") && data.WeightSpec <= 0) ||
				(slices.Contains(req, "avg_kg_stock") && data.AvgKgStock <= 0) {
				continue
			}
			if f.Uom == "pcs" {
				data.BasePrice, data.Extra = subGroup.PriceListGroup.PriceUnit, subGroup.ExtraPriceUnit
			} else {
				data.BasePrice, data.Extra = subGroup.PriceListGroup.PriceWeight, subGroup.ExtraPriceWeight
			}
			if skipFormulaWithoutBasePrice(f, data.BasePrice) {
				continue
			}
			value, err := CalculatePrice(priceDomain.PriceFormula{Expression: f.Expression, Params: f.Params, Rounding: f.Rounding}, data)
			if err != nil {
				return fmt.Errorf("subgroup %s formula %s: %w", c.SubGroupID, f.FormulaCode, err)
			}
			if f.Uom == "pcs" {
				data.Pcs, c.TotalNetPriceUnit = value, &value
			} else {
				data.Kg, c.TotalNetPriceWeight = value, &value
			}
			dirty[f.Uom] = true
		}
	}
	return nil
}

// manualVars คือตัวแปรสูตรที่ผู้ใช้กรอกเองใน change นี้
func manualVars(c models.UpdatePriceListSubGroupItem) map[string]bool {
	vars := map[string]bool{}
	if c.TotalNetPriceWeight != nil {
		vars["kg"] = true
	}
	if c.TotalNetPriceUnit != nil {
		vars["pcs"] = true
	}
	return vars
}

// dependentFormulas คัดสูตร kg/pcs ที่ไม่ใช่ input และขึ้นกับ manual (ตรงหรือต่อทอด) จาก formulas
// ที่เรียงตาม dependency แล้ว · default เป็น input = ไม่คำนวณอะไรเลย เหมือน update-latest
func dependentFormulas(formulas []models.PriceListSubGroupFormulasMap, manual map[string]bool) []models.PriceListFormulas {
	for _, f := range formulas {
		if f.IsDefault && f.PriceListFormulas.FormulaType == "input" {
			return nil
		}
	}
	dirty := maps.Clone(manual)
	var plan []models.PriceListFormulas
	for _, f := range formulas {
		pf := f.PriceListFormulas
		if pf.FormulaType == "input" || manual[pf.Uom] || (pf.Uom != "kg" && pf.Uom != "pcs") {
			continue
		}
		if slices.ContainsFunc(formulaRequiredVars(pf), func(v string) bool { return dirty[v] }) {
			plan = append(plan, pf)
			dirty[pf.Uom] = true
		}
	}
	return plan
}

// loadStockForSave ดึง weight_spec / avg_kg_stock เฉพาะ subgroup ที่ต้องใช้
// error ต้องคืนกลับ ไม่ใช่ log แล้วไปต่อแบบ update-latest เพราะจะได้ราคาจาก fallback ลง DB
func loadStockForSave(ctx context.Context, subGroupMap map[uuid.UUID]*models.PriceListSubGroup, need map[uuid.UUID]bool) (map[string]externalService.InventoryByProductCodeResponse, error) {
	result := map[string]externalService.InventoryByProductCodeResponse{}
	keyValues := []externalService.InventoryByProductCodeKeyValue{}
	companyCode := ""
	siteSet := map[string]bool{}
	for id := range need {
		subGroup := subGroupMap[id]
		if companyCode == "" {
			companyCode = subGroup.PriceListGroup.CompanyCode
		}
		if subGroup.PriceListGroup.SiteCode != "" {
			siteSet[subGroup.PriceListGroup.SiteCode] = true
		}
		for _, k := range subGroup.PriceListSubGroupKeys {
			keyValues = append(keyValues, externalService.InventoryByProductCodeKeyValue{
				ID: id.String(), GroupCode: k.Code, GroupValue: k.Value, Seq: k.Seq,
			})
		}
	}
	if len(keyValues) == 0 {
		return result, nil
	}

	resp, err := getInventoryWeightByKeyFunc(ctx, companyCode, slices.Collect(maps.Keys(siteSet)), keyValues)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory data: %w", err)
	}
	for _, r := range resp {
		result[r.ID] = r
	}
	return result, nil
}

// getValidationErrorMessage converts validator.ValidationError to user-friendly message
func getValidationErrorMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()
	tag := fieldError.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fieldError.Param())
	case "omitempty":
		return fmt.Sprintf("%s is invalid", field)
	case "uuid", "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", field)
	default:
		return fmt.Sprintf("%s failed validation for tag '%s'", field, tag)
	}
}
