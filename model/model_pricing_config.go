package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PricingValues is one model's configuration, keyed by the existing option
// names. A missing key inherits the engine's default; an explicit zero is free.
type PricingValues map[string]any

type ModelPricingChange struct {
	ModelName       string        `json:"model_name"`
	ExpectedVersion string        `json:"expected_version"`
	Pricing         PricingValues `json:"pricing"`
	Reset           bool          `json:"reset,omitempty"`
}

type ModelPricingEntry struct {
	ModelPricingDescription
	PluginVariants []ModelPricingPluginVariant          `json:"plugin_variants,omitempty"`
	ModelName      string                               `json:"model_name"`
	Version        string                               `json:"version"`
	Configured     PricingValues                        `json:"configured"`
	UsageSchema    map[string]jsplugin.UsageFieldSchema `json:"usage_schema,omitempty"`
}

type ModelPricingPluginVariant struct {
	PluginKey     string                               `json:"plugin_key"`
	PluginName    string                               `json:"plugin_name"`
	Icon          string                               `json:"icon,omitempty"`
	UsageSchema   map[string]jsplugin.UsageFieldSchema `json:"usage_schema"`
	UsageExamples []jsplugin.UsageExample              `json:"usage_examples,omitempty"`
	Configured    string                               `json:"configured"`
	Effective     string                               `json:"effective"`
	DefaultExpr   string                               `json:"default_expr,omitempty"`
	// Mapped expressions remain independent of an unsaved client-model draft.
	MappedProviderExpr string `json:"mapped_provider_expr,omitempty"`
	MappedModelExpr    string `json:"mapped_model_expr,omitempty"`
	Compatible         bool   `json:"compatible"`
	Stale              bool   `json:"stale,omitempty"`
}

type ModelPricingSnapshot struct {
	Entries      []ModelPricingEntry `json:"entries"`
	Options      map[string]string   `json:"options"`
	EmptyVersion string              `json:"empty_version"`
}

var ErrModelPricingConflict = errors.New("model pricing changed; reload before saving")

// Lock order is stable across instances. Creating missing option rows inside
// the transaction also serializes the first write to an unconfigured database.
var modelPricingOptionKeys = []string{
	"AudioCompletionRatio", "AudioRatio", "CacheRatio", "CompletionRatio",
	"CreateCacheRatio", "ImageRatio", "ModelPrice", "ModelRatio",
	"billing_setting.billing_expr", "billing_setting.billing_mode", billing_setting.PluginBillingExprOption,
}

var modelPricingMutationMu sync.Mutex

func IsModelPricingOption(key string) bool {
	return slices.Contains(modelPricingOptionKeys, key)
}

func ModelPricingVersion(values PricingValues) string {
	encoded, _ := common.Marshal(values)
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func defaultPricingMaps() map[string]map[string]any {
	result := make(map[string]map[string]any, len(modelPricingOptionKeys))
	for _, key := range modelPricingOptionKeys {
		result[key] = make(map[string]any)
	}
	for key, values := range ratio_setting.GetDefaultPricingMaps() {
		for name, value := range values {
			result[key][name] = value
		}
	}
	return result
}

func clonePricingMaps(values map[string]map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any, len(values))
	for key, entries := range values {
		result[key] = maps.Clone(entries)
	}
	return result
}

func readModelPricingMaps(db *gorm.DB) (map[string]map[string]any, map[string]bool, []string, error) {
	var rows []Option
	if err := db.Where(commonKeyCol+" IN ?", modelPricingOptionKeys).Find(&rows).Error; err != nil {
		return nil, nil, nil, err
	}
	values := defaultPricingMaps()
	existing := make(map[string]bool)
	counts := make(map[string]int)
	for _, row := range rows {
		var entries map[string]any
		if err := common.UnmarshalJsonStr(row.Value, &entries); err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", row.Key, err)
		}
		if entries == nil {
			return nil, nil, nil, fmt.Errorf("%s must be a JSON object", row.Key)
		}
		values[row.Key] = entries
		existing[row.Key] = true
		counts[row.Key]++
	}
	var duplicated []string
	for _, key := range modelPricingOptionKeys {
		if counts[key] > 1 {
			duplicated = append(duplicated, key)
		}
	}
	return values, existing, duplicated, nil
}

func modelPricingValues(values map[string]map[string]any, name string) PricingValues {
	result := make(PricingValues)
	for _, key := range modelPricingOptionKeys {
		if key == billing_setting.PluginBillingExprOption {
			variants := make(map[string]any)
			for variant, expression := range values[key] {
				if plugin, model, ok := billing_setting.SplitPluginBillingExprKey(variant); ok && model == name {
					variants[plugin] = expression
				}
			}
			if len(variants) > 0 {
				result[key] = variants
			}
			continue
		}
		if value, exists := values[key][name]; exists {
			result[key] = value
		}
	}
	return result
}

// replaceModelPricing writes one complete model draft into the option maps.
// Plugin expressions are grouped in the draft and flattened only in storage.
func replaceModelPricing(values map[string]map[string]any, name string, draft PricingValues) error {
	for key, value := range draft {
		if !IsModelPricingOption(key) {
			return fmt.Errorf("unsupported pricing field: %s", key)
		}
		if key == billing_setting.PluginBillingExprOption {
			variants, ok := value.(map[string]any)
			if !ok || variants == nil {
				return errors.New("plugin billing expressions must be a plugin-to-expression object")
			}
			for plugin := range variants {
				if !jsplugin.ValidPluginKey(plugin) {
					return fmt.Errorf("invalid plugin billing expression key: %s", plugin)
				}
			}
		}
	}
	for _, key := range modelPricingOptionKeys {
		if key == billing_setting.PluginBillingExprOption {
			for variant := range values[key] {
				if _, model, ok := billing_setting.SplitPluginBillingExprKey(variant); ok && model == name {
					delete(values[key], variant)
				}
			}
			variants, _ := draft[key].(map[string]any)
			for plugin, expr := range variants {
				values[key][billing_setting.PluginBillingExprKey(plugin, name)] = expr
			}
			continue
		}
		delete(values[key], name)
		if value, exists := draft[key]; exists {
			values[key][name] = value
		}
	}
	return nil
}

func effectiveModelPricing(values map[string]map[string]any, name string) PricingValues {
	result := modelPricingValues(values, name)
	// Legacy wildcard aliases are resolved by the same normalization as relay.
	alias := ratio_setting.FormatMatchingModelName(name)
	for _, key := range []string{"ModelPrice", "ModelRatio", "CompletionRatio", "AudioRatio", "AudioCompletionRatio"} {
		delete(result, key)
		if value, exists := values[key][alias]; exists {
			result[key] = value
		}
	}
	mode, _ := result["billing_setting.billing_mode"].(string)
	if mode == "" {
		_, hasPrice := result["ModelPrice"]
		_, hasRatio := result["ModelRatio"]
		if _, builtin := billing_setting.GetBuiltinBillingExpr(name); builtin && !hasPrice && !hasRatio {
			mode = "tiered_expr"
		}
	}
	if mode == "tiered_expr" {
		result["billing_setting.billing_mode"] = mode
		if _, exists := result["billing_setting.billing_expr"]; !exists {
			if expression, ok := billing_setting.GetBuiltinBillingExpr(name); ok {
				result["billing_setting.billing_expr"] = expression
			}
		}
		return result
	}
	if _, exists := result["ModelPrice"]; exists {
		return result
	}
	if _, exists := result["ModelRatio"]; !exists && operation_setting.SelfUseModeEnabled {
		result["ModelRatio"] = float64(37.5)
	}
	// Completion ratios include engine-enforced model defaults. Expose their
	// effective value without persisting them into the editable configuration.
	var configuredCompletion *float64
	if ratio, exists := result["CompletionRatio"].(float64); exists {
		configuredCompletion = &ratio
	}
	result["CompletionRatio"] = ratio_setting.ResolveCompletionRatio(name, configuredCompletion).Ratio
	for key, fallback := range map[string]float64{
		"CacheRatio":       ratio_setting.DefaultCacheRatio,
		"CreateCacheRatio": ratio_setting.DefaultCreateCacheRatio,
		"ImageRatio":       ratio_setting.DefaultImageRatio,
	} {
		if _, exists := result[key]; !exists {
			result[key] = fallback
		}
	}
	return result
}

// PreviewModelPricing resolves a complete editable draft using the same defaults
// as the saved-price display and conversion. It has no write side effects.
func PreviewModelPricing(name string, draft PricingValues) (PricingValues, error) {
	if draft == nil {
		return nil, errors.New("pricing draft is required")
	}
	values, _, _, err := readModelPricingMaps(DB)
	if err != nil {
		return nil, err
	}
	previous := clonePricingMaps(values)
	if err := replaceModelPricing(values, name, draft); err != nil {
		return nil, err
	}
	if err := validateModelPricing(name, values, previous); err != nil {
		return nil, err
	}
	return effectiveModelPricing(values, name), nil
}

func GetModelPricingSnapshot(names []string) (*ModelPricingSnapshot, error) {
	values, _, _, err := readModelPricingMaps(DB)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		nameSet := make(map[string]bool)
		for key, entries := range values {
			for name := range entries {
				if key == billing_setting.PluginBillingExprOption {
					_, model, ok := billing_setting.SplitPluginBillingExprKey(name)
					if !ok {
						continue
					}
					name = model
				}
				nameSet[name] = true
			}
		}
		for name := range billing_setting.GetBuiltinBillingExprCopy() {
			nameSet[name] = true
		}
		for _, plugin := range jsplugin.DefaultRegistry.Generation().Plugins() {
			for name := range plugin.Meta.ModelDefaults {
				nameSet[name] = true
			}
		}
		for name := range nameSet {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := &ModelPricingSnapshot{Entries: make([]ModelPricingEntry, 0, len(names)), Options: make(map[string]string), EmptyVersion: ModelPricingVersion(PricingValues{})}
	generation := jsplugin.DefaultRegistry.Generation()
	for _, name := range names {
		configured := modelPricingValues(values, name)
		entry := ModelPricingEntry{ModelName: name, Version: ModelPricingVersion(configured), Configured: configured,
			ModelPricingDescription: ModelPricingDescription{Effective: effectiveModelPricing(values, name)}}
		entry.CacheWriteMode = ResolveCacheWriteMode(name, configured)
		entry.BillingDetails = ResolveLegacyBillingDetails(name, entry.Effective, configured)
		plugins := generation.PluginsByModel(name)
		usageModel := name
		if len(plugins) == 0 {
			if target, ok := ResolveTaskModelAlias(generation, name); ok {
				if plugin, ok := generation.Get(target.PluginKey); ok {
					entry.UsageSchema, _ = plugin.Meta.UsageForModel(target.Declared)
					if target.Declared != "" {
						plugins = generation.PluginsByModel(target.Declared)
						usageModel = target.Declared
					}
				}
			}
		}
		if len(plugins) > 0 {
			entry.UsageSchema, _ = plugins[0].Meta.UsageForModel(usageModel)
		}
		mappedConfigured, mappedEffective := configured, entry.Effective
		if usageModel != name {
			mappedConfigured = modelPricingValues(values, usageModel)
			mappedEffective = effectiveModelPricing(values, usageModel)
		}
		configuredVariants, _ := configured[billing_setting.PluginBillingExprOption].(map[string]any)
		if len(plugins) > 0 || len(configuredVariants) > 0 {
			keys := make(map[string]bool, len(plugins)+len(configuredVariants))
			for _, plugin := range plugins {
				keys[plugin.Meta.Key] = true
			}
			for key := range configuredVariants {
				keys[key] = true
			}
			for _, key := range slices.Sorted(maps.Keys(keys)) {
				configuredValue, overridden := configuredVariants[key]
				configuredExpr, _ := configuredValue.(string)
				plugin, exists := generation.Get(key)
				if !exists || !slices.Contains(plugin.Meta.Models, usageModel) {
					variant := ModelPricingPluginVariant{
						PluginKey: key, PluginName: key, Configured: configuredExpr,
						UsageSchema: map[string]jsplugin.UsageFieldSchema{}, Stale: true,
					}
					if exists {
						variant.PluginName, variant.Icon = plugin.Meta.Name, plugin.Meta.Icon
					}
					entry.PluginVariants = append(entry.PluginVariants, variant)
					continue
				}
				schema, examples := plugin.Meta.UsageForModel(usageModel)
				if schema == nil {
					schema = map[string]jsplugin.UsageFieldSchema{}
				}
				expression := configuredExpr
				defaultExpr, _ := billing_setting.GetPluginDefaultBillingExpr(key, usageModel)
				var mappedProviderExpr, mappedModelExpr string
				if usageModel != name {
					mappedVariants, _ := mappedConfigured[billing_setting.PluginBillingExprOption].(map[string]any)
					mappedProviderExpr, _ = mappedVariants[key].(string)
					if mappedEffective["billing_setting.billing_mode"] == billing_setting.BillingModeTieredExpr {
						mappedModelExpr, _ = mappedEffective["billing_setting.billing_expr"].(string)
					}
				}
				if !overridden {
					if mappedProviderExpr != "" {
						expression = mappedProviderExpr
					} else if entry.Effective["billing_setting.billing_mode"] == billing_setting.BillingModeTieredExpr {
						expression, _ = entry.Effective["billing_setting.billing_expr"].(string)
					} else if mappedModelExpr != "" {
						expression = mappedModelExpr
					} else {
						_, explicitMode := configured["billing_setting.billing_mode"]
						legacyName := ratio_setting.FormatMatchingModelName(name)
						_, hasPrice := values["ModelPrice"][legacyName]
						_, hasRatio := values["ModelRatio"][legacyName]
						if !explicitMode && !hasPrice && !hasRatio {
							expression = defaultExpr
						}
					}
				}
				entry.PluginVariants = append(entry.PluginVariants, ModelPricingPluginVariant{
					PluginKey: plugin.Meta.Key, PluginName: plugin.Meta.Name, Icon: plugin.Meta.Icon,
					UsageSchema: schema, UsageExamples: examples, Configured: configuredExpr, Effective: expression, DefaultExpr: defaultExpr,
					MappedProviderExpr: mappedProviderExpr, MappedModelExpr: mappedModelExpr,
					Compatible: billing_setting.TaskExprCompatible(expression, schema),
				})
			}
		}
		result.Entries = append(result.Entries, entry)
	}
	// Preserve the existing settings editor's full-map interface. Built-in
	// expressions are display defaults only; per-model writes do not persist them.
	for name, expression := range billing_setting.GetBuiltinBillingExprCopy() {
		effective := effectiveModelPricing(values, name)
		if effective["billing_setting.billing_mode"] != "tiered_expr" {
			continue
		}
		if _, ok := values["billing_setting.billing_mode"][name]; !ok {
			values["billing_setting.billing_mode"][name] = "tiered_expr"
		}
		if _, ok := values["billing_setting.billing_expr"][name]; !ok {
			values["billing_setting.billing_expr"][name] = expression
		}
	}
	for key, entries := range values {
		encoded, err := common.Marshal(entries)
		if err != nil {
			return nil, err
		}
		result.Options[key] = string(encoded)
	}
	return result, nil
}

// Reasons a plugin provider cannot charge a task model with the saved prices.
const (
	ModelPricingIssueUnconfigured = "unconfigured"
	ModelPricingIssueIncompatible = "incompatible"
	ModelPricingIssueStale        = "stale"
)

type ModelPricingIssue struct {
	ModelName  string `json:"model_name"`
	PluginKey  string `json:"plugin_key"`
	PluginName string `json:"plugin_name"`
	Reason     string `json:"reason"`
}

// GetModelPricingIssues reads the provider variants of the pricing snapshot for
// every model a live plugin declares or a saved provider price names. A
// provider with neither an expression nor a per-call price is unconfigured, one
// whose expression reads usage its schema lacks is incompatible, and a saved
// provider price whose plugin is not installed or whose installed version no
// longer declares the model is stale. A plugin switched off keeps its prices
// for when it is switched back on. An empty pluginKey covers every plugin.
func GetModelPricingIssues(pluginKey string) ([]ModelPricingIssue, error) {
	generation := jsplugin.DefaultRegistry.Generation()
	names := make(map[string]bool)
	for _, plugin := range generation.Plugins() {
		if pluginKey == "" || plugin.Meta.Key == pluginKey {
			for _, name := range plugin.Meta.Models {
				names[name] = true
			}
		}
	}
	// Models of the installed versions of plugins with saved prices that are
	// missing from the live generation, such as plugins switched off.
	stopped := make(map[string][]string)
	for key := range billing_setting.GetPluginBillingExprCopy() {
		plugin, name, ok := billing_setting.SplitPluginBillingExprKey(key)
		if !ok || (pluginKey != "" && plugin != pluginKey) {
			continue
		}
		names[name] = true
		if _, live := generation.Get(plugin); live {
			continue
		}
		if _, read := stopped[plugin]; read {
			continue
		}
		models, err := installedTaskPluginModels(plugin)
		if err != nil {
			return nil, err
		}
		stopped[plugin] = models
	}
	issues := make([]ModelPricingIssue, 0)
	if len(names) == 0 {
		return issues, nil
	}
	snapshot, err := GetModelPricingSnapshot(slices.Collect(maps.Keys(names)))
	if err != nil {
		return nil, err
	}
	for _, entry := range snapshot.Entries {
		// A model in expression mode never falls back to its per-call price.
		_, perCall := entry.Effective["ModelPrice"]
		perCall = perCall && entry.Effective["billing_setting.billing_mode"] != billing_setting.BillingModeTieredExpr
		for _, variant := range entry.PluginVariants {
			if pluginKey != "" && variant.PluginKey != pluginKey {
				continue
			}
			reason := ModelPricingIssueUnconfigured
			switch {
			case variant.Stale && slices.Contains(stopped[variant.PluginKey], entry.ModelName):
				continue
			case variant.Stale:
				reason = ModelPricingIssueStale
			case variant.Compatible:
				continue
			// fixed() marks ordinary channel pricing: the provider has no task price.
			case variant.Effective != "" && !billingexpr.UsesFixedPricing(variant.Effective):
				reason = ModelPricingIssueIncompatible
			case perCall:
				continue
			}
			issues = append(issues, ModelPricingIssue{ModelName: entry.ModelName, PluginKey: variant.PluginKey, PluginName: variant.PluginName, Reason: reason})
		}
	}
	return issues, nil
}

// installedTaskPluginModels returns the models declared by the installed version
// of a plugin: its active override row, or else its built-in. A plugin that is
// not installed declares none, and neither does a stored source that no longer
// compiles.
func installedTaskPluginModels(key string) ([]string, error) {
	row, err := GetTaskPluginVersion(key, "")
	if err == nil {
		loaded, compileErr := jsplugin.CompilePlugin(string(row.Source), jsplugin.Options{Key: row.Key, Version: row.Version})
		if compileErr != nil {
			return nil, nil
		}
		return loaded.Meta.Models, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	for _, meta := range jsplugin.DefaultRegistry.Snapshot().Factory {
		if meta.Key == key {
			return meta.Models, nil
		}
	}
	return nil, nil
}

func ValidateModelPricing(name string, values PricingValues) error {
	previous := defaultPricingMaps()
	for key, expression := range billing_setting.GetPluginBillingExprCopy() {
		previous[billing_setting.PluginBillingExprOption][key] = expression
	}
	if expression, ok := billing_setting.GetBillingExpr(name); ok {
		previous["billing_setting.billing_expr"][name] = expression
	}
	draft := clonePricingMaps(previous)
	if err := replaceModelPricing(draft, name, values); err != nil {
		return err
	}
	return validateModelPricing(name, draft, previous)
}

// Writes pass the locked database snapshot here, so allowing an unchanged stale
// override cannot bypass validation through an out-of-date process-local cache.
func validateModelPricing(name string, draft, previousDraft map[string]map[string]any) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("model name is required")
	}
	values := modelPricingValues(draft, name)
	previous := modelPricingValues(previousDraft, name)
	generation := jsplugin.DefaultRegistry.Generation()
	usageModel := name
	plugins := generation.PluginsByModel(name)
	alias, aliased := TaskAliasTarget{}, false
	if len(plugins) == 0 {
		alias, aliased = ResolveTaskModelAlias(generation, name)
		if aliased && alias.Declared != "" {
			usageModel = alias.Declared
			plugins = generation.PluginsByModel(usageModel)
		}
	}
	previousVariants, _ := previous[billing_setting.PluginBillingExprOption].(map[string]any)
	variants := map[string]any{}
	if value, exists := values[billing_setting.PluginBillingExprOption]; exists {
		var ok bool
		variants, ok = value.(map[string]any)
		if !ok || variants == nil {
			return errors.New("plugin billing expressions must be a plugin-to-expression object")
		}
		for key, value := range variants {
			expression, ok := value.(string)
			if !ok || strings.TrimSpace(expression) == "" {
				return fmt.Errorf("model %s: plugin %s: billing expression is required", name, key)
			}
			plugin, exists := generation.Get(key)
			declared := exists && slices.Contains(plugin.Meta.Models, usageModel)
			// An unchanged override stays saveable after its plugin disappears or
			// stops declaring the model. A sole provider also keeps it after
			// narrowing its usage profile; relay checks shared models against
			// each profile, so their prices must follow it.
			if previousVariants[key] == expression && (!declared || !generation.SharedModel(usageModel)) {
				continue
			}
			if !declared {
				return fmt.Errorf("model %s: plugin %s does not declare this model", name, key)
			}
			schema, _ := plugin.Meta.UsageForModel(usageModel)
			if err := billing_setting.SmokeTestTaskExpr(expression, schema); err != nil {
				return fmt.Errorf("model %s: plugin %s: %w", name, key, err)
			}
		}
	}
	for key, value := range values {
		if key == billing_setting.PluginBillingExprOption {
			continue
		}
		if !IsModelPricingOption(key) {
			return fmt.Errorf("unsupported pricing field: %s", key)
		}
		if key == "billing_setting.billing_mode" {
			if value != "ratio" && value != "tiered_expr" {
				return errors.New("invalid billing mode")
			}
			continue
		}
		if key == "billing_setting.billing_expr" {
			expression, ok := value.(string)
			if !ok || strings.TrimSpace(expression) == "" {
				return errors.New("billing expression is required")
			}
			// Even a model expression currently shadowed by every provider must
			// compile; only its schema-specific smoke tests can be skipped.
			if _, err := billingexpr.CompileFromCache(expression); err != nil {
				return fmt.Errorf("model %s: %w", name, err)
			}
			// An unchanged expression a sole provider has been charging with keeps
			// its price after the plugin narrows its usage profile; relay checks
			// shared models against each profile, so their prices must follow it.
			unchanged := previous[key] == expression
			var err error
			if len(plugins) > 0 {
				for _, plugin := range plugins {
					_, overridden := variants[plugin.Meta.Key]
					_, wasOverridden := previousVariants[plugin.Meta.Key]
					if usageModel != name {
						mappedKey := billing_setting.PluginBillingExprKey(plugin.Meta.Key, usageModel)
						_, mappedOverride := draft[billing_setting.PluginBillingExprOption][mappedKey]
						_, previousMappedOverride := previousDraft[billing_setting.PluginBillingExprOption][mappedKey]
						overridden = overridden || mappedOverride
						wasOverridden = wasOverridden || previousMappedOverride
					}
					if overridden || (unchanged && !wasOverridden && len(plugins) == 1) {
						continue
					}
					schema, _ := plugin.Meta.UsageForModel(usageModel)
					if err = billing_setting.SmokeTestTaskExpr(expression, schema); err != nil {
						return fmt.Errorf("model %s: plugin %s: %w", name, plugin.Meta.Key, err)
					}
				}
			} else if aliased {
				if plugin, ok := generation.Get(alias.PluginKey); ok {
					if !unchanged || generation.SharedModel(alias.Declared) {
						schema, _ := plugin.Meta.UsageForModel(alias.Declared)
						err = billing_setting.SmokeTestTaskExpr(expression, schema)
					}
				} else {
					err = billing_setting.SmokeTestExpr(expression)
				}
			} else if !unchanged || len(billingexpr.UsedUsageKeys(expression)) == 0 {
				err = billing_setting.SmokeTestExpr(expression)
			}
			// With no remaining plugin, an unchanged stored usage expression has
			// no schema to test. Preserve it so removing stale overrides or saving
			// other model prices does not become impossible.
			if err != nil {
				return fmt.Errorf("model %s: %w", name, err)
			}
			continue
		}
		number, ok := value.(float64)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
			return fmt.Errorf("%s must be a finite, non-negative number", key)
		}
	}
	if values["billing_setting.billing_mode"] == "tiered_expr" {
		if _, exists := values["billing_setting.billing_expr"]; !exists {
			if _, builtin := billing_setting.GetBuiltinBillingExpr(name); !builtin {
				return errors.New("billing expression is required")
			}
		}
	}
	return nil
}

// Validate alias dependents against the complete locked draft when a mapped
// price changes, including overrides removed in the same batch.
func validateModelPricingDraft(values, previous map[string]map[string]any, names map[string]bool) error {
	generation := jsplugin.DefaultRegistry.Generation()
	for name := range values["billing_setting.billing_expr"] {
		if len(generation.PluginsByModel(name)) > 0 {
			continue
		}
		if alias, ok := ResolveTaskModelAlias(generation, name); ok && names[alias.Declared] {
			names[name] = true
		}
	}
	for _, name := range slices.Sorted(maps.Keys(names)) {
		if err := validateModelPricing(name, values, previous); err != nil {
			return err
		}
	}
	return nil
}

func UpdateModelPricing(changes []ModelPricingChange) error {
	if len(changes) == 0 {
		return errors.New("select model pricing changes before saving")
	}
	seen := make(map[string]bool)
	for _, change := range changes {
		if seen[change.ModelName] {
			return errors.New("duplicate model pricing change")
		}
		seen[change.ModelName] = true
		if change.ExpectedVersion == "" {
			return ErrModelPricingConflict
		}
	}
	return mutateModelPricingOptions(func(_ *gorm.DB, values map[string]map[string]any) error {
		defaults := defaultPricingMaps()
		previous := clonePricingMaps(values)
		for _, change := range changes {
			if ModelPricingVersion(modelPricingValues(previous, change.ModelName)) != change.ExpectedVersion {
				return fmt.Errorf("%w: %s", ErrModelPricingConflict, change.ModelName)
			}
			pricing := change.Pricing
			if change.Reset {
				pricing = modelPricingValues(defaults, change.ModelName)
			}
			if err := replaceModelPricing(values, change.ModelName, pricing); err != nil {
				return err
			}
		}
		return validateModelPricingDraft(values, previous, seen)
	})
}

// UpdateModelPricingOptions keeps legacy single-option callers on the same
// locking, validation and transaction path as the model-level API.
func UpdateModelPricingOptions(updates map[string]string) error {
	return mutateModelPricingOptions(func(_ *gorm.DB, values map[string]map[string]any) error {
		previous := clonePricingMaps(values)
		names := make(map[string]bool)
		for key, raw := range updates {
			if !IsModelPricingOption(key) {
				return fmt.Errorf("unsupported pricing field: %s", key)
			}
			var entries map[string]any
			if err := common.UnmarshalJsonStr(raw, &entries); err != nil {
				return err
			}
			if entries == nil {
				return fmt.Errorf("%s must be a JSON object", key)
			}
			for _, entriesForKey := range []map[string]any{values[key], entries} {
				for name := range entriesForKey {
					if key == billing_setting.PluginBillingExprOption {
						_, model, ok := billing_setting.SplitPluginBillingExprKey(name)
						if !ok {
							return fmt.Errorf("invalid plugin billing expression key: %s", name)
						}
						name = model
					}
					names[name] = true
				}
			}
			values[key] = entries
		}
		return validateModelPricingDraft(values, previous, names)
	})
}

func mutateModelPricingOptions(mutate func(*gorm.DB, map[string]map[string]any) error) error {
	modelPricingMutationMu.Lock()
	defer modelPricingMutationMu.Unlock()
	var committed map[string]map[string]any
	err := DB.Transaction(func(tx *gorm.DB) error {
		values, existing, duplicated, err := readModelPricingMaps(lockForUpdate(tx))
		if err != nil {
			return err
		}
		if len(duplicated) > 0 {
			common.SysError(common.LogText("options table has duplicate pricing keys [%s]; the table is missing a primary key", strings.Join(duplicated, ", ")))
		}
		defaults := defaultPricingMaps()
		for _, key := range modelPricingOptionKeys {
			if existing[key] {
				continue
			}
			encoded, err := common.Marshal(defaults[key])
			if err != nil {
				return err
			}
			row := Option{Key: key, Value: string(encoded)}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
		if err := mutate(tx, values); err != nil {
			return err
		}
		for _, key := range modelPricingOptionKeys {
			encoded, err := common.Marshal(values[key])
			if err != nil {
				return err
			}
			if err := tx.Model(&Option{}).Where(commonKeyCol+" = ?", key).Update("value", string(encoded)).Error; err != nil {
				return err
			}
		}
		committed = values
		return nil
	})
	if err != nil {
		return err
	}
	for _, key := range modelPricingOptionKeys {
		encoded, _ := common.Marshal(committed[key])
		if err := updateOptionMap(key, string(encoded)); err != nil {
			return err
		}
	}
	RefreshPricing()
	ratio_setting.InvalidateExposedDataCache()
	return nil
}
