package plugins_test

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	"github.com/QuantumNous/new-api/relay"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	taskplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadLecAPIPlugin(t *testing.T) *jsplugin.LoadedPlugin {
	t.Helper()
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	return plugin
}

func lecAPIHook(t *testing.T, plugin *jsplugin.LoadedPlugin, hook string, args ...any) map[string]any {
	t.Helper()
	value, err := plugin.Engine.Call(t.Context(), hook, args...)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, common.Unmarshal(encoded, &result))
	return result
}

func lecAPIContext(model string, body map[string]any) map[string]any {
	return map[string]any{"model": model, "upstreamModel": model, "requestBody": body, "baseUrl": "https://lec.example/", "apiKey": "test-key"}
}

func TestLecAPICatalogDelistsUnavailableModelsAndPreservesHistoricalQueries(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, model := range []string{
		"shuikule-seedance-2-5-m", "mf-video-upscale", "vg-seedance-2-5-pa", "ap-seedance-2-5-ot",
		"haya-seedance-2-5-m", "mg-seedance-2-5-6",
		"ap-muse-video-720-10s", "muse-zr-700",
	} {
		assert.Contains(t, plugin.Meta.Models, model)
	}
	const pooled = "seedance-2-5-01"
	request := lecAPIHook(t, plugin, "buildSubmitRequest", lecAPIContext(pooled, map[string]any{"prompt": "boat", "duration": 5}))
	var routes []string
	for _, alternative := range request["lecapiRouteAlternatives"].([]any) {
		routes = append(routes, alternative.(map[string]any)["model"].(string))
	}
	assert.NotContains(t, routes, "lec-mj-seedance-2-5-fd-720p")
	assert.NotContains(t, routes, "lec-zy-seedance-2-5-discount")
	assert.NotContains(t, routes, "lec-shuikule-seedance-2-5-bd", "the same 720p price does not imply the same request contract")
	defaults, found := plugin.Meta.DefaultsForModel(pooled)
	require.True(t, found)
	for _, text := range []string{"支持 4–30 秒", "可选 720p", "images：参考图片", "最多 30 张", "videos：参考视频", "最多 10 段", "audios：参考音频"} {
		assert.Contains(t, defaults.Description["zh"], text)
	}
	_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", lecAPIContext(pooled, map[string]any{
		"prompt": "boat", "resolution": "480p", "images": slices.Repeat([]any{"https://media.example/source.png"}, 21),
	}))
	require.Error(t, err)
	for _, old := range []string{
		"mm-seedance-2-5", "seedance-2-5-request-480",
		"haya-seedance-2-5-vc-480p", "mf-seedance-2-5-special", "mj-seedance-2-5-fd-720p",
		"mw-seedance-2-0-full", "omni-video-redraw", "shuikule-seedance-2-5-u",
		"vg-seedance-2-5-op", "vp-seedance-2-5-fp", "vp-seedance-2-5-vip-37",
	} {
		assert.NotContains(t, plugin.Meta.Models, old)
		ctx := lecAPIContext(old, map[string]any{"prompt": "boat"})
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
		require.Error(t, err)
		ctx["taskId"] = "historical/id"
		assert.Equal(t, "https://lec.example/v1/videos/historical%2Fid", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
	}
}

func TestLecAPICurrentVideoModelsExposePricingAndRequestContracts(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model, upstream string
		body            map[string]any
		price           float64
	}{
		{"am-minimax-h3-official", "lec-am-minimax-h3-official", map[string]any{"prompt": "boat"}, 1.8},
		{"am-minimax-h3-official", "lec-am-minimax-h3-official", map[string]any{"prompt": "boat", "duration": 15, "resolution": "1440p", "first_frame": "https://media.example/first.png", "last_frame": "https://media.example/last.png"}, 1.8},
		{"ap-minimax-h3", "lec-ap-minimax-h3", map[string]any{"prompt": "boat"}, 1},
		{"ap-seedance-2-0-image-audio", "lec-ap-seedance-2-0-image-audio", map[string]any{"prompt": "boat", "duration": 5}, 1.6},
		{"mf-seedance-2-0-fast-gz", "lec-mf-seedance-2-0-fast-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "720p"}, 1.45},
		{"mf-seedance-2-0-fast-gz", "lec-mf-seedance-2-0-fast-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "720p", "videos": []any{"https://media.example/ref.mp4"}}, 2.9},
		{"mf-seedance-2-0-gz", "lec-mf-seedance-2-0-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "1080p"}, 2.85},
		{"mf-seedance-2-0-gz", "lec-mf-seedance-2-0-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "1080p", "videos": []any{"https://media.example/ref.mp4"}}, 5.65},
		{"mf-seedance-2-5-flash-gz", "lec-mf-seedance-2-5-flash-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "720p"}, 1.8},
		{"mf-seedance-2-5-flash-gz", "lec-mf-seedance-2-5-flash-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "720p", "videos": []any{"https://media.example/ref.mp4"}}, 3.6},
		{"mf-seedance-2-5-gz", "lec-mf-seedance-2-5-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "1080p"}, 3.6},
		{"mf-seedance-2-5-gz", "lec-mf-seedance-2-5-gz", map[string]any{"prompt": "boat", "duration": 5, "resolution": "1080p", "videos": []any{"https://media.example/ref.mp4"}}, 7.2},
		{"ml-wan-3", "lec-ml-wan-3", map[string]any{"prompt": "boat", "duration": 5, "resolution": "720p"}, 1.7},
		{"vg-seedance-2-0-fast-cc", "lec-vg-seedance-2-0-fast-cc", map[string]any{"prompt": "boat", "audios": []any{"https://media.example/audio.mp3"}}, 1.6},
		{"md-seedance-2-5-900-720p", "lec-md-seedance-2-5-900-720p", map[string]any{"prompt": "boat", "process_face": false}, 5.5},
		{"md-seedance-2-0-900-720p", "lec-md-seedance-2-0-900-720p", map[string]any{"prompt": "boat", "duration": 10}, 1.5},
		{"vw-seedance-2-0-fast", "lec-vw-seedance-2-0-fast", map[string]any{"prompt": "boat", "images": []any{"https://media.example/ref.png"}}, 1.6},
		{"vp-wan-3-0-native", "lec-vp-wan-3-0-native", map[string]any{"prompt": "boat", "duration": 5, "resolution": "1080p"}, 5.5},
		{"ap-kling-4-yz", "lec-ap-kling-4-yz", map[string]any{"prompt": "boat"}, 0.72},
		{"ap-wan-3-0-720p", "lec-ap-wan-3-0-720p", map[string]any{"prompt": "boat", "duration": 2}, 4},
		{"qs-wan-3-0-cc-720p", "lec-qs-wan-3-0-cc-720p", map[string]any{"prompt": "boat", "duration": 30}, 3},
		{"shuikule-seedance-2-5-bd", "lec-shuikule-seedance-2-5-bd", map[string]any{"prompt": "boat", "duration": 4}, 6},
		{"vp-seedance-2-5-bd", "lec-vp-seedance-2-5-bd", map[string]any{"prompt": "boat", "resolution": "480p"}, 4.2},
		{"vp-seedance-2-5-bd", "lec-vp-seedance-2-5-bd", map[string]any{"prompt": "boat"}, 5.5},
		{"vp-seedance-2-5-flash-t", "lec-vp-seedance-2-5-flash-t", map[string]any{"prompt": "boat"}, 3.5},
	} {
		t.Run(tc.model+"/"+tc.upstream, func(t *testing.T) {
			assert.Contains(t, plugin.Meta.Models, tc.model)
			defaults, found := plugin.Meta.DefaultsForModel(tc.model)
			require.True(t, found)
			require.NotEmpty(t, defaults.BillingExpr)

			ctx := lecAPIContext(tc.model, tc.body)
			request := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)
			body := request["body"].(map[string]any)
			assert.Equal(t, tc.upstream, body["model"])
			facts := lecAPIHook(t, plugin, "extractUsage", ctx)
			switch tc.model {
			case "vg-seedance-2-0-fast-cc", "md-seedance-2-5-900-720p", "md-seedance-2-0-900-720p", "vw-seedance-2-0-fast",
				"ap-kling-4-yz", "ap-wan-3-0-720p", "qs-wan-3-0-cc-720p", "shuikule-seedance-2-5-bd":
				assert.NotContains(t, body, "resolution", "upstream fixes the resolution")
				assert.Equal(t, "720p", facts["resolution"])
			case "vp-seedance-2-5-flash-t":
				assert.NotContains(t, body, "resolution", "upstream fixes the resolution")
				assert.Equal(t, "480p", facts["resolution"])
			}
			switch tc.model {
			case "am-minimax-h3-official":
				if tc.body["resolution"] == nil {
					assert.Equal(t, "768p", body["resolution"])
					assert.Equal(t, float64(5), body["duration"])
					assert.Equal(t, "16:9", body["aspect_ratio"])
				} else {
					assert.Equal(t, "1440p", body["resolution"])
					assert.Equal(t, float64(15), body["duration"])
					assert.Equal(t, tc.body["first_frame"], body["first_frame"])
					assert.Equal(t, tc.body["last_frame"], body["last_frame"])
				}
			case "md-seedance-2-5-900-720p":
				assert.Equal(t, float64(30), body["duration"])
				assert.Equal(t, false, body["process_face"])
			case "vw-seedance-2-0-fast":
				assert.NotContains(t, body, "duration", "upstream fixes the duration")
				assert.Equal(t, float64(15), facts["seconds"])
			}
			cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.price, cost, 1e-12)
		})
	}
}

func TestLecAPIMLWan3FrameInputsAndReferenceVideoLimit(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	const firstFrame = "https://media.example/start.png"
	const lastFrame = "https://media.example/end.png"
	ctx := lecAPIContext("ml-wan-3", map[string]any{"prompt": "boat", "start_image": firstFrame, "end_image": lastFrame})
	request := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, firstFrame, request["start_image"])
	assert.Equal(t, lastFrame, request["end_image"])

	defaults, found := plugin.Meta.DefaultsForModel("ml-wan-3")
	require.True(t, found)
	assert.Contains(t, defaults.Description["zh"], "参考视频累计最长 15 秒")

	ctx = lecAPIContext("ml-wan-3", map[string]any{
		"prompt": "boat",
		"videos": []any{"https://media.example/first.mp4", "https://media.example/second.mp4"},
	})
	probe := jsplugin.WithMediaProber(t.Context(), func(_ context.Context, _ jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		return jsplugin.FetchResponse{Status: http.StatusOK, Body: []byte("8")}, nil
	})
	_, err := plugin.Engine.Call(probe, "buildSubmitRequest", ctx)
	require.ErrorContains(t, err, "combined reference video duration exceeds the supported range")
}

func TestLecAPIOpenAIVideoFrameRoles(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	const first, last = "https://media.example/a.png", "https://media.example/b.png"
	for _, name := range []string{"ap-seedance-2-0-ot", "ap-seedance-2-0-ot-fast", "ap-seedance-2-0-ot-mini", "ap-seedance-2-5-ot", "ap-muse-video-720-10s", "am-minimax-h3-official", "ml-wan-3", "gt-seedance-2-0-full", "ms-minimax-h3"} {
		t.Run(name, func(t *testing.T) {
			for _, pair := range [][2]string{{first, ""}, {first, last}, {last, first}} {
				fields := map[string]any{"model": name, "prompt": "transition", "first_frame": pair[0]}
				if pair[1] != "" {
					fields["last_frame"] = pair[1]
				}
				decoded, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{"body": map[string]any{"kind": "json", "value": fields}})
				require.NoError(t, err)
				request := lecAPIHook(t, plugin, "buildSubmitRequest", lecAPIContext(name, decoded.(map[string]any)["requestBody"].(map[string]any)))
				body := request["body"].(map[string]any)
				if name == "gt-seedance-2-0-full" || name == "ms-minimax-h3" {
					want, mode := []any{pair[0]}, "singleImage2video"
					if pair[1] != "" {
						want, mode = append(want, pair[1]), "frames2video"
					}
					assert.Equal(t, want, body["images"])
					if name == "gt-seedance-2-0-full" {
						assert.Equal(t, mode, body["mode"])
					} else {
						assert.NotContains(t, body, "mode")
					}
					assert.NotContains(t, body, "first_frame")
					continue
				}
				startKey, endKey := "first_frame", "last_frame"
				if name == "ml-wan-3" {
					startKey, endKey = "start_image", "end_image"
				}
				assert.Equal(t, pair[0], body[startKey])
				if pair[1] == "" {
					assert.NotContains(t, body, endKey)
				} else {
					assert.Equal(t, pair[1], body[endKey])
				}
				assert.NotContains(t, body, "images")
			}
			refs := []any{first, last}
			body := lecAPIHook(t, plugin, "buildSubmitRequest", lecAPIContext(name, map[string]any{"prompt": "reference", "images": refs}))["body"].(map[string]any)
			assert.Equal(t, refs, body["images"])
			assert.NotContains(t, body, "first_frame")
			assert.NotContains(t, body, "start_image")
			assert.NotContains(t, body, "mode")
		})
	}
	for _, fields := range []map[string]any{{"last_frame": last}, {"first_frame": first, "images": []any{last}}, {"first_frame": first, "mode": "mixed2video"}, {"first_frame": first, "metadata": map[string]any{"first_frame_url": last}}} {
		fields["prompt"] = "transition"
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", lecAPIContext("gt-seedance-2-0-full", fields))
		require.Error(t, err)
	}
}

func TestLecAPITemporaryMarkupCapAppliesOnlyToPerRequestPricing(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model string
		body  map[string]any
		want  float64
	}{
		{"mw-seedance-2-0-mini", map[string]any{"prompt": "boat"}, 1.1},
		{"ap-kling-4-yz", map[string]any{"prompt": "boat", "duration": 3}, 0.72},
		{"vp-seedance-2-5-bd", map[string]any{"prompt": "boat"}, 5.5},
		{"shuikule-seedance-2-5-m", map[string]any{"prompt": "boat", "duration": 5, "resolution": "1080p"}, 3.6},
	} {
		t.Run(tc.model, func(t *testing.T) {
			ctx := lecAPIContext(tc.model, tc.body)
			request := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)
			assert.Equal(t, "lec-"+tc.model, request["body"].(map[string]any)["model"])
			facts := lecAPIHook(t, plugin, "extractUsage", ctx)
			defaults, found := plugin.Meta.DefaultsForModel(tc.model)
			require.True(t, found)
			cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.want, cost, 1e-12)
		})
	}
}

func TestLecAPIVideoPoolsBillAndRouteByRequestedResolution(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		entry, resolution, vendor string
		body                      map[string]any
		price                     float64
	}{
		{"seedance-2-5-01", "720p", "lec-mm-seedance-2-5", nil, 6},
		{"seedance-2-5-02", "480p", "lec-ty-seedance-2-5-1050-flxd", map[string]any{"images": []any{"http://media.example/image.png"}}, 4.5},
		{"seedance-2-5-02", "720p", "lec-ty-seedance-2-5-1050-flxd", map[string]any{"images": []any{"http://media.example/image.png"}}, 6.5},
		{"seedance-2-0-01", "720p", "lec-vp-seedance-2-0-903", nil, 4},
		{"ty-seedance-2-0-full-933-w-720p", "720p", "lec-ty-seedance-2-0-full-933-w-720p", map[string]any{"images": []any{"https://media.example/image.png"}, "videos": []any{"https://media.example/ref.mp4"}}, 3.5},
	} {
		t.Run(tc.entry+"/"+tc.resolution, func(t *testing.T) {
			body := map[string]any{"prompt": "boat", "resolution": tc.resolution}
			maps.Copy(body, tc.body)
			ctx := lecAPIContext(tc.entry, body)
			request := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, tc.vendor, request["model"])
			ctx["upstream"] = map[string]any{"kind": "new_api"}
			chained := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, tc.entry, chained["model"])
			assert.Equal(t, tc.resolution, chained["resolution"])
			defaults, found := plugin.Meta.DefaultsForModel(tc.entry)
			require.True(t, found)
			facts := lecAPIHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, tc.resolution, facts["resolution"])
			cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.price, cost, 1e-12)
		})
	}
	for _, tc := range []struct {
		entry, resolution string
	}{
		{"seedance-2-0-01", "480p"},
		{"seedance-2-5-02", "4k"},
		{"seedance-2-5-01", "480p"},
		{"seedance-2-5-01", "1080p"},
	} {
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", lecAPIContext(tc.entry, map[string]any{"prompt": "boat", "resolution": tc.resolution}))
		require.Error(t, err)
	}
}

func TestLecAPISeedancePoolsPublishShortIDsAndSupportMappedLegacyIDs(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct{ short, display, legacy, vendor string }{
		{"seedance-2-5-01", "Seedance 2.5 · 多参考", "seedance-2-5-request-660", "lec-mm-seedance-2-5"},
		{"seedance-2-5-02", "Seedance 2.5 · 人像参考", "seedance-2-5-request-720", "lec-mf-seedance-2-5-30ref"},
		{"seedance-2-0-01", "Seedance 2.0 满血 · 720p", "seedance-2-0-request-420", "lec-vp-seedance-2-0-903"},
		{"seedance-2-0-02", "Seedance 2.0 · 720p 多参考", "seedance-2-0-request-540", "lec-vp-seedance-2-0-933-s2"},
	} {
		t.Run(tc.short, func(t *testing.T) {
			assert.Contains(t, plugin.Meta.Models, tc.short)
			assert.NotContains(t, plugin.Meta.Models, tc.legacy)
			defaults, ok := plugin.Meta.DefaultsForModel(tc.short)
			require.True(t, ok)
			assert.Equal(t, tc.display, defaults.DisplayName)
			ctx := lecAPIContext(tc.legacy, map[string]any{"model": tc.legacy, "prompt": "boat"})
			ctx["upstreamModel"] = tc.short
			request := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, tc.vendor, request["model"])
			ctx["taskId"] = "old/task"
			ctx["upstreamModel"] = tc.legacy
			ctx["upstream"] = map[string]any{"kind": "new_api"}
			assert.Equal(t, "https://lec.example/lecapi/v1/videos/old%2Ftask", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
		})
	}
	for _, name := range []string{
		"seedance-2-5-03", "seedance-2-5-request-600-480p-480-1080p-960-4k-1080",
		"seedance-2-5-04", "seedance-2-5-request-600-480p-600",
	} {
		assert.NotContains(t, plugin.Meta.Models, name)
		ctx := lecAPIContext(name, map[string]any{"prompt": "boat"})
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
		require.ErrorContains(t, err, "No LecAPI route is currently available")
		ctx["taskId"] = "old/task"
		assert.Equal(t, "https://lec.example/v1/videos/old%2Ftask", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
	}

	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	prefix := `"lec-mf-seedance-2-5-30ref": {` + "\n    kind:"
	require.Contains(t, source, prefix)
	source = strings.Replace(source, prefix, strings.Replace(prefix, "kind:", "retired: true,\n    kind:", 1), 1)
	updated, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	require.Contains(t, updated.Meta.Models, "seedance-2-5-02")
	ctx := lecAPIContext("seedance-2-5-request-720", map[string]any{"prompt": "boat", "images": []any{"http://media.example/ref.png"}, "resolution": "720p"})
	ctx["upstreamModel"] = "seedance-2-5-02"
	request := lecAPIHook(t, updated, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, "lec-ty-seedance-2-5-1050-flxd", request["model"])
}

func TestLecAPISeedance25RoutesByResolutionAndMaterials(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	const name = "seedance-2-5-02"
	defaults, found := plugin.Meta.DefaultsForModel(name)
	require.True(t, found)
	for _, text := range []string{"480p：1–10 张图 + 最多 5 段音频", "720p：最多 30 张图 + 最多 10 段视频 + 最多 10 段音频（合计最多 30 项", "videos：参考视频"} {
		assert.Contains(t, defaults.Description["zh"], text)
	}
	const referenceVideo = "https://media.example/reference.mp4"
	videoCtx := lecAPIContext(name, map[string]any{"prompt": "boat", "videos": []any{referenceVideo}})
	probe := jsplugin.WithMediaProber(t.Context(), func(_ context.Context, request jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		assert.Equal(t, referenceVideo, request.URL)
		return jsplugin.FetchResponse{Status: http.StatusOK, Body: []byte("8")}, nil
	})
	value, err := plugin.Engine.Call(probe, "buildSubmitRequest", videoCtx)
	require.NoError(t, err)
	videoBody := value.(map[string]any)["body"].(map[string]any)
	assert.Equal(t, "lec-mf-seedance-2-5-30ref", videoBody["model"])
	assert.Equal(t, []any{referenceVideo}, videoBody["videos"])
	assert.NotContains(t, videoBody, "resolution", "MF fixes the resolution upstream")
	facts, err := plugin.Engine.Call(probe, "extractUsage", videoCtx)
	require.NoError(t, err)
	assert.Equal(t, "720p", facts.(map[string]any)["resolution"])
	cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts.(map[string]any)})
	require.NoError(t, err)
	assert.InDelta(t, 6.5, cost, 1e-12)
	videoCtx["requestBody"].(map[string]any)["images"] = slices.Repeat([]any{"https://media.example/ref.png"}, 30)
	_, err = plugin.Engine.Call(probe, "buildSubmitRequest", videoCtx)
	require.ErrorContains(t, err, "supports this combination")

	for _, renderer := range []struct {
		export string
		path   []string
	}{
		{"native", []string{"task"}},
		{"protocols", []string{"openai_video", "render"}},
	} {
		result, err := plugin.Engine.CallPath(t.Context(), renderer.export, renderer.path, map[string]any{"model": name}, map[string]any{
			"data": map[string]any{"model": "lec-mf-seedance-2-5-30ref", "result": map[string]any{"model_name": "lec-mf-seedance-2-5-30ref"}},
		})
		require.NoError(t, err)
		encoded, err := common.Marshal(result)
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "mf-seedance")
		assert.Contains(t, string(encoded), `"model":"seedance-2-5-02"`)
	}
	for _, input := range []map[string]any{
		{"resolution": "1080p"},
		{"metadata": map[string]any{"duration": 31}},
		{"metadata": map[string]any{"model": "lec-hc-seedance-2-5-3010"}},
		{"resolution": "480p", "videos": []any{"http://media.example/source.mp4"}},
	} {
		input["prompt"] = "boat"
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", lecAPIContext(name, input))
		require.Error(t, err)
	}
	for _, old := range []string{"mj-seedance-2-5-fd-720p", "lec-ty-seedance-2-5-1050-flxd"} {
		assert.NotContains(t, plugin.Meta.Models, strings.TrimPrefix(old, "lec-"))
		ctx := lecAPIContext(old, map[string]any{"prompt": "boat"})
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
		require.Error(t, err)
		mapped := lecAPIContext(name, map[string]any{"prompt": "boat"})
		mapped["upstreamModel"] = old
		_, err = plugin.Engine.Call(t.Context(), "buildSubmitRequest", mapped)
		require.Error(t, err)
		_, err = plugin.Engine.CallPath(t.Context(), "native", []string{"createVideo"}, map[string]any{
			"body": map[string]any{"kind": "json", "value": map[string]any{"model": old, "prompt": "boat"}},
		})
		require.Error(t, err)
		ctx["taskId"] = "historical/id"
		assert.Equal(t, "https://lec.example/v1/videos/historical%2Fid", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
	}
}

func TestLecAPIInlineImageArtifactsRemainAvailableForStorage(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	data := map[string]any{"data": []any{map[string]any{"b64_json": "aW1hZ2U="}, map[string]any{"url": "https://cdn.example/second.png"}}}
	ctx := lecAPIContext("image2", nil)
	ctx["status"], ctx["action"], ctx["data"] = "SUCCESS", "imagine", data
	ctx["state"] = map[string]any{"queryable": false}
	ctx["upstreamTaskId"] = "sync-local-id"
	artifacts, err := plugin.Engine.Call(t.Context(), "listArtifacts", ctx)
	require.NoError(t, err)
	require.Len(t, artifacts, 2)
	for index, expected := range []string{"data:image/png;base64,aW1hZ2U=", "https://cdn.example/second.png"} {
		ctx["artifactKey"] = []string{"image-b64-1", "image-1"}[index]
		descriptor := lecAPIHook(t, plugin, "buildContentRequest", ctx)
		assert.Equal(t, expected, descriptor["url"])
		assert.Equal(t, true, descriptor["credentialless"])
		assert.NotContains(t, descriptor, "headers")
	}
}

func TestLecAPIBindsCatalogModelsToTheirMediaProtocol(t *testing.T) {
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	registry := jsplugin.NewRegistry()
	plugin, err := registry.RegisterFactory(source, jsplugin.Options{Key: "lecapi"})
	require.NoError(t, err)
	assert.Len(t, plugin.Meta.Models, 77)
	assert.Contains(t, plugin.Meta.RequiredCapabilities, "submit-no-retry@1")
	assert.True(t, plugin.Meta.SupportsUpstream(jsplugin.UpstreamKindNewAPI))
	priced := 0
	displayNames := make(map[string]struct{})
	for _, modelName := range plugin.Meta.Models {
		assert.False(t, strings.HasPrefix(modelName, "lec-"), "vendor prefixes are private")
		defaults, found := plugin.Meta.ModelDefaults[modelName]
		require.True(t, found)
		assert.NotEmpty(t, defaults.DisplayName, modelName)
		assert.False(t, strings.HasPrefix(defaults.DisplayName, "Stolas "), modelName)
		assert.NotRegexp(t, ` · [0-9]{2}$`, defaults.DisplayName, modelName)
		assert.NotContains(t, displayNames, defaults.DisplayName, "duplicate public title for %s", modelName)
		displayNames[defaults.DisplayName] = struct{}{}
		if defaults.BillingExpr != "" {
			priced++
		}
		for _, description := range defaults.Description {
			assert.NotContains(t, description, "lec-")
		}
	}
	assert.Equal(t, len(plugin.Meta.Models), priced)
	assert.Equal(t, "Wan 3.0 · 多参考按秒", plugin.Meta.ModelDefaults["aj-wan-3-0"].DisplayName)
	for _, name := range []string{
		"ap-seedance-2-0", "haya-seedance-2-0-vc-720p", "jl-wan-3-0-1055-720p",
		"mj-seedance-2-5-zd", "nzt-wan-3-0-prime",
		"vp-seedance-2-9-301010", "image2", "nano-banana-2",
		"mf-seedance-2-5-t1", "h3video-2k", "am-wan-3-0", "jm-seedance-2-5", "jm-seedance-2-5-flash", "MiniMax-H3", "gt-seedance-2-5-720p",
		"ac-seedance-2-0-fast-2-720p", "wan-3-0",
		"vg-seedance-2-5-vg-720p", "haya-seedance-2-5-flash",
		"59-seedance-2-5-flash", "59-seedance-2-0-mini", "59-seedance-2-0-full", "59-seedance-2-0-fast",
		"haya-seedance-2-0-xg-720", "vg-seedance-2-0-full-933", "vg-seedance-2-0-mini-cc",
		"mj-seedance-2-0-full-933-jd",
		"jl-seedance-2-0-fast-auto", "jl-seedance-2-0-full",
		"hc-seedance-2-0-903", "vg-wan-3-0", "md-seedance-2-0-fast-900-720p",
		"zy-seedance-2-5-discount2", "zy-seedance-2-5-discount", "mf-seedance-2-5-x", "hc-seedance-2-5-3010",
		"vv-seedance-2-5-r2", "vv-seedance-2-5-r2-plus", "haya-seedance-2-5-xg-480p", "mf-seedance-2-5-std",
		"seedance-2-5-301010", "mj-seedance-2-5-qn", "vp-seedance-2-5-flash", "vp-seedance-2-5-flash-m",
		"seedance-2-0-request-156", "seedance-2-0-request-240", "seedance-2-0-request-360",
	} {
		assert.NotContains(t, plugin.Meta.Models, name)
		assert.NotContains(t, plugin.Meta.ModelDefaults, name)
		_, bound := registry.Generation().LookupEndpoint("POST", "/v1/videos", name)
		assert.False(t, bound, name)
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, lecAPIContext(name, map[string]any{"prompt": "boat"}))
			require.ErrorContains(t, err, "is no longer available", name)
		}
		_, err := plugin.Engine.CallPath(t.Context(), "native", []string{"createVideo"}, map[string]any{
			"body": map[string]any{"kind": "json", "value": map[string]any{"model": name, "prompt": "boat"}},
		})
		require.ErrorContains(t, err, "is no longer available", name)
		query := lecAPIContext(name, nil)
		query["taskId"] = "historical/id"
		assert.Equal(t, "https://lec.example/v1/videos/historical%2Fid", lecAPIHook(t, plugin, "buildQueryRequest", query)["url"])
	}
	for _, tc := range []struct{ name, vendor string }{
		{"aj-wan-3-0", "Wan"}, {"ms-minimax-h3", "Minimax"}, {"ac-image-2", "OpenAI"},
		{"ac-seedream-5-0-lite", "ByteDance"}, {"gt-seedance-2-0-full", "ByteDance"},
		{"nw-nano-banana-2-1", "Google"}, {"sl-midjourney", "Midjourney"},
	} {
		assert.Equal(t, tc.vendor, plugin.Meta.ModelDefaults[tc.name].Vendor, tc.name)
	}
	schema, _ := plugin.Meta.UsageForModel("aj-wan-3-0")
	assert.ElementsMatch(t, []string{"480p", "720p", "1080p"}, schema["resolution"].Enum)
	for _, name := range []string{"nw-nano-banana-2-1", "tinysnow-banana-3"} {
		facts := lecAPIHook(t, plugin, "extractUsage", lecAPIContext(name, map[string]any{"prompt": "boat"}))
		schema, _ := plugin.Meta.UsageForModel(name)
		assert.Contains(t, schema["resolution"].Enum, facts["resolution"], "omitting an optional resolution must remain valid for %s", name)
	}
	assert.Contains(t, plugin.Meta.Models, "aj-wan-3-0")
	for _, name := range []string{
		"seedance-2-5-02", "seedance-2-0-01", "mw-vidu-q4-nopeak",
		"mw-wan-3-0-s", "rs-seedance-2-0-fast", "seedance-2-0-mini-b1-480p", "seedance-2-0-mini-c9-933-720p",
	} {
		assert.Contains(t, plugin.Meta.Models, name)
		binding, found := registry.Generation().LookupEndpoint("POST", "/v1/videos", name)
		require.True(t, found, name)
		assert.Equal(t, "lecapi", binding.Plugin.Meta.Key)
		_, bound := registry.Generation().LookupEndpoint("POST", "/v1/images/generations", name)
		assert.False(t, bound, name)
	}
	for _, spelling := range []string{"AJ-WAN-3-0", "aj-wan-3-0"} {
		canonical, found := registry.Generation().CanonicalModel(spelling)
		require.True(t, found)
		assert.Equal(t, "aj-wan-3-0", canonical)
		binding, found := registry.Generation().LookupEndpoint("POST", "/v1/videos", canonical)
		require.True(t, found)
		assert.Equal(t, "lecapi", binding.Plugin.Meta.Key)
	}
	imageModels := []string{"ac-image-2-5-flare", "ac-image-2-5-sunburst", "ac-image-2", "nw-image-2", "nw-image-2-5", "nw-image-2-6", "nw-nano-banana-2-1", "sl-midjourney", "sl-midjourney-niji", "ac-seedream-5-0-lite", "ac-seedream-5-0-pro", "ty-seedream-5-pro", "tinysnow-banana-3-1", "tinysnow-banana-3", "tinysnow-image-2", "tinysnow-image-2-5-flare", "tinysnow-image-2-5-sunburst", "image-2", "ty-seedream-5-5"}
	for _, modelName := range plugin.Meta.Models {
		path, protocol := "/v1/videos", "openai_video"
		for _, imageModel := range imageModels {
			if modelName == imageModel {
				path, protocol = "/v1/images/generations", "openai_image"
				break
			}
		}
		binding, found := registry.Generation().LookupEndpoint("POST", path, modelName)
		require.True(t, found, modelName)
		assert.Equal(t, protocol, binding.Protocol, modelName)
	}
	for _, tc := range []struct{ model, path, protocol string }{
		{"image-2", "/v1/images/generations", "openai_image"},
		{"ac-image-2-5-flare", "/v1/images/generations", "openai_image"},
		{"ty-seedream-5-pro", "/v1/images/generations", "openai_image"},
		{"ty-seedream-5-5", "/v1/images/generations", "openai_image"},
		{"seedance-2-0-01", "/v1/videos", "openai_video"},
		{"seedance-2-5-01", "/v1/videos", "openai_video"},

		{"ap-seedance-2-0-ot", "/v1/videos", "openai_video"},
		{"ap-seedance-2-0-ot-fast", "/v1/videos", "openai_video"},
		{"ap-seedance-2-0-ot-mini", "/v1/videos", "openai_video"},
		{"ms-minimax-h3-upscale", "/v1/videos", "openai_video"},
	} {
		binding, found := registry.Generation().LookupEndpoint("POST", tc.path, tc.model)
		require.True(t, found, tc.model)
		assert.Equal(t, tc.protocol, binding.Protocol)
	}
}

func TestLecAPIPublicModelsPreserveVendorRoutingAndGatewayChaining(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct{ public, vendor string }{
		{"aj-wan-3-0", "lec-aj-wan-3-0"}, {"ac-image-2", "lec-ac-image-2"}, {"ty-seedream-5-5", "lec-ty-seedream-5-5"},
		{"image-2", "lec-image-2"},

		{"ap-seedance-2-0-ot", "lec-ap-seedance-2-0-ot"},
		{"ap-seedance-2-0-ot-fast", "lec-ap-seedance-2-0-ot-fast"},
		{"ap-seedance-2-0-ot-mini", "lec-ap-seedance-2-0-ot-mini"},
	} {
		ctx := lecAPIContext(tc.public, map[string]any{"prompt": "boat"})
		assert.Equal(t, tc.vendor, lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"])
		ctx["upstream"] = map[string]any{"kind": "new_api"}
		assert.Equal(t, tc.public, lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"], "a chained gateway accepts the public plugin identity")
		ctx["model"], ctx["upstreamModel"] = "local-alias", tc.vendor
		delete(ctx, "upstream")
		assert.Equal(t, tc.vendor, lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"], "an explicit final vendor mapping retains its original wire identity")
		ctx["upstream"] = map[string]any{"kind": "new_api"}
		assert.Equal(t, tc.public, lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"])
	}
	data := map[string]any{"model": "lec-aj-wan-3-0", "status": "completed", "url": "https://media.example/a.mp4", "result": map[string]any{"model_name": "lec-ac-image-2"}}
	value, err := plugin.Engine.CallPath(t.Context(), "native", []string{"task"}, map[string]any{}, map[string]any{"task_id": "task_public", "status": "SUCCESS", "data": data})
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), `"lec-aj-wan-3-0"`)
	assert.NotContains(t, string(encoded), `"lec-ac-image-2"`)
	assert.Contains(t, string(encoded), `"model":"aj-wan-3-0"`)
	value, err = plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "render"}, map[string]any{"model": "client-alias"}, map[string]any{"data": data})
	require.NoError(t, err)
	assert.Equal(t, "client-alias", value.(map[string]any)["model"])
}

func TestLecAPIBuildsAndMetersTheFinalMappedBody(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := lecAPIContext("aj-wan-3-0", map[string]any{"prompt": "A red boat", "metadata": map[string]any{"duration": 7, "resolution": "1080p"}})
	ctx["model"] = "local-video-alias"
	submit := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "https://lec.example/v1/videos", submit["url"])
	assert.NotContains(t, submit, "model")
	body := submit["body"].(map[string]any)
	assert.Equal(t, "lec-aj-wan-3-0", body["model"])
	assert.Equal(t, float64(7), body["duration"])
	assert.Equal(t, "1080p", body["resolution"])
	facts := lecAPIHook(t, plugin, "extractUsage", ctx)
	assert.Equal(t, float64(7), facts["seconds"])
	assert.Equal(t, "1080p", facts["resolution"])
	assert.Equal(t, float64(1), facts["requests"])
	ctx["upstream"] = map[string]any{"kind": "new_api"}
	assert.Equal(t, "https://lec.example/lecapi/v1/videos", lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["url"])
	for _, invalid := range []map[string]any{
		{"duration": -1}, {"duration": 31}, {"duration": 5.5}, {"n": 2}, {"parameters": map[string]any{"duration": 9000000000000000000}},
		{"extra": map[string]any{"parameters": map[string]any{"n": 128}}}, {"videos": []any{"http://media.example/ref.mp4"}},
	} {
		request := maps.Clone(invalid)
		request["prompt"] = "A red boat"
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, lecAPIContext("aj-wan-3-0", request))
			require.Error(t, err, "%s must validate all parameter containers: %v", hook, invalid)
		}
	}
}

func TestLecAPIOfficialSeedanceReservesSmartDurationAndPreservesMode(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	images := []any{"https://media.example/first.png", "https://media.example/last.png"}
	ctx := lecAPIContext("gt-seedance-2-0-full", map[string]any{"prompt": "boat", "images": images, "metadata": map[string]any{
		"duration": -1, "mode": "frames2video", "aspect_ratio": "adaptive",
	}})
	body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, float64(-1), body["duration"])
	assert.Equal(t, "frames2video", body["mode"])
	assert.Equal(t, "adaptive", body["aspect_ratio"])
	assert.Equal(t, images, body["images"])
	facts := lecAPIHook(t, plugin, "extractUsage", ctx)
	assert.Equal(t, float64(15), facts["seconds"], "smart duration reserves the longest supported output")
	defaults, found := plugin.Meta.DefaultsForModel("gt-seedance-2-0-full")
	require.True(t, found)
	cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
	require.NoError(t, err)
	assert.InDelta(t, 8.1, cost, 1e-12)
	completion := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 7})
	maps.Copy(facts, completion)
	cost, _, err = billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
	require.NoError(t, err)
	assert.InDelta(t, 3.78, cost, 1e-12)
	ctx["requestBody"] = map[string]any{"prompt": "boat", "images": images}
	body = lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.NotContains(t, body, "mode", "omitted mode lets upstream infer production mode from references")
}

func TestLecAPIConvertsImageSizesWithoutChangingGenerationSettings(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, protocol := range plugin.Meta.Protocols {
		if protocol.Name != jsplugin.ProtocolOpenAIImage {
			continue
		}
		require.NotEmpty(t, protocol.Models)
		for _, name := range protocol.Models {
			t.Run(name, func(t *testing.T) {
				ctx := lecAPIContext(name, map[string]any{"prompt": "boat", "size": "1792x1024"})
				body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
				assert.Equal(t, "16:9", body["aspect_ratio"])
				assert.NotContains(t, body, "size")
				facts := lecAPIHook(t, plugin, "extractUsage", ctx)
				if resolution, ok := body["resolution"]; ok {
					assert.Equal(t, resolution, facts["resolution"])
				}
				assert.Equal(t, float64(1), facts["image_count"])
			})
		}
	}
	for _, tc := range []struct{ size, ratio string }{
		{"1024x1024", "1:1"}, {"3840x2160", "16:9"}, {"2160x3840", "9:16"},
		{"2048x1536", "4:3"}, {"1536x2048", "3:4"}, {"1536x1024", "3:2"}, {"1024x1536", "2:3"},
		{" 3840 X 2160 ", "16:9"}, {"3840×2160", "16:9"}, {"16:9", "16:9"},
		{"1792x1024", "16:9"}, {"1024x1792", "9:16"}, {"1100x1000", "1:1"},
		{"1200x1000", "4:3"}, {"1000x1200", "3:4"},
		{"1280x1024", "4:3"}, {"1024x1280", "3:4"},
	} {
		t.Run(tc.size, func(t *testing.T) {
			ctx := lecAPIContext("tinysnow-image-2-5-flare", map[string]any{"prompt": "boat", "size": tc.size, "resolution": "4k", "quality": "high", "n": 2})
			body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, "lec-tinysnow-image-2-5-flare", body["model"])
			assert.Equal(t, tc.ratio, body["aspect_ratio"])
			assert.Equal(t, "4K", body["resolution"])
			assert.Equal(t, "high", body["quality"])
			assert.Equal(t, float64(2), body["n"])
			assert.NotContains(t, body, "size")
			facts := lecAPIHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, body["resolution"], facts["resolution"])
			assert.Equal(t, body["n"], facts["image_count"])
		})
	}
	for _, name := range []string{"tinysnow-image-2", "tinysnow-image-2-5-sunburst"} {
		ctx := lecAPIContext(name, map[string]any{"prompt": "boat", "ratio": "16:9", "metadata": map[string]any{
			"size": "1920x1080", "extra": map[string]any{"parameters": map[string]any{"resolution_name": "2k", "quality": "HIGH", "n": 10}},
		}})
		ctx["model"] = "canvas-alias"
		body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
		assert.Equal(t, "16:9", body["aspect_ratio"])
		assert.Equal(t, "2K", body["resolution"])
		assert.Equal(t, "high", body["quality"])
		assert.NotContains(t, body, "size")
		assert.Equal(t, float64(10), lecAPIHook(t, plugin, "extractUsage", ctx)["image_count"])
		ctx["upstream"] = map[string]any{"kind": "new_api"}
		assert.Equal(t, name, lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"])
	}
	for _, tc := range []struct{ model, size, resolution string }{
		{"tinysnow-image-2", "4k", "4K"}, {"tinysnow-image-2", "1536x1024", "1K"},
		{"ac-image-2", "3840x2160", "4K"}, {"ac-seedream-5-0-pro", "1920x1080", "2K"},
		{"aj-wan-3-0", "1080p", "1080p"},
	} {
		request := map[string]any{"prompt": "boat", "size": tc.size}
		if tc.model == "ac-image-2" || tc.model == "ac-seedream-5-0-pro" {
			request["resolution"] = strings.ToLower(tc.resolution)
		}
		ctx := lecAPIContext(tc.model, request)
		body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
		assert.Equal(t, tc.resolution, body["resolution"], "%s must retain its resolution rules", tc.model)
		assert.Equal(t, tc.resolution, lecAPIHook(t, plugin, "extractUsage", ctx)["resolution"])
		if tc.model == "ac-image-2" || tc.model == "ac-seedream-5-0-pro" {
			assert.Equal(t, "16:9", body["aspect_ratio"])
			assert.NotContains(t, body, "size")
		}
	}
	for _, name := range []string{"sl-midjourney", "sl-midjourney-niji"} {
		for _, tc := range []struct{ size, ratio string }{
			{"2520x1080", "21:9"}, {"1792x768", "21:9"}, {"2500x1080", "21:9"},
			{"1024x1280", "4:5"}, {"1280x1024", "5:4"}, {"1000x1280", "4:5"}, {"1280x1000", "5:4"},
			{"4:5", "4:5"}, {"5:4", "5:4"},
		} {
			body := lecAPIHook(t, plugin, "buildSubmitRequest", lecAPIContext(name, map[string]any{"prompt": "boat", "size": tc.size}))["body"].(map[string]any)
			assert.Equal(t, tc.ratio, body["aspect_ratio"], "%s must match the closest supported canvas ratio for %s", name, tc.size)
		}
	}
	seedream := lecAPIHook(t, plugin, "buildSubmitRequest", lecAPIContext("ty-seedream-5-pro", map[string]any{"prompt": "boat", "size": "1536x1024"}))["body"].(map[string]any)
	assert.Equal(t, "4:3", seedream["aspect_ratio"], "the nearest ratio must also be supported by the selected model")
	for _, invalid := range []map[string]any{
		{"size": "0x2160"}, {"size": "3840x-2160"}, {"size": "9007199254740992x9007199254740992"},
		{"size": "auto"}, {"size": "3840x2160", "aspect_ratio": "1:1"},
		{"size": "3840x2160", "metadata": map[string]any{"ratio": "9:16"}},
		{"size": "4k", "resolution": "2K"}, {"size": "3840x2160", "quality": "ultra"},
		{"size": "3840x2160", "resolution": "8K"}, {"size": "3840x2160", "metadata": map[string]any{"n": 11}},
	} {
		request := maps.Clone(invalid)
		request["prompt"] = "boat"
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, lecAPIContext("tinysnow-image-2", request))
			require.Error(t, err, "%s must reject invalid or conflicting settings: %v", hook, invalid)
		}
	}
}

func TestLecAPIEnforcesModelSpecificMaterialsAndOptions(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model string
		body  map[string]any
	}{
		{"am-minimax-h3-official", map[string]any{"prompt": "boat", "metadata": map[string]any{"duration": 16}}},
		{"am-minimax-h3-official", map[string]any{"prompt": "boat", "last_frame": "https://media.example/last.png"}},
		{"am-minimax-h3-official", map[string]any{"prompt": "boat", "first_frame": "https://media.example/first.png", "videos": []any{"https://media.example/source.mp4"}}},
		{"ad-minimax-h3-replace", map[string]any{"images": []any{"https://media.example/person.png"}}},
		{"mf-video-upscale", map[string]any{"prompt": "upscale", "videos": []any{"https://media.example/source.mp4"}}},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "duration": 16, "resolution": "2K"}},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "resolution": "480p", "videos": []any{"https://media.example/source.mp4"}}},
		{"sl-midjourney", map[string]any{"prompt": "boat", "action": "blend"}},
		{"sl-midjourney", map[string]any{"prompt": "boat", "version": "8.2", "draft": true}},
		{"tinysnow-image-2", map[string]any{"prompt": "boat", "n": 11}},
		{"image-2", map[string]any{"prompt": "boat", "metadata": map[string]any{"n": 11}}},
		{"ap-muse-video-720-10s", map[string]any{"prompt": "boat", "duration": 5}},
		{"seedance-2-0-01", map[string]any{"prompt": "boat", "images": []any{"https://media.example/source.png"}, "duration": 16}},
		{"mw-vidu-q4-nopeak", map[string]any{"prompt": "boat"}},
		{"mw-vidu-q4-nopeak", map[string]any{"prompt": "boat", "images": []any{"https://media.example/ref.png"}, "metadata": map[string]any{"duration": 17}}},
		{"seedance-2-0-01", map[string]any{"prompt": "boat", "videos": []any{"https://media.example/ref.mp4"}}},
		{"wan-3-0-h2", map[string]any{"prompt": "boat", "resolution": "720p"}},
		{"gt-seedance-2-0-full", map[string]any{"prompt": "boat", "metadata": map[string]any{"duration": -2}}},
		{"gt-seedance-2-0-full", map[string]any{"prompt": "boat", "duration": 0}},
		{"gt-seedance-2-0-full", map[string]any{"prompt": "boat", "duration": 3}},
		{"gt-seedance-2-0-full", map[string]any{"prompt": "boat", "duration": 16}},
		{"gt-seedance-2-0-full", map[string]any{"prompt": "boat", "mode": "frames2video", "images": []any{"https://media.example/ref.png"}}},
		{"gt-seedance-2-0-full", map[string]any{"prompt": "boat", "videos": []any{"https://media.example/ref.mp4"}}},
		{"gt-seedance-2-0-full", map[string]any{"prompt": "boat", "audios": []any{"https://media.example/ref.mp3"}}},

		{"ty-seedream-5-5", map[string]any{"prompt": "boat", "resolution": "4K"}},
		{"ac-seedream-5-0-pro", map[string]any{"prompt": "boat", "resolution": "4K"}},
		{"haya-seedance-2-5-m", map[string]any{"prompt": "boat", "videos": []any{"https://media.example/source.mp4"}}},
		{"shuikule-seedance-2-5-m", map[string]any{"prompt": "boat", "duration": 4}},
		{"mf-video-upscale", map[string]any{"tier": "fast", "resolution": "8k", "videos": []any{"https://media.example/source.mp4"}}},
		{"mf-video-upscale", map[string]any{"fps": 240, "videos": []any{"https://media.example/source.mp4"}}},

		{"vg-seedance-2-0-fast-cc", map[string]any{"prompt": "boat", "videos": []any{"https://media.example/ref.mp4"}}},
		{"md-seedance-2-5-900-720p", map[string]any{"prompt": "boat", "duration": 20}},
		{"md-seedance-2-5-900-720p", map[string]any{"prompt": "boat", "resolution": "1080p"}},
		{"rs-seedance-2-0-fast", map[string]any{"prompt": "boat", "parameters": map[string]any{"duration": 7}}},
		{"rs-seedance-2-0-fast", map[string]any{"prompt": "boat", "images": []any{"data:image/png;base64,aW1hZ2U="}}},
		{"rs-seedance-2-0-fast", map[string]any{"prompt": "boat", "images": slices.Repeat([]any{"https://media.example/source.png"}, 10)}},

		{"ap-seedance-2-0-ot", map[string]any{"prompt": "boat", "duration": 16}},
		{"ap-seedance-2-0-ot-fast", map[string]any{"prompt": "boat", "resolution": "1080p"}},
		{"ap-seedance-2-0-ot-mini", map[string]any{"prompt": "boat", "metadata": map[string]any{"tokens": 1}}},
		{"ap-seedance-2-0-ot", map[string]any{"prompt": "boat", "first_frame": "https://media.example/first.png", "videos": []any{"https://media.example/source.mp4"}}},
		{"ap-seedance-2-0-ot", map[string]any{"prompt": "boat", "first_frame": "https://media.example/first.png", "last_frame": "https://media.example/last.png", "images": []any{"https://media.example/source.png"}}},
		{"ap-seedance-2-0-ot", map[string]any{"prompt": "boat", "draft": true}},
		{"ap-kling-4-yz", map[string]any{"prompt": "boat", "metadata": map[string]any{"duration": 31}}},
		{"vp-seedance-2-5-flash-t", map[string]any{"prompt": "boat", "images": slices.Repeat([]any{"https://media.example/ref.png"}, 11)}},

		{"ty-seedream-5-pro", map[string]any{"prompt": "boat", "images": []any{"https://media.example/1.png", "https://media.example/2.png", "https://media.example/3.png", "https://media.example/4.png", "https://media.example/5.png", "https://media.example/6.png", "https://media.example/7.png", "https://media.example/8.png", "https://media.example/9.png"}}},
	} {
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, lecAPIContext(tc.model, tc.body))
			require.Error(t, err, "%s %s", hook, tc.model)
		}
	}
	ctx := lecAPIContext("sl-midjourney", map[string]any{"prompt": "boat", "seed": 0, "stylize": 0, "style": false, "hd": false})
	body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, float64(0), body["seed"])
	assert.Equal(t, float64(0), body["stylize"])
	assert.Equal(t, false, body["style"])
	assert.Equal(t, false, body["hd"])
}

func TestLecAPIVideoReferenceLimitsMatchTheCatalog(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model                  string
		images, videos, audios int
		image                  string
	}{
		{"seedance-2-0-mini-b1-480p", 9, 0, 3, "https://media.example/source.png"},
		{"rs-seedance-2-0-fast", 9, 0, 0, "http://media.example/source.png"},
		{"shuikule-seedance-2-5-m", 30, 10, 10, "https://media.example/source.png"},
		{"haya-seedance-2-5-m", 30, 0, 10, "https://media.example/source.png"},

		{"ap-seedance-2-0-ot", 9, 3, 3, "https://media.example/source.png"},
		{"ap-seedance-2-0-ot-fast", 9, 3, 3, "https://media.example/source.png"},
		{"ap-seedance-2-0-ot-mini", 9, 3, 3, "https://media.example/source.png"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			request := map[string]any{"prompt": "boat", "images": slices.Repeat([]any{tc.image}, tc.images)}
			if tc.videos > 0 {
				request["videos"] = slices.Repeat([]any{"https://media.example/source.mp4"}, tc.videos)
			}
			if tc.audios > 0 {
				request["audios"] = slices.Repeat([]any{"https://media.example/source.mp3"}, tc.audios)
			}
			ctx := lecAPIContext(tc.model, map[string]any{"metadata": request})
			body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, request["images"], body["images"])
			if tc.videos > 0 {
				assert.Equal(t, request["videos"], body["videos"])
			}
			if tc.audios > 0 {
				assert.Equal(t, request["audios"], body["audios"])
			}
			ctx["upstream"] = map[string]any{"kind": "new_api"}
			assert.Equal(t, tc.model, lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"])
		})
	}
	t.Run("qs-wan-3-0-cc-720p", func(t *testing.T) {
		const ref = "https://media.example/source.mp4"
		driver := lecAPIContext("qs-wan-3-0-cc-720p", map[string]any{"prompt": "boat", "videos": []any{ref}})
		for _, duration := range []string{"15", "15.01"} {
			ctx := jsplugin.WithMediaProber(t.Context(), func(_ context.Context, request jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
				assert.Equal(t, ref, request.URL)
				return jsplugin.FetchResponse{Status: http.StatusOK, Body: []byte(duration)}, nil
			})
			for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
				_, err := plugin.Engine.Call(ctx, hook, driver)
				if duration == "15" {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, "combined reference video duration exceeds")
				}
			}
		}
	})
}

func TestLecAPICurrentModelParametersAndDescriptions(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	images := []any{"https://media.example/first.png", "https://media.example/second.png"}
	for _, tc := range []struct {
		model      string
		duration   float64
		resolution string
	}{
		{"ap-muse-video-720-10s", 10, "720p"},

		{"mw-vidu-q4-nopeak", 5, "720p"},
		{"mw-wan-3-0-s", 5, "720p"},

		{"wan-3-0-h2", 4, "480p"},

		{"ap-seedance-2-0-ot", 4, "720p"},
		{"ap-seedance-2-0-ot-fast", 4, "720p"},
		{"ap-seedance-2-0-ot-mini", 4, "720p"},
	} {
		ctx := lecAPIContext(tc.model, map[string]any{"prompt": "boat", "images": images})
		body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
		assert.Equal(t, "lec-"+tc.model, body["model"])
		assert.Equal(t, tc.duration, body["duration"], tc.model)
		assert.Equal(t, tc.resolution, body["resolution"], tc.model)
		assert.Equal(t, images, body["images"], "reference order must survive normalization")
		facts := lecAPIHook(t, plugin, "extractUsage", ctx)
		assert.Equal(t, tc.duration, facts["seconds"], tc.model)
		assert.Equal(t, tc.resolution, facts["resolution"], tc.model)
	}
	description := plugin.Meta.ModelDefaults["ad-minimax-h3-replace-multi"].Description["zh"]
	for _, text := range []string{
		"prompt：提示词，字符串类型，用来描述生成内容",
		"subjects：主体列表，数组类型，支持 1–9 个主体",
		"duration：生成时长，整数类型，支持 5–25 秒，默认 5 秒",
		"aspect_ratio：画面比例，可选 16:9 / 9:16，默认 9:16",
		"images：参考图片，数组类型，必填，支持 1–9 张，链接须使用公网 HTTPS 地址",
		"videos：参考视频，数组类型，必填，需传入 1 段，链接须使用公网 HTTPS 地址",
		"参考素材按传入顺序使用，请保持顺序",
	} {
		assert.Contains(t, description, text)
	}
	assert.NotContains(t, description, "* 为必填")
}

func TestLecAPIMidjourneySelectsAndBillsActionsFromReferenceImages(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, name := range []string{"sl-midjourney", "sl-midjourney-niji"} {
		for _, tc := range []struct {
			name   string
			body   map[string]any
			action string
			images int
			price  float64
		}{
			{"prompt only", map[string]any{}, "imagine", 0, 0.54},
			{"empty references", map[string]any{"images": []any{}}, "imagine", 0, 0.54},
			{"single reference", map[string]any{"images": []any{"https://media.example/a.png"}}, "edits", 1, 0.576},
			{"single image alias", map[string]any{"image": map[string]any{"url": "https://media.example/a.png"}}, "edits", 1, 0.576},
			{"multiple references", map[string]any{"images": []any{"https://media.example/a.png", "https://media.example/b.png"}}, "blend", 2, 0.576},
			{"nested references", map[string]any{"image_url": "https://media.example/a.png", "metadata": map[string]any{"extra": map[string]any{"image_urls": []any{"https://media.example/b.png"}}}}, "blend", 2, 0.576},
			{"explicit imagine", map[string]any{"action": "imagine", "images": []any{"https://media.example/a.png"}}, "imagine", 1, 0.54},
			{"explicit nested edits", map[string]any{"images": []any{"https://media.example/a.png", "https://media.example/b.png"}, "parameters": map[string]any{"action": "edits"}}, "edits", 2, 0.576},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				request := maps.Clone(tc.body)
				request["prompt"], request["size"] = "A red boat", "3840x2160"
				ctx := lecAPIContext(name, request)
				ctx["model"] = "canvas-mj-alias"
				submit := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)
				body := submit["body"].(map[string]any)
				assert.Equal(t, "https://lec.example/v1/images/generations", submit["url"])
				assert.Equal(t, tc.action, submit["action"])
				assert.Equal(t, tc.action, body["action"])
				assert.Equal(t, "16:9", body["aspect_ratio"])
				if tc.images > 0 {
					assert.Len(t, body["images"], tc.images)
				} else {
					assert.NotContains(t, body, "images")
				}
				facts := lecAPIHook(t, plugin, "extractUsage", ctx)
				assert.Equal(t, tc.action, facts["action"])
				assert.Equal(t, float64(1), facts["requests"])
				defaults, ok := plugin.Meta.DefaultsForModel(name)
				require.True(t, ok)
				cost, trace, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
				require.NoError(t, err)
				assert.Equal(t, tc.action, trace.MatchedTier)
				assert.InDelta(t, tc.price, cost, 1e-12)
				ctx["upstream"] = map[string]any{"kind": "new_api"}
				chained := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)
				assert.Equal(t, "https://lec.example/lecapi/v1/images/tasks", chained["url"])
				assert.Equal(t, tc.action, chained["body"].(map[string]any)["action"])
			})
		}
		references := make([]any, 10)
		for i := range references {
			references[i] = "https://media.example/reference.png"
		}
		ctx := lecAPIContext(name, map[string]any{"prompt": "boat", "images": references})
		assert.Equal(t, "blend", lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["action"])
		ctx = lecAPIContext(name, map[string]any{"prompt": "boat", "images": append(references, "https://media.example/extra.png")})
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, ctx)
			require.ErrorContains(t, err, "unsupported number of items")
			_, err = plugin.Engine.Call(t.Context(), hook, lecAPIContext(name, map[string]any{"images": []any{"https://media.example/a.png"}}))
			require.ErrorContains(t, err, "prompt is required")
		}
	}
}

func TestLecAPIImageSettlementCountsPayloads(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := lecAPIContext("tinysnow-image-2", map[string]any{"prompt": "boat", "n": 3})
	assert.Equal(t, float64(3), lecAPIHook(t, plugin, "extractUsage", ctx)["image_count"])
	for _, tc := range []struct {
		data  any
		count float64
	}{
		{map[string]any{"url": "https://media.example/a.png"}, 1},
		{map[string]any{"url": "https://media.example/a.png", "status": "ready"}, 1},
		{[]any{map[string]any{"url": "https://media.example/a.png"}, map[string]any{"b64_json": "YWJj"}, map[string]any{"revised_prompt": "boat"}}, 1},
		{[]any{map[string]any{"url": "https://media.example/a.png"}, map[string]any{"url": "https://media.example/b.png"}}, 2},
		{[]any{map[string]any{"revised_prompt": "boat"}}, 0},
	} {
		facts := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"data": tc.data})
		if tc.count == 0 {
			assert.NotContains(t, facts, "image_count")
		} else {
			assert.Equal(t, tc.count, facts["image_count"])
		}
	}
}

func TestLecAPIImageRoutesRetainOnlyAsyncJobs(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, route := range plugin.Meta.Routes {
		if route.Path == "/lecapi/v1/images/generations" {
			require.NotNil(t, route.RetainResult)
			assert.False(t, *route.RetainResult, "synchronous Base64 results may exceed persisted task limits")
		}
	}
	for _, name := range []string{"ty-seedream-5-pro", "ty-seedream-5-5"} {
		ctx := lecAPIContext(name, map[string]any{"prompt": "boat"})
		parsed := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"id": "seedream-job", "status": "queued"}})
		assert.Equal(t, "seedream-job", parsed["taskId"])
		assert.Equal(t, true, parsed["state"].(map[string]any)["queryable"])
		assert.NotContains(t, parsed, "immediate")
		ctx["upstream"] = map[string]any{"kind": "new_api"}
		assert.Equal(t, "https://lec.example/lecapi/v1/images/tasks", lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["url"])
		ctx["taskId"] = parsed["taskId"]
		assert.Equal(t, "https://lec.example/lecapi/v1/images/seedream-job", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
	}
	ctx := lecAPIContext("ac-image-2", map[string]any{"prompt": "boat"})
	parsed := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"data": []any{map[string]any{"url": "https://media.example/image.png"}}}})
	content := map[string]any{"model": "ac-image-2", "upstreamModel": "ac-image-2", "baseUrl": "https://lec.example", "artifactKey": "image-1", "upstreamTaskId": parsed["taskId"], "state": parsed["state"], "data": parsed["taskData"]}
	descriptor := lecAPIHook(t, plugin, "buildContentRequest", content)
	assert.Equal(t, "https://media.example/image.png", descriptor["url"], "a synchronous request UUID is not a queryable upstream job")

	for _, tc := range []struct {
		name, artifactKey, expectedURL string
		image                          map[string]any
	}{
		{"URL result", "image-1", "https://media.example/seedream.png", map[string]any{"url": "https://media.example/seedream.png"}},
		{"Base64 result", "image-b64-1", "data:image/png;base64,YWJj", map[string]any{"b64_json": "YWJj"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			submitCtx := lecAPIContext("ty-seedream-5-5", map[string]any{"prompt": "boat"})
			submitCtx["publicTaskId"] = "task_gateway-generated"
			parsed := lecAPIHook(t, plugin, "parseSubmitResponse", submitCtx, map[string]any{"body": map[string]any{"completed_at": time.Now().Add(-126 * time.Second).Unix(), "data": []any{tc.image}}})
			require.Equal(t, "task_gateway-generated", parsed["taskId"])
			assert.Equal(t, false, parsed["state"].(map[string]any)["queryable"])
			assert.Equal(t, "SUCCESS", parsed["immediate"].(map[string]any)["status"])

			for _, withState := range []bool{true, false} {
				contentCtx := lecAPIContext("ty-seedream-5-5", nil)
				contentCtx["artifactKey"] = tc.artifactKey
				contentCtx["upstreamTaskId"] = parsed["taskId"]
				contentCtx["publicTaskId"] = parsed["taskId"]
				contentCtx["data"] = parsed["taskData"]
				if withState {
					contentCtx["state"] = parsed["state"]
				}
				queries := 0
				callCtx := jsplugin.WithFetcher(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
					queries++
					return jsplugin.FetchResponse{Status: 400, Body: []byte(`{"error":"invented task id"}`)}, nil
				})
				value, err := plugin.Engine.Call(callCtx, "buildContentRequest", contentCtx)
				require.NoError(t, err)
				assert.Zero(t, queries)
				assert.Equal(t, tc.expectedURL, value.(map[string]any)["url"])
			}
		})
	}

	asyncCtx := lecAPIContext("ty-seedream-5-5", nil)
	asyncCtx["upstream"] = map[string]any{"kind": "new_api"}
	asyncCtx["upstreamTaskId"] = "seedream-job"
	asyncCtx["artifactKey"] = "image-1"
	asyncCtx["state"] = map[string]any{"kind": "image", "queryable": true}
	asyncCtx["data"] = map[string]any{"id": "seedream-job", "status": "queued"}
	asyncCallCtx := jsplugin.WithFetcher(t.Context(), func(_ context.Context, request jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		assert.Equal(t, "https://lec.example/lecapi/v1/images/seedream-job", request.URL)
		return jsplugin.FetchResponse{Status: 200, Body: []byte(`{"object":"lecapi.task","id":"seedream-job","status":"completed","output":{"data":[{"url":"https://media.example/refreshed-seedream.png"}]}}`)}, nil
	})
	value, err := plugin.Engine.Call(asyncCallCtx, "buildContentRequest", asyncCtx)
	require.NoError(t, err)
	assert.Equal(t, "https://media.example/refreshed-seedream.png", value.(map[string]any)["url"])
}

func TestLecAPICompletionDurationsStayWithinTheModelContract(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model   string
		seconds any
	}{
		{"aj-wan-3-0", 31}, {"aj-wan-3-0", 5.5}, {"ms-minimax-h3", 31}, {"mf-video-upscale", 2},
		{"59-seedance-2-0-fast", 12}, {"mj-seedance-2-5-fd-720p", 6}, {"gt-seedance-2-0-full", -1},
	} {
		facts := lecAPIHook(t, plugin, "extractUsageOnComplete", lecAPIContext(tc.model, nil), map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": tc.seconds})
		assert.Empty(t, facts, "invalid or output-dependent source usage must retain the reserved facts: %s", tc.model)
	}
}

func TestLecAPIPollingRejectsUnknownAndMissingArtifacts(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := map[string]any{"model": "aj-wan-3-0", "upstreamModel": "aj-wan-3-0", "taskId": "upstream/id", "baseUrl": "https://lec.example", "apiKey": "key"}
	assert.Equal(t, "https://lec.example/v1/videos/upstream%2Fid", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
	for _, tc := range []struct {
		body   map[string]any
		status string
	}{
		{map[string]any{"status": "unrecognized"}, "UNKNOWN"},
		{map[string]any{"status": "in_progress", "progress": 42}, "IN_PROGRESS"},
		{map[string]any{"status": "failed", "error": map[string]any{"message": "reference rejected"}}, "FAILURE"},
		{map[string]any{"status": "completed"}, "FAILURE"},
		{map[string]any{"status": "completed", "url": "https://media.example/video.mp4"}, "SUCCESS"},
	} {
		result := lecAPIHook(t, plugin, "parseTaskResult", ctx, tc.body)
		assert.Equal(t, tc.status, result["status"])
	}
	parsed := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"id": "task_upstream", "status": "queued"}})
	assert.Equal(t, "task_upstream", parsed["taskId"])
	ctx["upstream"] = map[string]any{"kind": "new_api"}
	assert.Equal(t, "https://lec.example/lecapi/v1/videos/upstream%2Fid", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
	for _, name := range []string{"hc-seedance-2-5-3010", "vg-seedance-2-0-full-933", "vg-seedance-2-0-mini-cc", "vg-seedance-2-5-pa"} {
		ctx["model"], ctx["upstreamModel"] = "historical-video-alias", "lec-"+name
		assert.Equal(t, "https://lec.example/lecapi/v1/videos/upstream%2Fid", lecAPIHook(t, plugin, "buildQueryRequest", ctx)["url"])
		result := lecAPIHook(t, plugin, "parseTaskResult", ctx, map[string]any{"status": "completed", "url": "https://media.example/video.mp4"})
		assert.Equal(t, "SUCCESS", result["status"], "retiring a model must preserve historical task polling: %s", name)
		facts := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, result, map[string]any{"seconds": 5})
		assert.Equal(t, float64(5), facts["seconds"], "historical tasks must retain completion usage: %s", name)
	}
}

func TestLecAPIChargesThePublishedBillingUnit(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model string
		body  map[string]any
		want  float64
	}{
		{"wan-3-0-h2", map[string]any{"prompt": "boat", "duration": 12}, 3},
		{"aj-wan-3-0", map[string]any{"prompt": "boat", "duration": 7, "resolution": "1080p"}, 5.18},

		{"seedance-2-0-mini-b1-480p", map[string]any{"prompt": "boat"}, 1.8},
		{"seedance-2-0-mini-c9-933-720p", map[string]any{"prompt": "boat", "videos": []any{"https://media.example/ref.mp4"}}, 3},

		{"mw-vidu-q4-nopeak", map[string]any{"prompt": "boat", "resolution": "480p", "images": []any{"https://media.example/ref.png"}}, 0.9},
		{"mw-vidu-q4-nopeak", map[string]any{"prompt": "boat", "images": []any{"https://media.example/ref.png"}}, 1.1},
		{"mw-vidu-q4-nopeak", map[string]any{"prompt": "boat", "resolution": "1080p", "images": []any{"https://media.example/ref.png"}}, 1.4},
		{"mw-wan-3-0-s", map[string]any{"prompt": "boat", "duration": 30, "resolution": "1080p"}, 4},

		{"tinysnow-image-2", map[string]any{"prompt": "boat", "n": 3}, 0.18},
		{"image-2", map[string]any{"prompt": "boat", "n": 3, "resolution": "4K", "quality": "high"}, 0.12},
		{"ac-image-2-5-flare", map[string]any{"prompt": "boat", "size": "3840x2160", "resolution": "4k"}, 0.12},
		{"ac-image-2-5-flare", map[string]any{"prompt": "boat", "size": "3840x2160"}, 0.06},
		{"nw-image-2", map[string]any{"prompt": "boat", "size": "1920x1080", "resolution": "2k", "quality": "high"}, 0.036},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "duration": 16, "resolution": "480p"}, 2.72},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "duration": 16, "resolution": "768P"}, 3.84},
		{"ap-muse-video-720-10s", map[string]any{"prompt": "boat"}, 0.6},
		{"rs-seedance-2-0-fast", map[string]any{"prompt": "boat"}, 1.2},
		{"shuikule-seedance-2-5-m", map[string]any{"prompt": "boat"}, 3.9},

		{"minimax-h3-768p", map[string]any{"prompt": "boat", "duration": 15}, 1.2},

		{"ty-seedream-5-5", map[string]any{"prompt": "boat", "resolution": "2K"}, 0.156},

		{"ac-seedream-5-0-pro", map[string]any{"prompt": "boat", "resolution": "2K"}, 0.18},
	} {
		facts := lecAPIHook(t, plugin, "extractUsage", lecAPIContext(tc.model, tc.body))
		defaults, ok := plugin.Meta.DefaultsForModel(tc.model)
		require.True(t, ok)
		require.NotEmpty(t, defaults.BillingExpr)
		cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
		require.NoError(t, err)
		assert.InDelta(t, tc.want, cost, 1e-12)
	}
}

func TestLecAPITokenVideoReservationAndSettlement(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model, resolution string
		seconds           int
		video             bool
		tokens, unitPrice float64
	}{
		{"ap-seedance-2-0-ot", "720p", 4, false, 86400, 29.1456},
		{"ap-seedance-2-0-ot", "720p", 4, true, 172800, 17.7408},
		{"ap-seedance-2-0-ot", "1080p", 4, false, 194400, 32.3136},
		{"ap-seedance-2-0-ot", "1080p", 15, true, 1312200, 19.6416},
		{"ap-seedance-2-0-ot-fast", "720p", 4, false, 86400, 23.4432},
		{"ap-seedance-2-0-ot-fast", "720p", 15, true, 583200, 13.9392},
		{"ap-seedance-2-0-ot-mini", "720p", 4, false, 86400, 14.5728},
		{"ap-seedance-2-0-ot-mini", "720p", 4, true, 172800, 8.8704},
	} {
		t.Run(tc.model+"/"+tc.resolution+"/"+map[bool]string{false: "text", true: "video"}[tc.video], func(t *testing.T) {
			request := map[string]any{"prompt": "boat", "duration": tc.seconds, "resolution": tc.resolution}
			if tc.video {
				request["videos"] = []any{"https://media.example/reference.mp4"}
			}
			ctx := lecAPIContext(tc.model, map[string]any{"metadata": request})
			ctx["model"] = "token-video-alias"
			body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, "lec-"+tc.model, body["model"])
			assert.Equal(t, float64(tc.seconds), body["duration"])
			facts := lecAPIHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, tc.tokens, facts["tokens"])
			assert.Equal(t, tc.resolution, facts["resolution"])
			assert.Equal(t, map[bool]string{false: "none", true: "video"}[tc.video], facts["video_input"])
			schema, _ := plugin.Meta.UsageForModel(tc.model)
			assert.Equal(t, "token", schema["tokens"].Unit)
			defaults, found := plugin.Meta.DefaultsForModel(tc.model)
			require.True(t, found)
			cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.tokens*tc.unitPrice/1000000, cost, 1e-10)
			complete := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{
				"seconds": 4, "resolution": "480p", "usage": map[string]any{"total_tokens": 1000000},
			})
			assert.Equal(t, map[string]any{"tokens": float64(1000000)}, complete, "completion must preserve the frozen resolution and video-input tier")
			maps.Copy(facts, complete)
			cost, _, err = billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.unitPrice, cost, 1e-10)
		})
	}
	ctx := lecAPIContext("ap-seedance-2-5-ot", nil)
	for _, value := range []any{nil, false, "", "1e6", -1, 1.5, 2147483648} {
		facts := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 8, "usage": map[string]any{"total_tokens": value}})
		assert.Empty(t, facts, "missing or malformed tokens must retain the reservation: %v", value)
	}
	for _, tc := range []struct {
		body   map[string]any
		tokens float64
	}{
		{map[string]any{"usage": map[string]any{"total_tokens": 0}}, 0},
		{map[string]any{"usage": map[string]any{"total_tokens": "120000"}}, 120000},
		{map[string]any{"total_tokens": 240000}, 240000},
		{map[string]any{"object": "lecapi.task", "output": map[string]any{"usage": map[string]any{"total_tokens": 360000}}}, 360000},
	} {
		facts := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, tc.body)
		assert.Equal(t, map[string]any{"tokens": tc.tokens}, facts)
	}
	assert.Empty(t, lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "FAILURE"}, map[string]any{"usage": map[string]any{"total_tokens": 1000000}}))
}

func TestLecAPINativeGatewayRepliesRoundTripAsyncImages(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	const modelName = "ty-seedream-5-pro"
	ctx := lecAPIContext(modelName, map[string]any{"prompt": "boat"})
	upstream := map[string]any{"id": "private-id", "status": "completed", "data": []any{map[string]any{"url": "https://media.example/a.png"}}}
	native, err := plugin.Engine.CallPath(t.Context(), "native", []string{"task"}, map[string]any{}, map[string]any{"task_id": "task_public", "status": "SUCCESS", "data": upstream})
	require.NoError(t, err)
	ctx["taskId"] = "task_public"
	result := lecAPIHook(t, plugin, "parseTaskResult", ctx, native)
	assert.Equal(t, "SUCCESS", result["status"])
	parsed := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": native})
	assert.Equal(t, "task_public", parsed["taskId"])
	assert.Equal(t, "SUCCESS", parsed["immediate"].(map[string]any)["status"])
	assert.Equal(t, float64(1), lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, result, native)["image_count"])
	images, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_image", "render"}, map[string]any{}, map[string]any{"data": native})
	require.NoError(t, err)
	encoded, err := common.Marshal(images)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":[{"url":"https://media.example/a.png"}]}`, string(encoded))
	rootURL, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_image", "render"}, map[string]any{}, map[string]any{"data": map[string]any{"id": "lec-image-task", "object": "video", "status": "completed", "url": "https://media.example/async-image.png"}})
	require.NoError(t, err)
	encoded, err = common.Marshal(rootURL)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":[{"url":"https://media.example/async-image.png"}]}`, string(encoded), "the shared async task envelope can contain an image URL at the root")
	failed := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"object": "lecapi.task", "id": "task_failed", "status": "failed", "output": map[string]any{}, "error": map[string]any{"message": "reference rejected"}}})
	require.Contains(t, failed, "immediate")
	assert.Equal(t, "FAILURE", failed["immediate"].(map[string]any)["status"])
	assert.Equal(t, "reference rejected", failed["immediate"].(map[string]any)["reason"])
	pending := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"id": "lec-pending", "status": "queued", "url": "https://media.example/preview.png"}})
	assert.NotContains(t, pending, "immediate", "a preview on a queued job is not a completed output")
}

func TestLecAPIResolvesOwnedOriginsAndPreservesZeroIndex(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := lecAPIContext("sl-midjourney-niji", map[string]any{"action": "hd_upscale", "original_task_id": "task_original", "index": 0})
	_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
	require.ErrorContains(t, err, "completed task")
	ctx["originTasks"] = []any{map[string]any{"taskId": "task_original", "upstreamTaskId": "lec-original", "status": "SUCCESS", "data": map[string]any{"model": "sl-midjourney-niji"}}}
	body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, "lec-original", body["original_task_id"])
	assert.Equal(t, float64(0), body["index"])
	ctx = lecAPIContext("ms-minimax-h3-upscale", map[string]any{"source_task_id": "task_original"})
	ctx["originTasks"] = []any{map[string]any{"taskId": "task_original", "upstreamTaskId": "lec-source", "status": "SUCCESS", "data": map[string]any{"model": "ms-minimax-h3", "resolution": "768P", "seconds": "12"}}}
	assert.Equal(t, float64(12), lecAPIHook(t, plugin, "extractUsage", ctx)["seconds"])
	assert.Equal(t, "lec-source", lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["source_task_id"])

}

func TestLecAPIMaterialUploadsKeepFileRefsAndRejectCountBypasses(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := lecAPIContext("ac-image-2-5-flare", map[string]any{"prompt": "boat"})
	ctx["files"] = []any{map[string]any{"ref": "request_file:image[]", "field": "image[]", "filename": "boat.png", "mimeType": "image/png", "size": 12}}
	body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	images := body["images"].([]any)
	require.Len(t, images, 1)
	assert.Equal(t, "request_file:image[]", images[0].(map[string]any)["__fileRef"])
	assert.Equal(t, "dataUrl", images[0].(map[string]any)["encoding"])
	ctx["files"].([]any)[0].(map[string]any)["size"] = 52428800
	_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
	require.ErrorContains(t, err, "size limit")
}

func TestLecAPIOmitsHostCountForModelsWithoutAnUpstreamCountParameter(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := lecAPIContext("ty-seedream-5-pro", map[string]any{"prompt": "boat", "n": 1})
	assert.NotContains(t, lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any), "n")
	assert.Equal(t, float64(1), lecAPIHook(t, plugin, "extractUsage", ctx)["image_count"])
}

func TestLecAPIRefreshesArtifactsWithoutSendingCredentialsToStorage(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := map[string]any{"model": "aj-wan-3-0", "upstreamModel": "aj-wan-3-0", "baseUrl": "https://lec.example", "apiKey": "secret", "upstreamTaskId": "lec-task", "artifactKey": "video", "data": map[string]any{"url": "https://old.example/expired.mp4"}}
	callCtx := jsplugin.WithFetcher(context.Background(), func(_ context.Context, request jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		assert.Equal(t, "https://lec.example/v1/videos/lec-task", request.URL)
		assert.Equal(t, "Bearer secret", request.Headers["Authorization"])
		return jsplugin.FetchResponse{Status: 200, Body: []byte(`{"id":"lec-task","status":"completed","metadata":{"url":"https://storage.example/video.mp4?signature=abc"}}`)}, nil
	})
	value, err := plugin.Engine.Call(callCtx, "buildContentRequest", ctx)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `{"url":"https://storage.example/video.mp4?signature=abc","method":"GET","credentialless":true}`, string(encoded))
}

func TestLecAPIRecentCompletedArtifactsAvoidDuplicateQueries(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	now := time.Now().Unix()
	signedAt := time.Unix(now-120, 0).UTC().Format("20060102T150405Z")
	for _, tc := range []struct {
		name        string
		completedAt int64
		url         string
		refresh     bool
	}{
		{"just completed", now - 5, "https://storage.example/current.mp4", false},
		{"historical result with valid signed URL", now - 3600, fmt.Sprintf("https://storage.example/current.mp4?Expires=%d&Signature=abc", now+600), false},
		{"historical result with valid relative signed URL", now - 3600, fmt.Sprintf("https://storage.example/current.mp4?X-Amz-Date=%s&X-Amz-Expires=3600&X-Amz-Signature=abc", signedAt), false},
		{"signed URL expired", now - 5, fmt.Sprintf("https://storage.example/expired.mp4?expires=%d&signature=abc", now-60), true},
		{"signed URL near expiry", now - 5, fmt.Sprintf("https://storage.example/soon.mp4?expires=%d&signature=abc", now+10), true},
		{"historical result", 1, "https://storage.example/expired.mp4", true},
		{"missing completion time", 0, "https://storage.example/expired.mp4", true},
		{"missing result", now - 5, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := lecAPIContext("aj-wan-3-0", nil)
			ctx["upstreamTaskId"], ctx["artifactKey"] = "lec-task", "video"
			ctx["data"] = map[string]any{"id": "lec-task", "status": "completed", "completed_at": tc.completedAt, "url": tc.url}
			queries := 0
			callCtx := jsplugin.WithFetcher(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
				queries++
				if !tc.refresh {
					return jsplugin.FetchResponse{Status: 429, Body: []byte(`{"error":"query rate limit"}`)}, nil
				}
				return jsplugin.FetchResponse{Status: 200, Body: []byte(`{"id":"lec-task","status":"completed","url":"https://storage.example/refreshed.mp4"}`)}, nil
			})
			value, err := plugin.Engine.Call(callCtx, "buildContentRequest", ctx)
			require.NoError(t, err)
			encoded, err := common.Marshal(value)
			require.NoError(t, err)
			if tc.refresh {
				assert.Equal(t, 1, queries)
				assert.Contains(t, string(encoded), "refreshed.mp4")
			} else {
				assert.Zero(t, queries)
				var descriptor map[string]any
				require.NoError(t, common.Unmarshal(encoded, &descriptor))
				assert.Equal(t, tc.url, descriptor["url"])
				assert.Equal(t, "GET", descriptor["method"])
				assert.Equal(t, true, descriptor["credentialless"])
			}
		})
	}
}

func TestLecAPIStoredTaskContentUsesPersistedModelMapping(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		name, model, key, path, response string
		channelType                      int
	}{
		{"vendor video", "aj-wan-3-0", "video", "/v1/videos/upstream-id", `{"id":"upstream-id","object":"video","status":"completed","url":"https://cdn.example/fresh.mp4"}`, constant.ChannelTypeTaskPlugin},
		{"gateway image", "ty-seedream-5-pro", "image-1", "/lecapi/v1/images/upstream-id", `{"id":"upstream-id","object":"lecapi.task","status":"SUCCESS","output":{"data":[{"url":"https://cdn.example/fresh.png"}]}}`, constant.ChannelTypeNewAPI},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.path, r.URL.Path)
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.response)
			}))
			defer server.Close()
			adaptor := taskplugin.New(plugin)
			adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: tc.channelType, ChannelBaseUrl: server.URL, ApiKey: "test-key"}})
			task := &model.Task{TaskID: "task-public", Status: model.TaskStatusSuccess,
				Properties:  model.Properties{OriginModelName: "client-alias", UpstreamModelName: tc.model},
				PrivateData: model.TaskPrivateData{UpstreamTaskID: "upstream-id"}, Data: []byte(`{"url":"https://cdn.example/expired"}`)}
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				descriptor, err := adaptor.BuildContentRequest(t.Context(), task, tc.key, relaychannel.TaskArtifactClientRequest{Method: method, Headers: map[string]string{"Range": "bytes=0-1"}})
				require.NoError(t, err)
				require.NotNil(t, descriptor)
				assert.True(t, descriptor.Credentialless)
				assert.Empty(t, descriptor.Headers)
				assert.Contains(t, descriptor.URL, "/fresh.")
			}
		})
	}
}

func TestLecAPIHistoricalSynchronousImageDoesNotQueryInventedTaskID(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	ctx := lecAPIContext("ac-image-2", nil)
	ctx["upstreamTaskId"], ctx["artifactKey"] = "generated-local-id", "image-1"
	ctx["data"] = map[string]any{"data": []any{map[string]any{"url": "https://cdn.example/image.png"}}}
	callCtx := jsplugin.WithFetcher(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		t.Error("historical synchronous image has no upstream query task")
		return jsplugin.FetchResponse{Status: 404}, nil
	})
	value, err := plugin.Engine.Call(callCtx, "buildContentRequest", ctx)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `{"url":"https://cdn.example/image.png","method":"GET","credentialless":true}`, string(encoded))
}

func TestLecAPIContentRefreshRejectsInvalidResults(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"query forbidden", `{"message":"secret https://cdn.example/?signature=secret"}`, 403},
		{"wrong id", `{"id":"wrong","status":"completed","url":"https://cdn.example/video.mp4"}`, 200},
		{"missing id", `{"status":"completed","url":"https://cdn.example/video.mp4"}`, 200},
		{"unfinished", `{"id":"vendor-id","status":"running","url":"https://cdn.example/video.mp4"}`, 200},
		{"no deliverable", `{"id":"vendor-id","status":"completed"}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := lecAPIContext("aj-wan-3-0", nil)
			ctx["upstreamTaskId"], ctx["artifactKey"] = "vendor-id", "video"
			callCtx := jsplugin.WithFetcher(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
				return jsplugin.FetchResponse{Status: tc.status, Body: []byte(tc.body)}, nil
			})
			_, err := plugin.Engine.Call(callCtx, "buildContentRequest", ctx)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "secret")
			assert.NotContains(t, err.Error(), "https://")
		})
	}
}

func TestLecAPIReferenceVideoBillingUsesMeasuredDuration(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	video := yueqiReferenceVideoFixture(t, 3.5)
	longVideo := yueqiReferenceVideoFixture(t, 1000.25)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"), "media probes must not forward channel credentials")
		w.Header().Set("Content-Type", "video/mp4")
		if r.URL.Path == "/broken.mp4" {
			_, _ = w.Write([]byte("invalid video"))
			return
		}
		if r.URL.Path == "/long.mp4" {
			_, _ = w.Write(longVideo)
			return
		}
		_, _ = w.Write(video)
	}))
	defer server.Close()
	fetch := system_setting.GetFetchSetting()
	previous := *fetch
	fetch.EnableSSRFProtection = false
	service.InitHttpClient()
	t.Cleanup(func() { *fetch = previous; service.InitHttpClient() })
	client := service.GetHttpClient()
	previousTransport := client.Transport
	client.Transport = server.Client().Transport
	t.Cleanup(func() { client.Transport = previousTransport })
	for _, tc := range []struct {
		model string
		price float64
	}{
		{"aj-wan-3-0", 0.39}, {"aj-wan-3-0-prime", 0.6},
	} {
		t.Run(tc.model, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
			c.Set("task_request", map[string]any{"prompt": "boat", "metadata": map[string]any{
				"duration": 7, "videos": []any{server.URL + "/first.mp4", server.URL + "/second.mp4"},
			}})
			info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}, OriginModelName: "local-wan-alias", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tc.model, ApiKey: "test-key", ChannelBaseUrl: "https://lec.example"}}
			adaptor := taskplugin.New(plugin)
			adaptor.Init(info)
			facts, err := adaptor.ExtractUsageFactsValidated(c, info)
			require.NoError(t, err)
			assert.Equal(t, float64(7), facts["seconds"])
			assert.Equal(t, float64(8), facts["reference_seconds"], "each 3.5-second clip rounds up separately")
			defaults, ok := plugin.Meta.DefaultsForModel(tc.model)
			require.True(t, ok)
			cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.price*15, cost, 1e-12)
			completion := lecAPIHook(t, plugin, "extractUsageOnComplete", lecAPIContext(tc.model, nil), map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 5})
			maps.Copy(facts, completion)
			cost, _, err = billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.price*13, cost, 1e-12, "completion preserves the frozen reference duration")
			for _, videos := range [][]any{
				{server.URL + "/broken.mp4"},
				{server.URL + "/long.mp4", server.URL + "/long.mp4", server.URL + "/long.mp4", server.URL + "/long.mp4"},
			} {
				c.Set("task_request", map[string]any{"prompt": "boat", "videos": videos})
				_, err := adaptor.ExtractUsageFactsValidated(c, info)
				require.Error(t, err, "unmeasurable or excessive reference duration must fail before reservation")
			}
		})
	}
	t.Run("Video upscaling", func(t *testing.T) {
		const name = "mf-video-upscale"
		defaults, ok := plugin.Meta.DefaultsForModel(name)
		require.True(t, ok)
		for _, tc := range []struct {
			params                map[string]any
			tier, resolution, fps string
			price                 float64
		}{
			{nil, "fast", "720p", "30", 0.01},
			{map[string]any{"tier": "standard", "resolution": "6K", "fps": 60}, "standard", "6k", "60", 0.70},
			{map[string]any{"tier": "generative", "resolution": "4k", "fps": 120}, "generative", "4k", "120", 2.14},
		} {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
			body := map[string]any{"videos": []any{server.URL + "/source.mp4"}}
			maps.Copy(body, tc.params)
			c.Set("task_request", body)
			info := &relaycommon.RelayInfo{
				TaskRelayInfo: &relaycommon.TaskRelayInfo{}, OriginModelName: name,
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: name, ApiKey: "test-key", ChannelBaseUrl: "https://lec.example"},
			}
			adaptor := taskplugin.New(plugin)
			adaptor.Init(info)
			facts, err := adaptor.ExtractUsageFactsValidated(c, info)
			require.NoError(t, err)
			assert.Equal(t, float64(4), facts["seconds"])
			assert.Equal(t, tc.tier, facts["upscale_tier"])
			assert.Equal(t, tc.resolution, facts["resolution"])
			assert.Equal(t, tc.fps, facts["fps"])
			cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.price*4, cost, 1e-12)

			ctx := lecAPIContext(name, body)
			submission := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"id": "task-upscale", "status": "queued"}})
			ctx["state"] = submission["state"]
			completion := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 3.5})
			maps.Copy(facts, completion)
			cost, _, err = billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.price*3.5, cost, 1e-12)
		}
	})
}

type lecAPIFixtureTransport func(*http.Request) (*http.Response, error)

func (transport lecAPIFixtureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestLecAPIPoolOnlyRetriesDefinitiveRouteRejections(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	t.Setenv("SQL_DSN", "")
	previousDB, previousPath, previousMaster := model.DB, common.SQLitePath, common.IsMasterNode
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	common.SQLitePath = filepath.Join(t.TempDir(), "lecapi-route-metrics.db")
	common.IsMasterNode = false
	require.NoError(t, model.InitDB())
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		model.DB, common.SQLitePath, common.IsMasterNode = previousDB, previousPath, previousMaster
		common.SetDatabaseTypes(previousMain, previousLog)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, model.DB.AutoMigrate(&model.PerfMetric{}))
	service.InitHttpClient()
	client := service.GetHttpClient()
	previous := client.Transport
	t.Cleanup(func() { client.Transport = previous })
	for _, tc := range []struct {
		name          string
		status        int
		response      string
		attempts      []string
		prefer        string
		gatewayTaskID string
	}{
		{name: "explicit throttling", status: 429, response: `{"error":{"code":"rate_limit_exceeded"}}`, attempts: []string{"lec-mf-seedance-2-5-30ref", "lec-ty-seedance-2-5-1050-flxd"}},
		{name: "gateway error with upstream task id", status: 502, response: `{"task_id":"recoverable","error":{"message":"bad gateway"}}`, attempts: []string{"lec-mf-seedance-2-5-30ref"}, gatewayTaskID: "recoverable"},
		{name: "gateway timeout without task id", status: 504, response: `{"id":"proxy-trace-id","error":{"message":"timeout"}}`, attempts: []string{"lec-mf-seedance-2-5-30ref"}, gatewayTaskID: "task_public"},
		{name: "ambiguous server failure", status: 503, response: `{"error":{"message":"upstream timed out"}}`, attempts: []string{"lec-mf-seedance-2-5-30ref"}},
		{name: "task id present", status: 429, response: `{"task_id":"accepted","error":{"code":"rate_limit_exceeded"}}`, attempts: []string{"lec-mf-seedance-2-5-30ref"}},
		{name: "business rejection", status: 400, response: `{"error":{"code":"content_policy_violation"}}`, attempts: []string{"lec-mf-seedance-2-5-30ref"}},
		{name: "last compatible route rejection", status: 429, response: `{"error":{"code":"rate_limit_exceeded"}}`, attempts: []string{"lec-ty-seedance-2-5-1050-flxd"}, prefer: "lec-mf-seedance-2-5-30ref"},
		{name: "higher success rate first", status: 202, response: `{"id":"accepted","status":"queued"}`, attempts: []string{"lec-ty-seedance-2-5-1050-flxd"}, prefer: "lec-ty-seedance-2-5-1050-flxd"},
		{name: "unmatched route cannot override resolution", status: 202, response: `{"id":"accepted","status":"queued"}`, attempts: []string{"lec-ty-seedance-2-5-1050-flxd"}, prefer: "lec-mf-seedance-2-5-30ref"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Only the ranking cases load counters; other cases keep the same order.
			channelID := 0
			if tc.prefer != "" {
				channelID = 31
				if tc.prefer == "lec-mf-seedance-2-5-30ref" {
					channelID++
				}
				require.NoError(t, model.RecordLecAPIRouteOutcome(channelID, tc.prefer, true))
			}
			var attempts []string
			client.Transport = lecAPIFixtureTransport(func(request *http.Request) (*http.Response, error) {
				var body map[string]any
				require.NoError(t, common.DecodeJson(request.Body, &body))
				attempts = append(attempts, body["model"].(string))
				status, response := tc.status, tc.response
				if len(attempts) > 1 {
					status, response = 202, `{"id":"accepted","status":"queued"}`
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(response)), Request: request}, nil
			})
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
			params := map[string]any{"prompt": "boat", "duration": 5, "images": []any{"https://media.example/ref.png"}}
			if tc.prefer == "lec-mf-seedance-2-5-30ref" {
				params["resolution"] = "480p"
			}
			ctx.Set("task_request", params)
			info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"}, OriginModelName: "seedance-2-5-02",
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "seedance-2-5-02", ChannelId: channelID, ApiKey: "stub", ChannelBaseUrl: "https://lec.example"}}
			adaptor := taskplugin.New(plugin)
			adaptor.Init(info)
			body, err := adaptor.BuildRequestBody(ctx, info)
			require.NoError(t, err)
			response, err := adaptor.DoRequest(ctx, info, body)
			require.NoError(t, err)
			defer response.Body.Close()
			assert.Equal(t, tc.attempts, attempts)
			if response.StatusCode == http.StatusAccepted {
				assert.Equal(t, http.StatusAccepted, response.StatusCode)
				parsed, taskErr := adaptor.ParseResponse(ctx, response, info)
				require.Nil(t, taskErr)
				var pluginState map[string]any
				require.NoError(t, common.Unmarshal(parsed.PluginState, &pluginState))
				assert.Equal(t, tc.attempts[len(tc.attempts)-1], pluginState["lecapiSubmitRoute"])
				if tc.gatewayTaskID != "" {
					assert.Nil(t, parsed.Immediate)
					assert.Equal(t, tc.gatewayTaskID, parsed.UpstreamTaskID)
					assert.Equal(t, tc.gatewayTaskID == "task_public", parsed.LecAPIPoll.CreateGatewayError)
					assert.Greater(t, parsed.LecAPIPoll.RecheckAt, time.Now().UnixMilli())
					assert.NotEmpty(t, parsed.LecAPIRetryPayload)
				}
				if tc.name == "higher success rate first" {
					assert.NotEmpty(t, pluginState["lecapiRetryCandidates"], "compatible fallback routes must survive the submission barrier")
				}
			} else if tc.name == "last compatible route rejection" {
				parsed, taskErr := adaptor.ParseResponse(ctx, response, info)
				require.Nil(t, taskErr)
				require.NotNil(t, parsed.Immediate)
				var metric model.PerfMetric
				require.NoError(t, model.DB.Where("model_name = ? AND `group` = ?", model.LecAPIRouteMetricPrefix+"lec-ty-seedance-2-5-1050-flxd", "32").First(&metric).Error)
				assert.Equal(t, int64(1), metric.RequestCount, "submit refusal and immediate failure must not double count")
				assert.Zero(t, metric.SuccessCount)
			} else {
				data, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tc.response, string(data))
			}
		})
	}
}

func TestLecAPIPoolRetriesTerminalPollFailureOnRemainingRoute(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	t.Setenv("SQL_DSN", "")
	previousDB, previousPath, previousMaster := model.DB, common.SQLitePath, common.IsMasterNode
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	common.SQLitePath = filepath.Join(t.TempDir(), "lecapi-delayed-retry.db")
	common.IsMasterNode = false
	require.NoError(t, model.InitDB())
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		model.DB, common.SQLitePath, common.IsMasterNode = previousDB, previousPath, previousMaster
		common.SetDatabaseTypes(previousMain, previousLog)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, model.DB.AutoMigrate(&model.PerfMetric{}, &model.Task{}))
	service.InitHttpClient()
	client := service.GetHttpClient()
	previousTransport := client.Transport
	t.Cleanup(func() { client.Transport = previousTransport })

	const channelID = 79
	for _, route := range []string{"lec-mm-seedance-2-5", "lec-mj-seedance-2-5-bd-720p", "lec-mm-seedance-2-5-full"} {
		require.NoError(t, model.DB.Where("model_name = ? AND `group` = ?", model.LecAPIRouteMetricPrefix+route, "79").Delete(&model.PerfMetric{}).Error)
	}
	require.NoError(t, model.RecordLecAPIRouteOutcome(channelID, "lec-mj-seedance-2-5-bd-720p", false))
	require.NoError(t, model.RecordLecAPIRouteOutcome(channelID, "lec-mm-seedance-2-5-full", true))
	largeImage := "data:image/png;base64," + strings.Repeat("AAAA", 400_000)
	retainedBodies, err := common.Marshal(map[string]any{
		"shared": map[string]any{"prompt": "boat", "duration": 5, "images": []string{largeImage}},
		"routes": map[string]any{
			"lec-mj-seedance-2-5-bd-720p": map[string]any{"model": "lec-mj-seedance-2-5-bd-720p", "resolution": "720p"},
			"lec-mm-seedance-2-5-full":    map[string]any{"model": "lec-mm-seedance-2-5-full", "resolution": "720p"},
		},
	})
	require.NoError(t, err)
	var requestedRoutes []string
	client.Transport = lecAPIFixtureTransport(func(request *http.Request) (*http.Response, error) {
		var body map[string]any
		require.NoError(t, common.DecodeJson(request.Body, &body))
		route, _ := body["model"].(string)
		requestedRoutes = append(requestedRoutes, route)
		image, ok := body["images"].([]any)
		require.True(t, ok)
		require.Len(t, image, 1)
		assert.True(t, image[0] == largeImage, "retained reference must be replayed byte for byte")
		assert.Equal(t, "720p", body["resolution"])
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://lec.example/v1/videos", request.URL.String())
		assert.Equal(t, "Bearer stub", request.Header.Get("Authorization"))
		if route == "lec-mm-seedance-2-5-full" {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":{"code":"rate_limit_exceeded"}}`)), Request: request}, nil
		}
		return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"replacement-task","status":"queued"}`)), Request: request}, nil
	})

	info := &relaycommon.RelayInfo{OriginModelName: "seedance-2-5-01", ChannelMeta: &relaycommon.ChannelMeta{
		ChannelId: channelID, ApiKey: "stub", ChannelBaseUrl: "https://lec.example",
	}}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(info)
	task := &model.Task{
		TaskID: "task_public", Platform: "lecapi", Status: model.TaskStatusInProgress, ChannelId: channelID, SubmitTime: time.Now().Unix(),
		Properties: model.Properties{OriginModelName: "seedance-2-5-01", UpstreamModelName: "seedance-2-5-01"},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID:     "failed-task",
			LecAPIRetryPayload: retainedBodies,
			Execution:          &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{Key: "lecapi"}},
			PluginState:        []byte(`{"kind":"video","queryable":true,"lecapiSubmitRoute":"lec-mm-seedance-2-5","lecapiRetryURL":"https://lec.example/v1/videos","lecapiRetryCandidates":[{"route":"lec-mj-seedance-2-5-bd-720p","body":{"model":"lec-mj-seedance-2-5-bd-720p"}},{"route":"lec-mm-seedance-2-5-full","body":{"model":"lec-mm-seedance-2-5-full"}}]}`),
		},
	}
	require.NoError(t, model.DB.Create(task).Error)
	// A fresh poll reads the retained material after the original request is gone.
	require.NoError(t, model.DB.First(task, task.ID).Error)
	ready, alreadyInFlight := adaptor.PrepareTaskFailureRetry(task, false)
	require.True(t, ready)
	assert.False(t, alreadyInFlight)
	task.Status = model.TaskStatusSubmitted
	claimed, err := task.UpdateWithStatus(model.TaskStatusInProgress)
	require.NoError(t, err)
	require.True(t, claimed)
	result := relaycommon.FailTaskInfo("upstream generation failed")
	retried, err := adaptor.RetryTaskAfterFailure(context.Background(), task, result)
	require.NoError(t, err)
	require.True(t, retried)
	assert.Equal(t, []string{"lec-mm-seedance-2-5-full", "lec-mj-seedance-2-5-bd-720p"}, requestedRoutes)
	assert.Equal(t, "replacement-task", task.GetUpstreamTaskID())
	assert.EqualValues(t, model.TaskStatusSubmitted, task.Status)

	var state map[string]any
	require.NoError(t, common.Unmarshal(task.PrivateData.PluginState, &state))
	assert.Equal(t, "lec-mj-seedance-2-5-bd-720p", state["lecapiSubmitRoute"])
	assert.Empty(t, state["lecapiRetryCandidates"])
	var metric model.PerfMetric
	require.NoError(t, model.DB.Where("model_name = ? AND `group` = ?", model.LecAPIRouteMetricPrefix+"lec-mm-seedance-2-5", "79").First(&metric).Error)
	assert.Equal(t, int64(1), metric.RequestCount)
	assert.Zero(t, metric.SuccessCount)
	var rejected model.PerfMetric
	require.NoError(t, model.DB.Where("model_name = ? AND `group` = ?", model.LecAPIRouteMetricPrefix+"lec-mm-seedance-2-5-full", "79").First(&rejected).Error)
	assert.Equal(t, int64(2), rejected.RequestCount)
	assert.Equal(t, int64(1), rejected.SuccessCount)

	t.Run("restart resumes only a route that was never dispatched", func(t *testing.T) {
		retryAdaptor := taskplugin.New(plugin)
		retryAdaptor.Init(info)
		ambiguousTask := &model.Task{
			TaskID: "task_ambiguous", Platform: "lecapi", Status: model.TaskStatusInProgress, ChannelId: channelID,
			Properties: task.Properties,
			PrivateData: model.TaskPrivateData{
				UpstreamTaskID: "failed-task",
				Execution:      task.PrivateData.Execution,
				PluginState:    []byte(`{"kind":"video","queryable":true,"lecapiSubmitRoute":"lec-mm-seedance-2-5","lecapiRetryURL":"https://lec.example/v1/videos","lecapiRetryCandidates":[{"route":"lec-mj-seedance-2-5-bd-720p","body":{"model":"lec-mj-seedance-2-5-bd-720p","prompt":"boat","resolution":"720p","duration":5}},{"route":"lec-mm-seedance-2-5-full","body":{"model":"lec-mm-seedance-2-5-full","prompt":"boat","resolution":"720p","duration":5}}]}`),
			},
		}
		require.NoError(t, model.DB.Create(ambiguousTask).Error)
		ready, alreadyInFlight := retryAdaptor.PrepareTaskFailureRetry(ambiguousTask, false)
		require.True(t, ready)
		assert.False(t, alreadyInFlight)
		ambiguousTask.Status = model.TaskStatusSubmitted
		ambiguousTask.PrivateData.LecAPIPoll = model.LecAPIPollState{
			Revision:      ambiguousTask.PrivateData.LecAPIPoll.Revision,
			RetryInFlight: true, RetryUntil: time.Now().Add(-time.Minute).UnixMilli(),
		}
		claimed, err := ambiguousTask.UpdateWithStatus(model.TaskStatusInProgress)
		require.NoError(t, err)
		require.True(t, claimed)

		var routes []string
		client.Transport = lecAPIFixtureTransport(func(req *http.Request) (*http.Response, error) {
			var requestBody map[string]any
			require.NoError(t, common.DecodeJson(req.Body, &requestBody))
			route, ok := requestBody["model"].(string)
			require.True(t, ok)
			routes = append(routes, route)
			var dispatched model.Task
			require.NoError(t, model.DB.First(&dispatched, ambiguousTask.ID).Error)
			require.True(t, dispatched.PrivateData.LecAPIPoll.RetryDispatched, "dispatch must be durable before POST")
			deadline, ok := req.Context().Deadline()
			require.True(t, ok)
			assert.GreaterOrEqual(t, dispatched.PrivateData.LecAPIPoll.RetryUntil, deadline.Add(model.LecAPIGatewayRecheckDelay).UnixMilli())
			if len(routes) == 1 {
				return nil, fmt.Errorf("connection reset after write")
			}
			return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"replacement-after-restart","status":"queued"}`)), Request: req}, nil
		})
		for range 2 {
			var reloaded model.Task
			require.NoError(t, model.DB.First(&reloaded, ambiguousTask.ID).Error)
			ambiguousTask = &reloaded
			retryAdaptor = taskplugin.New(plugin)
			retryAdaptor.Init(info)
			ready, alreadyInFlight := retryAdaptor.PrepareTaskFailureRetry(ambiguousTask, true)
			require.True(t, ready)
			require.False(t, alreadyInFlight)
			ambiguousTask.PrivateData.LecAPIPoll = model.LecAPIPollState{
				Revision:      ambiguousTask.PrivateData.LecAPIPoll.Revision,
				RetryInFlight: true, RetryUntil: time.Now().Add(time.Minute).UnixMilli(),
			}
			claimed, err := ambiguousTask.UpdateWithStatus(model.TaskStatusSubmitted)
			require.NoError(t, err)
			require.True(t, claimed)

			result := relaycommon.FailTaskInfo("upstream generation failed")
			retried, err := retryAdaptor.RetryTaskAfterFailure(context.Background(), ambiguousTask, result)
			require.True(t, retried)
			if len(routes) == 1 {
				require.Error(t, err)
				ready, alreadyInFlight := retryAdaptor.PrepareTaskFailureRetry(ambiguousTask, false)
				assert.False(t, ready)
				assert.True(t, alreadyInFlight)
				ambiguousTask.PrivateData.LecAPIPoll.RetryUntil = time.Now().Add(-time.Minute).UnixMilli()
			} else {
				require.NoError(t, err)
			}
			updated, err := ambiguousTask.UpdateWithStatus(model.TaskStatusSubmitted)
			require.NoError(t, err)
			require.True(t, updated)
		}
		assert.Equal(t, []string{"lec-mm-seedance-2-5-full", "lec-mj-seedance-2-5-bd-720p"}, routes)
		assert.Equal(t, "task_ambiguous", ambiguousTask.TaskID)
		assert.Equal(t, "replacement-after-restart", ambiguousTask.GetUpstreamTaskID())
		assert.False(t, ambiguousTask.PrivateData.LecAPIPoll.RetryInFlight)
	})
}

func TestLecAPIHostReservesAndSettlesDefaultPricesWithoutSavedConfiguration(t *testing.T) {
	plugin, found := jsplugin.DefaultRegistry.Get("lecapi")
	require.True(t, found)
	source, err := builtinplugins.Source("lecapi")
	require.NoError(t, err)
	require.NoError(t, i18n.Init())
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error { saved[key] = value; return nil }))
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(saved)) })
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`, "billing_setting.billing_expr": `{}`, "billing_setting.plugin_billing_expr": `{}`,
	}))
	service.InitHttpClient()
	client := service.GetHttpClient()
	previousTransport := client.Transport
	t.Cleanup(func() { client.Transport = previousTransport })
	for _, tc := range []struct {
		name      string
		update    bool
		current   bool
		disable   bool
		free      bool
		unitPrice float64
	}{
		{name: "active version", unitPrice: 0.06},
		{name: "updated after pinning", update: true, unitPrice: 0.06},
		{name: "new request after update", update: true, current: true, unitPrice: 0.12},
		{name: "disabled after pinning", disable: true, unitPrice: 0.06},
		{name: "explicit free override after update", update: true, free: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pinned := jsplugin.PinnedPlugin{Plugin: plugin, Generation: jsplugin.DefaultRegistry.Generation()}
			if tc.update {
				updated := strings.Replace(source, "const PRICE_MARKUP = 1.2;", "const PRICE_MARKUP = 2.4;", 1)
				require.NotEqual(t, source, updated)
				newPlugin, err := jsplugin.DefaultRegistry.Register(updated, jsplugin.Options{Key: "lecapi"})
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, jsplugin.DefaultRegistry.Unregister("lecapi")) })
				if tc.current {
					pinned = jsplugin.PinnedPlugin{Plugin: newPlugin, Generation: jsplugin.DefaultRegistry.Generation()}
				}
			}
			if tc.disable {
				enabled := jsplugin.DefaultRegistry.Enabled()
				jsplugin.DefaultRegistry.SetEnabled(false)
				t.Cleanup(func() { jsplugin.DefaultRegistry.SetEnabled(enabled) })
			}
			if tc.free {
				require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
					"billing_setting.plugin_billing_expr": `{"lecapi::local-lecapi-alias":"tier(\"free\", u(\"image_count\") * 0)"}`,
				}))
				t.Cleanup(func() {
					require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"billing_setting.plugin_billing_expr": `{}`}))
				})
			}
			calls := 0
			// The fixture transport never opens a socket or calls a vendor endpoint.
			client.Transport = lecAPIFixtureTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				assert.Equal(t, "https://lecapi-fixture.invalid/v1/images/generations", request.URL.String())
				assert.Equal(t, "Bearer test-key", request.Header.Get("Authorization"))
				var body map[string]any
				require.NoError(t, common.DecodeJson(request.Body, &body))
				assert.Equal(t, "lec-tinysnow-image-2", body["model"])
				assert.Equal(t, float64(3), body["n"])
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"data":[{"url":"https://media.example/a.png"},{"url":"https://media.example/b.png"},{"revised_prompt":"two pictures"}]}`)), Request: request}, nil
			})
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			c.Set("task_request", map[string]any{"model": "local-lecapi-alias", "prompt": "boat", "n": 3})
			c.Set("model_mapping", `{"local-lecapi-alias":"tinysnow-image-2"}`)
			c.Set(jsplugin.ContextKeyPinnedPlugin, pinned)
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeTaskPlugin)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://lecapi-fixture.invalid")
			common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
			reservation := &xogpuReservation{}
			info := &relaycommon.RelayInfo{OriginModelName: "local-lecapi-alias", UserGroup: "default", UsingGroup: "default", Billing: reservation, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			result, taskErr := relay.RelayTaskSubmit(c, info)
			require.Nil(t, taskErr, "%+v", taskErr)
			require.NotNil(t, result)
			assert.Equal(t, 1, calls)
			assert.Equal(t, common.QuotaRound(tc.unitPrice*3*common.QuotaPerUnit), reservation.held)
			assert.Equal(t, common.QuotaRound(tc.unitPrice*2*common.QuotaPerUnit), result.Quota)
			assert.Equal(t, float64(2), info.TieredBillingSnapshot.UsageFacts["image_count"])
			assert.Equal(t, "local-lecapi-alias", info.OriginModelName)
		})
	}
	t.Run("merged route keeps the advertised price", func(t *testing.T) {
		const entry = "seedance-2-5-02"
		reservation := &xogpuReservation{}
		want := common.QuotaRound(4.5 * common.QuotaPerUnit)
		calls := 0
		client.Transport = lecAPIFixtureTransport(func(request *http.Request) (*http.Response, error) {
			calls++
			assert.Equal(t, "https://lecapi-fixture.invalid/v1/videos", request.URL.String())
			assert.Equal(t, want, reservation.held)
			var body map[string]any
			require.NoError(t, common.DecodeJson(request.Body, &body))
			assert.Equal(t, "lec-ty-seedance-2-5-1050-flxd", body["model"])
			assert.Equal(t, "480p", body["resolution"])
			return &http.Response{
				StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
				Body:    io.NopCloser(strings.NewReader(`{"id":"vendor-video","status":"completed","url":"https://media.example/video.mp4","seconds":5}`)),
				Request: request,
			}, nil
		})
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
		c.Set("task_request", map[string]any{"model": entry, "prompt": "boat", "resolution": "480p", "images": []any{"https://media.example/ref.png"}})
		c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{Plugin: plugin, Generation: jsplugin.DefaultRegistry.Generation()})
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeTaskPlugin)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://lecapi-fixture.invalid")
		common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
		info := &relaycommon.RelayInfo{OriginModelName: entry, UserGroup: "default", UsingGroup: "default", Billing: reservation, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
		result, taskErr := relay.RelayTaskSubmit(c, info)
		require.Nil(t, taskErr, "%+v", taskErr)
		require.NotNil(t, result)
		assert.Equal(t, 1, calls)
		assert.Equal(t, want, result.Quota)
		assert.Equal(t, "480p", info.TieredBillingSnapshot.UsageFacts["resolution"])
	})
	t.Run("AP token reservation and completion quota", func(t *testing.T) {
		reservation := &xogpuReservation{}
		calls := 0
		client.Transport = lecAPIFixtureTransport(func(request *http.Request) (*http.Response, error) {
			calls++
			assert.Equal(t, "https://lecapi-fixture.invalid/v1/videos", request.URL.String())
			assert.Equal(t, common.QuotaRound(2.51817984*common.QuotaPerUnit), reservation.held, "estimated tokens must be reserved before the upstream call")
			var body map[string]any
			require.NoError(t, common.DecodeJson(request.Body, &body))
			assert.Equal(t, "lec-ap-seedance-2-0-ot", body["model"])
			assert.Equal(t, float64(4), body["duration"])
			return &http.Response{
				StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
				Body:    io.NopCloser(strings.NewReader(`{"id":"vendor-video","status":"completed","url":"https://media.example/video.mp4","usage":{"total_tokens":100000}}`)),
				Request: request,
			}, nil
		})
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
		c.Set("task_request", map[string]any{"model": "local-lecapi-token-alias", "prompt": "boat", "duration": 4})
		c.Set("model_mapping", `{"local-lecapi-token-alias":"ap-seedance-2-0-ot"}`)
		c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{Plugin: plugin, Generation: jsplugin.DefaultRegistry.Generation()})
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeTaskPlugin)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://lecapi-fixture.invalid")
		common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
		info := &relaycommon.RelayInfo{OriginModelName: "local-lecapi-token-alias", UserGroup: "default", UsingGroup: "default", Billing: reservation, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
		result, taskErr := relay.RelayTaskSubmit(c, info)
		require.Nil(t, taskErr, "%+v", taskErr)
		require.NotNil(t, result)
		assert.Equal(t, 1, calls)
		assert.Equal(t, common.QuotaRound(2.91456*common.QuotaPerUnit), result.Quota)
		assert.Greater(t, result.Quota, reservation.held, "the completion quota carries the increase to the host submission barrier")
		assert.Equal(t, float64(100000), info.TieredBillingSnapshot.UsageFacts["tokens"])
	})
}
