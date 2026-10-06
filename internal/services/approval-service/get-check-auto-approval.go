package approvalService

import (
	"context"
	"encoding/json"
	"errors"
	"prime-erp-core/internal/db"
	authenticationService "prime-erp-core/internal/services/authentication-service"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type CheckAutoApprovalRequest struct {
	RequestUserCode string  `json:"request_user_code"`
	ModuleCode      string  `json:"module_code"`
	TopicCode       string  `json:"topic_code"`
	MdItemCode      string  `json:"md_item_code"`
	CondRangeMin    float64 `json:"cond_range_min"`
}

type CheckAutoApprovalResponse struct {
	IsAutoApproved bool   `json:"is_auto_approved"`
	ResponseCode   int    `json:"response_code"`
	Message        string `json:"message"`
}

// CheckAutoApprovalRest เป็นเส้นที่ไม่มีตัวตนฝั่ง server ส่งเข้า CheckAutoApproval เลย
// ตัวตนมาจาก request_user_code ใน body เท่านั้น ไม่ส่งมา = ตอบ 400 ตามเดิม
//
// ของเดิมประกาศตัวแปร user แล้วอ่าน ctx.GetString("user_code") ซึ่งไม่มี middleware ตัวไหน
// เคย c.Set คีย์นั้นเลย (มีแต่ "user") จึงได้ค่าว่างเสมอมาตั้งแต่ต้น — เป็น fallback ที่หลอกคนอ่าน
// ว่ามีอยู่จริง จึงลบทิ้ง ไม่ได้เปลี่ยนพฤติกรรมอะไร
//
// เจตนาที่ไม่เอา user จาก token มาเป็น fallback: เส้นนี้เป็นด่านสิทธิ์ ถ้าให้มันเดาตัวตนจากคนที่
// login อยู่ request ที่ลืมส่ง request_user_code จะเปลี่ยนจากถูกปฏิเสธ ไปเป็นอนุมัติผ่านเงียบๆ
// (เจ้าของตัดสินใจไว้ 2026-09-26) ส่วนผู้เรียกภายในทั้ง 7 จุด — sale, quotation, purchase,
// pre-purchase, credit — เรียก CheckAutoApproval ตรงๆ พร้อมส่ง user ของตัวเองมาให้อยู่แล้ว
func CheckAutoApprovalRest(ctx context.Context, jsonPayload string) (interface{}, error) {
	req := CheckAutoApprovalRequest{}

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	// "" คือ "ไม่มีตัวตนจากฝั่ง server" — ดูเหตุผลใน doc comment ด้านบน
	return CheckAutoApproval(ctx, gormx, req, "")
}

func CheckAutoApproval(ctx context.Context, gormx *gorm.DB, req CheckAutoApprovalRequest, user string) (*CheckAutoApprovalResponse, error) {
	_ = gormx

	res := &CheckAutoApprovalResponse{
		IsAutoApproved: false,
		ResponseCode:   200,
		Message:        "success",
	}

	requestUserCode := strings.TrimSpace(req.RequestUserCode)
	if requestUserCode == "" {
		requestUserCode = strings.TrimSpace(user)
	}

	moduleCode := strings.TrimSpace(req.ModuleCode)
	topicCode := strings.TrimSpace(req.TopicCode)
	mdItemCode := strings.TrimSpace(req.MdItemCode)

	if requestUserCode == "" {
		res.ResponseCode = 400
		res.Message = "request_user_code is required"
		return res, nil
	}
	if moduleCode == "" {
		res.ResponseCode = 400
		res.Message = "module_code is required"
		return res, nil
	}
	if topicCode == "" {
		res.ResponseCode = 400
		res.Message = "topic_code is required"
		return res, nil
	}
	if mdItemCode == "" {
		res.ResponseCode = 400
		res.Message = "md_item_code is required"
		return res, nil
	}

	active := true
	isNotExpired := true

	requestData := map[string]interface{}{
		"user_code":         []string{requestUserCode},
		"module_code":       []string{moduleCode},
		"module_topic_code": []string{topicCode},
		"module_item_code":  []string{mdItemCode},
		"active":            active,
		"is_not_expired":    isNotExpired,
	}

	userApproval, err := authenticationService.GetUserApproval(ctx, requestData)
	if err != nil {
		return nil, err
	}

	if len(userApproval.Data) == 0 {
		res.IsAutoApproved = false
		res.Message = "not found user approval"
		return res, nil
	}

	sort.Slice(userApproval.Data, func(i, j int) bool {
		if userApproval.Data[i].CondRangeMin == nil && userApproval.Data[j].CondRangeMin == nil {
			return false
		}
		if userApproval.Data[i].CondRangeMin == nil {
			return false
		}
		if userApproval.Data[j].CondRangeMin == nil {
			return true
		}
		return *userApproval.Data[i].CondRangeMin > *userApproval.Data[j].CondRangeMin
	})

	var matched *authenticationService.UserApprovalDataResult
	for i := range userApproval.Data {
		item := &userApproval.Data[i]
		if item.CondRangeMin == nil {
			continue
		}

		// เลือก tier แรกที่เข้าเงื่อนไข แล้วหยุดทันที
		if req.CondRangeMin >= float64(*item.CondRangeMin) {
			matched = item
			break
		}
	}

	if matched == nil {
		res.IsAutoApproved = false
		res.Message = "approval required"
		return res, nil
	}

	if matched.AutoApprove {
		res.IsAutoApproved = true
		res.Message = "auto approved"
		return res, nil
	}

	res.IsAutoApproved = false
	res.Message = "approval required"
	return res, nil
}
