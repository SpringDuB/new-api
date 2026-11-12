package billing_setting

import (
	"fmt"
	"maps"
	"math"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/samber/lo"
)

const (
	BillingModeRatio        = "ratio"
	BillingModeTieredExpr   = "tiered_expr"
	BillingModeField        = "billing_mode"
	BillingExprField        = "billing_expr"
	PluginBillingExprOption = "billing_setting.plugin_billing_expr"
)

// BillingSetting is managed by config.GlobalConfig.Register.
// DB keys: billing_setting.billing_mode, billing_setting.billing_expr,
// billing_setting.plugin_billing_expr
type BillingSetting struct {
	BillingMode       map[string]string `json:"billing_mode"`
	BillingExpr       map[string]string `json:"billing_expr"`
	PluginBillingExpr map[string]string `json:"plugin_billing_expr"`
}

var billingSetting = BillingSetting{
	BillingMode:       make(map[string]string),
	BillingExpr:       make(map[string]string),
	PluginBillingExpr: make(map[string]string),
}

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	if mode, ok := billingSetting.BillingMode[model]; ok {
		return mode
	}
	if _, ok := builtinBillingExpr[model]; ok {
		// Existing administrator-configured legacy prices take precedence over
		// a newly introduced built-in expression unless a mode was explicit.
		if ratio_setting.HasConfiguredModelRatio(model) {
			return BillingModeRatio
		}
		if _, configured := ratio_setting.GetModelPrice(model, false); configured {
			return BillingModeRatio
		}
		return BillingModeTieredExpr
	}
	return BillingModeRatio
}

func GetBillingExpr(model string) (string, bool) {
	if expr, ok := billingSetting.BillingExpr[model]; ok {
		return expr, true
	}
	if GetBillingMode(model) == BillingModeTieredExpr {
		expr, ok := builtinBillingExpr[model]
		return expr, ok
	}
	return "", false
}

func GetBuiltinBillingExpr(model string) (string, bool) {
	expression, ok := builtinBillingExpr[model]
	return expression, ok
}

func PluginBillingExprKey(pluginKey, model string) string {
	return pluginKey + "::" + model
}

func SplitPluginBillingExprKey(key string) (plugin, model string, ok bool) {
	plugin, model, ok = strings.Cut(key, "::")
	if !ok || !jsplugin.ValidPluginKey(plugin) || strings.TrimSpace(model) == "" {
		return "", "", false
	}
	return plugin, model, true
}

func GetPluginBillingExprCopy() map[string]string {
	return maps.Clone(billingSetting.PluginBillingExpr)
}

func GetPluginBillingExpr(pluginKey, model string) (string, bool) {
	expression, ok := billingSetting.PluginBillingExpr[PluginBillingExprKey(pluginKey, model)]
	return expression, ok
}

// GetPluginDefaultBillingExpr reads a price from the active plugin generation.
// It never saves defaults into administrator pricing or ordinary relay prices.
func GetPluginDefaultBillingExpr(pluginKey string, models ...string) (string, bool) {
	plugin, exists := jsplugin.DefaultRegistry.Get(pluginKey)
	if !exists {
		return "", false
	}
	defaults, exists := plugin.Meta.DefaultsForModel(models...)
	if !exists || strings.TrimSpace(defaults.BillingExpr) == "" {
		return "", false
	}
	return defaults.BillingExpr, true
}

// ResolveTaskBillingExpr prefers administrator prices to the executing plugin's
// defaults, retaining alias fallback and explicit legacy modes, including free.
func ResolveTaskBillingExpr(pluginKey, model, mappedModel string) (string, bool) {
	if pluginKey != "" {
		if expr, ok := GetPluginBillingExpr(pluginKey, model); ok {
			return expr, true
		}
		if mappedModel != "" && mappedModel != model {
			if expr, ok := GetPluginBillingExpr(pluginKey, mappedModel); ok {
				return expr, true
			}
		}
	}
	if GetBillingMode(model) == BillingModeTieredExpr {
		return GetBillingExpr(model)
	}
	if mappedModel != "" && mappedModel != model && GetBillingMode(mappedModel) == BillingModeTieredExpr {
		expression, ok := GetBillingExpr(mappedModel)
		return expression, ok && strings.TrimSpace(expression) != ""
	}
	// Per-call billing reads the client model only. A legacy price on the
	// mapped name cannot suppress a fallback that is otherwise usable here.
	if _, configured := billingSetting.BillingMode[model]; configured {
		return "", false
	}
	if ratio_setting.HasConfiguredModelRatio(model) {
		return "", false
	}
	if _, configured := ratio_setting.GetModelPrice(model, false); configured {
		return "", false
	}
	return GetPluginDefaultBillingExpr(pluginKey, mappedModel, model)
}

// TaskExprCompatible checks the schema contract even for usage references in
// branches that the current request would not evaluate.
func TaskExprCompatible(expression string, schema map[string]jsplugin.UsageFieldSchema) bool {
	if strings.TrimSpace(expression) == "" {
		return false
	}
	if _, err := billingexpr.CompileFromCache(expression); err != nil {
		return false
	}
	for key := range billingexpr.UsedUsageKeys(expression) {
		if _, exists := schema[key]; !exists {
			return false
		}
	}
	return !billingexpr.UsesFixedPricing(expression)
}

func GetBuiltinBillingExprCopy() map[string]string {
	return lo.Assign(builtinBillingExpr)
}

func GetBillingModeCopy() map[string]string {
	modes := lo.Assign(billingSetting.BillingMode)
	for model := range builtinBillingExpr {
		if _, configured := modes[model]; !configured && GetBillingMode(model) == BillingModeTieredExpr {
			modes[model] = BillingModeTieredExpr
		}
	}
	return modes
}

func GetBillingExprCopy() map[string]string {
	expressions := lo.Assign(billingSetting.BillingExpr)
	for model := range builtinBillingExpr {
		if _, configured := expressions[model]; configured {
			continue
		}
		if expression, ok := GetBillingExpr(model); ok {
			expressions[model] = expression
		}
	}
	return expressions
}

func GetPricingSyncData(base map[string]any) map[string]any {
	extra := make(map[string]any, 2)
	if modes := GetBillingModeCopy(); len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if exprs := GetBillingExprCopy(); len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	return lo.Assign(base, extra)
}

// ---------------------------------------------------------------------------
// Smoke test (called externally for validation before save)
// ---------------------------------------------------------------------------

func SmokeTestExpr(exprStr string) error {
	return smokeTestExpr(exprStr)
}

func smokeTestExpr(exprStr string) error {
	if _, err := billingexpr.CompileFromCache(exprStr); err != nil {
		return err
	}
	usageKeys := billingexpr.UsedUsageKeys(exprStr)
	if len(usageKeys) > 0 {
		sortedKeys := make([]string, 0, len(usageKeys))
		for key := range usageKeys {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)
		return fmt.Errorf("expression references usage keys %v but the model has no task plugin usage schema", sortedKeys)
	}

	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
		{P: 300, C: 100, Len: 1000, CR: 100, Img: 400, ImgCR: 200},
		{P: 800, C: 50, Len: 1000, AI: 200, AO: 50},
		{Len: math.MaxInt32, ImgCR: math.MaxInt32},
	}

	for _, v := range vectors {
		for _, request := range billingExprSmokeRequests() {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if math.IsNaN(result) || math.IsInf(result, 0) || result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result must be finite and non-negative, got %f", v.P, v.C, result)
			}
		}
	}
	return nil
}

// SmokeTestTaskExpr validates task prices with the same contract used for
// plugin-supplied model defaults.
func SmokeTestTaskExpr(exprStr string, schema map[string]jsplugin.UsageFieldSchema) error {
	return jsplugin.ValidateTaskBillingExpr(exprStr, schema)
}

func billingExprSmokeRequests() []billingexpr.RequestInput {
	return []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{
				"anthropic-beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast","stream_options":{"include_usage":true},"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	}
}
