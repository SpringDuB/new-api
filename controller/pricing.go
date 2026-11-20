package controller

import (
	"maps"
	"reflect"
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if common.StringsContains(item.EnableGroup, "all") {
			filtered = append(filtered, item)
			continue
		}
		for _, group := range item.EnableGroup {
			if _, ok := usableGroup[group]; ok {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

// Equivalent mapped aliases remain callable but need only one catalog card.
func distinctPricingCatalogModels(pricing []model.Pricing) []model.Pricing {
	byName := make(map[string]model.Pricing, len(pricing))
	for _, item := range pricing {
		byName[item.ModelName] = item
	}
	result := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if target, ok := byName[item.CatalogAliasOf]; ok && item.DisplayName != "" && target.CatalogAliasOf == "" {
			alias := item
			alias.ModelName, alias.CatalogAliasOf, alias.PricingVersion = target.ModelName, "", target.PricingVersion
			if slices.Equal(slices.Sorted(slices.Values(alias.EnableGroup)), slices.Sorted(slices.Values(target.EnableGroup))) {
				alias.EnableGroup = target.EnableGroup
			}
			if reflect.DeepEqual(alias, target) {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	userId, exists := c.Get("id")
	usableGroup := map[string]string{}
	groupRatio := map[string]float64{}
	maps.Copy(groupRatio, ratio_setting.GetGroupRatioCopy())
	var group string
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio_setting.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	usableGroup = service.GetUserUsableGroups(group)
	pricing = distinctPricingCatalogModels(filterPricingByUsableGroups(pricing, usableGroup))
	modelNames := make([]string, len(pricing))
	for i := range pricing {
		modelNames[i] = pricing[i].ModelName
	}
	// check groupRatio contains usableGroup
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	c.JSON(200, gin.H{
		"success":                 true,
		"data":                    pricing,
		"model_square_display":    model.GetModelSquareDisplay().ForModels(modelNames),
		"model_square_categories": model.GetModelSquareCategories(modelNames),
		"vendors":                 model.GetVendors(),
		"group_ratio":             groupRatio,
		"usable_group":            usableGroup,
		"supported_endpoint":      model.GetSupportedEndpointMap(),
		"auto_groups":             service.GetUserAutoGroup(group),
		"pricing_version":         "a42d372ccf0b5dd13ecf71203521f9d2",
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccessT(c, "Model ratios reset", nil)
}
