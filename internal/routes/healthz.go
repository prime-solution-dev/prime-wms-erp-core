package routes

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RegisterHealthz adds GET /healthz (liveness: the process is up and serving).
// Call it BEFORE middleware.RegisterMiddlewares: Gin binds middleware to a route when the
// route is registered, so registering first keeps /healthz outside auth and request logging.
func RegisterHealthz(engine *gin.Engine) {
	engine.GET("/healthz", func(c *gin.Context) {
		if c.Query("action") == "version" {
			info, _ := debug.ReadBuildInfo()
			c.JSON(http.StatusOK, healthzVersion(info))
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// healthzVersion ดึง commit จากข้อมูลที่ go build ฝังไว้เอง (vcs.*) — มีเมื่อ build จากโฟลเดอร์ที่มี .git
// ซึ่ง Dockerfile ทุกตัว COPY เข้าไปอยู่แล้ว; sha ว่าง = build แบบไม่มี .git (เช่น go test)
// modified = ตอน build มีไฟล์ต่างจาก commit (deploy script เขียน cmd/.env ทับ จึงเป็น true ได้เสมอ)
func healthzVersion(info *debug.BuildInfo) gin.H {
	v := gin.H{"status": "ok", "sha": "", "time": "", "modified": false}
	if info == nil {
		return v
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			v["sha"] = s.Value
		case "vcs.time":
			v["time"] = s.Value
		case "vcs.modified":
			v["modified"] = s.Value == "true"
		}
	}
	return v
}
