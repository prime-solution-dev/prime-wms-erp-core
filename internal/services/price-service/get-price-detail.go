package priceService

import (
	"encoding/json"
	"fmt"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	groupService "prime-erp-core/internal/services/group-service"
	priceDomain "prime-erp-core/internal/services/price-service/domain"
	pricePatterns "prime-erp-core/internal/services/price-service/patterns"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// getGroupAndItemMappings gets group and group item mappings for value name resolution
func getGroupAndItemMappings() (map[string]models.GetGroupResponse, map[string]models.GetGroupItemResponse, map[string]GetPaymentTermResponse, error) {
	// Get groups using group service
	groupReq := models.GetGroupRequest{
		GroupCodes: []string{},
		ItemCodes:  []string{},
	}

	groupReqJson, err := json.Marshal(groupReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal group request: %w", err)
	}

	groupReqString := string(groupReqJson)

	// Note: We need a gin.Context for this call, but we're in a helper function
	// Let's create a minimal context or use nil if the function supports it
	resp, err := groupService.GetGroup(nil, groupReqString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get groups: %w", err)
	}

	groupResp, ok := resp.([]models.GetGroupResponse)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to cast group response")
	}

	groupMap := map[string]models.GetGroupResponse{}
	groupItemMap := map[string]models.GetGroupItemResponse{}
	for _, g := range groupResp {
		groupMap[g.GroupCode] = g
		for _, item := range g.Items {
			groupItemMap[item.ItemCode] = item
		}
	}

	// Get payment terms
	termReq := GetPaymentTermRequest{
		TermCode: []string{},
		TermType: []string{},
	}

	termReqJson, err := json.Marshal(termReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal payment term request: %w", err)
	}

	termReqString := string(termReqJson)

	termResp, err := GetPaymentTerm(nil, termReqString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get payment terms: %w", err)
	}

	paymentTerms, ok := termResp.([]GetPaymentTermResponse)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to cast payment term response")
	}

	paymentTermMap := map[string]GetPaymentTermResponse{}
	for _, pt := range paymentTerms {
		paymentTermMap[pt.TermCode] = pt
	}

	return groupMap, groupItemMap, paymentTermMap, nil
}

// loadPriceData loads price list data from database using GetPriceList
func loadPriceData(sqlx *sqlx.DB, req priceDomain.GetPriceDetailRequest) ([]models.GetPriceListResponse, error) {
	// Build GetPriceListGroupRequest from GetPriceDetailRequest
	priceListReq := GetPriceListGroupRequest{
		CompanyCode:       req.CompanyCode,
		SiteCodes:         req.SiteCodes,
		GroupCodes:        req.GroupCodes,
		EffectiveDateFrom: req.EffectiveDateFrom,
		EffectiveDateTo:   req.EffectiveDateTo,
	}

	// Get price list groups with subgroups
	groupSubGroup, err := getGroupSubGroup(sqlx, priceListReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get group sub group: %w", err)
	}

	// Get terms
	groupSubGroup, err = getTerms(sqlx, groupSubGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get terms: %w", err)
	}

	// Get extras
	groupSubGroup, err = getExtras(sqlx, groupSubGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get extras: %w", err)
	}

	// Transform to GetPriceListResponse format (same as GetPriceList API)
	result, err := transformToGetPriceListResponse(groupSubGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to transform response: %w", err)
	}

	return result, nil
}

// transformToGetPriceListResponse transforms internal response to API response format
func transformToGetPriceListResponse(responses []GetPriceListGroupResponse) ([]models.GetPriceListResponse, error) {
	// Get group and group item mappings
	groupMap, groupItemMap, _, err := getGroupAndItemMappings()
	if err != nil {
		return nil, fmt.Errorf("failed to get mappings: %w", err)
	}

	result := []models.GetPriceListResponse{}
	for _, resp := range responses {
		// Extract GroupKey from first subgroup's subgroup_key (first part before "|")
		groupKey := ""
		GroupName := ""
		if len(resp.SubGroups) > 0 && resp.SubGroups[0].SubGroupKey != "" {
			groupKey = pricePatterns.ExtractGroupKey(resp.SubGroups[0].SubGroupKey)
			// Get GroupName from group item map using the last part of group_key
			if groupKey != "" {
				// Extract the last part (e.g., "GROUP_1_ITEM_1" -> "GROUP_1_ITEM_1")
				// Actually, groupKey is already the first part, so we can use it directly
				if item, ok := groupItemMap[groupKey]; ok {
					GroupName = item.ItemName
				}
			}
		}

		priceListResp := models.GetPriceListResponse{
			ID:                resp.ID.String(),
			CompanyCode:       resp.CompanyCode,
			SiteCode:          resp.SiteCode,
			GroupCode:         resp.GroupCode,
			PriceUnit:         resp.PriceUnit,
			PriceWeight:       resp.PriceWeight,
			BeforePriceUnit:   resp.BeforePriceUnit,
			BeforePriceWeight: resp.BeforePriceWeight,
			Currency:          resp.Currency,
			Remark:            resp.Remark,
			GroupKey:          groupKey,
			GroupName:         GroupName,
		}

		// Format effective date
		if !resp.EffectiveDate.IsZero() {
			effectiveDate := resp.EffectiveDate.Format(time.RFC3339)
			priceListResp.EffectiveDate = &effectiveDate
		}

		// Transform subgroups
		subGroups := []models.PriceListSubGroupResponse{}
		for _, sg := range resp.SubGroups {
			subGroupKeys := []models.PriceListSubGroupKeyResponse{}
			for _, sgk := range sg.GroupKeys {
				subGroupKeys = append(subGroupKeys, models.PriceListSubGroupKeyResponse{
					ID:         uuid.New().String(),
					SubGroupID: sg.ID.String(),
					GroupCode:  sgk.Code,
					GroupName:  groupMap[sgk.Code].GroupName,
					ValueCode:  sgk.Value,
					ValueName:  groupItemMap[sgk.Value].ItemName,
					Seq:        sgk.Seq,
				})
			}

			sgEffectiveDate := ""
			if !sg.EffectiveDate.IsZero() {
				sgEffectiveDate = sg.EffectiveDate.Format(time.RFC3339)
			}

			subGroups = append(subGroups, models.PriceListSubGroupResponse{
				ID:                        sg.ID.String(),
				PriceListGroupID:          resp.ID.String(),
				SubgroupKey:               sg.SubGroupKey,
				IsTrading:                 sg.IsTrading,
				PriceUnit:                 sg.PriceUnit,
				ExtraPriceUnit:            sg.ExtraPriceUnit,
				TotalNetPriceUnit:         sg.TotalNetPriceUnit,
				PriceWeight:               sg.PriceWeight,
				ExtraPriceWeight:          sg.ExtraPriceWeight,
				TermPriceWeight:           sg.TermPriceWeight,
				TotalNetPriceWeight:       sg.TotalNetPriceWeight,
				BeforePriceUnit:           sg.BeforePriceUnit,
				BeforeExtraPriceUnit:      sg.BeforeExtraPriceUnit,
				BeforeTermPriceUnit:       sg.BeforeTermPriceUnit,
				BeforeTotalNetPriceUnit:   sg.BeforeTotalNetPriceUnit,
				BeforePriceWeight:         sg.BeforePriceWeight,
				BeforeExtraPriceWeight:    sg.BeforeExtraPriceWeight,
				BeforeTermPriceWeight:     sg.BeforeTermPriceWeight,
				BeforeTotalNetPriceWeight: sg.BeforeTotalNetPriceWeight,
				EffectiveDate:             &sgEffectiveDate,
				Remark:                    sg.Remark,
				UdfJson:                   sg.UdfJson,
				SubGroupKeys:              subGroupKeys,
			})
		}

		priceListResp.SubGroups = subGroups
		result = append(result, priceListResp)
	}

	return result, nil
}

func GetPriceDetail(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	// Parse request
	var req priceDomain.GetPriceDetailRequest
	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request JSON: %w", err)
	}

	// Validate required fields
	if req.CompanyCode == "" {
		return nil, fmt.Errorf("company_code is required")
	}
	if len(req.SiteCodes) == 0 {
		return nil, fmt.Errorf("site_codes is required")
	}
	if len(req.GroupCodes) == 0 {
		return nil, fmt.Errorf("group_codes is required")
	}

	// Load configuration using the first groupCode to extract requiredGroupCodes
	groupCode := req.GroupCodes[0]
	config, err := pricePatterns.LoadConfiguration(groupCode)
	if err != nil {
		// If config loading fails, continue with original groupCodes
		// This ensures backward compatibility
	} else {
		// Extract requiredGroupCodes from configuration
		requiredGroupCodes := pricePatterns.ExtractRequiredGroupCodes(config)
		// Merge requiredGroupCodes into req.GroupCodes (avoid duplicates)
		groupCodeMap := make(map[string]bool)
		for _, gc := range req.GroupCodes {
			groupCodeMap[gc] = true
		}
		for _, gc := range requiredGroupCodes {
			if !groupCodeMap[gc] {
				req.GroupCodes = append(req.GroupCodes, gc)
				groupCodeMap[gc] = true
			}
		}
	}

	// Connect to database
	sqlx, err := db.ConnectSqlx(`prime_erp`)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer sqlx.Close()

	// Load price data
	priceListData, err := loadPriceData(sqlx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to load price data: %w", err)
	}

	if len(priceListData) == 0 {
		return pricePatterns.PriceListDetailApiResponse{
			Id:   uuid.New(),
			Name: "Price List Detail",
			Tabs: []pricePatterns.PriceListDetailTabConfig{},
		}, nil
	}

	// Determine the pattern handler from the group's code
	dataGroupCode := priceListData[0].GroupCode
	if dataGroupCode == "" {
		return nil, fmt.Errorf("GroupCode is missing in price list data")
	}

	type priceTableHandler func([]models.GetPriceListResponse, string) (pricePatterns.PriceListDetailApiResponse, error)

	handlers := map[string]priceTableHandler{
		"GROUP_1_ITEM_1": func(data []models.GetPriceListResponse, _ string) (pricePatterns.PriceListDetailApiResponse, error) {
			return pricePatterns.BuildGroup1Item1Response(data)
		},
		"GROUP_1_ITEM_2": pricePatterns.BuildGroup1Item2Response,
		"GROUP_1_ITEM_3": pricePatterns.BuildGroup1Item3Response,
		"GROUP_1_ITEM_4": pricePatterns.BuildGroup1Item4Response,
		"GROUP_1_ITEM_5": pricePatterns.BuildGroup1Item5Response,
		"GROUP_1_ITEM_6": pricePatterns.BuildGroup1Item6Response,
		"GROUP_1_ITEM_7": pricePatterns.BuildGroup1Item7Response,
		"GROUP_1_ITEM_8": pricePatterns.BuildGroup1Item8Response,
		"GROUP_1_ITEM_9": pricePatterns.BuildGroup1Item9Response,
		"GROUP_1_ITEM_10": func(data []models.GetPriceListResponse, code string) (pricePatterns.PriceListDetailApiResponse, error) {
			return pricePatterns.BuildGroup1Item10Response(data, code)
		},
		"GROUP_1_ITEM_11": pricePatterns.BuildGroup1Item11Response,
		"GROUP_1_ITEM_12": pricePatterns.BuildGroup1Item12Response,
		"GROUP_1_ITEM_13": pricePatterns.BuildGroup1Item13Response,
		"GROUP_1_ITEM_14": pricePatterns.BuildGroup1Item12Response,
		"GROUP_1_ITEM_15": pricePatterns.BuildGroup1Item12Response,
		"GROUP_1_ITEM_16": pricePatterns.BuildGroup1Item12Response,
		"GROUP_1_ITEM_17": pricePatterns.BuildGroup1Item12Response,
		"GROUP_1_ITEM_18": pricePatterns.BuildGroup1Item12Response,
		"GROUP_1_ITEM_19": pricePatterns.BuildGroup1Item12Response,
		"GROUP_1_ITEM_20": pricePatterns.BuildGroup1Item12Response,
	}

	handler, ok := handlers[dataGroupCode]
	if !ok {
		return nil, fmt.Errorf("unsupported GroupCode: %s", dataGroupCode)
	}

	response, err := handler(priceListData, dataGroupCode)
	if err != nil {
		return nil, err
	}

	return response, nil
}
