package authenticationService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type GetUserApprovalRequest struct {
	UserCode        []string `json:"user_code"`
	ModuleCode      []string `json:"module_code"`
	ModuleTopicCode []string `json:"module_topic_code"`
	ModuleItemCode  []string `json:"module_item_code"`
	Active          *bool    `json:"active"`
	IsNotExpired    *bool    `json:"is_not_expired"`
}
type UserApprovalDataResult struct {
	ID                   string                           `json:"id"`
	UserCode             string                           `json:"user_code"` //คนที่ขอ approve
	ModuleCode           string                           `json:"module_code"`
	ModuleTopicCode      string                           `json:"module_topic_code"`
	ModuleItemCode       string                           `json:"module_item_code"`
	ModuleItemID         string                           `json:"module_item_id"`
	ModuleItemName       string                           `json:"module_item_name"`
	ModuleItemSection    string                           `json:"module_item_section"`
	ModuleItemSubSection string                           `json:"module_item_sub_section"`
	CondRangeMin         *int                             `json:"cond_range_min"`
	Active               bool                             `json:"active"`
	AutoApprove          bool                             `json:"auto_approve"`
	ApproveCode          string                           `json:"approve_code"` //คนที่ มีสิทธิอนุมัติ
	Secondarys           []UserApprovalDataScondaryResult `json:"secondarys"`
}
type UserApprovalDataScondaryResult struct {
	ID            string  `json:"id"`
	ApproveCode   string  `json:"approve_code"` //คนที่ มีสิทธิอนุมัติ 2,3,4
	EffectiveDate string  `json:"effective_date"`
	ExpireDate    *string `json:"expire_date"`
}
type GetUserApprovalResponse struct {
	ResponseCode int                      `json:"response_code"`
	Message      string                   `json:"message"`
	Data         []UserApprovalDataResult `json:"data"`
}

func GetUserApproval(requestData map[string]interface{}) (GetUserApprovalResponse, error) {

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		errors.New("Error marshalling data :")
	}

	reqHttp, err := http.NewRequest("POST", os.Getenv("base_url_authorization")+"/author/get-user-approval", bytes.NewBuffer(jsonData))
	if err != nil {
		return GetUserApprovalResponse{}, errors.New("Error parsing DateTo: " + err.Error())
	}

	reqHttp.Header.Set("Content-Type", "application/json")

	// Create a client and execute the request
	client := &http.Client{}
	resp, err := client.Do(reqHttp)
	if err != nil {
		return GetUserApprovalResponse{}, errors.New("Error parsing DateTo : " + err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Response Status:", err)
	}
	var requesters GetUserApprovalResponse
	err = json.Unmarshal(body, &requesters)
	if err != nil {
		fmt.Println("Response Status:", err)
	}

	fmt.Println("Response Status:", resp.Status)

	return requesters, nil

}
