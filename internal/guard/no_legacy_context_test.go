package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bannedPattern is one legacy pattern this migration removed, plus what to use instead.
// exemptFileSuffixes is an explicit, narrowly-scoped allow-list for that one pattern only —
// a file listed here is still banned from every other pattern in this test.
type bannedPattern struct {
	pattern            string
	reason             string
	exemptFileSuffixes []string
}

// ห้ามกลับไปใช้ของเดิมที่ทำให้ user หายระหว่างทาง หรือ token ไม่ถูกส่งต่อ
func TestNoLegacyContextUsage(t *testing.T) {
	banned := []bannedPattern{
		{
			pattern: `http.NewRequest(`,
			reason:  "ใช้ utils.NewRequest(ctx, ...) แทน ไม่งั้น token ไม่ถูกส่งต่อ",
			// get-deposit.go และ get-deposits.go ยิง form-urlencoded body ไปที่
			// https://tmi.trcloud.co/... (get-deposit.go) และไป URL ที่ผู้เรียกส่งมาเอง
			// (get-deposits.go) ถ้าเปลี่ยนไปใช้ utils.NewRequest(ctx, ...) จะพ่วง JWT ของ
			// พนักงานออกไปนอกองค์กร แถม Content-Type ก็ผิด (utils.NewRequest ตั้ง JSON แต่
			// สองเส้นนี้ต้อง application/x-www-form-urlencoded) อย่า "แก้ให้เหมือนที่อื่น" ตรงนี้
			exemptFileSuffixes: []string{
				"interface-service/get-deposit.go",
				"interface-service/get-deposits.go",
			},
		},
		{
			pattern: `ctx.GetString("user")`,
			reason:  "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
		},
		{
			pattern: `c.GetString("user")`,
			reason:  "ใช้ requestcontext.GetUserOrDefault(ctx) แทน",
			// request-handler-gin.go: buildContext() คือจุดเดียวในระบบที่ค่าข้ามจาก gin
			// เข้า context.Context — เป็นสะพาน gin→context ตัวเดียวที่มีสิทธิ์อ่านที่เก็บของ gin
			// เอง (c.GetString) และอ่านเฉพาะตอนที่ context ยังไม่มี user เลย (กันไว้เผื่อมี
			// middleware ที่ยังทำแค่ c.Set ไม่ได้ยัดลง context) พฤติกรรมนี้ถูกพินไว้ด้วย
			// TestProcessContextRequestBridgesLegacyGinUser แล้ว โค้ดที่เหลือทั้งหมดที่อยาก
			// ได้ user ให้เรียก requestcontext.GetUserOrDefault(ctx) — ยกเว้นให้แค่ pattern
			// นี้ pattern เดียว ไฟล์นี้ยังห้าม c.Get("user")/ctx.Get("user")/ctx.GetString("user")
			// เหมือนที่อื่นทุกประการ
			exemptFileSuffixes: []string{
				"utils/request-handler-gin.go",
			},
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

	// roots คือทุกอย่างใต้ internal กับ external — ไม่ใช่แค่ services เพราะรั้วที่มีรู
	// (เช่น เว้น routes/middleware/utils) สอนคนว่ารูนั้นใช้ได้ ยกเว้น internal/guard เอง
	// ที่ต้องข้ามไม่งั้นสตริง pattern ในไฟล์นี้จะทำให้เทสฟ้องตัวเอง
	roots := []string{"..", "../../external"}

	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			slashPathForSkip := filepath.ToSlash(path)
			if strings.HasPrefix(slashPathForSkip, "../guard/") || slashPathForSkip == "../guard" {
				return nil
			}

			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(content)
			slashPath := filepath.ToSlash(path)

			for _, bp := range banned {
				if isExempt(slashPath, bp.exemptFileSuffixes) {
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

// isExempt checks a file against one pattern's own allow-list only — never a blanket
// per-file exemption. A file exempted for one pattern is still checked against every
// other pattern in the list.
func isExempt(slashPath string, exemptFileSuffixes []string) bool {
	for _, suffix := range exemptFileSuffixes {
		if strings.HasSuffix(slashPath, suffix) {
			return true
		}
	}
	return false
}
