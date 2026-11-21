package controller

import (
	"cmp"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

type canvasCatalogModel struct {
	ID           string                          `json:"id"`
	DisplayName  string                          `json:"display_name,omitempty"`
	Description  string                          `json:"description,omitempty"`
	Tags         []canvasCatalogTag              `json:"tags,omitempty"`
	Family       string                          `json:"family,omitempty"`
	Icon         string                          `json:"icon,omitempty"`
	Capabilities []string                        `json:"capabilities"`
	Featured     bool                            `json:"featured,omitempty"`
	Parameters   map[string]canvasMediaParameter `json:"parameters,omitempty"`
	Price        *canvasCatalogPrice             `json:"price,omitempty"`
}

type canvasCatalogTag struct {
	Text  string `json:"text"`
	Color string `json:"color"`
}

// CanvasModelCatalog exposes presentation metadata, not channel configuration.
// Both the bridge and the user's current default model key are required.
func CanvasModelCatalog(c *gin.Context) {
	if !common.IsStolasCanvasClient(c.GetHeader("X-Stolas-Client-Secret")) {
		writeCanvasError(c, service.ErrAuthTokenInvalid)
		return
	}
	if _, err := service.CanvasIdentity(c.GetInt("id"), c.GetInt("token_id")); err != nil {
		writeCanvasError(c, err)
		return
	}
	groups, err := getModelListGroups(c)
	if err != nil {
		writeCanvasError(c, err)
		return
	}
	groupSet := make(map[string]string, len(groups.ownerGroups))
	for _, group := range groups.ownerGroups {
		groupSet[group] = group
	}
	pricing := model.GetPricing()
	if pricing == nil {
		writeCanvasError(c, service.ErrCanvasDisabled)
		return
	}
	limits, _ := common.GetContextKeyType[map[string]bool](c, constant.ContextKeyTokenModelLimit)
	limited := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	candidates := make([]model.Pricing, 0)
	ids := make([]string, 0)
	for _, item := range filterPricingByUsableGroups(pricing, groupSet) {
		name := item.ModelName
		if limited && !limits[name] && !limits[ratio_setting.RoutingMatchModelName(name)] {
			continue
		}
		if !common.IsImageGenerationModel(name) &&
			!slices.Contains(item.SupportedEndpointTypes, constant.EndpointTypeImageGeneration) &&
			!slices.Contains(item.SupportedEndpointTypes, constant.EndpointTypeOpenAIVideo) {
			continue
		}
		ids = append(ids, name)
		candidates = append(candidates, item)
	}
	channels, err := model.GetEnabledModelChannels(groups.ownerGroups, ids)
	if err != nil {
		writeCanvasError(c, err)
		return
	}
	display := model.GetModelSquareDisplay()
	positions := make(map[string]int, len(display.Order))
	for index, name := range display.Order {
		positions[name] = index + 1
	}
	result := make([]canvasCatalogModel, 0, len(candidates))
	generation := jsplugin.DefaultRegistry.Generation()
	// Keep callable aliases in the allowlist for existing Canvas mappings. An
	// alias is not made a recommendation simply because its target is featured.
	for _, item := range candidates {
		capabilities, documented, plugins := canvasMediaCapabilities(generation, item.ModelName, channels[item.ModelName])
		if len(capabilities) == 0 {
			continue
		}
		family := canvasModelFamily(item.ModelName, item.DisplayName, display.Models[item.ModelName].Family)
		decoration := display.Models[item.ModelName]
		entry := canvasCatalogModel{
			ID: item.ModelName, DisplayName: item.DisplayName, Description: canvasCatalogDescription(item.Description), Tags: canvasCatalogTags(decoration), Icon: canvasModelIcon(family, item.Icon),
			Family:       family,
			Capabilities: capabilities, Featured: decoration.Highlight,
		}
		if documented && len(capabilities) == 1 {
			entry.Parameters = canvasDocumentedParameters(item.Description, capabilities[0])
		}
		entry.Price = canvasPriceSummary(item, groups, plugins, entry.Parameters["resolution"].Values)
		result = append(result, entry)
	}
	slices.SortFunc(result, func(a, b canvasCatalogModel) int {
		if a.Featured != b.Featured {
			if a.Featured {
				return -1
			}
			return 1
		}
		aPos, bPos := positions[a.ID], positions[b.ID]
		if aPos == 0 {
			aPos = len(positions) + 1
		}
		if bPos == 0 {
			bPos = len(positions) + 1
		}
		return cmp.Or(cmp.Compare(aPos, bPos), cmp.Compare(a.ID, b.ID))
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func canvasCatalogDescription(description string) string {
	if heading := canvasParamHeading.FindStringIndex(description); heading != nil {
		description = description[:heading[0]]
	}
	return strings.Join(strings.Fields(description), " ")
}

func canvasCatalogTags(decoration model.ModelSquareDecoration) []canvasCatalogTag {
	badge := strings.TrimSpace(decoration.Badge)
	if badge == "" {
		return nil
	}
	color := map[string]string{"red": "pink", "amber": "orange", "blue": "blue", "emerald": "green", "violet": "purple"}[decoration.Color]
	if color == "" {
		color = "blue"
	}
	return []canvasCatalogTag{{Text: badge, Color: color}}
}

var canvasFamilySeparators = regexp.MustCompile(`[^a-z0-9]+`)

// Public display names help classify gateway aliases (e.g. ac-image-2).
// Match a specific variant before falling back to the site's broader series.
func canvasModelFamily(id, name, series string) string {
	label := "-" + canvasFamilySeparators.ReplaceAllString(strings.ToLower(id+" "+name+" "+series), "-") + "-"
	for _, family := range []string{"seedance-2-5-flash", "seedance-2-0-fast", "seedance-2-0-mini", "seedance-2-5", "seedance-2-0"} {
		if strings.Contains(label, "-"+family+"-") {
			return "seedance-" + strings.Replace(strings.TrimPrefix(family, "seedance-"), "-", ".", 1)
		}
	}
	for _, family := range []struct{ key, label string }{
		{"wan-3", "wan3"}, {"wan3", "wan3"}, {"minimax-h3", "MiniMax H3"},
		{"gemini-omni", "Gemini Omni"}, {"gemini-omini", "Gemini Omni"}, {"muse", "Muse"},
		{"seedream", "seedream"}, {"banana", "banana"}, {"image-2", "gpt-image-2"},
	} {
		if strings.Contains(label, "-"+family.key+"-") {
			return family.label
		}
	}
	return series
}

// The model library's endpoints aggregate across channels (including plugin
// channel types). Determine what the user's enabled routes actually serve.
func canvasMediaCapabilities(generation *jsplugin.RoutingGeneration, name string, routes []*model.Channel) ([]string, bool, map[string]bool) {
	if len(routes) == 0 {
		return nil, false, nil
	}
	lookup := name
	if declared, ok := generation.CanonicalModel(name); ok {
		lookup = declared
	} else if alias, ok := model.ResolveTaskModelAlias(generation, name); ok && alias.Declared != "" {
		lookup = alias.Declared
	}
	priced := helper.HasModelBillingConfig(name)
	var capabilities []string
	documented := true
	plugins := make(map[string]bool)
	for _, media := range []struct{ kind, path string }{
		{"image", "/v1/images/generations"},
		{"video", "/v1/videos"},
	} {
		bindings := generation.LookupEndpointCandidates(http.MethodPost, media.path, lookup)
		supported := false
		for _, route := range routes {
			if constant.IsAdvancedCustomChannel(route.Type) {
				config := route.GetOtherSettings().AdvancedCustom
				if priced && config != nil && config.SupportsPathForModel(media.path, name) {
					supported, documented = true, false
				}
				continue
			}
			pluginRoute := false
			for _, binding := range bindings {
				meta := binding.Plugin.Meta
				if route.Type == constant.ChannelTypeNewAPI && !meta.SupportsUpstream(jsplugin.UpstreamKindNewAPI) {
					continue
				}
				ok, _ := model.ChannelSatisfiesFilters(route, name, []dto.ChannelFilter{{
					Kind: dto.FilterTaskPluginIdentity, TaskPluginKey: meta.Key, TaskPluginChannelTypes: meta.ChannelTypes,
				}})
				if ok && (priced || helper.HasTaskModelBillingConfig(name, []*model.Channel{route})) {
					plugins[meta.Key] = true
					pluginRoute, supported = true, true
					documented = documented && meta.Key == "lecapi"
				}
			}
			nativeImage := media.kind == "image" && priced && route.Type != constant.ChannelTypeTaskPlugin && common.IsImageGenerationModel(name)
			if nativeImage && !pluginRoute {
				supported, documented = true, false
			}
		}
		if supported {
			capabilities = append(capabilities, media.kind)
		}
	}
	return capabilities, documented, plugins
}
