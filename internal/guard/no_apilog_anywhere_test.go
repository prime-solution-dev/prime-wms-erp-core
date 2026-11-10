package guard

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// นโยบายของทีม: repo นี้ใช้ **prime-service-x/servicelog ตัวเดียว** ส่วน apilog เป็นของอีกทีม
// ห้ามเอากลับเข้ามา
//
// apilog เคยอยู่ใน repo นี้จริง (commit e91a06d, 27faa6b) ทำหน้าที่บันทึก request ที่ erp-core ยิง
// ออกไปหา service อื่นลง MongoDB แล้วถูกถอดออกทั้งหมด เหลือ utils.NewOutboundLogTransport ไว้เป็น
// shim ที่คืน http.DefaultTransport เฉยๆ เพื่อไม่ต้องแก้ call site 33 จุด (ดู
// internal/utils/log-transport.go)
//
// รั้วนี้เดิน AST อ่าน import จริงของทุกไฟล์ .go ไม่ได้ค้นสตริง เพราะคอมเมนต์หลายที่ในโค้ด
// (รวมไฟล์นี้) พูดถึงชื่อ apilog ตรงๆ เพื่ออธิบายว่าทำไมมันไม่อยู่แล้ว การค้นสตริงเปล่าๆ จะฟ้อง
// คอมเมนต์ของตัวเอง
const (
	// apilogImportPath คือ path ที่ห้ามปรากฏใน import ของไฟล์ไหนก็ตามใน repo นี้
	apilogImportPath = "github.com/prime-solution-dev/prime-service-x/apilog"

	// servicelogImportPath คือตัวที่อนุญาต — มีไว้ให้รั้วพิสูจน์ได้ว่ามัน "เห็น" import ของ
	// prime-service-x จริง ไม่ใช่เขียวเพราะ parse ไม่เจออะไรเลย
	servicelogImportPath = "github.com/prime-solution-dev/prime-service-x/servicelog"

	// minParsedGoFiles คือพื้นที่เป็น "ค่าคงที่" ไม่ใช่ len() ของอะไร — ถ้า walker พัง (root ผิด,
	// filter กินไฟล์หมด, parse error เงียบ) จำนวนไฟล์ที่อ่านได้จะร่วงต่ำกว่านี้แล้วรั้วต้องแดง
	// ไม่ใช่เขียวเพราะไม่ได้ตรวจอะไร repo นี้มี 312 ไฟล์ตอนเขียนรั้ว เผื่อให้ลบได้พอสมควร
	minParsedGoFiles = 250

	// minServicelogImporters คือจำนวนไฟล์ที่ต้อง import servicelog เป็นอย่างน้อย — ถ้าเป็น 0
	// แปลว่ารั้วอ่าน import ไม่ออกเลย (เช่นเทียบ path ผิดรูป) ไม่ใช่ว่า repo สะอาด
	minServicelogImporters = 2
)

// repoRoot คือรากของ repo เทียบจาก internal/guard (ที่ไฟล์เทสนี้อยู่)
const repoRoot = "../.."

func TestNoApilogImportAnywhereInRepo(t *testing.T) {
	fset := token.NewFileSet()

	parsed := 0
	servicelogImporters := 0
	offenders := []string{}

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" || name == ".superpowers" {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		// ParseFile โหมด ImportsOnly — เร็วกว่าและพอสำหรับงานนี้
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			// ไฟล์ .go ที่ parse ไม่ผ่านต้องดัง ไม่ใช่ข้ามเงียบๆ เพราะไฟล์ที่ข้ามไปคือไฟล์ที่
			// รั้วไม่ได้ตรวจ
			t.Errorf("parse %s: %v", path, perr)
			return nil
		}

		parsed++

		for _, imp := range f.Imports {
			got := strings.Trim(imp.Path.Value, `"`)

			switch got {
			case apilogImportPath:
				offenders = append(offenders, path)
			case servicelogImportPath:
				servicelogImporters++
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("เดินไฟล์ไม่สำเร็จ: %v", err)
	}

	if parsed < minParsedGoFiles {
		t.Fatalf("อ่านไฟล์ .go ได้แค่ %d ไฟล์ (ต้องอย่างน้อย %d) — รั้วนี้ไม่ได้ตรวจอะไรเลย ไม่ใช่ว่า repo สะอาด", parsed, minParsedGoFiles)
	}

	if servicelogImporters < minServicelogImporters {
		t.Fatalf("เจอไฟล์ที่ import servicelog แค่ %d ไฟล์ (ต้องอย่างน้อย %d) — รั้วอ่าน import ของ prime-service-x ไม่ออก เทียบ path ผิดรูปหรือเปล่า", servicelogImporters, minServicelogImporters)
	}

	if len(offenders) != 0 {
		t.Fatalf("เจอ import %q ใน %d ไฟล์: %v\n\nrepo นี้ใช้ servicelog ตัวเดียว apilog เป็นของอีกทีม ถ้าต้องการ outbound logging จริงๆ ให้คุยกับเจ้าของ repo ก่อน อย่าเติมกลับมาเฉยๆ", apilogImportPath, len(offenders), offenders)
	}
}
