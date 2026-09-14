package priceService

import (
	"math"
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

// extraConditionMatched เป็นตัวตัดสินราคาจริง operator ที่รับค่าตัวเดียวต้องอ่าน
// cond_range_max ให้ตรงกับช่องที่หน้าจอเปิดให้กรอก (ExtraPriceTable.vue ล็อคช่อง
// min ไว้ให้กรอกได้เฉพาะ "<>") · เดิม ">" อ่าน min ที่ถูกล็อคเป็น 0 ทำให้ "> 38"
// กลายเป็น "> 0" คือ match ทุกแถว
func TestExtraConditionMatched(t *testing.T) {
	tests := []struct {
		name     string
		val      float64
		operator string
		min, max float64
		want     bool
	}{
		// "> 38" ที่หน้าจอส่งมาเป็น min=0, max=38
		{"> ต่ำกว่าขอบ", 32, ">", 0, 38, false},
		{"> ที่ขอบพอดีต้องไม่ตรง", 38, ">", 0, 38, false},
		{"> เหนือขอบ", 40, ">", 0, 38, true},
		{"> ไม่สนใจค่า min ที่ตกค้าง", 40, ">", 999, 38, true},

		{">= ต่ำกว่าขอบ", 37, ">=", 0, 38, false},
		{">= ที่ขอบพอดีต้องตรง", 38, ">=", 0, 38, true},
		{">= เหนือขอบ", 40, ">=", 0, 38, true},

		{"< ต่ำกว่าขอบ", 37, "<", 0, 38, true},
		{"< ที่ขอบพอดีต้องไม่ตรง", 38, "<", 0, 38, false},
		{"< เหนือขอบ", 40, "<", 0, 38, false},

		{"<= ที่ขอบพอดีต้องตรง", 38, "<=", 0, 38, true},
		{"<= เหนือขอบ", 39, "<=", 0, 38, false},

		{"= ตรงค่า", 38, "=", 0, 38, true},
		{"= ไม่ตรงค่า", 37, "=", 0, 38, false},

		// "<> 30 to 38" ใช้ทั้งสองขอบและนับขอบเป็นของตัวเอง
		{"<> ต่ำกว่าช่วง", 25, "<>", 30, 38, false},
		{"<> ขอบล่าง", 30, "<>", 30, 38, true},
		{"<> กลางช่วง", 32, "<>", 30, 38, true},
		{"<> ขอบบน", 38, "<>", 30, 38, true},
		{"<> เหนือช่วง", 40, "<>", 30, 38, false},

		{"operator ไม่รู้จักต้องไม่ match", 38, "to", 30, 38, false},
		{"operator ว่างต้องไม่ match", 38, "", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extraConditionMatched(tt.val, tt.operator, tt.min, tt.max)
			if got != tt.want {
				t.Fatalf("extraConditionMatched(%v, %q, %v, %v) = %v ต้องได้ %v",
					tt.val, tt.operator, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

// ช่วงที่ติดกันตามเจตนาอย่าง "<> 30..38" คู่กับ "> 38" ต้องไม่ถือว่าทับกัน
// ไม่งั้นผู้ใช้ตั้งค่าถูกแล้วยังกด Update ไม่ผ่าน
func TestRangesOverlap(t *testing.T) {
	r := func(op string, min, max float64) condRange { return effectiveRange(op, min, max) }

	tests := []struct {
		name string
		a, b condRange
		want bool
	}{
		{"<> 30..38 กับ > 38 ติดกันไม่ทับ", r("<>", 30, 38), r(">", 0, 38), false},
		{"<> 30..38 กับ >= 38 ทับที่ 38", r("<>", 30, 38), r(">=", 0, 38), true},
		{"<> 30..38 กับ <> 35..40 ทับ", r("<>", 30, 38), r("<>", 35, 40), true},
		{"<> 30..38 กับ <> 39..45 ไม่ทับ", r("<>", 30, 38), r("<>", 39, 45), false},
		{"< 30 กับ >= 30 ติดกันไม่ทับ", r("<", 0, 30), r(">=", 0, 30), false},
		{"<= 30 กับ >= 30 ทับที่ 30", r("<=", 0, 30), r(">=", 0, 30), true},
		{"= 38 กับ <> 30..38 ทับ", r("=", 0, 38), r("<>", 30, 38), true},
		{"= 38 กับ = 40 ไม่ทับ", r("=", 0, 38), r("=", 0, 40), false},
		{"condition ว่างทับกับทุกช่วง", r("", 0, 0), r("<>", 30, 38), true},
		// operator ที่ไม่รู้จักถูก validateExtras ปฏิเสธไปก่อน ช่วงของมันต้องว่าง
		// ไม่ใช่ ±∞ ซึ่งจะกลายเป็น "ทับทุกช่วง" สวนทางกับ extraConditionMatched
		{"operator ไม่รู้จักไม่ทับกับอะไร", r("to", 30, 38), r("<>", 30, 38), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rangesOverlap(tt.a, tt.b); got != tt.want {
				t.Fatalf("rangesOverlap(%+v, %+v) = %v ต้องได้ %v", tt.a, tt.b, got, tt.want)
			}
			// การทับกันต้องสมมาตร ไม่ขึ้นกับลำดับที่ส่งเข้าไป
			if got := rangesOverlap(tt.b, tt.a); got != tt.want {
				t.Fatalf("สลับลำดับแล้วได้ %v ต้องได้ %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveRange_SingleOperandUsesMax(t *testing.T) {
	got := effectiveRange(">", 999, 38)
	if got.lo != 38 || !math.IsInf(got.hi, 1) || got.loInc {
		t.Fatalf("\"> 38\" ต้องเป็น (38, ∞) แต่ได้ %+v", got)
	}

	got = effectiveRange("<=", 999, 38)
	if !math.IsInf(got.lo, -1) || got.hi != 38 || !got.hiInc {
		t.Fatalf("\"<= 38\" ต้องเป็น (-∞, 38] แต่ได้ %+v", got)
	}
}

// checkForOverlappingConditions จัดกลุ่มด้วย group + extra_key + condition_code
// แถวที่ extra_key ต่างกันคนละกลุ่มสินค้า จึงตั้งช่วงทับกันได้โดยไม่ถือว่าชน
func TestCheckForOverlappingConditions(t *testing.T) {
	groupID := uuid.New()
	extra := func(extraKey, operator string, min, max float64) models.UpdatePriceListExtraRequest {
		return models.UpdatePriceListExtraRequest{
			PriceListGroupID: groupID,
			ExtraKey:         extraKey,
			ConditionCode:    "PG06",
			Operator:         operator,
			CondRangeMin:     min,
			CondRangeMax:     max,
		}
	}

	tests := []struct {
		name      string
		extras    []models.UpdatePriceListExtraRequest
		wantError bool
	}{
		{
			// เคสจริงที่ผู้ใช้ต้องตั้งได้: ช่วงหนึ่งจบที่ 38 อีกช่วงเริ่มเหนือ 38
			name: "extra_key เดียวกัน <> 30..38 กับ > 38 ติดกันต้องผ่าน",
			extras: []models.UpdatePriceListExtraRequest{
				extra("PG03_18", "<>", 30, 38),
				extra("PG03_18", ">", 0, 38),
			},
		},
		{
			name: "extra_key เดียวกัน <> 30..38 กับ >= 38 ทับที่ 38 ต้องไม่ผ่าน",
			extras: []models.UpdatePriceListExtraRequest{
				extra("PG03_18", "<>", 30, 38),
				extra("PG03_18", ">=", 0, 38),
			},
			wantError: true,
		},
		{
			name: "extra_key เดียวกัน ช่วงซ้อนกันต้องไม่ผ่าน",
			extras: []models.UpdatePriceListExtraRequest{
				extra("PG03_18", "<>", 30, 38),
				extra("PG03_18", "<>", 35, 45),
			},
			wantError: true,
		},
		{
			name: "extra_key ต่างกัน ช่วงเดียวกันต้องผ่าน",
			extras: []models.UpdatePriceListExtraRequest{
				extra("PG03_18", "<>", 30, 38),
				extra("PG03_21", "<>", 30, 38),
			},
		},
		{
			name:   "แถวเดียวต้องผ่าน",
			extras: []models.UpdatePriceListExtraRequest{extra("PG03_18", "<>", 30, 38)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkForOverlappingConditions(tt.extras)
			if tt.wantError && err == nil {
				t.Fatal("ต้องได้ error เรื่องช่วงทับกัน แต่ผ่านไปได้")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("ต้องผ่าน แต่ได้ error: %v", err)
			}
		})
	}
}

// condition_code กับ operator ต้องมีหรือไม่มีพร้อมกัน
// group ที่ config ไม่มีแกน condition (เช่น หมวดตัวซี: PG01, PG04) มี condition_code
// ว่างเสมอ ถ้าบังคับว่าห้ามว่างหน้าจอนั้นจะกด Update ไม่ผ่านถาวร
func TestValidateExtras_ConditionAndOperatorArePaired(t *testing.T) {
	build := func(conditionCode, operator string) []models.UpdatePriceListExtraRequest {
		return []models.UpdatePriceListExtraRequest{{
			PriceListGroupID: uuid.New(),
			ExtraKey:         "PG01_7|PG04_65",
			ConditionCode:    conditionCode,
			Operator:         operator,
			ValueInt:         0.1,
			PriceListGroupExtraKeys: []models.UpdatePriceListGroupExtraKeyRequest{
				{Code: "PG01", Value: "PG01_7", Seq: 1},
			},
		}}
	}

	if err := validateExtras(build("", "")); err != nil {
		t.Fatalf("condition ว่างคู่กับ operator ว่างต้องผ่าน แต่ถูกปฏิเสธ: %v", err)
	}
	if err := validateExtras(build("", "<>")); err == nil {
		t.Fatal("มี operator แต่ไม่มี condition ต้องถูกปฏิเสธ")
	}
	if err := validateExtras(build("PG06", "")); err == nil {
		t.Fatal("มี condition แต่ไม่มี operator ต้องถูกปฏิเสธ")
	}
	if err := validateExtras(build("PG06", "<>")); err != nil {
		t.Fatalf("condition คู่กับ operator ที่ถูกต้องต้องผ่าน แต่ถูกปฏิเสธ: %v", err)
	}
}

// helper สร้าง subgroup พร้อม key และ extra config ของ group
func subGroupWith(keys map[string]string, extras []models.PriceListGroupExtra, currentExtra float64) *models.PriceListSubGroup {
	sgKeys := make([]models.PriceListSubGroupKey, 0, len(keys))
	for code, value := range keys {
		sgKeys = append(sgKeys, models.PriceListSubGroupKey{Code: code, Value: value})
	}
	return &models.PriceListSubGroup{
		ExtraPriceWeight:      currentExtra,
		ExtraPriceUnit:        currentExtra,
		PriceListSubGroupKeys: sgKeys,
		PriceListGroup:        models.PriceListGroup{PriceListGroupExtras: extras},
	}
}

// pg06Values ห่อ value_int ของ PG06 ให้อยู่ในรูป map ที่ calculateExtraForSubGroup รับ
func pg06Values(values map[string]float64) groupItemValueInts {
	return groupItemValueInts{"PG06": values}
}

// ค่า extra เก่าที่ไม่ตรงเงื่อนไขใดเลยต้องถูกล้างเป็น 0
// เดิมเริ่มจากค่าเดิมแล้วเขียนทับเฉพาะตอน match ทำให้ค่าที่อัปโหลดมาค้างถาวร
// ข้อมูลจริงเคยมี LT ขนาด 40 ที่ได้ทั้ง 0 และ 1 ปนกันเพราะสาเหตุนี้
func TestCalculateExtraForSubGroup_ClearsStaleExtraWhenNothingMatches(t *testing.T) {
	groupItemValues := pg06Values(map[string]float64{"PG06_40": 40})

	extras := []models.PriceListGroupExtra{{
		ExtraKey:      "PG03_18",
		ConditionCode: "PG06",
		Operator:      "<>",
		CondRangeMin:  30,
		CondRangeMax:  38,
		ValueInt:      1,
		PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
			{Code: "PG03", Value: "PG03_18"},
		},
	}}

	sg := subGroupWith(map[string]string{"PG03": "PG03_18", "PG06": "PG06_40"}, extras, 1)

	weight, unit := calculateExtraForSubGroup(sg, groupItemValues)
	if weight != 0 || unit != 0 {
		t.Fatalf("ขนาด 40 อยู่นอกช่วง 30..38 ต้องได้ 0 แต่ได้ weight=%v unit=%v", weight, unit)
	}
}

// group ที่ยังไม่ได้ตั้ง extra ไว้เลย ต้องคงค่าที่อัปโหลดมาตามเดิม
func TestCalculateExtraForSubGroup_KeepsUploadedValueWhenNoExtrasConfigured(t *testing.T) {
	sg := subGroupWith(map[string]string{"PG03": "PG03_18"}, nil, 2.5)
	groupItemValues := groupItemValueInts{}

	weight, unit := calculateExtraForSubGroup(sg, groupItemValues)
	if weight != 2.5 || unit != 2.5 {
		t.Fatalf("group ที่ไม่มี extra config ต้องคงค่าเดิม 2.5 แต่ได้ weight=%v unit=%v", weight, unit)
	}
}

// subgroup ที่ไม่มี rule ไหน key ตรงเลย ต้องคงค่าที่อัปโหลดมา ไม่ใช่ถูกล้างเป็น 0
//
// ต่างจาก "มี rule แต่ไม่เข้าเงื่อนไข" ซึ่งต้องล้าง · ถ้าล้างเคสนี้ด้วย ข้อมูลของ
// เกรดที่ไม่ได้ตั้ง rule ไว้จะหายทั้งหมดตอน cascadeBasePriceToSubGroups ทำงาน
// และกู้คืนไม่ได้เพราะไม่มีแหล่งข้อมูลอื่น
func TestCalculateExtraForSubGroup_KeepsUploadedValueWhenNoRuleMatchesKeys(t *testing.T) {
	groupItemValues := pg06Values(map[string]float64{"PG06_40": 40})

	extras := []models.PriceListGroupExtra{{
		ExtraKey:      "PG03_18",
		ConditionCode: "PG06",
		Operator:      "<>",
		CondRangeMin:  30,
		CondRangeMax:  38,
		ValueInt:      1,
		PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
			{Code: "PG03", Value: "PG03_18"},
		},
	}}

	// เกรด PG03_99 ไม่มี rule ควบคุมอยู่เลย
	sg := subGroupWith(map[string]string{"PG03": "PG03_99", "PG06": "PG06_40"}, extras, 2.5)

	weight, unit := calculateExtraForSubGroup(sg, groupItemValues)
	if weight != 2.5 || unit != 2.5 {
		t.Fatalf("subgroup ที่ไม่มี rule ตรง key ต้องคงค่าเดิม 2.5 แต่ได้ weight=%v unit=%v", weight, unit)
	}
}

// มี rule ที่ key ตรง แต่ subgroup ไม่มี key ของแกน condition อยู่เลย
// ต้องไม่บวก extra และต้องล้างค่าเก่า เพราะถือว่ามี rule ควบคุมอยู่
func TestCalculateExtraForSubGroup_ClearsWhenConditionKeyMissing(t *testing.T) {
	groupItemValues := pg06Values(map[string]float64{"PG06_32": 32})

	extras := []models.PriceListGroupExtra{{
		ExtraKey:      "PG03_18",
		ConditionCode: "PG06",
		Operator:      "<>",
		CondRangeMin:  30,
		CondRangeMax:  38,
		ValueInt:      1,
		PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
			{Code: "PG03", Value: "PG03_18"},
		},
	}}

	sg := subGroupWith(map[string]string{"PG03": "PG03_18"}, extras, 1)

	weight, _ := calculateExtraForSubGroup(sg, groupItemValues)
	if weight != 0 {
		t.Fatalf("subgroup ไม่มี key ของแกน condition ต้องได้ 0 แต่ได้ %v", weight)
	}
}

// เงื่อนไขที่ตรงต้องบวก extra เข้าไป — เคสจริงจาก GROUP_1_ITEM_1 (S4 "<> 30..38")
func TestCalculateExtraForSubGroup_AppliesMatchingRange(t *testing.T) {
	groupItemValues := pg06Values(map[string]float64{
		"PG06_30": 30, "PG06_32": 32, "PG06_38": 38, "PG06_40": 40,
	})

	extras := []models.PriceListGroupExtra{{
		ExtraKey:      "PG03_18",
		ConditionCode: "PG06",
		Operator:      "<>",
		CondRangeMin:  30,
		CondRangeMax:  38,
		ValueInt:      1,
		PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
			{Code: "PG03", Value: "PG03_18"},
		},
	}}

	for _, tc := range []struct {
		size string
		want float64
	}{
		{"PG06_30", 1}, {"PG06_32", 1}, {"PG06_38", 1}, {"PG06_40", 0},
	} {
		t.Run(tc.size, func(t *testing.T) {
			sg := subGroupWith(map[string]string{"PG03": "PG03_18", "PG06": tc.size}, extras, 0)
			weight, _ := calculateExtraForSubGroup(sg, groupItemValues)
			if weight != tc.want {
				t.Fatalf("ขนาด %s ต้องได้ extra %v แต่ได้ %v", tc.size, tc.want, weight)
			}
		})
	}
}

// เคสจริงจาก GROUP_1_ITEM_1 (LT "> 38") ต้องเริ่มบวกที่ 40 ไม่ใช่ที่ 38
func TestCalculateExtraForSubGroup_GreaterThanIsExclusive(t *testing.T) {
	groupItemValues := pg06Values(map[string]float64{
		"PG06_32": 32, "PG06_38": 38, "PG06_40": 40,
	})

	extras := []models.PriceListGroupExtra{{
		ExtraKey:      "PG03_21",
		ConditionCode: "PG06",
		Operator:      ">",
		CondRangeMin:  0, // หน้าจอล็อคช่องนี้เป็น 0 เสมอ ต้องไม่ถูกใช้
		CondRangeMax:  38,
		ValueInt:      1,
		PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
			{Code: "PG03", Value: "PG03_21"},
		},
	}}

	for _, tc := range []struct {
		size string
		want float64
	}{
		{"PG06_32", 0}, {"PG06_38", 0}, {"PG06_40", 1},
	} {
		t.Run(tc.size, func(t *testing.T) {
			sg := subGroupWith(map[string]string{"PG03": "PG03_21", "PG06": tc.size}, extras, 0)
			weight, _ := calculateExtraForSubGroup(sg, groupItemValues)
			if weight != tc.want {
				t.Fatalf("ขนาด %s ต้องได้ extra %v แต่ได้ %v", tc.size, tc.want, weight)
			}
		})
	}
}

// extra ที่ไม่มี condition (group อย่างหมวดตัวซี) ต้องบวกทันทีเมื่อ key ตรงครบ
// เดิมถูก continue ข้ามไป ทำให้ extra ของ group เหล่านี้ไม่เคยถูกใช้เลย
func TestCalculateExtraForSubGroup_AppliesExtraWithoutCondition(t *testing.T) {
	extras := []models.PriceListGroupExtra{{
		ExtraKey:      "PG01_7|PG04_65",
		ConditionCode: "",
		Operator:      "",
		ValueInt:      0.1,
		PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
			{Code: "PG01", Value: "PG01_7"},
			{Code: "PG04", Value: "PG04_65"},
		},
	}}

	matching := subGroupWith(map[string]string{"PG01": "PG01_7", "PG04": "PG04_65"}, extras, 0)
	groupItemValues := groupItemValueInts{}
	weight, unit := calculateExtraForSubGroup(matching, groupItemValues)
	if weight != 0.1 || unit != 0.1 {
		t.Fatalf("key ตรงครบและไม่มีเงื่อนไข ต้องได้ 0.1 แต่ได้ weight=%v unit=%v", weight, unit)
	}

	other := subGroupWith(map[string]string{"PG01": "PG01_7", "PG04": "PG04_99"}, extras, 0)
	weight, _ = calculateExtraForSubGroup(other, groupItemValues)
	if weight != 0 {
		t.Fatalf("key ไม่ตรงต้องได้ 0 แต่ได้ %v", weight)
	}
}

// ไฟล์อัปโหลดถูก export จากตารางหน้าเว็บซึ่งแสดง BETWEEN เป็น label "to"
func TestNormalizeExtraOperator(t *testing.T) {
	for raw, want := range map[string]string{
		"to":   "<>",
		"To":   "<>",
		"TO":   "<>",
		" to ": "<>",
		"ถึง":  "<>",
		"<>":   "<>",
		">=":   ">=",
		"":     "",
	} {
		if got := normalizeExtraOperator(raw); got != want {
			t.Fatalf("normalizeExtraOperator(%q) = %q ต้องได้ %q", raw, got, want)
		}
	}
}

// extra ที่ไม่มี key เลย (ข้อมูลที่เสียจากบั๊ก extra_key ซ้ำใน upload path) เคยผ่าน
// การจับคู่ "ตรงทุกคีย์" โดยอัตโนมัติ เพราะ matchedAllKeys เริ่มที่ true แล้ววนลูป 0 รอบ
// จึงบวกให้ทุก subgroup ที่เข้าเงื่อนไขโดยไม่สนกลุ่มสินค้า (ITEM_4 บวกผิด 78/207 subgroup)
func TestCalculateExtraForSubGroup_SkipsExtraWithoutKeys(t *testing.T) {
	extras := []models.PriceListGroupExtra{{
		ExtraKey:                "PG01_7|PG04_65",
		ConditionCode:           "",
		ValueInt:                0.1,
		PriceListGroupExtraKeys: nil,
	}}

	sg := subGroupWith(map[string]string{"PG01": "PG01_99", "PG04": "PG04_99"}, extras, 5)
	weight, unit := calculateExtraForSubGroup(sg, groupItemValueInts{})
	if weight != 5 || unit != 5 {
		t.Fatalf("extra ที่ไม่มีคีย์ต้องถูกข้าม และคงค่าเดิม 5 ไว้ แต่ได้ weight=%v unit=%v", weight, unit)
	}
}

// extra ที่ไม่มีคีย์ต้องไม่ถูกนับเป็น rule ที่ควบคุม subgroup นี้ จึงต้องไม่ไป trigger
// การรีเซ็ตเป็น 0 ของ subgroup ที่ไม่มี rule ไหนคุมอยู่จริง
func TestCalculateExtraForSubGroup_KeylessExtraDoesNotGovern(t *testing.T) {
	extras := []models.PriceListGroupExtra{
		{ExtraKey: "x", ValueInt: 0.1},
		{
			ExtraKey: "PG01_7|PG04_65",
			ValueInt: 0.2,
			PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
				{Code: "PG01", Value: "PG01_7"},
				{Code: "PG04", Value: "PG04_65"},
			},
		},
	}

	governed := subGroupWith(map[string]string{"PG01": "PG01_7", "PG04": "PG04_65"}, extras, 9)
	if weight, _ := calculateExtraForSubGroup(governed, groupItemValueInts{}); weight != 0.2 {
		t.Fatalf("subgroup ที่ตรง rule จริงต้องได้ 0.2 แต่ได้ %v", weight)
	}

	ungoverned := subGroupWith(map[string]string{"PG01": "PG01_1", "PG04": "PG04_1"}, extras, 9)
	if weight, _ := calculateExtraForSubGroup(ungoverned, groupItemValueInts{}); weight != 9 {
		t.Fatalf("subgroup ที่ไม่มี rule คุมต้องคงค่าเดิม 9 แต่ได้ %v", weight)
	}
}
