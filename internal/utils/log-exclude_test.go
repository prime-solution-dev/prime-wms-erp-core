package utils_test

import (
	"testing"

	"prime-erp-core/internal/utils"
)

// ParseAPILogExcludePrefixes / IsExcludedPath เคยถูกเทสผ่าน outbound transport ของ apilog ซึ่งถูก
// ลบไปแล้ว เทสนี้จึงเข้ามาคุมสองฟังก์ชันตรงๆ ไม่ให้เหลือแค่เทส end-to-end ฝั่ง middleware ตัวเดียว
//
// ที่คุมจริงๆ คือ cron ของ repo นี้ที่ยิงทุกนาที — ถ้า parse หรือ match เพี้ยน MongoDB จะโดนถล่ม
func TestParseAPILogExcludePrefixes(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"ค่าว่างคือไม่ยกเว้นอะไรเลย", "", nil},
		{"มีแต่ comma ก็ยังไม่ยกเว้น", ",,,", []string{}},
		{"ตัวเดียว", "/credit/", []string{"/credit/"}},
		{"หลายตัวพร้อมช่องว่างรอบๆ", " /credit/ , /health ", []string{"/credit/", "/health"}},
		{"ตัวว่างกลางลิสต์ต้องถูกข้าม", "/credit/,,/health", []string{"/credit/", "/health"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := utils.ParseAPILogExcludePrefixes(c.raw)

			if len(got) != len(c.want) {
				t.Fatalf("ได้ %d ตัว %v, ต้องการ %d ตัว %v", len(got), got, len(c.want), c.want)
			}

			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("ตัวที่ %d = %q, ต้องการ %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestIsExcludedPath(t *testing.T) {
	prefixes := utils.ParseAPILogExcludePrefixes("/credit/,/health")

	cases := []struct {
		path string
		want bool
	}{
		{"/credit/GetCreditRequestCronjob", true},
		{"/credit/", true},
		{"/health", true},
		{"/healthz", true}, // prefix match ไม่ใช่ exact — ตั้งใจให้เป็นแบบนี้
		{"/credit", false}, // สั้นกว่า prefix จึงไม่ตรง
		{"/orders/create", false},
		{"", false},
	}

	for _, c := range cases {
		if got := utils.IsExcludedPath(c.path, prefixes); got != c.want {
			t.Errorf("IsExcludedPath(%q) = %v, ต้องการ %v", c.path, got, c.want)
		}
	}
}

// ไม่ได้ตั้ง API_LOG_EXCLUDE ไว้เลย = ไม่มี prefix = ต้องไม่ยกเว้น path ไหนทั้งนั้น
// (เคสนี้สำคัญ เพราะถ้าพลาดเป็น "ยกเว้นหมด" log จะหายเงียบทั้ง service)
func TestIsExcludedPathWithNoPrefixesExcludesNothing(t *testing.T) {
	for _, path := range []string{"/orders/create", "/credit/GetCredit", "/"} {
		if utils.IsExcludedPath(path, nil) {
			t.Errorf("IsExcludedPath(%q, nil) = true, ต้องเป็น false", path)
		}
	}
}
