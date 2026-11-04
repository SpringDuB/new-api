package plugins_test

import (
	"bytes"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relay/channel"
	taskplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func loadXOGPUPlugin(t *testing.T) *jsplugin.LoadedPlugin {
	t.Helper()
	source, err := builtinplugins.Source("xogpu")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "xogpu"})
	require.NoError(t, err)
	return plugin
}

func xogpuFile(field, filename string, size int) map[string]any {
	return map[string]any{"ref": "request_file:" + field, "field": field, "filename": filename, "mimeType": "application/octet-stream", "size": size}
}

func xogpuContext(modelName string, request map[string]any, files ...map[string]any) map[string]any {
	return map[string]any{
		"model": modelName, "upstreamModel": modelName, "requestBody": request, "files": files,
		"baseUrl": "https://xogpu.example", "apiKey": "test-key",
	}
}

func callXOGPUHook(t *testing.T, plugin *jsplugin.LoadedPlugin, hook string, args ...any) map[string]any {
	t.Helper()
	value, err := plugin.Engine.Call(t.Context(), hook, args...)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, common.Unmarshal(encoded, &result))
	return result
}

// Wrong mode defaults or a missing model would change either the request sent
// to XOGPU or the seconds charged before that request.
func TestXOGPUVideoModesAndSubmittedUsage(t *testing.T) {
	plugin := loadXOGPUPlugin(t)
	assert.True(t, plugin.Meta.SupportsUpstream(jsplugin.UpstreamKindNewAPI))
	assert.Empty(t, plugin.Meta.ChannelTypes)
	assert.Equal(t, map[string]jsplugin.UsageFieldSchema{
		"seconds": {Type: "number", Unit: "second", Description: jsplugin.LocalizedText{"en": "Video generation unit price", "zh": "视频生成单价"}},
	}, plugin.Meta.UsageSchema)
	var exercised []string
	for _, mode := range []struct {
		prefix, durationField string
		qualities             []string
		files                 []map[string]any
	}{
		{"xogpu-video-text", "duration", []string{"480p", "768p"}, nil},
		{"xogpu-video-frames", "duration", []string{"480p", "768p"}, []map[string]any{xogpuFile("first_frame", "first.png", 3), xogpuFile("last_frame", "last.webp", 4)}},
		{"xogpu-video-multiref-quality", "duration", []string{"480p", "768p"}, []map[string]any{xogpuFile("reference_images", "ref.jpg", 3)}},
		{"xogpu-video-multiref-fast", "duration", []string{"480p", "768p"}, []map[string]any{xogpuFile("reference_images", "ref.jpg", 3)}},
		{"xogpu-video-multiref-hd", "duration", []string{"480p", "768p", "1080p"}, nil},
		{"xogpu-video-lipsync", "audio_duration", []string{"480p", "768p", "1080p"}, []map[string]any{xogpuFile("reference_images", "face.png", 3), xogpuFile("reference_audios", "speech.mp3", 4)}},
	} {
		for _, quality := range mode.qualities {
			name := mode.prefix + "-" + quality
			exercised = append(exercised, name)
			t.Run(name, func(t *testing.T) {
				req := map[string]any{}
				if mode.durationField == "duration" {
					req["prompt"] = "  海边🌊  "
				}
				ctx := xogpuContext(name, req, mode.files...)
				submit := callXOGPUHook(t, plugin, "buildSubmitRequest", ctx)
				assert.Equal(t, "https://xogpu.example/v1/videos", submit["url"])
				assert.Equal(t, "multipart", submit["bodyType"])
				assert.Equal(t, map[string]any{"Authorization": "Bearer test-key"}, submit["headers"])
				fields := map[string]any{}
				var uploaded []string
				for _, raw := range submit["parts"].([]any) {
					part := raw.(map[string]any)
					if ref, ok := part["fileRef"].(string); ok {
						uploaded = append(uploaded, ref)
						continue
					}
					fields[part["name"].(string)] = part["value"]
				}
				want := map[string]any{"model": name, mode.durationField: "5", "resolution": quality + "竖"}
				if mode.durationField == "duration" {
					want["prompt"] = "海边🌊"
				}
				assert.Equal(t, want, fields)
				assert.Len(t, uploaded, len(mode.files))
				assert.Equal(t, map[string]any{"seconds": float64(5)}, callXOGPUHook(t, plugin, "extractUsage", ctx))
				longer := maps.Clone(req)
				longer[mode.durationField] = "7"
				assert.Equal(t, map[string]any{"seconds": float64(7)}, callXOGPUHook(t, plugin, "extractUsage", xogpuContext(name, longer, mode.files...)))
			})
		}
	}
	assert.ElementsMatch(t, exercised, plugin.Meta.Models)
}

// Every rejection must also happen at metering, so an alternate host entry
// point cannot price an unbounded or unsupported submission.
func TestXOGPURejectsInvalidSubmissionsBeforeBilling(t *testing.T) {
	plugin := loadXOGPUPlugin(t)
	for _, tc := range []struct {
		name, model string
		request     map[string]any
		files       []map[string]any
	}{
		{name: "unknown model", model: "xogpu-video-text"},
		{name: "missing prompt"},
		{name: "non-string prompt", request: map[string]any{"prompt": 1}},
		{name: "long prompt", request: map[string]any{"prompt": strings.Repeat("🌊", 10001)}},
		{name: "zero duration", request: map[string]any{"prompt": "p", "duration": 0}},
		{name: "negative duration", request: map[string]any{"prompt": "p", "duration": -1}},
		{name: "fractional duration", request: map[string]any{"prompt": "p", "duration": "1.5"}},
		{name: "exponent duration", request: map[string]any{"prompt": "p", "duration": "1e1"}},
		{name: "boolean duration", request: map[string]any{"prompt": "p", "duration": true}},
		{name: "oversized duration", request: map[string]any{"prompt": "p", "duration": "9999999999999999"}},
		{name: "hd duration over ten", model: "xogpu-video-multiref-hd-1080p", request: map[string]any{"prompt": "p", "duration": 11}},
		{name: "opposite duration field", request: map[string]any{"prompt": "p", "audio_duration": 5}},
		{name: "conflicting durations", request: map[string]any{"prompt": "p", "duration": 5, "audio_duration": 5}},
		{name: "resolution quality mismatch", request: map[string]any{"prompt": "p", "resolution": "768p竖"}},
		{name: "unsupported square hd", model: "xogpu-video-multiref-hd-480p", request: map[string]any{"prompt": "p", "resolution": "480p(1:1)"}},
		{name: "text seed", request: map[string]any{"prompt": "p", "seed": "0"}},
		{name: "frames missing last", model: "xogpu-video-frames-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("first_frame", "f.png", 1)}},
		{name: "duplicate first frame", model: "xogpu-video-frames-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("first_frame", "f.png", 1), xogpuFile("first_frame", "f.png", 1), xogpuFile("last_frame", "l.png", 1)}},
		{name: "quality missing image", model: "xogpu-video-multiref-quality-480p", request: map[string]any{"prompt": "p"}},
		{name: "fast missing image", model: "xogpu-video-multiref-fast-480p", request: map[string]any{"prompt": "p"}},
		{name: "lipsync missing audio", model: "xogpu-video-lipsync-480p", files: []map[string]any{xogpuFile("reference_images", "f.png", 1)}},
		{name: "lipsync forbids prompt", model: "xogpu-video-lipsync-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_images", "f.png", 1), xogpuFile("reference_audios", "a.wav", 1)}},
		{name: "text forbids media", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_images", "f.png", 1)}},
		{name: "empty upload", model: "xogpu-video-multiref-hd-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_images", "f.png", 0)}},
		{name: "oversized image", model: "xogpu-video-multiref-hd-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_images", "f.png", (20<<20)+1)}},
		{name: "oversized audio", model: "xogpu-video-multiref-hd-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_audios", "a.mp3", (50<<20)+1)}},
		{name: "unsupported extension", model: "xogpu-video-multiref-hd-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_images", "f.gif", 1)}},
		{name: "wrong file field", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_images[]", "f.png", 1)}},
		{name: "combined media over limit", model: "xogpu-video-multiref-hd-480p", request: map[string]any{"prompt": "p"}, files: []map[string]any{xogpuFile("reference_audios", "a.mp3", 50<<20), xogpuFile("reference_audios", "b.mp3", 46<<20)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := tc.model
			if name == "" {
				name = "xogpu-video-text-480p"
			}
			for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
				_, err := plugin.Engine.Call(t.Context(), hook, xogpuContext(name, tc.request, tc.files...))
				require.Error(t, err, hook)
			}
		})
	}
	for _, field := range []string{"seconds", "size", "width", "height", "ratio", "aspect_ratio", "n", "fps", "reference_video", "image_urls", "audio_urls", "content", "metadata", "reference_images"} {
		t.Run("unsupported field "+field, func(t *testing.T) {
			_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", xogpuContext("xogpu-video-text-480p", map[string]any{"prompt": "p", field: "1"}))
			require.Error(t, err)
		})
	}
}

func TestXOGPUMultipartDecodeAndSeedPrecision(t *testing.T) {
	plugin := loadXOGPUPlugin(t)
	value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
		"model": "Minimax-H3-frame-480p",
		"body":  map[string]any{"kind": "multipart", "fields": map[string][]string{"model": {"Minimax-H3-frame-480p"}, "prompt": {"p"}, "seed": {"9999999999999999"}}},
	})
	require.NoError(t, err)
	intent := value.(map[string]any)
	assert.Equal(t, "Minimax-H3-frame-480p", intent["model"])
	ctx := xogpuContext("Minimax-H3-frame-480p", intent["requestBody"].(map[string]any), xogpuFile("first_frame", "f.png", 1), xogpuFile("last_frame", "l.png", 1))
	ctx["upstreamModel"] = "xogpu-video-frames-480p"
	for _, tc := range []struct {
		seed any
		want string
	}{{"9999999999999999", "9999999999999999"}, {"0", "0"}, {0, "0"}} {
		seedCtx := maps.Clone(ctx)
		req := maps.Clone(intent["requestBody"].(map[string]any))
		req["seed"] = tc.seed
		seedCtx["requestBody"] = req
		submit := callXOGPUHook(t, plugin, "buildSubmitRequest", seedCtx)
		parts, err := common.Marshal(submit["parts"])
		require.NoError(t, err)
		assert.Contains(t, string(parts), `"name":"model","value":"xogpu-video-frames-480p"`)
		assert.Contains(t, string(parts), `"name":"seed","value":"`+tc.want+`"`)
	}
	for _, seed := range []any{"10000000000000000", "-1", "1.5", true, float64(10000000000000000)} {
		seedCtx := maps.Clone(ctx)
		req := maps.Clone(intent["requestBody"].(map[string]any))
		req["seed"] = seed
		seedCtx["requestBody"] = req
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", seedCtx)
		require.Error(t, err)
	}
	for _, body := range []map[string]any{
		{"kind": "json", "value": map[string]any{"prompt": "p"}},
		{"kind": "multipart", "fields": map[string][]string{"duration": {"5", "10"}}},
	} {
		_, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{"model": "xogpu-video-text-480p", "body": body})
		require.Error(t, err)
	}
}

func TestXOGPUReferenceLimits(t *testing.T) {
	plugin := loadXOGPUPlugin(t)
	wave := xogpuFile("reference_audios", "speech.wav", 1)
	wave["mimeType"] = "audio/vnd.wave"
	callXOGPUHook(t, plugin, "buildSubmitRequest", xogpuContext("xogpu-video-lipsync-480p", map[string]any{}, xogpuFile("reference_images", "face.jpg", 1), wave))
	files := []map[string]any{
		xogpuFile("reference_audios", "a.mp4", 50<<20),
		xogpuFile("reference_images", "a.jpeg", 20<<20),
		xogpuFile("reference_images", "b.PNG", 20<<20),
		xogpuFile("reference_audios", "b.flac", 5<<20),
	}
	ctx := xogpuContext("xogpu-video-multiref-hd-1080p", map[string]any{"prompt": "p", "duration": 10, "resolution": "1080p横"}, files...)
	assert.Equal(t, map[string]any{"seconds": float64(10)}, callXOGPUHook(t, plugin, "extractUsage", ctx))
	for _, tc := range []struct {
		field, filename string
		limit           int
	}{{"reference_images", "ref.webp", 9}, {"reference_audios", "ref.wav", 3}} {
		files := make([]map[string]any, tc.limit)
		for i := range files {
			files[i] = xogpuFile(tc.field, tc.filename, 1)
		}
		ctx := xogpuContext("xogpu-video-multiref-hd-480p", map[string]any{"prompt": "p"}, files...)
		callXOGPUHook(t, plugin, "buildSubmitRequest", ctx)
		invalid := maps.Clone(ctx)
		invalid["files"] = append(files, xogpuFile(tc.field, tc.filename, 1))
		_, err := plugin.Engine.Call(t.Context(), "extractUsage", invalid)
		require.Error(t, err)
	}
	for _, duration := range []int{1, 15} {
		ctx := xogpuContext("xogpu-video-text-768p", map[string]any{"prompt": strings.Repeat("🌊", 10000), "duration": duration, "resolution": "768p(1:1)"})
		assert.Equal(t, map[string]any{"seconds": float64(duration)}, callXOGPUHook(t, plugin, "extractUsage", ctx))
	}
}

func TestXOGPUAliasExposesVideoUnitPricing(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, database.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}))
	previousDB, previousCache := model.DB, common.MemoryCacheEnabled
	model.DB, common.MemoryCacheEnabled = database, false
	t.Cleanup(func() {
		model.DB = previousDB
		model.InitChannelCache()
		common.MemoryCacheEnabled = previousCache
	})
	const alias = "Minimax-H3-frame-480p"
	mapping := `{"Minimax-H3-frame-480p":"xogpu-video-frames-480p"}`
	setting := `{"task_plugin_key":"xogpu"}`
	ch := &model.Channel{Type: constant.ChannelTypeNewAPI, Name: "xogpu", Key: "key", Status: common.ChannelStatusEnabled, Models: alias, Group: "default", ModelMapping: &mapping, Setting: &setting}
	require.NoError(t, database.Create(ch).Error)
	require.NoError(t, database.Create(&model.Ability{Group: "default", Model: alias, ChannelId: ch.Id, Enabled: true}).Error)
	model.InitChannelCache()
	served, declared, ok := model.ResolveTaskPluginModel(jsplugin.DefaultRegistry.Generation(), alias, loadXOGPUPlugin(t).Meta.Models)
	require.True(t, ok)
	assert.Equal(t, alias, served)
	assert.Equal(t, "xogpu-video-frames-480p", declared)
	prices := model.GetPricing()
	require.Len(t, prices, 1)
	assert.Equal(t, alias, prices[0].ModelName)
	assert.Equal(t, "second", prices[0].BillingUsageSchema["seconds"].Unit)
	assert.Equal(t, "视频生成单价", prices[0].BillingUsageSchema["seconds"].Description["zh"])
}

type xogpuReservation struct{ held int }

func (b *xogpuReservation) Reserve(quota int) error             { b.held = quota; return nil }
func (b *xogpuReservation) ReserveBeforeSubmit(quota int) error { return b.Reserve(quota) }
func (b *xogpuReservation) GetPreConsumedQuota() int            { return b.held }
func (*xogpuReservation) Settle(int) error                      { return nil }
func (*xogpuReservation) Refund(*gin.Context)                   {}
func (*xogpuReservation) NeedsRefund() bool                     { return false }

// The real host meters a mapped alias, sends multipart bytes and suppresses
// resubmission after ambiguous transport, HTTP, and accepted-response failures.
func TestXOGPUHostSubmissionAndRetrySafety(t *testing.T) {
	gin.SetMode(gin.TestMode)
	plugin := loadXOGPUPlugin(t)
	service.InitHttpClient()
	require.NoError(t, i18n.Init())
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error { saved[key] = value; return nil }))
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(saved)) })
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"Minimax-H3-frame-480p":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"Minimax-H3-frame-480p":"tier(\"video\", 0.01 * u(\"seconds\"))"}`,
	}))
	for _, tc := range []struct {
		name, reply string
		status      int
	}{
		{"accepted", `{"id":"private-task","object":"video","status":"queued","duration":12}`, 202},
		{"server failure", `{"error":{"message":"unavailable"}}`, 503},
		{"rate limit", `{"error":{"message":"busy"}}`, 429},
		{"missing task id", `{"object":"video","status":"queued"}`, 202},
		{"malformed accepted response", `{"id":`, 202},
		{"transport failure", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assert.Equal(t, "/v1/videos", r.URL.Path)
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				if !assert.NoError(t, r.ParseMultipartForm(1<<20)) {
					return
				}
				defer r.MultipartForm.RemoveAll()
				assert.Equal(t, map[string][]string{"model": {"xogpu-video-frames-480p"}, "prompt": {"p"}, "duration": {"7"}, "resolution": {"480p竖"}}, r.MultipartForm.Value)
				for field, expected := range map[string]string{"first_frame": "first bytes", "last_frame": "last bytes"} {
					file, _, err := r.FormFile(field)
					if !assert.NoError(t, err) {
						return
					}
					data, err := io.ReadAll(file)
					_ = file.Close()
					assert.NoError(t, err)
					assert.Equal(t, expected, string(data))
				}
				if tc.status == 0 {
					panic(http.ErrAbortHandler)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.reply)
			}))
			defer server.Close()
			var input bytes.Buffer
			writer := multipart.NewWriter(&input)
			for key, value := range map[string]string{"model": "Minimax-H3-frame-480p", "prompt": "p", "duration": "7"} {
				require.NoError(t, writer.WriteField(key, value))
			}
			for field, value := range map[string]string{"first_frame": "first bytes", "last_frame": "last bytes"} {
				part, err := writer.CreateFormFile(field, field+".png")
				require.NoError(t, err)
				_, err = io.WriteString(part, value)
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &input)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())
			c.Set("model_mapping", `{"Minimax-H3-frame-480p":"xogpu-video-frames-480p"}`)
			c.Set("task_request", map[string]any{"model": "Minimax-H3-frame-480p", "prompt": "p", "duration": "7"})
			c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{Plugin: plugin})
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeNewAPI)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
			common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
			reservation := &xogpuReservation{}
			info := &relaycommon.RelayInfo{OriginModelName: "Minimax-H3-frame-480p", UserGroup: "default", UsingGroup: "default", Billing: reservation, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			result, taskErr := relay.RelayTaskSubmit(c, info)
			assert.Equal(t, int32(1), calls.Load())
			assert.Equal(t, "Minimax-H3-frame-480p", info.OriginModelName)
			assert.Equal(t, map[string]any{"seconds": float64(7)}, info.TieredBillingSnapshot.UsageFacts)
			assert.Equal(t, common.QuotaRound(0.07*common.QuotaPerUnit), reservation.held)
			if tc.name != "accepted" {
				require.NotNil(t, taskErr)
				assert.True(t, taskErr.NoRetry, "%+v", taskErr)
				assert.Nil(t, result)
				return
			}
			require.Nil(t, taskErr, "%+v", taskErr)
			require.NotNil(t, result)
			assert.Equal(t, "private-task", result.UpstreamTaskID)
			assert.Equal(t, reservation.held, result.Quota, "the upstream output duration cannot change the submitted charge")
		})
	}
}

func TestXOGPUTaskLifecycleAndAuthenticatedContent(t *testing.T) {
	plugin := loadXOGPUPlugin(t)
	adaptor := taskplugin.New(plugin)
	adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "test-key", ChannelBaseUrl: "https://xogpu.example"}})
	for upstream, status := range map[string]string{"queued": "QUEUED", "in_progress": "IN_PROGRESS", "completed": "SUCCESS", "failed": "FAILURE", "unknown": "UNKNOWN", "new_state": "UNKNOWN"} {
		t.Run(upstream, func(t *testing.T) {
			body := []byte(`{"id":"private","status":"` + upstream + `","progress":42,"duration":12,"error":{"message":"blocked"},"metadata":{"url":"https://private.invalid"}}`)
			task := &model.Task{TaskID: "task_public", Status: model.TaskStatus(status), Properties: model.Properties{OriginModelName: "local-alias"}}
			task.PrivateData.UpstreamTaskID = "private/with space"
			task.Data = body
			result, err := adaptor.ParseTaskResult(task, &http.Response{StatusCode: 200, Header: make(http.Header)}, body)
			require.NoError(t, err)
			assert.Equal(t, status, result.Status)
			assert.Empty(t, result.UsageFacts, "output duration must not replace reserved seconds")
			if upstream == "failed" {
				assert.Equal(t, "blocked", result.Reason)
			}
			artifacts, err := adaptor.ListArtifacts(task)
			require.NoError(t, err)
			if status != "SUCCESS" {
				assert.Empty(t, artifacts)
				return
			}
			assert.Equal(t, []channel.TaskArtifact{{Key: "video", Type: "video", MimeType: "video/mp4"}}, artifacts)
			descriptor, err := adaptor.BuildContentRequest(t.Context(), task, "video", channel.TaskArtifactClientRequest{Method: http.MethodGet, Headers: map[string]string{"Range": "bytes=0-9"}})
			require.NoError(t, err)
			assert.Equal(t, "https://xogpu.example/v1/videos/private%2Fwith%20space/content", descriptor.URL)
			assert.Equal(t, "Bearer test-key", descriptor.Headers["Authorization"])
			assert.False(t, descriptor.Credentialless)
			head, err := adaptor.BuildContentRequest(t.Context(), task, "video", channel.TaskArtifactClientRequest{Method: http.MethodHead})
			require.NoError(t, err)
			assert.Equal(t, http.MethodGet, head.Method, "XOGPU documents GET content retrieval; the host handles the client's HEAD response")
			encoded, err := adaptor.ConvertToOpenAIVideo(task)
			require.NoError(t, err)
			var video map[string]any
			require.NoError(t, common.Unmarshal(encoded, &video))
			assert.Equal(t, "task_public", video["id"])
			assert.Equal(t, "local-alias", video["model"])
		})
	}
	ctx := map[string]any{"baseUrl": "https://xogpu.example", "apiKey": "test-key", "taskId": "private/with space"}
	assert.Equal(t, map[string]any{"url": "https://xogpu.example/v1/videos/private%2Fwith%20space", "method": "GET", "headers": map[string]any{"Authorization": "Bearer test-key"}}, callXOGPUHook(t, plugin, "buildQueryRequest", ctx))
	for _, body := range []any{`<html>error</html>`, map[string]any{"id": ""}, map[string]any{"id": 42}, map[string]any{"id": "private", "error": map[string]any{"message": "blocked"}}} {
		_, err := plugin.Engine.Call(t.Context(), "parseSubmitResponse", ctx, map[string]any{"statusCode": 200, "body": body})
		require.Error(t, err)
	}
	service.InitHttpClient()
	var queries atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/videos/private", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		if queries.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		_, _ = io.WriteString(w, `{"id":"private","status":"completed","duration":12}`)
	}))
	defer server.Close()
	task := &model.Task{TaskID: "task_public"}
	task.PrivateData.UpstreamTaskID = "private"
	first, err := adaptor.FetchTask(server.URL, "test-key", task, "")
	require.NoError(t, err)
	assert.Equal(t, 503, first.StatusCode)
	require.NoError(t, first.Body.Close())
	second, err := adaptor.FetchTask(server.URL, "test-key", task, "")
	require.NoError(t, err)
	defer second.Body.Close()
	assert.Equal(t, 200, second.StatusCode)
	assert.Equal(t, int32(2), queries.Load(), "query failures retry the same task without submitting again")
}
