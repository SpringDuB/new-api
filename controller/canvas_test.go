package controller

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func canvasDatabase(t *testing.T, kind, dsn string) *gorm.DB {
	t.Helper()
	db, _ := newAuditTestDatabase(t, kind, dsn)
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	previousDB, previousLogDB, previousRedis := model.DB, model.LOG_DB, common.RedisEnabled
	previousType := common.MainDatabaseType()
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	if kind == "mysql" {
		common.SetMainDatabaseType(common.DatabaseTypeMySQL)
	} else if kind == "postgres" {
		common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
	} else {
		common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	}
	t.Cleanup(func() {
		connection, err := db.DB()
		if err == nil {
			_ = connection.Close()
		}
		model.DB, model.LOG_DB, common.RedisEnabled = previousDB, previousLogDB, previousRedis
		common.SetMainDatabaseType(previousType)
	})
	t.Setenv("STOLAS_CANVAS_URL", "https://canvas.stolas.cc")
	t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", "test-canvas-secret-with-at-least-32-characters")
	for range 2 {
		require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.AuthFlow{}, &model.UserSession{}))
	}
	versionQuery := "SELECT version()"
	if kind == "sqlite" {
		versionQuery = "SELECT sqlite_version()"
	}
	var version string
	require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
	t.Logf("%s engine: %s", kind, version)
	return db
}

func TestCanvasModelCatalogDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			require.NoError(t, db.AutoMigrate(&model.Token{}))
			model.InvalidatePricingCache()
			t.Cleanup(model.InvalidatePricingCache)
			withSelfUseModeDisabled(t)
			const secret = "fixture-only-canvas-secret-at-least-32-characters"
			t.Setenv("STOLAS_CANVAS_URL", "https://canvas.stolas.cc")
			t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", secret)
			user := model.User{Username: "catalog-user", Group: "default", Status: common.UserStatusEnabled, AffCode: "catalog-user"}
			require.NoError(t, db.Create(&user).Error)
			var token *model.Token
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				_, initialized, err := model.EnsureCanvasDefaultTokenWithTx(tx, user.Id)
				token = initialized
				return err
			}))
			other := model.Token{UserId: user.Id, Key: "nondefaultkey", Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true}
			require.NoError(t, db.Create(&other).Error)
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"gpt-image-hidden":1,"gpt-image-1":1,"gpt-image-private":1,"catalog-video":1}`))
			common.OptionMap[model.ModelSquareDisplayOption] = `{"order":["seedance-2-0-per-request","gpt-image-1"],"models":{"gpt-image-1":{"highlight":true,"tags":["image-extra"]},"seedance-2-0-per-request":{"badge":"推荐","color":"violet","tags":["video-extra"]}}}`
			channels := []model.Channel{
				{Name: "not-public-channel", Key: "not-public-upstream-key", Type: constant.ChannelTypeTaskPlugin, Group: "default", Models: "seedance-2-0-per-request", Status: common.ChannelStatusEnabled},
				{Name: "backup", Key: "backup-key", Type: constant.ChannelTypeTaskPlugin, Group: "default", Models: "seedance-2-0-per-request", Status: common.ChannelStatusEnabled},
				{Name: "other group", Key: "other-key", Type: constant.ChannelTypeOpenAI, Group: "private", Models: "gpt-image-private", Status: common.ChannelStatusEnabled},
				{Name: "image", Key: "image-key", Type: constant.ChannelTypeOpenAI, Group: "default", Models: "gpt-image-1,gpt-image-unpriced,gpt-image-hidden,catalog-video,seedance-2-0-per-request", Status: common.ChannelStatusEnabled},
			}
			for i := range 2 {
				channels[i].SetSetting(dto.ChannelSettings{TaskPluginKey: "lecapi"})
			}
			for i := range channels {
				require.NoError(t, channels[i].Insert())
			}
			metadata := []model.Model{
				{ModelName: "gpt-image-hidden", Status: 1},
				{ModelName: "gpt-image-1", Status: 1, Description: "Image generation model", Tags: "image,official"},
				{ModelName: "seedance-2-0-per-request", Status: 1, Description: "Video generation model; images: 最多 3", Tags: "video;cinematic"},
			}
			require.NoError(t, db.Create(&metadata).Error)
			require.NoError(t, db.Model(&metadata[0]).Update("status", 0).Error)
			engine := gin.New()
			engine.GET("/api/canvas/model-catalog", middleware.DisableCache(), middleware.TokenAuth(), CanvasModelCatalog)
			read := func(key, bridge string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/canvas/model-catalog", nil)
				req.Header.Set("Authorization", "Bearer sk-"+key)
				req.Header.Set("X-Stolas-Client-Secret", bridge)
				res := httptest.NewRecorder()
				engine.ServeHTTP(res, req)
				return res
			}
			for _, denied := range []*httptest.ResponseRecorder{read(token.Key, ""), read("invalid", secret), read(other.Key, secret)} {
				assert.Equal(t, http.StatusUnauthorized, denied.Code)
				assert.Contains(t, denied.Header().Get("Cache-Control"), "no-store")
				assert.NotContains(t, denied.Body.String(), "seedance-2-0-per-request")
			}
			catalog := func() []canvasCatalogModel {
				res := read(token.Key, secret)
				require.Equal(t, http.StatusOK, res.Code, res.Body.String())
				assert.Contains(t, res.Header().Get("Cache-Control"), "no-store")
				assert.NotContains(t, res.Body.String(), "not-public")
				assert.NotContains(t, res.Body.String(), "billing_expr")
				assert.NotContains(t, res.Body.String(), "plugin_key")
				var payload struct{ Data []canvasCatalogModel }
				require.NoError(t, common.Unmarshal(res.Body.Bytes(), &payload))
				return payload.Data
			}
			entries := catalog()
			require.Len(t, entries, 2)
			image, video := entries[0], entries[1]
			assert.Equal(t, "gpt-image-1", image.ID)
			assert.True(t, image.Featured)
			assert.Equal(t, "Image generation model", image.Description)
			assert.Empty(t, image.Tags)
			require.NotNil(t, image.Price)
			assert.Equal(t, []canvasPriceRate{{Unit: "request", Min: 1, Max: 1}}, image.Price.Rates)
			assert.Equal(t, "seedance-2-0-per-request", video.ID)
			assert.Equal(t, "Seedance 2.0 · 按次", video.DisplayName)
			assert.Equal(t, "Video generation model", video.Description)
			assert.Equal(t, []canvasCatalogTag{{Text: "推荐", Color: "purple"}}, video.Tags)
			assert.Equal(t, "seedance-2.0", video.Family)
			assert.Equal(t, "ByteDance.Color", video.Icon)
			assert.Equal(t, []string{"video"}, video.Capabilities)
			assert.Equal(t, 3, video.Parameters["images"].Max)
			assert.Equal(t, []string{"16:9", "9:16", "4:3", "3:4", "1:1"}, video.Parameters["aspect_ratio"].Values)
			assert.Equal(t, canvasMediaParameter{Min: 5, Max: 15, Default: "5", Required: true}, video.Parameters["duration"])
			require.NotNil(t, video.Price)
			assert.NotEmpty(t, video.Price.Rates)
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
			assert.Equal(t, []canvasCatalogModel{image, video}, catalog(), "a backup route keeps the same published ID")
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channels[1].Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
			assert.Equal(t, []canvasCatalogModel{image}, catalog(), "the last disabled route withdraws the model, even with warm pricing metadata")
			require.NoError(t, db.Model(token).Updates(map[string]any{"model_limits_enabled": true, "model_limits": "seedance-2-0-per-request"}).Error)
			assert.Equal(t, []canvasCatalogModel{}, catalog(), "model limits can produce a successful empty result")
			require.NoError(t, db.Model(token).Update("expired_time", time.Now().Add(-time.Minute).Unix()).Error)
			assert.Equal(t, http.StatusUnauthorized, read(token.Key, secret).Code)

			// The public square must not promise plugin limits for the generic OpenAI route above.
			mapping := `{"square-sd25":"seedance-2-5-per-second"}`
			square := model.Channel{Name: "square fixture", Key: "fixture", Type: constant.ChannelTypeTaskPlugin, Group: "default", Models: "square-sd25,seedance-2-5-per-request,ms-minimax-h3,ms-minimax-h3-upscale,ap-minimax-h3,st-gpt-image-2", ModelMapping: &mapping, Status: common.ChannelStatusEnabled}
			square.SetSetting(dto.ChannelSettings{TaskPluginKey: "lecapi"})
			require.NoError(t, square.Insert())
			model.InitChannelCache()
			model.InvalidatePricingCache()
			var public struct {
				Data []struct {
					ModelName  string             `json:"model_name"`
					VideoSpecs []catalogVideoSpec `json:"video_specs"`
					ImageSpecs []catalogImageSpec `json:"image_specs"`
				}
			}
			modelManagementRequest(t, GetPricing, http.MethodGet, "/api/pricing", nil, &public)
			entriesByName := make(map[string][]catalogVideoSpec)
			var pooledImage []catalogImageSpec
			for _, entry := range public.Data {
				entriesByName[entry.ModelName] = entry.VideoSpecs
				if entry.ModelName == "st-gpt-image-2" {
					pooledImage = entry.ImageSpecs
				}
			}
			require.Len(t, pooledImage, 3)
			require.Contains(t, entriesByName, "seedance-2-0-per-request")
			assert.Empty(t, entriesByName["seedance-2-0-per-request"])
			shared := entriesByName["seedance-2-5-per-request"]
			require.Len(t, shared, 1, "only the bound LecAPI channel can quote SD2.5")
			assert.Empty(t, shared[0].Provider)
			for _, row := range shared {
				assert.Equal(t, "720p", row.Resolution)
				assert.Equal(t, []canvasPriceAmount{{"request", 5.5}}, row.WithoutReference)
			}
			ms := entriesByName["ms-minimax-h3"]
			require.Len(t, ms, 5)
			for _, row := range ms {
				assert.Equal(t, 5, row.IncludedImages)
				assert.Contains(t, row.WithoutReference, canvasPriceAmount{"reference_image", 0.24})
				if row.Resolution == "480p" {
					assert.Empty(t, row.WithReference)
				}
				if row.Resolution == "768P" && row.DurationTier == "long" {
					assert.Equal(t, []canvasPriceAmount{{"second", 0.24}, {"reference_second", 0.6}, {"reference_image", 0.24}}, row.WithReference)
				}
				if row.Resolution == "2K" {
					assert.Equal(t, "short", row.DurationTier)
				}
			}
			require.Len(t, entriesByName["ms-minimax-h3-upscale"], 1)
			assert.True(t, entriesByName["ms-minimax-h3-upscale"][0].RequiresSourceTask)
			// Remove the unrelated generic route, then publish both actual task providers for SD2.
			for _, key := range []string{"lecapi", "yuanxiang"} {
				ch := model.Channel{Name: key, Key: "fixture", Type: constant.ChannelTypeTaskPlugin, Group: "default", Models: "seedance-2-0-per-request", Status: common.ChannelStatusEnabled}
				ch.SetSetting(dto.ChannelSettings{TaskPluginKey: key})
				require.NoError(t, ch.Insert())
			}
			require.NoError(t, db.Model(&channels[3]).Update("status", common.ChannelStatusManuallyDisabled).Error)
			require.NoError(t, model.UpdateAbilityStatus(channels[3].Id, false))
			model.InitChannelCache()
			model.InvalidatePricingCache()
			public.Data = nil // Decode a fresh response; omitted fields must not retain the previous catalog.
			modelManagementRequest(t, GetPricing, http.MethodGet, "/api/pricing", nil, &public)
			var sd2 []catalogVideoSpec
			for _, entry := range public.Data {
				if entry.ModelName == "seedance-2-0-per-request" {
					sd2 = entry.VideoSpecs
				}
			}
			require.Len(t, sd2, 2)
			var supportsVideo int
			for _, row := range sd2 {
				require.Equal(t, []canvasPriceAmount{{"request", 2.9}}, row.WithoutReference)
				if len(row.WithReference) > 0 {
					supportsVideo++
					assert.Equal(t, []canvasPriceAmount{{"request", 3.3}}, row.WithReference)
				}
			}
			assert.Equal(t, 1, supportsVideo)

			idx := slices.IndexFunc(entriesByName["square-sd25"], func(spec catalogVideoSpec) bool { return spec.Preferred })
			require.NotEqual(t, -1, idx, "plugin model mapping carries its preferred resolution to /api/pricing")
			selected := entriesByName["square-sd25"][idx]
			assert.Equal(t, "720p", selected.Resolution)
			assert.Equal(t, []canvasPriceAmount{{Unit: "second", Amount: 0.39}}, selected.WithoutReference)

		})
	}
}

func TestCanvasLecAPIVideoPoolRatios(t *testing.T) {
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		ratios []string
	}{
		{"seedance-2-5-per-second", []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}},
		{"seedance-2-5-per-token", []string{"16:9", "9:16", "1:1", "21:9", "4:3", "3:4", "adaptive"}},
		{"seedance-2-0-per-second", []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}},
		{"seedance-2-0-per-token", []string{"16:9", "9:16", "1:1", "21:9", "4:3", "3:4"}},
		{"seedance-2-0-fast-per-second", []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "auto"}},
		{"seedance-2-0-mini-per-second", []string{"16:9", "9:16", "1:1", "auto"}},
		{"wan-3-0-per-second", []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "adaptive"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaults, ok := plugin.Meta.DefaultsForModel(tc.name)
			require.True(t, ok)
			for _, lang := range []string{"en", "zh"} {
				params := canvasDocumentedParameters(defaults.Description[lang], "video")
				assert.Equal(t, tc.ratios, params["aspect_ratio"].Values, lang)
				assert.Equal(t, "16:9", params["aspect_ratio"].Default, lang)
				assert.NotEmpty(t, params["resolution"].Values, lang)
				assert.Positive(t, params["duration"].Max, lang)
			}
		})
	}
}

func TestCanvasLecAPIVideoFramesFollowUpstreamModel(t *testing.T) {
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	for _, name := range []string{"seedance-2-0-per-token", "seedance-2-0-fast-per-token", "seedance-2-0-mini-per-token", "seedance-2-5-per-token", "ap-muse-video-720-10s", "am-minimax-h3-official", "wan-3-0-per-second", "ms-minimax-h3"} {
		_, frames := canvasLecAPIVideoContract(plugin.Meta, name, &model.Channel{})
		assert.Equal(t, 2, frames, name)
	}
	mapping := `{"custom":"route","route":"lec-wan-3-0-per-second","wan-3-0-per-second":"wan-3-0-per-request"}`
	channel := &model.Channel{ModelMapping: &mapping}
	_, frames := canvasLecAPIVideoContract(plugin.Meta, "custom", channel)
	assert.Equal(t, 2, frames)
	for _, name := range []string{"wan-3-0-per-second", "seedance-2-5-01"} {
		_, frames := canvasLecAPIVideoContract(plugin.Meta, name, channel)
		assert.Zero(t, frames, name)
	}
}

func TestCanvasModelCatalogPriceSummary(t *testing.T) {
	group := modelListGroups{userGroup: "default", ownerGroups: []string{"default"}}
	for _, tc := range []struct {
		name, expression string
		want             []canvasPriceRate
		varies           bool
	}{
		{"fixed", `tier("x", fixed(0.12)) * image_count`, []canvasPriceRate{{Unit: "image", Min: 0.12, Max: 0.12}}, false},
		{"second", `u("resolution") == "720p" ? tier("x", u("seconds") * 0.4) : tier("y", u("seconds") * 0.84)`, []canvasPriceRate{{Unit: "second", Min: 0.4, Max: 0.84}}, true},
		{"token", `tier("x", u("tokens") * 26.625 / 1000000)`, []canvasPriceRate{{Unit: "video_tokens", Min: 26.625, Max: 26.625}}, false},
		{"dynamic", `tier("x", u("seconds") * 0.4) * (hour("UTC") == 9 ? 2 : 1)`, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := model.Pricing{EnableGroup: []string{"default"}, BillingExpr: tc.expression}
			if tc.name != "fixed" {
				item.BillingUsageSchema = map[string]jsplugin.UsageFieldSchema{"seconds": {Unit: "second"}, "tokens": {Unit: "token"}}
			}
			price := canvasPriceSummary(item, group, nil, nil, false, nil)
			require.NotNil(t, price)
			assert.Equal(t, tc.want, price.Rates)
			assert.Equal(t, tc.varies, price.Varies)
		})
	}
}

func TestCanvasModelCatalogCreditEstimateMetadata(t *testing.T) {
	settings := operation_setting.GetGeneralSetting()
	previous, previousRate := *settings, operation_setting.USDExchangeRate
	t.Cleanup(func() {
		*settings = previous
		operation_setting.USDExchangeRate = previousRate
	})
	operation_setting.USDExchangeRate = 7.3
	settings.CustomCurrencyExchangeRate = 2
	groups := modelListGroups{userGroup: "default", ownerGroups: []string{"default"}}
	item := model.Pricing{ModelName: "ap-muse-video-720-10s", ModelPrice: 0.6, QuotaType: 1, EnableGroup: []string{"default"}}
	for _, display := range []string{operation_setting.QuotaDisplayTypeUSD, operation_setting.QuotaDisplayTypeCNY, operation_setting.QuotaDisplayTypeCustom, operation_setting.QuotaDisplayTypeTokens} {
		settings.QuotaDisplayType = display
		price := canvasPriceSummary(item, groups, nil, nil, false, nil)
		require.NotNil(t, price)
		assert.Equal(t, []canvasPriceAmount{{Unit: "request", Amount: 0.6}}, price.Estimate, display)
	}
	assert.Equal(t, "Meta.Color", canvasModelIcon(canvasModelFamily(item.ModelName, "Muse Video · 720p", ""), "Video"))
}

func TestCanvasModelCatalogResolutionEstimate(t *testing.T) {
	groups := modelListGroups{userGroup: "default", ownerGroups: []string{"default"}}
	resolutions := []string{"480p", "720p"}
	schema := map[string]jsplugin.UsageFieldSchema{
		"resolution": {Enum: resolutions}, "video_input": {Enum: []string{"none", "video"}},
		"seconds": {Unit: "second"}, "requests": {Unit: "count"},
	}
	for _, tc := range []struct {
		name, expression string
		want             map[string][]canvasPriceAmount
	}{
		{"per second", `u("resolution") == "480p" ? tier("x", u("seconds") * 0.30) : tier("y", u("seconds") * 0.42)`, map[string][]canvasPriceAmount{
			"480p": {{Unit: "second", Amount: 0.30}}, "720p": {{Unit: "second", Amount: 0.42}},
		}},
		{"per request", `u("resolution") == "720p" ? tier("x", u("requests") * 6.6) : tier("y", u("requests") * 4.2)`, map[string][]canvasPriceAmount{
			"480p": {{Unit: "request", Amount: 4.2}}, "720p": {{Unit: "request", Amount: 6.6}},
		}},
		{"ambiguous only at 720p", `u("resolution") == "720p" && u("video_input") == "video" ? tier("x", u("seconds") * 0.42) : tier("y", u("seconds") * 0.30)`, map[string][]canvasPriceAmount{
			"480p": {{Unit: "second", Amount: 0.30}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := model.Pricing{EnableGroup: []string{"default"}, BillingExpr: tc.expression, BillingUsageSchema: schema}
			price := canvasPriceSummary(item, groups, nil, resolutions, true, nil)
			require.NotNil(t, price)
			assert.Empty(t, price.Estimate)
			assert.Equal(t, tc.want, price.EstimatesByResolution)
		})
	}
	t.Run("routes disagree", func(t *testing.T) {
		item := model.Pricing{EnableGroup: []string{"default"}, BillingPluginVariants: []model.PricingPluginVariant{
			{PluginKey: "one", BillingExpr: `tier("x", u("requests") * 4.2)`, BillingUsageSchema: schema},
			{PluginKey: "two", BillingExpr: `tier("x", u("requests") * 6.6)`, BillingUsageSchema: schema},
		}}
		price := canvasPriceSummary(item, groups, map[string]bool{"one": true, "two": true}, resolutions, true, nil)
		require.NotNil(t, price)
		assert.Empty(t, price.Estimate)
		assert.Empty(t, price.EstimatesByResolution)
	})
}

func TestCanvasLecAPIPriceEstimatesUseSelectedVideoInput(t *testing.T) {
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	groups := modelListGroups{userGroup: "default", ownerGroups: []string{"default"}}
	for _, tc := range []struct {
		name, resolution string
		plain, reference []canvasPriceAmount
		tokensPerSecond  int
	}{
		{"seedance-2-5-per-second", "720p", []canvasPriceAmount{{"second", 0.39}}, []canvasPriceAmount{{"second", 0.39}, {"reference_second", 0.31}}, 0},
		{"seedance-2-0-per-second", "720p", []canvasPriceAmount{{"second", 0.35}}, nil, 0},
		{"seedance-2-5-per-token", "1080p", []canvasPriceAmount{{"video_tokens", 48.7872}}, []canvasPriceAmount{{"video_tokens", 29.1456}}, 48_600},
		{"seedance-2-0-per-token", "720p", []canvasPriceAmount{{"video_tokens", 29.1456}}, []canvasPriceAmount{{"video_tokens", 17.7408}}, 21_600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaults, ok := plugin.Meta.DefaultsForModel(tc.name)
			require.True(t, ok)
			schema, _ := plugin.Meta.UsageForModel(tc.name)
			item := model.Pricing{EnableGroup: []string{"default"}, BillingExpr: defaults.BillingExpr, BillingUsageSchema: schema}
			price := canvasPriceSummary(item, groups, nil, schema["resolution"].Enum, true, defaults.VideoTokenEstimate)
			if tc.reference == nil {
				assert.Equal(t, tc.plain, price.EstimatesByResolution[tc.resolution])
				return
			}
			quote := price.VideoEstimatesByResolution[tc.resolution]
			for _, check := range []struct{ got, want []canvasPriceAmount }{{quote.WithoutReference, tc.plain}, {quote.WithReference, tc.reference}} {
				require.Len(t, check.got, len(check.want))
				for i, amount := range check.want {
					assert.Equal(t, amount.Unit, check.got[i].Unit)
					assert.InDelta(t, amount.Amount, check.got[i].Amount, 1e-9)
				}
			}
			if tc.tokensPerSecond > 0 {
				require.NotNil(t, price.VideoTokenEstimate)
				assert.Equal(t, tc.tokensPerSecond, price.VideoTokenEstimate.TokensPerSecond[tc.resolution])
				assert.Equal(t, 4, price.VideoTokenEstimate.ReferenceExtraSeconds)
				assert.Equal(t, 180, price.VideoTokenEstimate.ReferencePercent)
			}
		})
	}
}

func TestCatalogVideoSpecsOnlyQuoteEffectivePrices(t *testing.T) {
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	for _, tc := range []struct {
		model, resolution, unit string
		price                   float64
	}{
		{"seedance-2-5-per-second", "720p", "second", 0.39},
		{"seedance-2-5-per-second", "1080p", "second", 0.65},
		{"seedance-2-5-per-request", "720p", "request", 5.5},
		{"seedance-2-5-per-token", "1080p", "video_tokens", 48.7872},
		{"ap-minimax-h3", "720p", "request", 1},
		{"ms-minimax-h3-upscale", "2K", "second", 0.1},
	} {
		defaults, ok := plugin.Meta.DefaultsForModel(tc.model)
		require.True(t, ok)
		schema, _ := plugin.Meta.UsageForModel(tc.model)
		item := model.Pricing{VideoSpecs: defaults.VideoSpecs, BillingExpr: `tier("global", u("requests") * 99.0)`, BillingPluginVariants: []model.PricingPluginVariant{{BillingMode: "tiered_expr", BillingExpr: defaults.BillingExpr, BillingUsageSchema: schema}}}
		specs := catalogVideoSpecs(item)
		idx := slices.IndexFunc(specs, func(s catalogVideoSpec) bool { return s.Resolution == tc.resolution })
		require.NotEqual(t, -1, idx, "%s %s", tc.model, tc.resolution)
		spec := specs[idx]
		require.NotEmpty(t, spec.WithoutReference)
		assert.Equal(t, tc.unit, spec.WithoutReference[0].Unit)
		assert.InDelta(t, tc.price, spec.WithoutReference[0].Amount, 1e-9)
		if tc.model == "seedance-2-5-per-second" && tc.resolution == "720p" {
			assert.True(t, spec.Preferred)
			assert.Contains(t, spec.WithReference, canvasPriceAmount{Unit: "reference_second", Amount: 0.31})
		}
		item.BillingPluginVariants[0].BillingExpr = `tier("custom", u("seconds") * u("seconds"))`
		assert.Empty(t, catalogVideoSpecs(item), "never substitute the global price for an unquotable provider override")
	}
}

func TestCatalogImageSpecsQuoteImageAndRequestPrices(t *testing.T) {
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, res, size, action, unit string
		amount                        float64
		min, max, outputs             int
	}{
		{"st-gpt-image-2", "1K", "", "", "image", 0.032, 0, -1, 10},
		{"st-gpt-image-2", "4K", "", "", "image", 0.06, 0, -1, 10},
		{"ty-seedream-5-5", "2K", "", "", "request", 0.156, 0, 8, 0},
		{"sl-midjourney", "", "0.25", "imagine", "request", 0.54, 0, 10, 0},
		{"sl-midjourney", "", "0.25", "blend", "request", 0.576, 1, 10, 0},
	} {
		t.Run(tc.name+"/"+tc.res+tc.action, func(t *testing.T) {
			d, ok := plugin.Meta.DefaultsForModel(tc.name)
			require.True(t, ok)
			schema, _ := plugin.Meta.UsageForModel(tc.name)
			item := model.Pricing{ImageSpecs: d.ImageSpecs, BillingExpr: `u("requests")*99`, BillingPluginVariants: []model.PricingPluginVariant{{BillingExpr: d.BillingExpr, BillingUsageSchema: schema}}}
			rows := catalogImageSpecs(item)
			i := slices.IndexFunc(rows, func(s catalogImageSpec) bool {
				return s.Resolution == tc.res && s.ImageSize == tc.size && s.Action == tc.action
			})
			require.NotEqual(t, -1, i)
			row := rows[i]
			assert.Equal(t, canvasPriceAmount{Unit: tc.unit, Amount: tc.amount}, row.Rate)
			assert.Equal(t, tc.min, row.MinImages)
			assert.Equal(t, tc.outputs, row.MaxOutputs)
			if tc.max < 0 {
				assert.Nil(t, row.MaxImages)
			} else {
				require.NotNil(t, row.MaxImages)
				assert.Equal(t, tc.max, *row.MaxImages)
			}
			item.BillingPluginVariants[0].BillingExpr = `tier("override",u("requests")*1.25)`
			require.Equal(t, canvasPriceAmount{"request", 1.25}, catalogImageSpecs(item)[i].Rate)
			item.BillingPluginVariants[0].BillingExpr = `p * 2.0 + c * 5.0`
			assert.Empty(t, catalogImageSpecs(item), "token fees must not be presented as an image rate")
		})
	}
}

func TestCanvasModelFamilies(t *testing.T) {
	for _, tc := range []struct{ id, name, series, want string }{
		{"sd25f", "Seedance 2.5 Flash · 720p", "Seedance 2.5", "seedance-2.5-flash"},
		{"ap-seedance-2-0-ot-fast", "Seedance 2.0 Fast · Token", "Seedance 2.0", "seedance-2.0-fast"},
		{"seedance-2-0-mini-b1-480p", "", "", "seedance-2.0-mini"},
		{"aj-wan-3-0", "Wan 3.0", "", "wan3"},
		{"ms-minimax-h3", "MiniMax H3 · Generate", "", "MiniMax H3"},
		{"mf-gemini-omni", "Gemini Omni Flash", "", "Gemini Omni"},
		{"muse-zr-700", "Muse Video · Character", "", "Muse"},
		{"nw-image-2", "Image 2 · 画质可调", "", "gpt-image-2"},
		{"ac-seedream-5-0", "", "", "seedream"},
		{"tinysnow-banana-3", "Nano Banana Pro", "", "banana"},
		{"other-model", "", "自定义系列", "自定义系列"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			assert.Equal(t, tc.want, canvasModelFamily(tc.id, tc.name, tc.series))
		})
	}
}

func canvasRequest(t *testing.T, handler gin.HandlerFunc, bearer string, input any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := common.Marshal(input)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/canvas", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer "+bearer)
	handler(c)
	return response
}

func canvasUserSession(t *testing.T, db *gorm.DB) (*model.User, string) {
	t.Helper()
	user := &model.User{Username: "creator", Email: "creator@example.com", DisplayName: "Creator", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", AuthVersion: 1, AffCode: "creator", Quota: 1234567, UsedQuota: 12345}
	require.NoError(t, db.Create(user).Error)
	session := model.UserSession{SID: "canvas-browser-session", UserID: user.Id, Version: 1, UserAuthVersion: 1, Status: model.UserSessionStatusActive, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	require.NoError(t, db.Create(&session).Error)
	token, _, err := service.IssueAccessToken(service.AuthIdentity{UserID: user.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1})
	require.NoError(t, err)
	return user, token
}

func canvasAuthorizeCode(t *testing.T, bearer string) string {
	t.Helper()
	response := canvasRequest(t, CanvasAuthorize, bearer, map[string]any{
		"redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback", "state": "browser-bound-state",
		"code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", "code_challenge_method": "S256",
	})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var result struct {
		Data struct {
			RedirectURL string `json:"redirect_url"`
		}
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	redirect, err := url.Parse(result.Data.RedirectURL)
	require.NoError(t, err)
	assert.Equal(t, "https", redirect.Scheme)
	assert.Equal(t, "canvas.stolas.cc", redirect.Host)
	assert.Equal(t, "/api/auth/stolas/callback", redirect.Path)
	assert.Equal(t, "browser-bound-state", redirect.Query().Get("state"))
	assert.Len(t, redirect.Query(), 2, "redirect contains only code and browser state")
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.NotEmpty(t, redirect.Query().Get("code"))
	return redirect.Query().Get("code")
}

func canvasExchangeInput(code string) map[string]string {
	return map[string]string{"code": code, "code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", "redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback"}
}

func TestCanvasCreditMigrationDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			dsn := os.Getenv(dialect.env)
			if dialect.env != "" && dsn == "" {
				t.Skip(dialect.env + " is not configured")
			}
			db := canvasDatabase(t, dialect.kind, dsn)
			owner := model.User{Username: "migration-owner", AffCode: "origin", Status: common.UserStatusEnabled, Quota: 20}
			other := model.User{Username: "migration-other", AffCode: "other", Status: common.UserStatusEnabled}
			require.NoError(t, db.Create(&owner).Error)
			require.NoError(t, db.Create(&other).Error)
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.CanvasCreditMigration{}))
			}
			oldRate, oldUnit := operation_setting.USDExchangeRate, common.QuotaPerUnit
			t.Cleanup(func() { operation_setting.USDExchangeRate, common.QuotaPerUnit = oldRate, oldUnit })
			operation_setting.USDExchangeRate, common.QuotaPerUnit = 7.3, 500000
			router := gin.New()
			router.POST("/api/canvas/credit-migration", CanvasCreditMigration)
			apply := func(input model.CanvasCreditMigration, wantStatus int) {
				t.Helper()
				body, err := common.Marshal(input)
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "/api/canvas/credit-migration", bytes.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				secret := os.Getenv("STOLAS_CANVAS_CLIENT_SECRET")
				request.Header.Set("Authorization", "Bearer "+secret)
				request.Header.Set("X-Stolas-Client-Secret", secret)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				require.Equal(t, wantStatus, response.Code, response.Body.String())
			}
			request := model.CanvasCreditMigration{ID: strings.Repeat("a", 64), UserID: owner.Id, CNYMicros: 7_300_000}
			apply(request, http.StatusOK)
			operation_setting.USDExchangeRate = 9
			apply(request, http.StatusOK)
			request.UserID = other.Id
			apply(request, http.StatusConflict)
			require.NoError(t, db.First(&owner, owner.Id).Error)
			assert.Equal(t, 500020, owner.Quota)
			require.NoError(t, db.First(&other, other.Id).Error)
			assert.Zero(t, other.Quota)

			// A rejected credit cannot leave a receipt that would swallow delivery after the balance changes.
			request = model.CanvasCreditMigration{ID: strings.Repeat("b", 64), UserID: owner.Id, CNYMicros: 1}
			require.NoError(t, db.Model(&owner).UpdateColumn("quota", common.MaxWalletQuota).Error)
			apply(request, http.StatusConflict)
			require.NoError(t, db.First(&owner, owner.Id).Error)
			assert.Equal(t, common.MaxWalletQuota, owner.Quota)
			var receipt model.CanvasCreditMigration
			require.ErrorIs(t, db.First(&receipt, "id = ?", request.ID).Error, gorm.ErrRecordNotFound)
			require.NoError(t, db.Model(&owner).UpdateColumn("quota", common.MaxWalletQuota-1).Error)
			apply(request, http.StatusOK)
			apply(request, http.StatusOK)
			require.NoError(t, db.First(&owner, owner.Id).Error)
			assert.Equal(t, common.MaxWalletQuota, owner.Quota)
			require.NoError(t, db.First(&receipt, "id = ?", request.ID).Error)
			assert.Equal(t, 1, receipt.Quota)
		})
	}
}

func TestCanvasIdentityReadsCurrentRoleAndRequiresBridgeAndDefaultAssociation(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := canvasDatabase(t, dialect.kind, os.Getenv(dialect.env))
			user, _ := canvasUserSession(t, db)
			var token *model.Token
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				_, initialized, err := model.EnsureCanvasDefaultTokenWithTx(tx, user.Id)
				token = initialized
				return err
			}))
			engine := gin.New()
			verifiedTokenID := token.Id
			engine.GET("/api/canvas/identity", func(c *gin.Context) {
				c.Set("id", user.Id)
				c.Set("token_id", verifiedTokenID)
				CanvasIdentity(c)
			})
			read := func(secret, key string) *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodGet, "/api/canvas/identity", nil)
				request.Header.Set("Authorization", "Bearer "+key)
				request.Header.Set("X-Stolas-Client-Secret", secret)
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, request)
				return response
			}
			secret := os.Getenv("STOLAS_CANVAS_CLIENT_SECRET")
			assert.Equal(t, http.StatusUnauthorized, read("", "sk-"+token.Key).Code)
			verifiedTokenID = token.Id + 999
			assert.Equal(t, http.StatusUnauthorized, read(secret, "sk-"+token.Key).Code)
			verifiedTokenID = token.Id
			for _, role := range []int{common.RoleAdminUser, common.RoleRootUser, common.RoleCommonUser} {
				require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("role", role).Error)
				response := read(secret, "sk-"+token.Key)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				var result struct {
					Data service.CanvasUser `json:"data"`
				}
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
				var fields struct {
					Data map[string]any `json:"data"`
				}
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &fields))
				assert.Len(t, fields.Data, 3, "identity exposes only id, status and role")
				assert.Equal(t, role, result.Data.Role)
				assert.Equal(t, user.Id, result.Data.ID)
				assert.NotContains(t, response.Body.String(), token.Key)
				assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			}
			require.NoError(t, db.Model(&model.Token{}).Where("id = ?", token.Id).Update("expired_time", time.Now().Add(-time.Minute).Unix()).Error)
			assert.Equal(t, http.StatusUnauthorized, read(secret, "sk-"+token.Key).Code)
			require.NoError(t, db.Model(&model.Token{}).Where("id = ?", token.Id).Update("expired_time", -1).Error)
			require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("status", common.UserStatusDisabled).Error)
			assert.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, read(secret, "sk-"+token.Key).Code)
		})
	}
}

func TestCanvasAuthorizationSecurity(t *testing.T) {
	for _, tc := range []string{"success and replay", "wrong verifier", "wrong callback", "wrong client", "expired", "revoked session", "auth version changed", "disabled user", "revoked key", "deleted key", "cross user key", "PAT rejected", "disabled config"} {
		t.Run(tc, func(t *testing.T) {
			db := canvasDatabase(t, "sqlite", "")
			user, bearer := canvasUserSession(t, db)
			if tc == "PAT rejected" {
				response := canvasRequest(t, CanvasAuthorize, "management-pat", map[string]string{})
				assert.Equal(t, http.StatusUnauthorized, response.Code)
				return
			}
			if tc == "disabled config" {
				t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", "too-short")
				response := canvasRequest(t, CanvasAuthorize, bearer, map[string]string{})
				assert.Equal(t, http.StatusServiceUnavailable, response.Code)
				return
			}
			code := canvasAuthorizeCode(t, bearer)
			input, secret := canvasExchangeInput(code), os.Getenv("STOLAS_CANVAS_CLIENT_SECRET")
			switch tc {
			case "wrong verifier":
				input["code_verifier"] = strings.Repeat("x", 43)
			case "wrong callback":
				input["redirect_uri"] += "?evil=true"
			case "wrong client":
				secret = "wrong-client-secret"
			case "expired":
				require.NoError(t, db.Model(&model.AuthFlow{}).Where("user_id = ?", user.Id).Update("expires_at", time.Now().Add(-time.Minute)).Error)
			case "revoked session":
				require.NoError(t, db.Model(&model.UserSession{}).Where("user_id = ?", user.Id).Update("revoked_at", time.Now().Unix()).Error)
			case "auth version changed":
				require.NoError(t, db.Model(user).Update("auth_version", 2).Error)
			case "disabled user":
				require.NoError(t, db.Model(user).Update("status", common.UserStatusDisabled).Error)
			case "revoked key", "deleted key", "cross user key":
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error { _, _, err := model.EnsureCanvasDefaultTokenWithTx(tx, user.Id); return err }))
				var token model.Token
				require.NoError(t, db.Where("user_id = ?", user.Id).First(&token).Error)
				if tc == "deleted key" {
					require.NoError(t, db.Delete(&token).Error)
				} else if tc == "cross user key" {
					require.NoError(t, db.Model(&token).Update("user_id", user.Id+100).Error)
				} else {
					require.NoError(t, db.Model(&token).Update("status", common.TokenStatusDisabled).Error)
				}
			}
			response := canvasRequest(t, CanvasExchange, secret, input)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			if tc != "success and replay" {
				assert.NotEqual(t, http.StatusOK, response.Code, response.Body.String())
				assert.NotContains(t, response.Body.String(), `"key":`)
				if tc == "wrong verifier" || tc == "wrong callback" || tc == "wrong client" {
					retry := canvasRequest(t, CanvasExchange, os.Getenv("STOLAS_CANVAS_CLIENT_SECRET"), canvasExchangeInput(code))
					assert.Equal(t, http.StatusOK, retry.Code, retry.Body.String())
				}
				return
			}
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var result struct {
				Data struct {
					User struct {
						ID       int
						Username string
					}
					Token struct {
						ID  int
						Key string
					}
					Wallet struct {
						Quota        int
						UsedQuota    int     `json:"used_quota"`
						QuotaPerUnit float64 `json:"quota_per_unit"`
					}
				}
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, user.Id, result.Data.User.ID)
			assert.Equal(t, user.Username, result.Data.User.Username)
			assert.Equal(t, 1234567, result.Data.Wallet.Quota)
			assert.Equal(t, 12345, result.Data.Wallet.UsedQuota)
			assert.Positive(t, result.Data.Wallet.QuotaPerUnit)
			assert.True(t, strings.HasPrefix(result.Data.Token.Key, "sk-"))
			assert.NotContains(t, response.Body.String(), "password")
			assert.NotContains(t, response.Body.String(), "auth_version")
			replay := canvasRequest(t, CanvasExchange, secret, input)
			assert.NotEqual(t, http.StatusOK, replay.Code)
		})
	}
}

func TestCanvasExchangeDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := canvasDatabase(t, dialect.kind, os.Getenv(dialect.env))
			_, bearer := canvasUserSession(t, db)
			code := canvasAuthorizeCode(t, bearer)
			statuses := make(chan int, 2)
			var workers sync.WaitGroup
			for range 2 {
				workers.Go(func() {
					response := canvasRequest(t, CanvasExchange, os.Getenv("STOLAS_CANVAS_CLIENT_SECRET"), canvasExchangeInput(code))
					statuses <- response.Code
				})
			}
			workers.Wait()
			close(statuses)
			successes := 0
			for status := range statuses {
				if status == http.StatusOK {
					successes++
				}
			}
			assert.Equal(t, 1, successes, "authorization code must have exactly one successful consumer")
			var count int64
			require.NoError(t, db.Model(&model.Token{}).Count(&count).Error)
			assert.EqualValues(t, 1, count, "concurrent exchange cannot create duplicate defaults")
		})
	}
}

func TestCanvasRegistrationInitializesDefaultToken(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := canvasDatabase(t, dialect.kind, os.Getenv(dialect.env))
			user := model.User{Username: "new-creator", DisplayName: "Creator", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return user.InsertWithTx(tx, 0) }))
			var tokens []model.Token
			require.NoError(t, db.Where("user_id = ?", user.Id).Find(&tokens).Error)
			require.Len(t, tokens, 1, "registration must create its default model credential atomically")
			assert.True(t, tokens[0].UnlimitedQuota)
			assert.True(t, tokens[0].AllowUserQuotaQuery)
			assert.False(t, tokens[0].ModelLimitsEnabled)
			assert.Equal(t, int64(-1), tokens[0].ExpiredTime)
		})
	}
}

func TestCanvasDefaultTokenLifecycle(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })
	t.Run("password registration is atomic and does not duplicate legacy initialization", func(t *testing.T) {
		previousGenerate, previousEmail, previousRegister, previousPassword := constant.GenerateDefaultToken, common.EmailVerificationEnabled, common.RegisterEnabled, common.PasswordRegisterEnabled
		constant.GenerateDefaultToken, common.EmailVerificationEnabled, common.RegisterEnabled, common.PasswordRegisterEnabled = true, false, true, true
		t.Cleanup(func() {
			constant.GenerateDefaultToken, common.EmailVerificationEnabled, common.RegisterEnabled, common.PasswordRegisterEnabled = previousGenerate, previousEmail, previousRegister, previousPassword
		})
		response := canvasRequest(t, Register, "", map[string]string{"username": "password-creator", "password": "secure-test-password"})
		require.Equal(t, http.StatusOK, response.Code)
		var result struct{ Success bool }
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
		require.True(t, result.Success, response.Body.String())
		var user model.User
		require.NoError(t, db.Where("username = ?", "password-creator").First(&user).Error)
		require.NotNil(t, user.DefaultTokenID)
		var count int64
		require.NoError(t, db.Model(&model.Token{}).Where("user_id = ?", user.Id).Count(&count).Error)
		assert.EqualValues(t, 1, count)
	})
	t.Run("token initialization failure rolls registration back", func(t *testing.T) {
		require.NoError(t, db.Callback().Create().Before("gorm:create").Register("canvas_test_token_failure", func(tx *gorm.DB) {
			if tx.Statement.Table == "tokens" {
				tx.AddError(errors.New("test token persistence failure"))
			}
		}))
		t.Cleanup(func() { require.NoError(t, db.Callback().Create().Remove("canvas_test_token_failure")) })
		for _, transactional := range []bool{false, true} {
			user := model.User{Username: "rollback-creator", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
			var err error
			if transactional {
				err = db.Transaction(func(tx *gorm.DB) error { return user.InsertWithTx(tx, 0) })
			} else {
				err = user.Insert(0)
			}
			require.Error(t, err)
			var count int64
			require.NoError(t, db.Model(&model.User{}).Where("username = ?", "rollback-creator").Count(&count).Error)
			assert.Zero(t, count)
		}
	})
	t.Run("OAuth registration initializes the associated credential", func(t *testing.T) {
		previousRegister := common.RegisterEnabled
		common.RegisterEnabled = true
		t.Cleanup(func() { common.RegisterEnabled = previousRegister })
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/oauth/test", nil)
		user, _, err := findOrCreateOAuthUser(c, &authFlowTestOAuthProvider{}, &oauth.OAuthUser{ProviderUserID: "oauth-creator", Username: "oauth-creator", DisplayName: "OAuth Creator"}, &oauth.OAuthToken{}, "")
		require.NoError(t, err)
		require.NotNil(t, user.DefaultTokenID)
		var token model.Token
		require.NoError(t, db.Where("id = ? AND user_id = ?", *user.DefaultTokenID, user.Id).First(&token).Error)
		assert.True(t, token.UnlimitedQuota)
		assert.True(t, token.AllowUserQuotaQuery)
	})
	t.Run("incomplete config leaves registration unchanged", func(t *testing.T) {
		t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", "")
		user := model.User{Username: "ordinary-user", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return user.InsertWithTx(tx, 0) }))
		var count int64
		require.NoError(t, db.Model(&model.Token{}).Where("user_id = ?", user.Id).Count(&count).Error)
		assert.Zero(t, count)
		assert.Nil(t, user.DefaultTokenID)
	})
}

func TestCanvasAuthorizeRejectsInvalidClientParameters(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	_, bearer := canvasUserSession(t, db)
	for _, tc := range []struct{ name, field, value string }{
		{"plain PKCE", "code_challenge_method", "plain"},
		{"wrong challenge length", "code_challenge", "abcd"},
		{"missing state", "state", ""},
		{"cross origin redirect", "redirect_uri", "https://evil.example/api/auth/stolas/callback"},
		{"callback query", "redirect_uri", "https://canvas.stolas.cc/api/auth/stolas/callback?next=evil"},
		{"callback fragment", "redirect_uri", "https://canvas.stolas.cc/api/auth/stolas/callback#evil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]string{"redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback", "state": "browser-bound-state", "code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", "code_challenge_method": "S256"}
			input[tc.field] = tc.value
			response := canvasRequest(t, CanvasAuthorize, bearer, input)
			assert.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
	var count int64
	require.NoError(t, db.Model(&model.AuthFlow{}).Count(&count).Error)
	assert.Zero(t, count, "invalid requests must not create redeemable authorizations")
}

func TestCanvasAuthorizeOnlyAcceptsDashboardBearer(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	require.NoError(t, db.AutoMigrate(&model.UserAccessToken{}, &model.AuditLog{}))
	user, bearer := canvasUserSession(t, db)
	pat, _ := createScopedAccessToken(t, user.Id, 0, "api_key:reveal")
	router := gin.New()
	router.POST("/api/canvas/authorize", middleware.UserAuth(), CanvasAuthorize)
	input := `{"redirect_uri":"https://canvas.stolas.cc/api/auth/stolas/callback","state":"browser-bound-state","code_challenge":"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM","code_challenge_method":"S256"}`
	for _, tc := range []struct {
		name, authorization string
		want                int
	}{
		{"dashboard bearer", "Bearer " + bearer, http.StatusOK},
		{"valid management PAT", "Bearer " + pat, http.StatusForbidden},
		{"cookie only", "", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/canvas/authorize", strings.NewReader(input))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", tc.authorization)
			request.AddCookie(&http.Cookie{Name: service.RefreshCookieName, Value: "canvas-browser-session.cookie"})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assert.Equal(t, tc.want, response.Code, response.Body.String())
		})
	}
	var flow model.AuthFlow
	require.NoError(t, db.Where("purpose = ?", model.AuthFlowPurposeCanvas).First(&flow).Error)
	assert.WithinDuration(t, time.Now().Add(time.Minute), flow.ExpiresAt, 5*time.Second, "authorization must expire after one minute")
}

func TestCanvasDefaultCredentialNeverAppearsInSQLLogs(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	user := model.User{Username: "logging-creator", AffCode: "logging", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	var output bytes.Buffer
	verbose := db.Session(&gorm.Session{Logger: logger.New(log.New(&output, "", 0), logger.Config{LogLevel: logger.Info})})
	var token *model.Token
	require.NoError(t, verbose.Transaction(func(tx *gorm.DB) error {
		var err error
		_, token, err = model.EnsureCanvasDefaultTokenWithTx(tx, user.Id)
		return err
	}))
	require.NotNil(t, token)
	assert.NotContains(t, output.String(), token.Key, "credential creation must suppress plaintext SQL parameters even in debug mode")
}

func TestCanvasConfigurationPublicStatus(t *testing.T) {
	for _, tc := range []struct {
		name, origin, secret string
		enabled              bool
		wantOrigin           string
	}{
		{"configured", "https://canvas.stolas.cc/", strings.Repeat("s", 32), true, "https://canvas.stolas.cc"},
		{"missing secret", "https://canvas.stolas.cc", "", false, ""},
		{"short secret", "https://canvas.stolas.cc", "short", false, ""},
		{"http", "http://canvas.stolas.cc", strings.Repeat("s", 32), false, ""},
		{"userinfo", "https://evil@canvas.stolas.cc", strings.Repeat("s", 32), false, ""},
		{"path", "https://canvas.stolas.cc/path", strings.Repeat("s", 32), false, ""},
		{"query", "https://canvas.stolas.cc?x=y", strings.Repeat("s", 32), false, ""},
		{"fragment", "https://canvas.stolas.cc#", strings.Repeat("s", 32), false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STOLAS_CANVAS_URL", tc.origin)
			t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", tc.secret)
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
			GetStatus(c)
			var result struct {
				Data struct {
					Enabled bool   `json:"canvas_enabled"`
					URL     string `json:"canvas_url"`
				}
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, tc.enabled, result.Data.Enabled)
			assert.Equal(t, tc.wantOrigin, result.Data.URL)
			if tc.secret != "" {
				assert.NotContains(t, response.Body.String(), tc.secret)
			}
		})
	}
}

func TestCanvasDefaultTokenMigrationDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db, _ := newAuditTestDatabase(t, dialect.kind, os.Getenv(dialect.env))
			connection, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = connection.Close() })
			require.NoError(t, db.AutoMigrate(&releasedAuditUser{}, &model.Token{}))
			// rc.50 adds this persisted field to the rc.33 user fixture above;
			// its remaining differences are validation or ignored Go-only fields.
			require.NoError(t, db.Migrator().AddColumn(&model.User{}, "AccessTokenCreatedAt"))
			legacy := releasedAuditUser{Username: "legacy-creator", DisplayName: "Legacy", AffCode: "legacy", Status: common.UserStatusEnabled, AuthVersion: 1, Group: "default"}
			require.NoError(t, db.Create(&legacy).Error)
			existing := model.Token{UserId: legacy.Id, Key: "legacy-model-key", Name: "Stolas default", UnlimitedQuota: false, RemainQuota: 321, ModelLimitsEnabled: true, ModelLimits: "one-model", ExpiredTime: -1}
			require.NoError(t, db.Create(&existing).Error)
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))
			}
			var upgraded model.User
			require.NoError(t, db.First(&upgraded, legacy.Id).Error)
			assert.Nil(t, upgraded.DefaultTokenID, "old tokens are never guessed from names or ordering")
			previousDB, previousType := model.DB, common.MainDatabaseType()
			model.DB = db
			if dialect.kind == "mysql" {
				common.SetMainDatabaseType(common.DatabaseTypeMySQL)
			} else if dialect.kind == "postgres" {
				common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
			} else {
				common.SetMainDatabaseType(common.DatabaseTypeSQLite)
			}
			t.Cleanup(func() { model.DB = previousDB; common.SetMainDatabaseType(previousType) })
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { _, _, err := model.EnsureCanvasDefaultTokenWithTx(tx, legacy.Id); return err }))
			require.NoError(t, db.First(&upgraded, legacy.Id).Error)
			require.NotNil(t, upgraded.DefaultTokenID)
			assert.NotEqual(t, existing.Id, *upgraded.DefaultTokenID)
			var preserved model.Token
			require.NoError(t, db.First(&preserved, existing.Id).Error)
			assert.Equal(t, "legacy-model-key", preserved.Key)
			assert.Equal(t, 321, preserved.RemainQuota)
			assert.True(t, preserved.ModelLimitsEnabled)
			assert.False(t, preserved.AllowUserQuotaQuery)
			assert.False(t, preserved.UnlimitedQuota)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { _, _, err := model.EnsureCanvasDefaultTokenWithTx(tx, legacy.Id); return err }))
			var count int64
			require.NoError(t, db.Model(&model.Token{}).Count(&count).Error)
			assert.EqualValues(t, 2, count)
			duplicateToken := model.Token{UserId: legacy.Id, Key: existing.Key}
			assert.Error(t, db.Session(&gorm.Session{Logger: logger.Discard}).Create(&duplicateToken).Error, "migration must preserve model-key uniqueness")
			duplicateAssociation := model.User{Username: "duplicate-assoc", AffCode: "duplicate-assoc", DefaultTokenID: upgraded.DefaultTokenID}
			assert.Error(t, db.Session(&gorm.Session{Logger: logger.Discard}).Create(&duplicateAssociation).Error, "one default credential cannot be associated with two users")
		})
	}
}
