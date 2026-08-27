package priceListRepository

import (
	"encoding/json"
	"errors"
	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

// GetPriceListGroup
func GetPriceListGroup(companyCode string, siteCode string, groupCodes []string) ([]models.PriceListGroup, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	priceListGroups := []models.PriceListGroup{}
	query := gormx.Model(&models.PriceListGroup{}).
		Where("company_code = ? AND site_code = ?", companyCode, siteCode)

	if len(groupCodes) > 0 {
		query = query.Where("group_code IN ?", groupCodes)
	}

	if err := query.
		Preload("PriceListGroupTerms").
		Preload("PriceListGroupExtras.PriceListGroupExtraKeys").
		// Without an explicit order Postgres is free to return the sub groups in a
		// different order after an update, which reshuffles the detail table.
		Preload("PriceListSubGroups", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("subgroup_key").Order("id")
		}).
		Preload("PriceListSubGroups.PriceListSubGroupKeys").
		Find(&priceListGroups).Error; err != nil {
		return priceListGroups, err
	}

	return priceListGroups, nil
}

// GetPriceListGroupCodesByIDs returns the distinct group codes for the given
// price list group IDs. Used to cascade a base price update into the sub groups.
func GetPriceListGroupCodesByIDs(ids []uuid.UUID) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	groupCodes := []string{}
	if err := gormx.Model(&models.PriceListGroup{}).
		Where("id IN ?", ids).
		Distinct().
		Pluck("group_code", &groupCodes).Error; err != nil {
		return nil, err
	}

	return groupCodes, nil
}

// GetPriceListSubGroupByID loads a price list sub group (with keys) by sub group ID.
func GetPriceListSubGroupByID(subGroupID uuid.UUID) (*models.PriceListSubGroup, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	subGroup := models.PriceListSubGroup{}

	if err := gormx.Model(&models.PriceListSubGroup{}).
		Where("id = ?", subGroupID).
		Preload("PriceListGroup").
		Preload("PriceListSubGroupKeys").
		First(&subGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &subGroup, nil
}

// GetPriceListSubGroupsByIDs loads multiple price list sub groups (with keys) by sub group IDs.
func GetPriceListSubGroupsByIDs(subGroupIDs []uuid.UUID) ([]models.PriceListSubGroup, error) {
	if len(subGroupIDs) == 0 {
		return []models.PriceListSubGroup{}, nil
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	subGroups := []models.PriceListSubGroup{}

	if err := gormx.Model(&models.PriceListSubGroup{}).
		Where("id IN ?", subGroupIDs).
		Preload("PriceListGroup").
		Preload("PriceListGroup.PriceListGroupExtras.PriceListGroupExtraKeys").
		Preload("PriceListSubGroupKeys").
		Find(&subGroups).Error; err != nil {
		return nil, err
	}

	return subGroups, nil
}

// GetPriceListSubGroupsByGroupCodes loads price list sub groups (with keys and extras)
// for all price list groups matching the given group codes.
func GetPriceListSubGroupsByGroupCodes(groupCodes []string) ([]models.PriceListSubGroup, error) {
	if len(groupCodes) == 0 {
		return []models.PriceListSubGroup{}, nil
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	subGroups := []models.PriceListSubGroup{}

	if err := gormx.Model(&models.PriceListSubGroup{}).
		Joins("JOIN price_list_group ON price_list_group.id = price_list_sub_group.price_list_group_id").
		Where("price_list_group.group_code IN ?", groupCodes).
		Preload("PriceListGroup").
		Preload("PriceListGroup.PriceListGroupExtras.PriceListGroupExtraKeys").
		Preload("PriceListSubGroupKeys").
		Find(&subGroups).Error; err != nil {
		return nil, err
	}

	return subGroups, nil
}

// GetGroupItemValueInt looks up group_item.value_int for a given group_code (mapped from condition_code)
// and subgroup key value. It returns (valueInt, found, error).
func GetGroupItemValueInt(groupCode, value string) (float64, bool, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return 0, false, err
	}
	defer db.CloseGORM(gormx)

	// 1) find group by group_code
	var grp models.Group
	if err := gormx.Model(&models.Group{}).
		Where("group_code = ?", groupCode).
		First(&grp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}

	// 2) find group_item by group_id + item_code
	var item models.GroupItem
	if err := gormx.Model(&models.GroupItem{}).
		Where("group_id = ? AND item_code = ?", grp.ID, value).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}

	return item.ValueInt, true, nil
}

func GetPriceListExtraConfig(groupCodes []string) ([]models.PriceListExtraConfig, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	priceListExtraConfigs := []models.PriceListExtraConfig{}
	query := gormx.Model(&models.PriceListExtraConfig{})

	if len(groupCodes) > 0 {
		query = query.Where("group_code IN ?", groupCodes)
	}

	if err := query.Find(&priceListExtraConfigs).Error; err != nil {
		return priceListExtraConfigs, err
	}

	return priceListExtraConfigs, nil
}

// CreatePriceListGroup
func CreatePriceListBase(priceListGroups []models.PriceListGroup) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&priceListGroups).Error; err != nil {
			return err
		}
		return nil
	})
}

// UpdatePriceListGroup
func UpdatePriceListBase(priceListGroup []models.PriceListGroup) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, group := range priceListGroup {
			oldPriceListGroup := models.PriceListGroup{}

			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", group.ID).
				Find(&oldPriceListGroup).
				Error; err != nil {
				return err
			}

			now := time.Now().UTC()

			historyFormat := models.PriceListGroupHistory{
				ID:                uuid.New(),
				CompanyCode:       oldPriceListGroup.CompanyCode,
				SiteCode:          oldPriceListGroup.SiteCode,
				GroupCode:         oldPriceListGroup.GroupCode,
				PriceUnit:         oldPriceListGroup.PriceUnit,
				PriceWeight:       oldPriceListGroup.PriceWeight,
				BeforePriceUnit:   oldPriceListGroup.BeforePriceUnit,
				BeforePriceWeight: oldPriceListGroup.BeforePriceWeight,
				Currency:          oldPriceListGroup.Currency,
				EffectiveDate:     oldPriceListGroup.EffectiveDate,
				ExpiryDate:        &now,
				Remark:            oldPriceListGroup.Remark,
				CreateBy:          oldPriceListGroup.CreateBy,
				CreateDtm:         oldPriceListGroup.CreateDtm,
				UpdateBy:          oldPriceListGroup.UpdateBy,
				UpdateDtm:         oldPriceListGroup.UpdateDtm,
			}

			// Insert old record into history table
			if err := tx.Model(&models.PriceListGroupHistory{}).Create(&historyFormat).Error; err != nil {
				return err
			}

			// Update main table
			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", group.ID).
				Updates(map[string]interface{}{
					"price_unit":          group.PriceUnit,
					"price_weight":        group.PriceWeight,
					"before_price_unit":   oldPriceListGroup.PriceUnit,
					"before_price_weight": oldPriceListGroup.PriceWeight,
					"currency":            group.Currency,
					"effective_date":      group.EffectiveDate,
					"remark":              group.Remark,
					"update_by":           group.UpdateBy,
					"update_dtm":          group.UpdateDtm,
				}).Error; err != nil {
				return err
			}

			// Delete old Terms
			if termResult := tx.Where("price_list_group_id = ?", group.ID).Delete(&models.PriceListGroupTerm{}); termResult.Error != nil {
				return termResult.Error
			}

			// Insert new Terms
			for _, term := range group.PriceListGroupTerms {
				term.PriceListGroupID = group.ID
				if err := tx.Create(&term).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// UpdateExtra
func UpdateExtra(extras []models.PriceListGroupExtra) error {
	priceListGroupIDs := []uuid.UUID{}
	for _, extra := range extras {
		priceListGroupIDs = append(priceListGroupIDs, extra.PriceListGroupID)
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		// Delete old Extra
		if extraResult := tx.Where("price_list_group_id IN ?", priceListGroupIDs).
			Delete(&models.PriceListGroupExtra{}); extraResult.Error != nil {
			return extraResult.Error
		}

		// Insert new Extra
		if err := tx.Create(&extras).Error; err != nil {
			return err
		}

		return nil
	})
}

// DeletePriceListGroup
func DeletePriceListBase(ids []string) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)
	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			oldPriceListGroup := models.PriceListGroup{}

			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", id).
				Find(&oldPriceListGroup).
				Error; err != nil {
				return err
			}

			now := time.Now().UTC()

			historyFormat := models.PriceListGroupHistory{
				ID:                uuid.New(),
				CompanyCode:       oldPriceListGroup.CompanyCode,
				SiteCode:          oldPriceListGroup.SiteCode,
				GroupCode:         oldPriceListGroup.GroupCode,
				PriceUnit:         oldPriceListGroup.PriceUnit,
				PriceWeight:       oldPriceListGroup.PriceWeight,
				BeforePriceUnit:   oldPriceListGroup.BeforePriceUnit,
				BeforePriceWeight: oldPriceListGroup.BeforePriceWeight,
				Currency:          oldPriceListGroup.Currency,
				EffectiveDate:     oldPriceListGroup.EffectiveDate,
				ExpiryDate:        &now,
				Remark:            oldPriceListGroup.Remark,
				CreateBy:          oldPriceListGroup.CreateBy,
				CreateDtm:         oldPriceListGroup.CreateDtm,
				UpdateBy:          oldPriceListGroup.UpdateBy,
				UpdateDtm:         oldPriceListGroup.UpdateDtm,
			}

			// Insert old record into history table
			if err := tx.Model(&models.PriceListGroupHistory{}).Create(&historyFormat).Error; err != nil {
				return err
			}

			// Delete from main table
			if err := tx.Model(&models.PriceListGroup{}).
				Where("id = ?", id).
				Delete(&models.PriceListGroup{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdatePriceListSubGroup updates a price_list_sub_group record by ID
// It carefully merges udf_json to preserve existing keys while updating with new values
func UpdatePriceListSubGroups(reqs models.UpdatePriceListSubGroupRequest) error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	return gormx.Transaction(func(tx *gorm.DB) error {
		for _, req := range reqs.Changes {
			// Retrieve existing sub_group record
			oldSubGroup := models.PriceListSubGroup{}
			if err := tx.Model(&models.PriceListSubGroup{}).
				Where("id = ?", req.SubGroupID).
				Preload("PriceListSubGroupKeys").
				First(&oldSubGroup).Error; err != nil {
				return err
			}

			now := time.Now().UTC()

			// Create history record from old data
			historyRecord := models.PriceListSubGroupHistory{
				ID:                        uuid.New(),
				PriceListGroupID:          oldSubGroup.PriceListGroupID,
				SubgroupKey:               oldSubGroup.SubgroupKey,
				IsTrading:                 oldSubGroup.IsTrading,
				PriceUnit:                 oldSubGroup.PriceUnit,
				ExtraPriceUnit:            oldSubGroup.ExtraPriceUnit,
				TotalNetPriceUnit:         oldSubGroup.TotalNetPriceUnit,
				PriceWeight:               oldSubGroup.PriceWeight,
				ExtraPriceWeight:          oldSubGroup.ExtraPriceWeight,
				TermPriceWeight:           oldSubGroup.TermPriceWeight,
				TotalNetPriceWeight:       oldSubGroup.TotalNetPriceWeight,
				BeforePriceUnit:           oldSubGroup.BeforePriceUnit,
				BeforeExtraPriceUnit:      oldSubGroup.BeforeExtraPriceUnit,
				BeforeTermPriceUnit:       oldSubGroup.BeforeTermPriceUnit,
				BeforeTotalNetPriceUnit:   oldSubGroup.BeforeTotalNetPriceUnit,
				BeforePriceWeight:         oldSubGroup.BeforePriceWeight,
				BeforeExtraPriceWeight:    oldSubGroup.BeforeExtraPriceWeight,
				BeforeTermPriceWeight:     oldSubGroup.BeforeTermPriceWeight,
				BeforeTotalNetPriceWeight: oldSubGroup.BeforeTotalNetPriceWeight,
				EffectiveDate:             oldSubGroup.EffectiveDate,
				ExpiryDate:                &now,
				Remark:                    oldSubGroup.Remark,
				CreateBy:                  oldSubGroup.CreateBy,
				CreateDtm:                 oldSubGroup.CreateDtm,
				UpdateBy:                  oldSubGroup.UpdateBy,
				UpdateDtm:                 oldSubGroup.UpdateDtm,
			}

			// Insert old record into history table
			if err := tx.Model(&models.PriceListSubGroupHistory{}).Create(&historyRecord).Error; err != nil {
				return err
			}

			// Prepare update map
			updateMap := make(map[string]interface{})

			// Handle udf_json merging
			var mergedUdfJson json.RawMessage
			if len(req.UdfJson) > 0 {
				// Parse existing udf_json
				existingUdfMap := make(map[string]interface{})
				if len(oldSubGroup.UdfJson) > 0 {
					if err := json.Unmarshal(oldSubGroup.UdfJson, &existingUdfMap); err != nil {
						return err
					}
				}

				// Parse new udf_json
				newUdfMap := make(map[string]interface{})
				if err := json.Unmarshal(req.UdfJson, &newUdfMap); err != nil {
					return err
				}

				// Merge: new values override existing, but preserve all existing keys
				for key, value := range newUdfMap {
					existingUdfMap[key] = value
				}

				// Marshal merged map back to json.RawMessage
				mergedBytes, err := json.Marshal(existingUdfMap)
				if err != nil {
					return err
				}
				mergedUdfJson = json.RawMessage(mergedBytes)
				updateMap["udf_json"] = mergedUdfJson
			}

			// Handle other field updates
			if req.IsTrading != nil {
				updateMap["is_trading"] = *req.IsTrading
			}

			// Handle price unit fields - update before fields with old values
			if req.PriceUnit != nil {
				updateMap["before_price_unit"] = oldSubGroup.PriceUnit
				updateMap["price_unit"] = *req.PriceUnit
			}
			if req.ExtraPriceUnit != nil {
				updateMap["before_extra_price_unit"] = oldSubGroup.ExtraPriceUnit
				updateMap["extra_price_unit"] = *req.ExtraPriceUnit
			}
			if req.TotalNetPriceUnit != nil {
				updateMap["before_total_net_price_unit"] = oldSubGroup.TotalNetPriceUnit
				updateMap["total_net_price_unit"] = *req.TotalNetPriceUnit
			}

			// Handle price weight fields - update before fields with old values
			if req.PriceWeight != nil {
				updateMap["before_price_weight"] = oldSubGroup.PriceWeight
				updateMap["price_weight"] = *req.PriceWeight
			}
			if req.ExtraPriceWeight != nil {
				updateMap["before_extra_price_weight"] = oldSubGroup.ExtraPriceWeight
				updateMap["extra_price_weight"] = *req.ExtraPriceWeight
			}
			if req.TermPriceWeight != nil {
				updateMap["before_term_price_weight"] = oldSubGroup.TermPriceWeight
				updateMap["term_price_weight"] = *req.TermPriceWeight
			}
			if req.TotalNetPriceWeight != nil {
				updateMap["before_total_net_price_weight"] = oldSubGroup.TotalNetPriceWeight
				updateMap["total_net_price_weight"] = *req.TotalNetPriceWeight
			}

			if req.EffectiveDate != nil {
				updateMap["effective_date"] = req.EffectiveDate
			}

			if req.Remark != nil {
				updateMap["remark"] = *req.Remark
			}

			updateMap["update_by"] = "system" // TODO: get user from auth
			updateMap["update_dtm"] = now

			// Update the record
			if err := tx.Model(&models.PriceListSubGroup{}).
				Where("id = ?", req.SubGroupID).
				Updates(updateMap).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func GetPriceListSubGroupFormulasMapBySubGroupCode(subGroupCode string) ([]models.PriceListSubGroupFormulasMap, error) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return []models.PriceListSubGroupFormulasMap{}, err
	}
	defer db.CloseGORM(gormx)

	// get the latest price list formulas
	priceListSubGroupFormulasMap := []models.PriceListSubGroupFormulasMap{}
	if err := gormx.Model(&models.PriceListSubGroupFormulasMap{}).
		Where("price_list_subgroup_code = ?", subGroupCode).
		Order("create_dtm DESC").
		Preload("PriceListFormulas").
		Find(&priceListSubGroupFormulasMap).Error; err != nil {
		return priceListSubGroupFormulasMap, err
	}
	return priceListSubGroupFormulasMap, nil
}

// GetPriceListSubGroupFormulasMapBySubGroupCodes loads price list formulas for multiple sub group codes.
// Returns a map keyed by sub group code for efficient lookup.
func GetPriceListSubGroupFormulasMapBySubGroupCodes(subGroupCodes []string) (map[string][]models.PriceListSubGroupFormulasMap, error) {
	result := make(map[string][]models.PriceListSubGroupFormulasMap)

	if len(subGroupCodes) == 0 {
		return result, nil
	}

	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		return nil, err
	}
	defer sqlxDB.Close()

	// Build query to handle empty strings properly using COALESCE
	// This ensures empty strings are properly matched
	query := `
		SELECT 
			psfm.id,
			COALESCE(psfm.price_list_subgroup_code, '') as price_list_subgroup_code,
			psfm.price_list_formulas_code,
			psfm.is_default,
			psfm.create_dtm,
			pf.id,
			pf.formula_code,
			pf.name,
			pf.uom,
			pf.formula_type,
			pf.expression,
			pf.params,
			pf.rounding,
			pf.create_dtm
		FROM price_list_subgroup_formulas_map psfm
		LEFT JOIN price_list_formulas pf ON psfm.price_list_formulas_code = pf.formula_code
		WHERE COALESCE(psfm.price_list_subgroup_code, '') IN (?)
		ORDER BY psfm.create_dtm DESC
	`

	// Use sqlx.In to expand the IN clause
	query, args, err := sqlx.In(query, subGroupCodes)
	if err != nil {
		return nil, err
	}

	// Rebind to match the database driver's placeholder syntax
	query = sqlxDB.Rebind(query)

	rows, err := sqlxDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan results
	for rows.Next() {
		var fm models.PriceListSubGroupFormulasMap
		var pf models.PriceListFormulas

		err := rows.Scan(
			&fm.ID,
			&fm.PriceListSubGroupCode,
			&fm.PriceListFormulasCode,
			&fm.IsDefault,
			&fm.CreateDtm,
			&pf.ID,
			&pf.FormulaCode,
			&pf.Name,
			&pf.Uom,
			&pf.FormulaType,
			&pf.Expression,
			&pf.Params,
			&pf.Rounding,
			&pf.CreateDtm,
		)
		if err != nil {
			return nil, err
		}

		fm.PriceListFormulas = pf
		result[fm.PriceListSubGroupCode] = append(result[fm.PriceListSubGroupCode], fm)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
