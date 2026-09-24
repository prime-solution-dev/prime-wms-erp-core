package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bannedPattern is one legacy pattern this migration removed, plus what to use instead.
type bannedPattern struct {
	pattern string
	reason  string
}

// httpNewRequestExempt lists the only files allowed to keep http.NewRequest( in
// production code, with the reason written out so nobody "fixes" it by accident.
//
// get-deposit.go and get-deposits.go post form-urlencoded bodies to
// https://tmi.trcloud.co/... (get-deposit.go) and to a caller-supplied URL
// (get-deposits.go). Routing those through utils.NewRequest(ctx, ...) would
// forward the calling employee's JWT to an external, non-organization host,
// and utils.NewRequest sets a JSON Content-Type instead of the
// application/x-www-form-urlencoded these endpoints require. Keep them on the
// raw net/http client.
var httpNewRequestExempt = map[string]bool{
	"interface-service/get-deposit.go":  true,
	"interface-service/get-deposits.go": true,
}

// ห้ามกลับไปใช้ของเดิมที่ทำให้ user หายระหว่างทาง หรือ token ไม่ถูกส่งต่อ
func TestNoLegacyContextUsage(t *testing.T) {
	banned := []bannedPattern{
		{
			pattern: `http.NewRequest(`,
			reason:  "ใช้ utils.NewRequest(ctx, ...) แทน ไม่งั้น token ไม่ถูกส่งต่อ",
		},
		{
			pattern: `ctx.GetString("user")`,
			reason:  "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
		},
		{
			pattern: `c.GetString("user")`,
			reason:  "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
		},
		{
			pattern: `ctx.Get("user")`,
			reason:  "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
		},
		{
			pattern: `c.Get("user")`,
			reason:  "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
		},
		{
			pattern: `gin.Context{`,
			reason:  "ห้ามปลอม gin.Context เอง — รับ context.Context จริงจาก caller แทน (บริดจ์แบบนี้เคยลองแล้วโดน revert)",
		},
		{
			pattern: `gin.CreateTestContext`,
			reason:  "ห้ามปลอม gin.Context เอง — รับ context.Context จริงจาก caller แทน (บริดจ์แบบนี้เคยลองแล้วโดน revert)",
		},
	}

	// roots ต้องเป็น production code ของ services กับ external เท่านั้น — ไม่รวม cmd/routes/utils
	// ที่ยังต้องแตะ gin.Context ตรงๆ เพื่อทำสะพานให้ service ชั้นในไม่ต้องรู้จัก gin
	roots := []string{"../services", "../../external"}

	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(content)
			slashPath := filepath.ToSlash(path)

			for _, bp := range banned {
				if bp.pattern == `http.NewRequest(` && isHTTPNewRequestExempt(slashPath) {
					continue
				}

				if strings.Contains(text, bp.pattern) {
					t.Errorf("%s: เจอ %q — %s", path, bp.pattern, bp.reason)
				}
			}

			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

func isHTTPNewRequestExempt(slashPath string) bool {
	for suffix := range httpNewRequestExempt {
		if strings.HasSuffix(slashPath, suffix) {
			return true
		}
	}
	return false
}
