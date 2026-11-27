package controller

import (
	"maps"
	"reflect"
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
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

type catalogVideoSpec struct {
	Provider      string                 `json:"provider,omitempty"`
	DurationTier  string                 `json:"duration_tier,omitempty"`
	DurationLabel jsplugin.LocalizedText `json:"duration_label,omitempty"`
	jsplugin.VideoSpec
	canvasVideoPriceEstimate
}

type catalogImageSpec struct {
	jsplugin.ImageSpec
	Provider string            `json:"provider,omitempty"`
	Rate     canvasPriceAmount `json:"rate"`
}

// Pair each configuration with its own provider's effective sale expression.
// Never infer a per-image rate from a token or request-dependent formula.
func catalogImageSpecs(item model.Pricing) []catalogImageSpec {
	variants := item.BillingPluginVariants
	if len(variants) == 0 {
		variants = []model.PricingPluginVariant{{BillingExpr: item.BillingExpr, BillingUsageSchema: item.BillingUsageSchema}}
	}
	var result []catalogImageSpec
	for _, v := range variants {
		specs, provider := item.ImageSpecs, ""
		if len(variants) > 1 {
			specs, provider = v.ImageSpecs, v.PluginName
			if provider == "" {
				provider = v.PluginKey
			}
		}
		if len(specs) == 0 || v.BillingExpr == "" {
			return nil
		}
		parsed := parseCanvasPriceExpression(v.BillingExpr, v.BillingUsageSchema)
		for _, s := range specs {
			known := map[string]string{}
			if s.Resolution != "" {
				known["resolution"] = s.Resolution
			}
			if s.Action != "" {
				known["action"] = s.Action
			}
			rates := canvasPriceEstimate(canvasPriceTiers([]canvasPriceExpression{parsed}, known), []float64{1}, false)
			if len(rates) != 1 || (rates[0].Unit != "image" && rates[0].Unit != "request") {
				return nil
			}
			result = append(result, catalogImageSpec{ImageSpec: s, Provider: provider, Rate: rates[0]})
		}
	}
	return result
}

type catalogPricingModel struct {
	ImageSpecs []catalogImageSpec `json:"image_specs,omitempty"`
	model.Pricing
	VideoSpecs []catalogVideoSpec `json:"video_specs,omitempty"`
}

// These are base sale rates, not group-adjusted amounts or upstream costs. Only
// publish an interactive card when every selectable input mode can be quoted.
func catalogVideoSpecs(item model.Pricing) []catalogVideoSpec {
	if len(item.BillingPluginVariants) == 0 {
		return catalogProviderVideoSpecs(item.VideoSpecs, "", item.BillingExpr, item.BillingUsageSchema, item.BillingMode, item)
	}
	var result []catalogVideoSpec
	for _, variant := range item.BillingPluginVariants {
		limits, provider := item.VideoSpecs, ""
		if len(item.BillingPluginVariants) > 1 {
			limits, provider = variant.VideoSpecs, variant.PluginName
			if provider == "" {
				provider = variant.PluginKey
			}
		}
		rows := catalogProviderVideoSpecs(limits, provider, variant.BillingExpr, variant.BillingUsageSchema, variant.BillingMode, item)
		if len(rows) == 0 {
			return nil
		}
		result = append(result, rows...)
	}
	return result
}

func catalogProviderVideoSpecs(limits []jsplugin.VideoSpec, provider, expr string, schema map[string]jsplugin.UsageFieldSchema, mode string, item model.Pricing) []catalogVideoSpec {
	if len(limits) == 0 || (expr == "" && (mode == billing_setting.BillingModeTieredExpr || item.QuotaType != 1)) {
		return nil
	}
	parsed := parseCanvasPriceExpression(expr, schema)
	var specs []catalogVideoSpec
	for _, spec := range limits {
		bands := spec.DurationTiers
		if len(bands) == 0 {
			bands = []string{""}
		}
		for _, band := range bands {
			quote := catalogVideoSpec{VideoSpec: spec, Provider: provider, DurationTier: band, DurationLabel: schema["duration_tier"].EnumLabels[band]}
			for _, video := range []bool{false, true} {
				if !slices.ContainsFunc(spec.Combinations, func(c jsplugin.VideoInputCombination) bool {
					if video {
						return c.Maxima[1] > 0
					}
					return c.Minima[1] == 0
				}) {
					continue
				}
				tiers := []map[string]float64{{"request": item.ModelPrice}}
				if expr != "" {
					input := "none"
					if video {
						input = "video"
					}
					known := map[string]string{"resolution": spec.Resolution, "video_input": input}
					if band != "" {
						known["duration_tier"] = band
					}
					tiers = canvasPriceTiers([]canvasPriceExpression{parsed}, known)
					if !video {
						for _, tier := range tiers {
							delete(tier, "reference_second")
						}
					}
				}
				rates := canvasPriceEstimate(tiers, []float64{1}, true)
				// Keep exactly one output rate first, even for token quotes (which canvasPriceEstimate orders last).
				var primary []canvasPriceAmount
				var extras []canvasPriceAmount
				for _, rate := range rates {
					switch rate.Unit {
					case "request", "second", "video_tokens":
						primary = append(primary, rate)
					case "reference_second", "reference_image":
						if rate.Amount > 0 {
							extras = append(extras, rate)
						}
					default:
						return nil
					}
				}
				if len(primary) != 1 {
					return nil
				}
				output := append(primary, extras...)
				if video {
					quote.WithReference = output
				} else {
					quote.WithoutReference = output
				}
			}
			// Do not advertise a single image surcharge if a custom expression changes it with video input.
			if len(quote.WithReference) > 0 && len(quote.WithoutReference) > 0 {
				var imageFee [2]float64
				for i, rates := range [][]canvasPriceAmount{quote.WithoutReference, quote.WithReference} {
					for _, rate := range rates {
						if rate.Unit == "reference_image" {
							imageFee[i] = rate.Amount
						}
					}
				}
				if imageFee[0] != imageFee[1] {
					return nil
				}
			}
			specs = append(specs, quote)
		}
	}
	return specs
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
	// A manifest declaration is not a route. Only shared models need their
	// provider quotes narrowed to the live bindings in the viewer's groups.
	var shared []string
	for _, item := range pricing {
		if len(item.BillingPluginVariants) > 1 {
			shared = append(shared, item.ModelName)
		}
	}
	routes, err := model.GetEnabledModelChannels(append(slices.Collect(maps.Keys(usableGroup)), "all"), shared)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	catalog := make([]catalogPricingModel, len(pricing))
	modelNames := make([]string, len(pricing))
	generation := jsplugin.DefaultRegistry.Generation()
	for i := range pricing {
		item := pricing[i]
		modelNames[i] = item.ModelName
		if len(item.BillingPluginVariants) > 1 {
			var active []model.PricingPluginVariant
			for _, variant := range item.BillingPluginVariants {
				plugin, ok := generation.Get(variant.PluginKey)
				if !ok {
					continue
				}
				for _, route := range routes[item.ModelName] {
					if route.Type == constant.ChannelTypeNewAPI && !plugin.Meta.SupportsUpstream(jsplugin.UpstreamKindNewAPI) {
						continue
					}
					matches := slices.Contains(plugin.Meta.ChannelTypes, route.Type)
					if route.Type == constant.ChannelTypeTaskPlugin || route.Type == constant.ChannelTypeNewAPI {
						// These are partial channel records. Decode without GetSetting, which
						// repairs malformed settings by saving the channel back to the DB.
						var setting dto.ChannelSettings
						matches = route.Setting != nil && common.UnmarshalJsonStr(*route.Setting, &setting) == nil && setting.BindsTaskPlugin(variant.PluginKey)
					}
					if matches {
						active = append(active, variant)
						break
					}
				}
			}
			item.BillingPluginVariants = active
			if len(active) == 1 {
				item.VideoSpecs = active[0].VideoSpecs
				item.ImageSpecs = active[0].ImageSpecs
				item.BillingUsageSchema = active[0].BillingUsageSchema
				item.BillingUsageExamples = active[0].BillingUsageExamples
			}
		}
		catalog[i] = catalogPricingModel{Pricing: item, ImageSpecs: catalogImageSpecs(item), VideoSpecs: catalogVideoSpecs(item)}
	}
	// check groupRatio contains usableGroup
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	c.JSON(200, gin.H{
		"success":                 true,
		"data":                    catalog,
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
