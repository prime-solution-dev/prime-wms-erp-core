package approvalService

import (
	"context"
	"testing"
)

// เส้น REST ไม่มีตัวตนฝั่ง server ตัวตนต้องมาจาก request_user_code ใน body เท่านั้น
// ไม่ส่งมา = ปฏิเสธ ไม่ใช่เดาจากคนที่ login อยู่ (ด่านสิทธิ์ห้ามเดาตัวตน)
func TestCheckAutoApprovalRejectsWhenBodyHasNoUser(t *testing.T) {
	res, err := CheckAutoApproval(context.Background(), nil, CheckAutoApprovalRequest{
		ModuleCode: "CUSTOMIZE",
		TopicCode:  "CUSTOMIZE",
		MdItemCode: "CTM-CTM5",
	}, "")
	if err != nil {
		t.Fatalf("CheckAutoApproval: %v", err)
	}

	if res.ResponseCode != 400 || res.Message != "request_user_code is required" {
		t.Fatalf("ได้ %d %q — ต้องปฏิเสธเมื่อไม่มีตัวตน", res.ResponseCode, res.Message)
	}

	if res.IsAutoApproved {
		t.Fatal("ไม่มีตัวตนแต่อนุมัติผ่าน — ด่านสิทธิ์รั่ว")
	}
}

// ผู้เรียกภายใน 7 จุดส่ง user ของตัวเองมา ทางนั้นต้องไม่ถูกปฏิเสธ
// (ผ่านด่านตัวตนแล้วไปต่อที่ขอสิทธิ์จาก authentication service ซึ่งเทสนี้ไม่แตะ)
func TestCheckAutoApprovalAcceptsIdentityFromCaller(t *testing.T) {
	res, err := CheckAutoApproval(context.Background(), nil, CheckAutoApprovalRequest{
		ModuleCode: "CUSTOMIZE",
		TopicCode:  "CUSTOMIZE",
		MdItemCode: "CTM-CTM5",
	}, "somchai")

	if err == nil && res != nil && res.ResponseCode == 400 && res.Message == "request_user_code is required" {
		t.Fatal("caller ส่ง user มาแล้วยังถูกปฏิเสธว่าไม่มีตัวตน")
	}
}
