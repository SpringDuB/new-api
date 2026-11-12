package model

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Plugin defaults must work on every supported database without writing model
// metadata or prices. Explicit administrator settings still own their values.
func TestPluginModelDefaultsDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(":memory:")
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN is not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN is not configured")
				}
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			previousDB, previousType := DB, common.MainDatabaseType()
			previousConfig := config.GlobalConfig.ExportAllConfigs()
			previousPrice, previousRatio := ratio_setting.ModelPrice2JSONString(), ratio_setting.ModelRatio2JSONString()
			previousLanguage, previousCache := common.DefaultLanguage, common.MemoryCacheEnabled
			previousSelfUse := operation_setting.SelfUseModeEnabled
			operation_setting.SelfUseModeEnabled = false
			common.OptionMapRWMutex.Lock()
			previousOptions := common.OptionMap
			common.OptionMap = make(map[string]string)
			common.OptionMapRWMutex.Unlock()
			DB = db
			common.SetMainDatabaseType(common.DatabaseType(dialect))
			common.DefaultLanguage = "zh-CN"
			common.MemoryCacheEnabled = true
			initCol()
			t.Cleanup(func() {
				for _, key := range []string{"portable-defaults", "portable-second"} {
					if _, exists := jsplugin.DefaultRegistry.Get(key); exists {
						require.NoError(t, jsplugin.DefaultRegistry.Unregister(key))
					}
				}
				require.NoError(t, db.Migrator().DropTable(&Ability{}, &Channel{}, &Model{}, &Vendor{}, &Option{}))
				DB = previousDB
				common.SetMainDatabaseType(previousType)
				common.DefaultLanguage, common.MemoryCacheEnabled = previousLanguage, previousCache
				operation_setting.SelfUseModeEnabled = previousSelfUse
				common.OptionMapRWMutex.Lock()
				common.OptionMap = previousOptions
				common.OptionMapRWMutex.Unlock()
				initCol()
				require.NoError(t, config.GlobalConfig.LoadFromDB(previousConfig))
				require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrice))
				require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousRatio))
				InitChannelCache()
				InvalidatePricingCache()
				require.NoError(t, sqlDB.Close())
			})
			require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}, &Model{}, &Vendor{}, &Option{}))
			var version string
			versionQuery := "SELECT version()"
			if dialect == "sqlite" {
				versionQuery = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
			t.Logf("database version: %s", version)
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				"billing_setting.billing_mode": `{}`, "billing_setting.billing_expr": `{}`,
				billing_setting.PluginBillingExprOption: `{}`,
			}))
			const source = `
export const meta = {
 apiVersion:1,key:"portable-defaults",name:"Portable defaults",version:"1.0.0",author:{name:"Test"},
 models:["portable-video","portable-image"],fetchMode:"per_task",
 usageSchema:{seconds:{type:"number",unit:"second"}},
 usageProfiles:[{models:["portable-image"],schema:{image_count:{type:"number",unit:"count"}}}],
 modelDefaults:{
  "portable-video":{vendor:"bytedance",displayName:"Portable video",description:{en:"Video reference requirements",zh:"视频参考素材要求"},icon:"Doubao.Color",tags:["Video"],billingExpr:'tier("video", u("seconds") * 0.6)'},
  "portable-image":{vendor:"google",description:{en:"Image requirements",zh:"图片素材要求"},icon:"OpenAI.Color",tags:["Image"],billingExpr:'tier("image", u("image_count") * 0.12)'}
 }
};
export function buildSubmitRequest(){return {};}
export function parseSubmitResponse(){return {};}
export function buildQueryRequest(){return {};}
export function parseTaskResult(){return {};}
`
			_, err = jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
			require.NoError(t, err)
			vendor := Vendor{Name: "ByteDance", Status: 1}
			require.NoError(t, db.Create(&vendor).Error)
			mapping := `{"portable-alias":"portable-video"}`
			channel := &Channel{Type: constant.ChannelTypeTaskPlugin, Status: 1, Name: "portable-defaults", Models: "portable-video,portable-image,portable-alias", ModelMapping: &mapping}
			channel.SetSetting(dto.ChannelSettings{TaskPluginKey: "portable-defaults"})
			require.NoError(t, db.Create(channel).Error)
			for _, name := range channel.GetModels() {
				insertPricingEndpointAbility(t, channel.Id, name)
			}
			InitChannelCache()
			InvalidatePricingCache()
			prices := pricingByModel(GetPricing())
			assert.Equal(t, "视频参考素材要求", prices["portable-video"].Description)
			assert.Equal(t, "Doubao.Color", prices["portable-video"].Icon)
			assert.Equal(t, vendor.Id, prices["portable-video"].VendorID)
			assert.Negative(t, prices["portable-image"].VendorID, "a missing vendor is presentation data only")
			assert.Equal(t, "Video", prices["portable-video"].Tags)
			assert.Equal(t, "视频参考素材要求", prices["portable-alias"].Description)
			for _, name := range []string{"portable-video", "portable-alias"} {
				require.Len(t, prices[name].BillingPluginVariants, 1)
				assert.Equal(t, `tier("video", u("seconds") * 0.6)`, prices[name].BillingPluginVariants[0].BillingExpr)
			}
			expression, exists := billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-alias", "portable-video")
			require.True(t, exists)
			assert.Equal(t, `tier("video", u("seconds") * 0.6)`, expression)
			assert.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode("portable-video"), "ordinary relays must not inherit plugin usage billing")
			rows, _, err := SearchModelsWithChannels("portable-video", "", "", "", 0, 20)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, "视频参考素材要求", rows[0].Description)
			assert.Equal(t, vendor.Id, rows[0].VendorID)
			filteredRows, _, err := SearchModelsWithChannels("portable-video", fmt.Sprint(vendor.Id), "", "", 0, 20)
			require.NoError(t, err)
			require.Len(t, filteredRows, 1, "plugin vendor assignments participate in model filtering")
			assert.Zero(t, rows[0].Id, "plugin metadata is an unsaved fallback")
			snapshot, err := GetModelPricingSnapshot(nil)
			require.NoError(t, err)
			var imageEntry *ModelPricingEntry
			for i := range snapshot.Entries {
				if snapshot.Entries[i].ModelName == "portable-image" {
					imageEntry = &snapshot.Entries[i]
				}
			}
			require.NotNil(t, imageEntry, "installed defaults must appear in the pricing editor")
			require.Len(t, imageEntry.PluginVariants, 1)
			assert.Equal(t, `tier("image", u("image_count") * 0.12)`, imageEntry.PluginVariants[0].Effective)
			assert.True(t, imageEntry.PluginVariants[0].Compatible)
			assert.Empty(t, imageEntry.PluginVariants[0].Configured)
			assert.Equal(t, imageEntry.PluginVariants[0].Effective, imageEntry.PluginVariants[0].DefaultExpr)
			operation_setting.SelfUseModeEnabled = true
			selfUse, err := GetModelPricingSnapshot([]string{"portable-image"})
			require.NoError(t, err)
			assert.Equal(t, imageEntry.PluginVariants[0].DefaultExpr, selfUse.Entries[0].PluginVariants[0].Effective, "self-use fallback ratio must not hide plugin pricing")
			operation_setting.SelfUseModeEnabled = false
			var modelRows, optionRows int64
			require.NoError(t, db.Model(&Model{}).Count(&modelRows).Error)
			require.NoError(t, db.Model(&Option{}).Count(&optionRows).Error)
			assert.Zero(t, modelRows)
			assert.Zero(t, optionRows)
			var vendorRows int64
			require.NoError(t, db.Model(&Vendor{}).Count(&vendorRows).Error)
			assert.EqualValues(t, 1, vendorRows, "plugin metadata reads never create vendors")
			secondSource := strings.ReplaceAll(strings.ReplaceAll(source, "portable-defaults", "portable-second"), `* 0.6`, `* 2`)
			secondSource = strings.ReplaceAll(secondSource, "seconds", "credits")
			_, err = jsplugin.DefaultRegistry.Register(secondSource, jsplugin.Options{})
			require.NoError(t, err)
			InvalidatePricingCache()
			assert.Len(t, pricingByModel(GetPricing())["portable-alias"].BillingPluginVariants, 2, "an alias retains every eligible provider's price")
			shared, err := GetModelPricingSnapshot([]string{"portable-alias"})
			require.NoError(t, err)
			require.Len(t, shared.Entries, 1)
			assert.Len(t, shared.Entries[0].PluginVariants, 2)
			assert.ErrorContains(t, ValidateModelPricing("portable-alias", PricingValues{"billing_setting.billing_expr": `u("seconds")`}), "portable-second")
			assert.NoError(t, ValidateModelPricing("portable-alias", PricingValues{
				billing_setting.PluginBillingExprOption: map[string]any{"portable-defaults": `u("seconds") * 4`},
			}))
			assert.NoError(t, ValidateModelPricing("portable-alias", PricingValues{
				"billing_setting.billing_expr":          `u("seconds")`,
				billing_setting.PluginBillingExprOption: map[string]any{"portable-second": `u("credits") * 4`},
			}))
			assert.ErrorContains(t, ValidateModelPricing("portable-alias", PricingValues{"unsupported": float64(1)}), "unsupported pricing field")
			assert.ErrorContains(t, ValidateModelPricing("portable-alias", PricingValues{billing_setting.PluginBillingExprOption: "invalid"}), "plugin-to-expression object")
			for _, key := range []string{"not valid!", "portable-defaults::other-model"} {
				assert.ErrorContains(t, ValidateModelPricing("portable-alias", PricingValues{
					billing_setting.PluginBillingExprOption: map[string]any{key: `invalid(`},
				}), "invalid plugin billing expression key")
			}
			expression, exists = billing_setting.ResolveTaskBillingExpr("portable-second", "portable-alias", "portable-video")
			assert.True(t, exists)
			assert.Equal(t, `tier("video", u("credits") * 2)`, expression)
			require.NoError(t, jsplugin.DefaultRegistry.Unregister("portable-second"))

			require.NoError(t, db.Create(&Model{ModelName: "portable-video", Description: "Administrator description", Status: 1}).Error)
			InvalidatePricingCache()
			assert.Equal(t, "Administrator description", pricingByModel(GetPricing())["portable-video"].Description)
			assert.Zero(t, pricingByModel(GetPricing())["portable-video"].VendorID, "an administrator's explicit no-vendor setting wins")
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"portable-video":0}`))
			_, exists = billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-video", "")
			assert.False(t, exists, "explicit free per-call pricing overrides the plugin default")
			expression, exists = billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-alias", "portable-video")
			assert.True(t, exists, "mapped legacy prices do not suppress defaults because per-call billing only reads the client model")
			assert.Equal(t, `tier("video", u("seconds") * 0.6)`, expression)
			aliasSnapshot, err := GetModelPricingSnapshot([]string{"portable-alias"})
			require.NoError(t, err)
			require.Len(t, aliasSnapshot.Entries[0].PluginVariants, 1)
			assert.Equal(t, expression, aliasSnapshot.Entries[0].PluginVariants[0].Effective)
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				billing_setting.PluginBillingExprOption: `{"portable-defaults::portable-video":"u(\"seconds\") * 2"}`,
			}))
			expression, exists = billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-video", "")
			require.True(t, exists)
			assert.Equal(t, `u("seconds") * 2`, expression)
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				billing_setting.PluginBillingExprOption: `{}`,
				"billing_setting.billing_mode":          `{"portable-video":"tiered_expr"}`,
				"billing_setting.billing_expr":          `{"portable-video":"u(\"seconds\") * 3"}`,
			}))
			expression, exists = billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-video", "")
			require.True(t, exists)
			assert.Equal(t, `u("seconds") * 3`, expression)
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				"billing_setting.billing_mode": `{}`, "billing_setting.billing_expr": `{}`,
			}))
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
			updated := strings.ReplaceAll(strings.ReplaceAll(source, `version:"1.0.0"`, `version:"1.1.0"`), `* 0.6`, `* 0.9`)
			_, err = jsplugin.DefaultRegistry.Register(updated, jsplugin.Options{})
			require.NoError(t, err)
			expression, exists = billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-video", "")
			require.True(t, exists)
			assert.Equal(t, `tier("video", u("seconds") * 0.9)`, expression)
			require.NoError(t, db.Model(&Option{}).Count(&optionRows).Error)
			assert.Zero(t, optionRows, "plugin updates never save their defaults into administrator settings")
			aliasSnapshot, err = GetModelPricingSnapshot([]string{"portable-alias"})
			require.NoError(t, err)
			require.NoError(t, UpdateModelPricing([]ModelPricingChange{{
				ModelName: "portable-alias", ExpectedVersion: aliasSnapshot.Entries[0].Version,
				Pricing: PricingValues{billing_setting.PluginBillingExprOption: map[string]any{"portable-defaults": `u("seconds") * 4`}},
			}}))
			expression, exists = billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-alias", "portable-video")
			assert.True(t, exists)
			assert.Equal(t, `u("seconds") * 4`, expression)
			aliasSnapshot, err = GetModelPricingSnapshot([]string{"portable-alias"})
			require.NoError(t, err)
			assert.Equal(t, expression, aliasSnapshot.Entries[0].PluginVariants[0].Configured)
			require.NoError(t, UpdateModelPricing([]ModelPricingChange{{
				ModelName: "portable-alias", ExpectedVersion: aliasSnapshot.Entries[0].Version, Pricing: PricingValues{},
			}}))
			_, err = jsplugin.DefaultRegistry.Register(secondSource, jsplugin.Options{})
			require.NoError(t, err)
			for _, aliasFirst := range []bool{true, false} {
				locked, err := GetModelPricingSnapshot([]string{"portable-alias", "portable-video"})
				require.NoError(t, err)
				changes := []ModelPricingChange{
					{ModelName: "portable-alias", ExpectedVersion: locked.Entries[0].Version, Pricing: PricingValues{
						"billing_setting.billing_mode": "tiered_expr", "billing_setting.billing_expr": `u("seconds") * 3`,
					}},
					{ModelName: "portable-video", ExpectedVersion: locked.Entries[1].Version, Pricing: PricingValues{
						"billing_setting.billing_mode": "tiered_expr", "billing_setting.billing_expr": `u("seconds") * 7`,
						billing_setting.PluginBillingExprOption: map[string]any{"portable-second": `u("credits") * 9`},
					}},
				}
				if !aliasFirst {
					slices.Reverse(changes)
				}
				require.NoError(t, UpdateModelPricing(changes), "validate the entire final draft independently of edit order")
				expression, exists = billing_setting.ResolveTaskBillingExpr("portable-second", "portable-alias", "portable-video")
				require.True(t, exists)
				assert.Equal(t, `u("credits") * 9`, expression)
				assert.NoError(t, ValidateModelPricing("portable-alias", PricingValues{"billing_setting.billing_expr": `u("seconds") * 3`}))
				locked, err = GetModelPricingSnapshot([]string{"portable-alias", "portable-video"})
				require.NoError(t, err)
				for _, variant := range locked.Entries[0].PluginVariants {
					assert.Equal(t, `u("seconds") * 7`, variant.MappedModelExpr)
					assert.Empty(t, variant.Configured)
					if variant.PluginKey == "portable-second" {
						assert.Equal(t, `u("credits") * 9`, variant.MappedProviderExpr)
						assert.Equal(t, variant.MappedProviderExpr, variant.Effective)
					} else {
						assert.Empty(t, variant.MappedProviderExpr)
						assert.Equal(t, `u("seconds") * 3`, variant.Effective)
					}
				}
				assert.ErrorContains(t, UpdateModelPricing([]ModelPricingChange{{
					ModelName: "portable-video", ExpectedVersion: locked.Entries[1].Version, Pricing: PricingValues{},
				}}), "portable-second", "removing a mapped override must validate the alias expression it exposes")
				assert.ErrorContains(t, UpdateModelPricingOptions(map[string]string{billing_setting.PluginBillingExprOption: `{}`}), "portable-second")
				for i := range changes {
					changes[i].Pricing = PricingValues{}
					for _, entry := range locked.Entries {
						if entry.ModelName == changes[i].ModelName {
							changes[i].ExpectedVersion = entry.Version
						}
					}
				}
				require.NoError(t, UpdateModelPricing(changes), "the complete removal draft is compatible in either order")
			}
			require.NoError(t, jsplugin.DefaultRegistry.Unregister("portable-second"))
			prefix := Model{ModelName: "portable-", NameRule: NameRulePrefix, Description: "Administrator prefix description", Status: 1}
			require.NoError(t, db.Create(&prefix).Error)
			rows, _, err = SearchModelsWithChannels("portable-image", "", "", "", 0, 20)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, "portable-image", rows[0].ModelName)
			assert.Equal(t, "Administrator prefix description", rows[0].Description)
			assert.Zero(t, rows[0].Id, "matching a prefix must not reuse that record's identity")
			require.NoError(t, db.Delete(&prefix).Error)
			require.NoError(t, jsplugin.DefaultRegistry.Unregister("portable-defaults"))
			_, exists = billing_setting.ResolveTaskBillingExpr("portable-defaults", "portable-image", "")
			assert.False(t, exists)
			InvalidatePricingCache()
			assert.Empty(t, pricingByModel(GetPricing())["portable-image"].Description)
		})
	}
}

func pricingUsagePluginSource(version, usageSchema string) string {
	return fmt.Sprintf(`
export const meta = {
  apiVersion: 1, key: "pricing-usage-probe", name: "Pricing Usage Probe", version: %q, author: {name: "Test"},
  models: ["pricing-usage-model"], fetchMode: "per_task", usageSchema: %s
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {}; }
`, version, usageSchema)
}

func TestPricingCarriesTaskUsageSchemaAndRefreshesWithPluginGeneration(t *testing.T) {
	resetPricingEndpointTestTables(t)
	const pluginKey = "pricing-usage-probe"
	initialSource := pricingUsagePluginSource("1.0.0", `{
  seconds: {type: "number", unit: "second", description: "Estimated duration."},
  action: {enum:["video"],enumLabels:{video:{en:"Generate video",zh:"生成视频"}}}
}`)
	_, err := jsplugin.DefaultRegistry.Register(initialSource, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(pluginKey) })

	insertPricingEndpointChannel(t, 901, constant.ChannelTypeTaskPlugin, dto.ChannelOtherSettings{})
	insertPricingEndpointAbility(t, 901, "pricing-usage-model")
	insertPricingEndpointAbility(t, 901, "ordinary-model")

	initialPricing := pricingByModel(GetPricing())
	require.Contains(t, initialPricing, "pricing-usage-model")
	require.Contains(t, initialPricing, "ordinary-model")
	assert.Equal(t, "second", initialPricing["pricing-usage-model"].BillingUsageSchema["seconds"].Unit)
	assert.Equal(t, "Estimated duration.", initialPricing["pricing-usage-model"].BillingUsageSchema["seconds"].Description["en"])
	assert.Equal(t, "生成视频", initialPricing["pricing-usage-model"].BillingUsageSchema["action"].EnumLabels["video"]["zh"])
	assert.Nil(t, initialPricing["ordinary-model"].BillingUsageSchema)

	updatedSource := pricingUsagePluginSource("1.1.0", `{
  seconds: {type: "number", unit: "second", description: "Measured duration."},
  clips: {type: "number", unit: "count", description: "Generated clip count."}
}`)
	_, err = jsplugin.DefaultRegistry.Register(updatedSource, jsplugin.Options{})
	require.NoError(t, err)
	lastGetPricingTime = time.Now().Add(-2 * time.Minute)

	refreshedPricing := pricingByModel(GetPricing())
	require.Len(t, refreshedPricing["pricing-usage-model"].BillingUsageSchema, 2)
	assert.Equal(t, "Measured duration.", refreshedPricing["pricing-usage-model"].BillingUsageSchema["seconds"].Description["en"])
	assert.Equal(t, "count", refreshedPricing["pricing-usage-model"].BillingUsageSchema["clips"].Unit)
}

func TestPricingAliasCarriesPluginUsageSchemaAndTailExpr(t *testing.T) {
	resetPricingEndpointTestTables(t)
	const pluginKey = "pricing-usage-probe"
	source := pricingUsagePluginSource("1.0.0", `{
  seconds: {type: "number", unit: "second", description: "Estimated duration."}
}`)
	_, err := jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(pluginKey) })

	mapping := `{"alias-model":"pricing-usage-model"}`
	channel := &Channel{
		Id:           910,
		Type:         constant.ChannelTypeTaskPlugin,
		Key:          "key-910",
		Status:       1,
		Name:         "channel-910",
		Models:       "alias-model,pricing-usage-model",
		ModelMapping: &mapping,
	}
	require.NoError(t, DB.Create(channel).Error)
	insertPricingEndpointAbility(t, 910, "alias-model")
	insertPricingEndpointAbility(t, 910, "pricing-usage-model")
	InitChannelCache()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"pricing-usage-model":"tiered_expr","alias-own-expr":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"pricing-usage-model":"u(\"seconds\")","alias-own-expr":"u(\"seconds\") * 2"}`,
	}))
	InvalidatePricingCache()

	pricing := pricingByModel(GetPricing())
	require.Contains(t, pricing, "alias-model")
	require.Contains(t, pricing, "pricing-usage-model")
	assert.Equal(t, "second", pricing["alias-model"].BillingUsageSchema["seconds"].Unit)
	assert.Equal(t, "Estimated duration.", pricing["alias-model"].BillingUsageSchema["seconds"].Description["en"])
	assert.Equal(t, "tiered_expr", pricing["alias-model"].BillingMode)
	assert.Equal(t, `u("seconds")`, pricing["alias-model"].BillingExpr)
	assert.Equal(t, "tiered_expr", pricing["pricing-usage-model"].BillingMode)
	assert.Equal(t, `u("seconds")`, pricing["pricing-usage-model"].BillingExpr)

	ownMapping := `{"alias-own-expr":"pricing-usage-model"}`
	own := &Channel{
		Id:           911,
		Type:         constant.ChannelTypeTaskPlugin,
		Key:          "key-911",
		Status:       1,
		Name:         "channel-911",
		Models:       "alias-own-expr,pricing-usage-model",
		ModelMapping: &ownMapping,
	}
	require.NoError(t, DB.Create(own).Error)
	insertPricingEndpointAbility(t, 911, "alias-own-expr")
	InitChannelCache()
	InvalidatePricingCache()

	refreshed := pricingByModel(GetPricing())
	assert.Equal(t, `u("seconds") * 2`, refreshed["alias-own-expr"].BillingExpr)
	assert.Equal(t, "second", refreshed["alias-own-expr"].BillingUsageSchema["seconds"].Unit)
}

func pricingByModel(pricings []Pricing) map[string]Pricing {
	result := make(map[string]Pricing, len(pricings))
	for _, pricing := range pricings {
		result[pricing.ModelName] = pricing
	}
	return result
}

func TestPricingUsageProfilesAndAliases(t *testing.T) {
	resetPricingEndpointTestTables(t)
	require.NoError(t, DB.AutoMigrate(&Option{}))
	const key = "pricing-profiles"
	const source = `
export const meta = {
 apiVersion:1,key:"pricing-profiles",name:"Pricing Profiles",version:"1.0.0",author:{name:"Test"},
 models:["profile-image","profile-video"],fetchMode:"per_task",
 usageSchema:{fallback:{type:"number",unit:"count"}},
 usageExamples:[{label:"default",facts:{fallback:1}}],
 usageProfiles:[
  {models:["profile-image"],schema:{image_count:{type:"number",unit:"count"}}},
  {models:["profile-video"],schema:{seconds:{type:"number",unit:"second"}},examples:[{label:"5 seconds",facts:{seconds:5}}]}
 ]
};
export function buildSubmitRequest(){return {};}
export function parseSubmitResponse(){return {};}
export function buildQueryRequest(){return {};}
export function parseTaskResult(){return {};}
`
	_, err := jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(key) })
	for index, mapping := range []string{
		`{"image-alias":"profile-image","video-alias":"profile-video","ambiguous-alias":"profile-image"}`,
		`{"ambiguous-alias":"profile-video"}`,
	} {
		id := 920 + index
		require.NoError(t, DB.Create(&Channel{
			Id: id, Type: constant.ChannelTypeTaskPlugin, Status: 1, Name: fmt.Sprintf("channel-%d", id),
			Models: "profile-image,profile-video,image-alias,video-alias,ambiguous-alias", ModelMapping: &mapping,
		}).Error)
		for _, name := range []string{"profile-image", "profile-video", "image-alias", "video-alias", "ambiguous-alias"} {
			insertPricingEndpointAbility(t, id, name)
		}
	}
	InitChannelCache()
	pricing := pricingByModel(GetPricing())
	names := []string{"profile-image", "profile-video", "image-alias", "video-alias", "ambiguous-alias"}
	snapshot, err := GetModelPricingSnapshot(names)
	require.NoError(t, err)
	entries := make(map[string]ModelPricingEntry, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		entries[entry.ModelName] = entry
	}
	for _, tc := range []struct {
		name, field, unit string
		wantExamples      int
	}{
		{"profile-image", "image_count", "count", 0},
		{"profile-video", "seconds", "second", 1},
		{"image-alias", "image_count", "count", 0},
		{"video-alias", "seconds", "second", 1},
		{"ambiguous-alias", "fallback", "count", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := map[string]jsplugin.UsageFieldSchema{tc.field: {Type: "number", Unit: tc.unit}}
			assert.Equal(t, expected, pricing[tc.name].BillingUsageSchema)
			assert.Equal(t, expected, entries[tc.name].UsageSchema)
			assert.Len(t, pricing[tc.name].BillingUsageExamples, tc.wantExamples)
			require.NoError(t, ValidateModelPricing(tc.name, PricingValues{"billing_setting.billing_expr": `u("` + tc.field + `")`}))
			require.ErrorContains(t, ValidateModelPricing(tc.name, PricingValues{"billing_setting.billing_expr": `u("missing")`}), "not declared")
		})
	}
	require.ErrorContains(t, ValidateModelPricing("profile-image", PricingValues{"billing_setting.billing_expr": `u("seconds")`}), "not declared")
	require.ErrorContains(t, ValidateModelPricing("video-alias", PricingValues{"billing_setting.billing_expr": `u("image_count")`}), "not declared")

	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(saved)) })
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_expr":          `{"profile-image":"u(\"image_count\")"}`,
		billing_setting.PluginBillingExprOption: `{"pricing-profiles::profile-image":"u(\"image_count\") * 2"}`,
	}))
	updated := strings.Replace(source, `version:"1.0.0"`, `version:"1.1.0"`, 1)
	updated = strings.Replace(updated, `image_count:{type:"number",unit:"count"}`, `images:{type:"number",unit:"count"}`, 1)
	_, err = jsplugin.DefaultRegistry.Register(updated, jsplugin.Options{})
	require.NoError(t, err)
	// Stored prices that read a fact the narrowed profile dropped stay saveable
	// while unchanged; edited ones must follow the new profile.
	stored := PricingValues{"billing_setting.billing_expr": `u("image_count")`, billing_setting.PluginBillingExprOption: map[string]any{key: `u("image_count") * 2`}}
	require.NoError(t, ValidateModelPricing("profile-image", stored))
	require.ErrorContains(t, ValidateModelPricing("profile-image", PricingValues{"billing_setting.billing_expr": `u("image_count") * 3`}), "not declared")
	stored[billing_setting.PluginBillingExprOption] = map[string]any{key: `u("image_count") * 3`}
	require.ErrorContains(t, ValidateModelPricing("profile-image", stored), "not declared")
	InvalidatePricingCache()
	refreshed := pricingByModel(GetPricing())
	assert.Equal(t, map[string]jsplugin.UsageFieldSchema{"images": {Type: "number", Unit: "count"}}, refreshed["image-alias"].BillingUsageSchema)
	assert.Equal(t, "second", refreshed["profile-video"].BillingUsageSchema["seconds"].Unit)
}

func TestPricingSharedPluginVariantsUseEachSchemaAndExpression(t *testing.T) {
	resetPricingEndpointTestTables(t)
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(saved)) })
	for _, spec := range []struct{ key, field, unit string }{{"pricing-alpha", "seconds", "second"}, {"pricing-beta", "credits", "credit"}} {
		source := strings.ReplaceAll(pricingUsagePluginSource("1.0.0", `{`+spec.field+`:{type:"number",unit:"`+spec.unit+`"}}`), "pricing-usage-probe", spec.key)
		_, err := jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
		require.NoError(t, err)
		t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(spec.key) })
	}
	insertPricingEndpointChannel(t, 930, constant.ChannelTypeTaskPlugin, dto.ChannelOtherSettings{})
	insertPricingEndpointAbility(t, 930, "pricing-usage-model")
	const expression = `tier("base", u("seconds") * 0.4)`
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":          `{"pricing-usage-model":"tiered_expr"}`,
		"billing_setting.billing_expr":          `{"pricing-usage-model":"tier(\"base\", u(\"seconds\") * 0.4)"}`,
		billing_setting.PluginBillingExprOption: `{}`,
	}))
	prices := pricingByModel(GetPricing())
	variants := prices["pricing-usage-model"].BillingPluginVariants
	require.Len(t, variants, 2)
	assert.Equal(t, "pricing-alpha", variants[0].PluginKey)
	assert.Equal(t, "pricing-beta", variants[1].PluginKey)
	assert.Equal(t, expression, variants[0].BillingExpr)
	assert.Empty(t, variants[1].BillingExpr)
	assert.Equal(t, "credit", variants[1].BillingUsageSchema["credits"].Unit)
	assert.Equal(t, expression, prices["pricing-usage-model"].BillingExpr)
	assert.Contains(t, prices["pricing-usage-model"].BillingUsageSchema, "seconds")
	draft := PricingValues{"billing_setting.billing_expr": expression}
	require.ErrorContains(t, ValidateModelPricing("pricing-usage-model", draft), "plugin pricing-beta")
	draft[billing_setting.PluginBillingExprOption] = map[string]any{"pricing-beta": `tier("beta", u("credits") * 2)`}
	require.NoError(t, ValidateModelPricing("pricing-usage-model", draft))
	for _, invalid := range []any{map[string]any{"missing": "1"}, map[string]any{"pricing-beta": float64(1)}, map[string]any{"pricing-beta": `u("seconds")`}, map[string]any(nil), []any{}, "invalid"} {
		draft[billing_setting.PluginBillingExprOption] = invalid
		assert.Error(t, ValidateModelPricing("pricing-usage-model", draft))
	}
	draft = PricingValues{
		"billing_setting.billing_expr":          "invalid expression(",
		billing_setting.PluginBillingExprOption: map[string]any{"pricing-alpha": expression, "pricing-beta": `u("credits")`},
	}
	require.ErrorContains(t, ValidateModelPricing("pricing-usage-model", draft), "compile")
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		billing_setting.PluginBillingExprOption: `{"pricing-beta::pricing-usage-model":"tier(\"beta\", u(\"credits\") * 2)"}`,
	}))
	InvalidatePricingCache()
	priced := pricingByModel(GetPricing())["pricing-usage-model"].BillingPluginVariants
	require.Len(t, priced, 2)
	assert.Equal(t, `tier("beta", u("credits") * 2)`, priced[1].BillingExpr)
	previousPrice := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrice)) })
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"pricing-usage-model":0.25}`))
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"billing_setting.billing_mode": `{"pricing-usage-model":"ratio"}`}))
	InvalidatePricingCache()
	mixed := pricingByModel(GetPricing())["pricing-usage-model"].BillingPluginVariants
	require.Len(t, mixed, 2)
	assert.Equal(t, billing_setting.BillingModeRatio, mixed[0].BillingMode)
	assert.Equal(t, billing_setting.BillingModeTieredExpr, mixed[1].BillingMode)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{billing_setting.PluginBillingExprOption: `{}`}))
	InvalidatePricingCache()
	perCall := pricingByModel(GetPricing())["pricing-usage-model"]
	assert.Empty(t, perCall.BillingPluginVariants)
	assert.Equal(t, 1, perCall.QuotaType)
	assert.Equal(t, 0.25, perCall.ModelPrice)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		billing_setting.PluginBillingExprOption: `{"pricing-beta::pricing-usage-model":"tier(\"beta\", u(\"credits\") * 2)"}`,
	}))
	require.NoError(t, jsplugin.DefaultRegistry.Unregister("pricing-alpha"))
	InvalidatePricingCache()
	single := pricingByModel(GetPricing())["pricing-usage-model"]
	require.Len(t, single.BillingPluginVariants, 1)
	assert.Equal(t, "pricing-beta", single.BillingPluginVariants[0].PluginKey)
	assert.Equal(t, `tier("beta", u("credits") * 2)`, single.BillingPluginVariants[0].BillingExpr)
	encoded, err := common.Marshal(single)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), "billing_plugin_variants")
}

// Prices saved while a plugin fit stop fitting when it changes: a dropped model
// leaves its provider price stale, and a new provider without the usage the
// model expression reads cannot charge it. Unpriced models are listed too,
// including those whose only expression prices ordinary channels.
func TestModelPricingIssuesFollowPluginChanges(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Option{}))
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		require.NoError(t, DB.Where(commonKeyCol+" IN ?", modelPricingOptionKeys).Delete(&Option{}).Error)
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	pluginSource := func(key, name, version, models, usageSchema string) string {
		source := strings.ReplaceAll(pricingUsagePluginSource(version, usageSchema), "pricing-usage-probe", key)
		source = strings.Replace(source, "Pricing Usage Probe", name, 1)
		return strings.Replace(source, `["pricing-usage-model"]`, models, 1)
	}
	savePricing := func(options map[string]string) {
		for key, value := range options {
			require.NoError(t, DB.Save(&Option{Key: key, Value: value}).Error)
		}
		require.NoError(t, config.GlobalConfig.LoadFromDB(options))
	}
	const seconds = `{seconds:{type:"number",unit:"second"}}`
	_, err := jsplugin.DefaultRegistry.Register(pluginSource("issue-alpha", "Alpha", "1.0.0", `["issue-priced","issue-per-call","issue-unpriced","issue-ordinary","issue-shared","issue-dropped"]`, seconds), jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister("issue-alpha") })
	savePricing(map[string]string{
		"ModelPrice":                            `{"issue-per-call":0.3}`,
		"billing_setting.billing_mode":          `{"issue-priced":"tiered_expr","issue-ordinary":"tiered_expr","issue-shared":"tiered_expr"}`,
		"billing_setting.billing_expr":          `{"issue-priced":"u(\"seconds\") * 0.1","issue-ordinary":"tier(\"image\", fixed(0.02)) * image_count","issue-shared":"u(\"seconds\") * 0.2"}`,
		billing_setting.PluginBillingExprOption: `{"issue-alpha::issue-dropped":"u(\"seconds\") * 0.5"}`,
	})
	unpriced := ModelPricingIssue{ModelName: "issue-unpriced", PluginKey: "issue-alpha", PluginName: "Alpha", Reason: ModelPricingIssueUnconfigured}
	ordinary := ModelPricingIssue{ModelName: "issue-ordinary", PluginKey: "issue-alpha", PluginName: "Alpha", Reason: ModelPricingIssueUnconfigured}
	// Other tests leave their plugins in the shared registry.
	ownIssues := func() []ModelPricingIssue {
		issues, err := GetModelPricingIssues("")
		require.NoError(t, err)
		return slices.DeleteFunc(issues, func(issue ModelPricingIssue) bool { return !strings.HasPrefix(issue.PluginKey, "issue-") })
	}

	assert.Equal(t, []ModelPricingIssue{ordinary, unpriced}, ownIssues())

	_, err = jsplugin.DefaultRegistry.Register(pluginSource("issue-alpha", "Alpha", "1.1.0", `["issue-priced","issue-per-call","issue-unpriced","issue-ordinary","issue-shared"]`, seconds), jsplugin.Options{})
	require.NoError(t, err)
	_, err = jsplugin.DefaultRegistry.Register(pluginSource("issue-beta", "Beta", "1.0.0", `["issue-shared"]`, `{credits:{type:"number",unit:"credit"}}`), jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister("issue-beta") })
	incompatible := ModelPricingIssue{ModelName: "issue-shared", PluginKey: "issue-beta", PluginName: "Beta", Reason: ModelPricingIssueIncompatible}

	assert.Equal(t, []ModelPricingIssue{
		{ModelName: "issue-dropped", PluginKey: "issue-alpha", PluginName: "Alpha", Reason: ModelPricingIssueStale},
		ordinary,
		incompatible,
		unpriced,
	}, ownIssues())
	issues, err := GetModelPricingIssues("issue-beta")
	require.NoError(t, err)
	assert.Equal(t, []ModelPricingIssue{incompatible}, issues)

	// A provider price that reads the provider's own usage replaces the model
	// expression for that provider only.
	savePricing(map[string]string{
		billing_setting.PluginBillingExprOption: `{"issue-alpha::issue-dropped":"u(\"seconds\") * 0.5","issue-beta::issue-shared":"u(\"credits\") * 2"}`,
	})
	issues, err = GetModelPricingIssues("issue-beta")
	require.NoError(t, err)
	assert.Empty(t, issues)
}
