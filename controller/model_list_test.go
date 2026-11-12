package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type listModelsResponse struct {
	Success bool               `json:"success"`
	Data    []dto.OpenAIModels `json:"data"`
	Object  string             `json:"object"`
}

type userModelsResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}

func setupModelListControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	initModelListColumnNames(t)

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func initModelListColumnNames(t *testing.T) {
	t.Helper()

	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))

	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

func withTieredBillingConfig(t *testing.T, modes map[string]string, exprs map[string]string) {
	t.Helper()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if strings.HasPrefix(key, "billing_setting.") {
			saved[key] = value
		}
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		model.InvalidatePricingCache()
	})

	modeBytes, err := common.Marshal(modes)
	require.NoError(t, err)
	exprBytes, err := common.Marshal(exprs)
	require.NoError(t, err)

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": string(modeBytes),
		"billing_setting.billing_expr": string(exprBytes),
	}))
	model.InvalidatePricingCache()
}

func withSelfUseModeDisabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = false
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func withSelfUseModeEnabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func decodeListModelsPayload(t *testing.T, recorder *httptest.ResponseRecorder) listModelsResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload listModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, "list", payload.Object)
	return payload
}

func decodeListModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]struct{} {
	t.Helper()

	payload := decodeListModelsPayload(t, recorder)
	ids := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		ids[item.Id] = struct{}{}
	}
	return ids
}

func pricingByModelName(pricings []model.Pricing) map[string]model.Pricing {
	byName := make(map[string]model.Pricing, len(pricings))
	for _, pricing := range pricings {
		byName[pricing.ModelName] = pricing
	}
	return byName
}

func decodeUserModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload userModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	return payload.Data
}

func TestGetUserModelsFiltersByRequestedGroup(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1002,
		Username: "playground-model-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-default-only-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-disabled-model", ChannelId: 1, Enabled: false},
	}).Error)

	defaultRecorder := httptest.NewRecorder()
	defaultContext, _ := gin.CreateTestContext(defaultRecorder)
	defaultContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=default", nil)
	defaultContext.Set("id", 1002)

	GetUserModels(defaultContext)

	defaultModels := decodeUserModelsResponse(t, defaultRecorder)
	require.ElementsMatch(t, []string{"zz-default-only-model"}, defaultModels)

	vipRecorder := httptest.NewRecorder()
	vipContext, _ := gin.CreateTestContext(vipRecorder)
	vipContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=vip", nil)
	vipContext.Set("id", 1002)

	GetUserModels(vipContext)

	require.Empty(t, decodeUserModelsResponse(t, vipRecorder))
}

func TestGetUserModelsExpandsAutoGroupsInConfiguredOrder(t *testing.T) {
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalSpecialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.ReadAll()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		specialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
		specialGroups.Clear()
		specialGroups.AddAll(originalSpecialGroups)
	})

	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["vip","default","unavailable"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"auto":"自动分组","default":"默认分组","unavailable":"不可用分组"}`))
	specialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	specialGroups.Clear()
	specialGroups.Set("default", map[string]string{
		"+:vip":         "VIP 分组",
		"-:unavailable": "",
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1003,
		Username: "playground-auto-model-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "vip", Model: "zz-vip-model", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "zz-shared-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-default-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-shared-model", ChannelId: 2, Enabled: true},
		{Group: "unavailable", Model: "zz-unavailable-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=auto", nil)
	context.Set("id", 1003)

	GetUserModels(context)

	models := decodeUserModelsResponse(t, recorder)
	require.Len(t, models, 3)
	assert.ElementsMatch(t, []string{"zz-vip-model", "zz-shared-model"}, models[:2])
	assert.Equal(t, "zz-default-model", models[2])
}

func TestListModelsIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-tiered-visible-model":      "tiered_expr",
		"zz-tiered-empty-expr-model":   "tiered_expr",
		"zz-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-tiered-empty-expr-model": "   ",
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1001,
		Username: "model-list-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-tiered-visible-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-empty-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-missing-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-unpriced-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("id", 1001)

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-tiered-visible-model")
	require.NotContains(t, ids, "zz-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-unpriced-model")

	pricingByName := pricingByModelName(model.GetPricing())
	visiblePricing, ok := pricingByName["zz-tiered-visible-model"]
	require.True(t, ok)
	require.Equal(t, "tiered_expr", visiblePricing.BillingMode)
	require.NotEmpty(t, visiblePricing.BillingExpr)

	emptyExprPricing, ok := pricingByName["zz-tiered-empty-expr-model"]
	require.True(t, ok)
	require.Empty(t, emptyExprPricing.BillingMode)
	require.Empty(t, emptyExprPricing.BillingExpr)

	missingExprPricing, ok := pricingByName["zz-tiered-missing-expr-model"]
	require.True(t, ok)
	require.Empty(t, missingExprPricing.BillingMode)
	require.Empty(t, missingExprPricing.BillingExpr)
}

func TestListModelsIncludesPluginDefaultPricesDatabaseMatrix(t *testing.T) {
	withSelfUseModeDisabled(t)
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			_, err := jsplugin.DefaultRegistry.Register(`
export const meta = {
 apiVersion:1,key:"model-list-defaults",name:"Model list defaults",version:"1.0.0",author:{name:"Test"},
 models:["zz-bundled-video","zz-bundled-free","zz-bundled-image","zz-bundled-unpriced","zz-bundled-override","zz-bundled-blocked","zz-bundled-invalid","zz-bundled-other-group","zz-bundled-ordinary","zz-bundled-wrong-plugin","zz-bundled-disabled-channel","zz-bundled-new-api","zz-bundled-legacy"],
 channelTypes:[99001],upstreams:["new_api"],
 fetchMode:"per_task",usageSchema:{seconds:{type:"number",unit:"second"}},
 usageProfiles:[{models:["zz-bundled-image"],schema:{image_count:{type:"number",unit:"count"}}}],
 modelDefaults:{
  "zz-bundled-video":{billingExpr:'tier("video", u("seconds") * 0.384)'},
  "zz-bundled-free":{billingExpr:'tier("free", u("seconds") * 0)'},
  "zz-bundled-image":{billingExpr:'tier("image", u("image_count") * 0.12)'},
  "zz-bundled-blocked":{billingExpr:'tier("video", u("seconds") * 1)'},
  "zz-bundled-invalid":{billingExpr:'tier("video", u("seconds") * 1)'},
  "zz-bundled-other-group":{billingExpr:'tier("video", u("seconds") * 1)'},
  "zz-bundled-ordinary":{billingExpr:'tier("video", u("seconds") * 1)'},
  "zz-bundled-wrong-plugin":{billingExpr:'tier("video", u("seconds") * 1)'},
  "zz-bundled-disabled-channel":{billingExpr:'tier("video", u("seconds") * 1)'},
  "zz-bundled-new-api":{billingExpr:'tier("video", u("seconds") * 1)'},
  "zz-bundled-legacy":{billingExpr:'tier("video", u("seconds") * 1)'}
 }
};
export function buildSubmitRequest(){return {};}
export function parseSubmitResponse(){return {};}
export function buildQueryRequest(){return {};}
export function parseTaskResult(){return {};}
`, jsplugin.Options{})
			require.NoError(t, err)
			t.Cleanup(func() {
				if _, exists := jsplugin.DefaultRegistry.Get("model-list-defaults"); exists {
					require.NoError(t, jsplugin.DefaultRegistry.Unregister("model-list-defaults"))
				}
				require.NoError(t, jsplugin.DefaultRegistry.Unregister("model-list-unpriced"))
				model.InvalidatePricingCache()
			})
			_, err = jsplugin.DefaultRegistry.Register(`
export const meta = {
 apiVersion:1,key:"model-list-unpriced",name:"Unpriced provider",version:"1.0.0",author:{name:"Test"},
 models:["zz-bundled-wrong-plugin"],fetchMode:"per_task",usageSchema:{credits:{type:"number",unit:"credit"}}
};
export function buildSubmitRequest(){return {};}
export function parseSubmitResponse(){return {};}
export function buildQueryRequest(){return {};}
export function parseTaskResult(){return {};}
`, jsplugin.Options{})
			require.NoError(t, err)
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				"billing_setting.billing_mode":        `{"zz-bundled-blocked":"tiered_expr"}`,
				"billing_setting.billing_expr":        `{"zz-bundled-blocked":""}`,
				"billing_setting.plugin_billing_expr": `{"model-list-defaults::zz-bundled-override":"tier(\"override\", u(\"seconds\") * 0.5)","model-list-defaults::zz-bundled-invalid":"u(\"undeclared\")"}`,
			}))
			mapping := `{"zz-bundled-alias":"zz-bundled-video","zz-bundled-group-alias":"zz-bundled-video"}`
			channel := model.Channel{
				Name: "Bundled prices", Type: constant.ChannelTypeTaskPlugin, Status: common.ChannelStatusEnabled,
				Group: "media", Models: "zz-bundled-video,zz-bundled-free,zz-bundled-image,zz-bundled-unpriced,zz-bundled-override,zz-bundled-blocked,zz-bundled-invalid,zz-bundled-alias,zz-bundled-group-alias", ModelMapping: &mapping,
			}
			channel.SetSetting(dto.ChannelSettings{TaskPluginKey: "model-list-defaults"})
			require.NoError(t, db.Create(&channel).Error)
			require.NoError(t, channel.UpdateAbilities(db))
			otherMapping := `{"zz-bundled-group-alias":"zz-bundled-unpriced"}`
			otherChannel := model.Channel{
				Name: "Other group", Type: constant.ChannelTypeTaskPlugin, Status: common.ChannelStatusEnabled,
				Group: "other", Models: "zz-bundled-other-group,zz-bundled-group-alias", ModelMapping: &otherMapping,
			}
			otherChannel.SetSetting(dto.ChannelSettings{TaskPluginKey: "model-list-defaults"})
			require.NoError(t, db.Create(&otherChannel).Error)
			require.NoError(t, otherChannel.UpdateAbilities(db))
			for _, channel := range []model.Channel{
				{Name: "Ordinary only", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Group: "media", Models: "zz-bundled-ordinary,zz-bundled-other-group,zz-bundled-disabled-channel"},
				{Name: "Gateway without plugin binding", Type: constant.ChannelTypeNewAPI, Status: common.ChannelStatusEnabled, Group: "media", Models: "zz-bundled-ordinary"},
				{Name: "Wrong plugin", Type: constant.ChannelTypeTaskPlugin, Status: common.ChannelStatusEnabled, Group: "media", Models: "zz-bundled-wrong-plugin"},
				{Name: "Disabled plugin channel", Type: constant.ChannelTypeTaskPlugin, Status: common.ChannelStatusManuallyDisabled, Group: "media", Models: "zz-bundled-disabled-channel"},
				{Name: "Extended gateway", Type: constant.ChannelTypeNewAPI, Status: common.ChannelStatusEnabled, Group: "media", Models: "zz-bundled-new-api"},
				{Name: "Legacy plugin channel", Type: 99001, Status: common.ChannelStatusEnabled, Group: "media", Models: "zz-bundled-legacy"},
			} {
				pluginKey := "model-list-defaults"
				if channel.Name == "Wrong plugin" {
					pluginKey = "model-list-unpriced"
				}
				if channel.Type == constant.ChannelTypeTaskPlugin {
					channel.SetSetting(dto.ChannelSettings{TaskPluginKey: pluginKey})
				}
				if channel.Name == "Extended gateway" {
					channel.SetSetting(dto.ChannelSettings{TaskExtendPluginKeys: []string{pluginKey}})
				}
				require.NoError(t, db.Create(&channel).Error)
				require.NoError(t, channel.UpdateAbilities(db))
			}
			model.InitChannelCache()

			for _, tc := range []struct {
				name    string
				group   string
				limits  map[string]bool
				disable bool
				want    []string
			}{
				{name: "bundled prices and provider overrides", want: []string{"zz-bundled-video", "zz-bundled-free", "zz-bundled-image", "zz-bundled-override", "zz-bundled-alias", "zz-bundled-group-alias", "zz-bundled-new-api", "zz-bundled-legacy"}},
				{name: "token limits", limits: map[string]bool{"zz-bundled-video": true, "zz-bundled-unpriced": true, "zz-bundled-other-group": true}, want: []string{"zz-bundled-video"}},
				{name: "another token group uses its own mapped prices", group: "other", want: []string{"zz-bundled-other-group"}},
				{name: "disabled plugin", disable: true, want: []string{}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					if tc.disable {
						require.NoError(t, jsplugin.DefaultRegistry.Unregister("model-list-defaults"))
					}
					for _, memoryCache := range []bool{false, true} {
						common.MemoryCacheEnabled = memoryCache
						model.InitChannelCache()
						recorder := httptest.NewRecorder()
						ctx, _ := gin.CreateTestContext(recorder)
						ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
						common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
						group := tc.group
						if group == "" {
							group = "media"
						}
						common.SetContextKey(ctx, constant.ContextKeyTokenGroup, group)
						if tc.limits != nil {
							common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
							common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, tc.limits)
						}
						ListModels(ctx, constant.ChannelTypeOpenAI)
						ids := make([]string, 0)
						for name := range decodeListModelsResponse(t, recorder) {
							ids = append(ids, name)
						}
						assert.ElementsMatch(t, tc.want, ids, "memory cache: %v", memoryCache)
					}
				})
			}
		})
	}
}

func TestListModelsUsesAdvancedCustomEndpointTypesFromPricingCache(t *testing.T) {
	withSelfUseModeEnabled(t)
	db := setupModelListControllerTestDB(t)

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		model.InvalidatePricingCache()
	})

	require.NoError(t, db.Create(&model.User{
		Id:       1003,
		Username: "advanced-custom-model-list-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)

	channel := &model.Channel{
		Id:     701,
		Type:   constant.ChannelTypeAdvancedCustom,
		Key:    "advanced-custom-key",
		Status: common.ChannelStatusEnabled,
		Name:   "advanced-custom-channel",
		Group:  "default",
		Models: "gemini-3.5-flash",
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{
				{
					IncomingPath: "/v1/chat/completions",
					UpstreamPath: "/v1/chat/completions",
				},
				{
					IncomingPath: "/v1/responses",
					UpstreamPath: "/v1beta/models/{model}:generateContent",
					Converter:    "openai_responses_to_gemini_generate_content",
					Models:       []string{"re:^gemini-"},
				},
			},
		},
	})
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "default",
		Model:     "gemini-3.5-flash",
		ChannelId: 701,
		Enabled:   true,
	}).Error)

	model.InitChannelCache()
	model.GetPricing()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("id", 1003)

	ListModels(ctx, constant.ChannelTypeOpenAI)

	payload := decodeListModelsPayload(t, recorder)
	require.Len(t, payload.Data, 1)
	require.Equal(t, "gemini-3.5-flash", payload.Data[0].Id)
	require.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
	}, payload.Data[0].SupportedEndpointTypes)
}

func TestListModelsTokenLimitIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-token-tiered-visible-model":      "tiered_expr",
		"zz-token-tiered-empty-expr-model":   "tiered_expr",
		"zz-token-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-token-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-token-tiered-empty-expr-model": "",
	})
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-token-tiered-visible-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-token-tiered-empty-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-token-tiered-missing-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-token-unpriced-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, map[string]bool{
		"zz-token-tiered-visible-model":      true,
		"zz-token-tiered-empty-expr-model":   true,
		"zz-token-tiered-missing-expr-model": true,
		"zz-token-unpriced-model":            true,
	})

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-token-tiered-visible-model")
	require.NotContains(t, ids, "zz-token-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-token-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-token-unpriced-model")
}

func TestListModelsTokenLimitUsesResolvedCustomAutoGroups(t *testing.T) {
	withSelfUseModeEnabled(t)
	originalMax := setting.GetMaxTokenAutoGroups()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("5"))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(fmt.Sprintf("%d", originalMax)))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "vip", Model: "zz-vip-allowed", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "zz-vip-denied", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-default-outside-snapshot", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenGroup, "auto")
	common.SetContextKey(ctx, constant.ContextKeyTokenAutoGroups, []string{"vip"})
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, map[string]bool{
		"zz-vip-allowed":              true,
		"zz-default-outside-snapshot": true,
		"zz-not-enabled":              true,
	})

	ListModels(ctx, constant.ChannelTypeOpenAI)
	ids := decodeListModelsResponse(t, recorder)
	require.Equal(t, map[string]struct{}{"zz-vip-allowed": {}}, ids)

	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`))
	emptyRecorder := httptest.NewRecorder()
	emptyCtx, _ := gin.CreateTestContext(emptyRecorder)
	emptyCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(emptyCtx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenGroup, "auto")
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenAutoGroups, []string{"vip"})
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenModelLimit, map[string]bool{"zz-vip-allowed": true})

	require.NotPanics(t, func() {
		ListModels(emptyCtx, constant.ChannelTypeAnthropic)
	})
	var anthropicResponse struct {
		Data    []dto.AnthropicModel `json:"data"`
		FirstID string               `json:"first_id"`
		LastID  string               `json:"last_id"`
	}
	require.NoError(t, common.Unmarshal(emptyRecorder.Body.Bytes(), &anthropicResponse))
	require.Empty(t, anthropicResponse.Data)
	require.Empty(t, anthropicResponse.FirstID)
	require.Empty(t, anthropicResponse.LastID)
}

func TestSetupLoginDoesNotTouchPasswordWhenPasswordFieldOmitted(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}, &model.AuditLog{}, &model.UserSession{}, &model.TwoFA{}, &model.PasskeyCredential{}))

	hashedPassword, err := common.Password2Hash("CurrentPassword123")
	require.NoError(t, err)
	user := &model.User{
		Username: "twofa-user",
		Password: hashedPassword,
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(user).Error)

	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		setupLogin(&model.User{
			Id:          user.Id,
			AuthVersion: user.AuthVersion,
			Username:    user.Username,
			Role:        user.Role,
			Status:      user.Status,
			Group:       user.Group,
		}, nil, c)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, hashedPassword, stored.Password)
}
