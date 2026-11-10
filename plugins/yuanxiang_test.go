package plugins_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	taskplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadYuanxiangPlugin(t *testing.T) *jsplugin.LoadedPlugin {
	t.Helper()
	source, err := builtinplugins.Source("yuanxiang")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "yuanxiang"})
	require.NoError(t, err)
	return plugin
}

func yuanxiangContext(model string, body map[string]any) map[string]any {
	return map[string]any{"model": model, "upstreamModel": model, "requestBody": body, "baseUrl": "https://yuanxiang.example/", "apiKey": "test-key", "publicTaskId": "task_public"}
}

func yuanxiangHook(t *testing.T, plugin *jsplugin.LoadedPlugin, hook string, args ...any) map[string]any {
	t.Helper()
	value, err := plugin.Engine.Call(t.Context(), hook, args...)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, common.Unmarshal(encoded, &result))
	return result
}

func TestYuanxiangCanvasMappingAndBilling(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	ctx := yuanxiangContext("Wan 3.0 ·  官方直连 · 特价按秒-YX", map[string]any{"model": "Wan 3.0 ·  官方直连 · 特价按秒-YX", "prompt": "A paper boat", "resolution_name": "720p", "seconds": "5", "size": "16:9"})
	submit := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "https://yuanxiang.example/v1/videos", submit["url"])
	assert.NotContains(t, submit, "model", "the pinned billing identity stays suffixed")
	body := submit["body"].(map[string]any)
	assert.Equal(t, "Wan 3.0 ·  官方直连 · 特价按秒", body["model"])
	assert.Equal(t, float64(5), body["duration"])
	assert.Equal(t, "720p", body["resolution"])
	assert.Equal(t, "16:9", body["ratio"])
	assert.Equal(t, false, body["wait"])
	for _, field := range []string{"seconds", "size", "resolution_name"} {
		assert.NotContains(t, body, field)
	}
	facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
	assert.Equal(t, float64(5), facts["seconds"])
	assert.Equal(t, "720p", facts["resolution"])
	assert.Equal(t, float64(1), facts["requests"])
	assert.Equal(t, float64(0), facts["reference_seconds"])
	cost, _, err := billingexpr.RunExprWithRequest(`u("resolution") == "720p" ? tier("hd", u("seconds") * 0.2 + u("reference_seconds") * 0.1) : tier("base", u("requests") * 1)`, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
	require.NoError(t, err)
	assert.InDelta(t, 1, cost, 1e-12)
	ctx["usagePurpose"] = "billing_ratios"
	assert.Equal(t, map[string]any{"requests": float64(1)}, yuanxiangHook(t, plugin, "extractUsage", ctx))
	delete(ctx, "usagePurpose")
	ctx["upstreamModel"] = "MiniMax H3 · 特价按秒"
	ctx["model"] = "canvas-custom-YX"
	assert.Equal(t, "MiniMax H3 · 特价按秒", yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"])
	ctx["upstream"] = map[string]any{"kind": "new_api"}
	gateway := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "https://yuanxiang.example/yuanxiang/v1/videos", gateway["url"])
	assert.Equal(t, "MiniMax H3 · 特价按秒-YX", gateway["body"].(map[string]any)["model"])
}

func TestYuanxiangModelFamiliesAndImageProtocol(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	videoModels := []string{"SD 2.5 · 官方直连（满血版）按秒", "SD 2.0 满血版 按秒", "SD 2.5 · 特价按次", "Sd 2.0 · 特价按次 · 漫剧短剧专线", "SD 2.0 · 特价按次 · 电商专线", "Muse · 特价按次 ·【原生】", "MiniMax H3 · 特价按秒", "MiniMax H3 · 特价按次", "Wan 3.0 ·  官方直连 · 特价按秒"}
	for _, model := range videoModels {
		t.Run(model, func(t *testing.T) {
			ctx := yuanxiangContext(model+"-YX", map[string]any{"prompt": "A boat"})
			assert.Equal(t, model, yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"])
			schema, _ := plugin.Meta.UsageForModels(model + "-YX")
			assert.Equal(t, []string{"duration_tier", "reference_seconds", "requests", "resolution", "seconds", "tokens", "video_input"}, slices.Sorted(maps.Keys(schema)))
		})
	}
	for _, model := range []string{"GT Image 2.5"} {
		t.Run(model, func(t *testing.T) {
			ctx := yuanxiangContext(model+"-YX", map[string]any{"prompt": "A boat", "n": 1, "size": "16:9", "resolution_name": "4K", "quality": "high", "options": map[string]any{"web_search": false, "version": "flare"}})
			submit := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)
			assert.Equal(t, "https://yuanxiang.example/v1/images/generations", submit["url"])
			body := submit["body"].(map[string]any)
			assert.Equal(t, model, body["model"])
			assert.Equal(t, "16:9", body["ratio"])
			assert.Equal(t, false, body["options"].(map[string]any)["web_search"])
			facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, float64(1), facts["image_count"])
			assert.Equal(t, "4k", facts["resolution"])
			assert.Equal(t, "high", body["quality"])
			assert.Equal(t, "low", facts["quality"], "retired conditions retain their first declared value for stored expressions")
			schema, _ := plugin.Meta.UsageForModels(model + "-YX")
			assert.Equal(t, []string{"image_count", "resolution"}, slices.Sorted(maps.Keys(schema)))
			result, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_image", "render"}, map[string]any{}, map[string]any{"status": "SUCCESS", "data": map[string]any{"data": []any{map[string]any{"url": "https://cdn.example/image.png"}}}})
			require.NoError(t, err)
			encoded, err := common.Marshal(result)
			require.NoError(t, err)
			assert.JSONEq(t, `{"data":[{"url":"https://cdn.example/image.png"}]}`, string(encoded))
		})
	}
	for _, model := range plugin.Meta.Models {
		assert.True(t, strings.HasSuffix(model, "-YX"), model)
	}
	assert.NotContains(t, plugin.Meta.Models, "wan3.0-video-nsfw-YX", "the user's example is not an upstream model")
	assert.Len(t, plugin.Meta.Models, 10)
}

func TestYuanxiangResolutionBranchesRemainIndependent(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	for _, tc := range []struct {
		model, quantity, resolution, fallback, quality string
	}{
		{"Wan 3.0 ·  官方直连 · 特价按秒-YX", "requests", "720p", "1080p", "default"},
		{"GT Image 2.5-YX", "image_count", "4k", "2k", "low"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			schema, _ := plugin.Meta.UsageForModels(tc.model)
			assert.True(t, schema["resolution"].AllowCustom)
			expression := `u("resolution") == "` + tc.resolution + `" ? tier("resolution", u("` + tc.quantity + `") * 0.2) : tier("fallback", u("` + tc.quantity + `") * 0.1)`
			require.NoError(t, billing_setting.SmokeTestTaskExpr(expression, schema))
			for _, field := range []string{"quality", "version", "channel", "has_reference_video"} {
				assert.NotContains(t, schema, field)
				require.Error(t, billing_setting.SmokeTestTaskExpr(`u("`+field+`") == "custom" ? tier("custom", 1) : tier("base", 0.1)`, schema))
			}
			for _, tier := range []struct {
				resolution, name string
				cost             float64
			}{
				{tc.resolution, "resolution", 0.2},
				{tc.fallback, "fallback", 0.1},
			} {
				for _, customOptions := range []bool{false, true} {
					body := map[string]any{"prompt": "boat", "resolution_name": strings.ToUpper(tier.resolution)}
					if customOptions {
						body["quality"], body["version"], body["channel"] = "high", "custom", "alternate"
						if tc.quantity == "requests" {
							body["seconds"], body["video_b64s"] = 15, []string{"unmeasured-reference"}
						}
					}
					ctx := yuanxiangContext(tc.model, body)
					facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
					cost, trace, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
					require.NoError(t, err)
					assert.InDelta(t, tier.cost, cost, 1e-12)
					assert.Equal(t, tier.name, trace.MatchedTier)
					assert.Equal(t, tc.quality, facts["quality"])
					assert.Equal(t, "default", facts["version"])
					assert.Equal(t, "auto", facts["channel"])
					if tc.quantity == "requests" {
						assert.Equal(t, fmt.Sprint(facts["seconds"]), facts["duration_tier"])
						assert.Equal(t, false, facts["has_reference_video"])
					}
				}
			}
		})
	}
}

func TestYuanxiangRejectsConflictingAndUnboundedParameters(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	for _, body := range []map[string]any{
		{"seconds": -1}, {"seconds": 0}, {"seconds": "1e99"}, {"seconds": 61}, {"seconds": true},
		{"seconds": 5, "duration": 6}, {"seconds": 5, "metadata": map[string]any{"duration": 6}},
		{"resolution_name": "720p", "parameters": map[string]any{"resolution": "1080p"}},
		{"size": "16:9", "ratio": "9:16"}, {"size": "0:9"},
		{"options": map[string]any{"duration": -1}}, {"extra": map[string]any{"n": 129}},
		{"metadata": map[string]any{"max_tokens": 1e99}}, {"n": 2}, {"metadata": "bad"},
		{"version": "bad\x00value"}, {"channel": "bad\x7fvalue"},
	} {
		body["prompt"] = "boat"
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, yuanxiangContext("Wan 3.0 ·  官方直连 · 特价按秒-YX", body))
			require.Error(t, err, "%s: %v", hook, body)
		}
	}
	ctx := yuanxiangContext("Wan 3.0 ·  官方直连 · 特价按秒-YX", map[string]any{"prompt": "boat", "metadata": map[string]any{"seconds": "8", "size": "9:16", "resolution_name": "1080P", "options": map[string]any{"version": "Mini", "generate_audio": false}}})
	body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, float64(8), body["duration"])
	assert.Equal(t, "9:16", body["ratio"])
	assert.Equal(t, "1080p", body["resolution"])
	assert.Equal(t, "Mini", body["options"].(map[string]any)["version"])
}

func TestYuanxiangImageSizeMapping(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	for _, tc := range []struct {
		name       string
		fields     map[string]any
		ratio      string
		resolution string
	}{
		{"reported request", map[string]any{"size": "1824x1024", "resolution": "1k", "n": 1, "output_format": "png"}, "16:9", "1k"},
		{"portrait", map[string]any{"size": "1024x1824", "resolution": "2k"}, "9:16", "2k"},
		{"square default resolution", map[string]any{"size": "1024x1024"}, "1:1", "1k"},
		{"landscape", map[string]any{"size": "1536x1024", "resolution_name": "4K"}, "3:2", "4k"},
		{"uppercase separator", map[string]any{"size": " 1824X1024 "}, "16:9", "1k"},
		{"matching explicit ratio", map[string]any{"size": "1824x1024", "ratio": "16:9"}, "16:9", "1k"},
		{"explicit ratio takes precedence", map[string]any{"size": "1824x1024", "ratio": "9:16"}, "9:16", "1k"},
		{"nested explicit ratio takes precedence", map[string]any{"size": "1824x1024", "metadata": map[string]any{"aspect_ratio": "9:16"}}, "9:16", "1k"},
		{"nested dimensions", map[string]any{"resolution": "2k", "metadata": map[string]any{"parameters": map[string]any{"size": "1024x1824"}}}, "9:16", "2k"},
		{"resolution label", map[string]any{"size": "2K"}, "", "2k"},
		{"documented request", map[string]any{"ratio": "1:1"}, "1:1", "1k"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.fields["prompt"] = "A corgi wearing an astronaut helmet"
			tc.fields["quality"] = "high"
			ctx := yuanxiangContext("GT Image 2.5-YX", tc.fields)
			body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, tc.resolution, body["resolution"])
			assert.Equal(t, "high", body["quality"])
			if tc.ratio == "" {
				assert.NotContains(t, body, "ratio")
			} else {
				assert.Equal(t, tc.ratio, body["ratio"])
			}
			assert.NotContains(t, body, "size")
			facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, tc.resolution, facts["resolution"])
			assert.Equal(t, "low", facts["quality"], "retired quality conditions retain their value for stored expressions")
			assert.Equal(t, float64(1), facts["image_count"])
		})
	}
	for _, fields := range []map[string]any{
		{"size": "0x1024"}, {"size": "1024x0"}, {"size": "-1024x1024"},
		{"size": "1024.5x1024"}, {"size": "99999999999999999999x1024"},
		{"size": "1824x1024", "metadata": map[string]any{"size": "1024x1824"}},
		{"size": "1824x1024", "resolution": "1k", "metadata": map[string]any{"resolution_name": "4k"}},
	} {
		fields["prompt"] = "A corgi wearing an astronaut helmet"
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, yuanxiangContext("GT Image 2.5-YX", fields))
			require.Error(t, err, "%s: %v", hook, fields)
		}
	}
}

func TestYuanxiangReferenceBillingUsesMediaAndSkipsDisabledMeasurements(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	video := base64.StdEncoding.EncodeToString(yueqiReferenceVideoFixture(t, 3.5))
	ctx := yuanxiangContext("MiniMax H3 · 特价按秒-YX", map[string]any{"prompt": "boat", "video_b64s": []string{video}})
	ctx["usageKeys"] = []string{"reference_seconds"}
	assert.Equal(t, 3.5, yuanxiangHook(t, plugin, "extractUsage", ctx)["reference_seconds"], "bill the video track, not the longer audio track")
	ctx["requestBody"] = map[string]any{"prompt": "boat", "video_b64s": []string{"not-a-video"}}
	_, err := plugin.Engine.Call(t.Context(), "extractUsage", ctx)
	require.Error(t, err)
	ctx["usageKeys"] = []string{"seconds"}
	assert.Zero(t, yuanxiangHook(t, plugin, "extractUsage", ctx)["reference_seconds"])
	ctx["requestBody"] = map[string]any{"prompt": "boat"}
	ctx["usageKeys"] = []string{"reference_seconds"}
	ctx["files"] = []any{map[string]any{"ref": "request_file:videos", "field": "videos", "filename": "ref.mp4", "mimeType": "video/mp4", "size": 100}}
	probe := jsplugin.WithMediaProber(t.Context(), func(_ context.Context, req jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		assert.Equal(t, "request_file:videos", req.FileRef)
		return jsplugin.FetchResponse{Status: 200, Body: []byte("4.5")}, nil
	})
	value, err := plugin.Engine.Call(probe, "extractUsage", ctx)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"reference_seconds":4.5`)
	body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, []any{map[string]any{"__fileRef": "request_file:videos", "encoding": "base64"}}, body["video_b64s"])
}

func TestYuanxiangTaskLifecycleAndStableSubmissionFacts(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	ctx := yuanxiangContext("Wan 3.0 ·  官方直连 · 特价按秒-YX", map[string]any{"prompt": "boat"})
	ctx["taskId"] = "private/id"
	for _, tc := range []struct {
		body map[string]any
		want string
	}{
		{map[string]any{"status": "queued"}, "QUEUED"},
		{map[string]any{"status": "running"}, "IN_PROGRESS"},
		{map[string]any{"status": "succeeded", "video_url": "https://cdn.example/video.mp4"}, "SUCCESS"},
		{map[string]any{"status": "succeeded"}, "IN_PROGRESS"},
		{map[string]any{"status": "failed", "error": map[string]any{"message": "rejected"}}, "FAILURE"},
		{map[string]any{"status": "cancelled"}, "FAILURE"},
		{map[string]any{"status": "future_status"}, "UNKNOWN"},
		{map[string]any{"id": "wrong", "status": "succeeded", "video_url": "https://cdn.example/video.mp4"}, "UNKNOWN"},
	} {
		assert.Equal(t, tc.want, yuanxiangHook(t, plugin, "parseTaskResult", ctx, tc.body)["status"])
	}
	assert.Equal(t, "https://yuanxiang.example/v1/videos/private%2Fid", yuanxiangHook(t, plugin, "buildQueryRequest", ctx)["url"])
	submitted := yuanxiangHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"id": "private/id", "status": "queued"}})
	assert.Equal(t, "private/id", submitted["taskId"])
	assert.NotContains(t, submitted, "immediate")
	immediate := yuanxiangHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"id": "private/id", "status": "failed", "error": map[string]any{"message": "rejected"}}})
	assert.Equal(t, "FAILURE", immediate["immediate"].(map[string]any)["status"])
	// Yuanxiang charges the requested duration tier (Muse can return a shorter
	// video), so echoed duration and wall time cannot lower the frozen charge.
	assert.Empty(t, yuanxiangHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"duration": 1, "reference_seconds": 0}))
	ctx["model"], ctx["upstreamModel"] = "GT Image 2.5-YX", "GT Image 2.5-YX"
	for _, data := range []any{map[string]any{"url": "https://cdn.example/a.png"}, []any{map[string]any{"url": "https://cdn.example/a.png"}, map[string]any{"b64_json": "YWJj"}, map[string]any{"revised_prompt": "boat"}}} {
		assert.Equal(t, map[string]any{"image_count": float64(1)}, yuanxiangHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"data": data}))
	}
}

func TestYuanxiangDecodersReferencesAndArtifactRoutes(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	signedURL := "https://media.example/reference.png?Expires=1794225387&OSSAccessKeyId=test-id&Signature=abc%2Bdef%3D"
	for _, reference := range []any{map[string]any{"image_url": signedURL}, map[string]any{"image_url": map[string]any{"url": signedURL}}} {
		value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
			"model": "MiniMax H3 · 特价按秒-YX",
			"body":  map[string]any{"kind": "json", "value": map[string]any{"model": "MiniMax H3 · 特价按秒-YX", "prompt": "boat", "seconds": "5", "resolution_name": "720p", "size": "16:9", "input_reference": reference}},
		})
		require.NoError(t, err)
		encoded, err := common.Marshal(value)
		require.NoError(t, err)
		var intent map[string]any
		require.NoError(t, common.Unmarshal(encoded, &intent))
		ctx := yuanxiangContext("MiniMax H3 · 特价按秒-YX", intent["requestBody"].(map[string]any))
		submit := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)
		assert.Equal(t, []any{signedURL}, submit["body"].(map[string]any)["image_urls"], "preserve encoded signatures and query parameters byte for byte")
	}
	for _, kind := range []string{"json", "multipart"} {
		body := map[string]any{"kind": kind, "value": map[string]any{"model": "MiniMax H3 · 特价按秒-YX", "prompt": "boat", "seconds": "5", "size": "16:9"}}
		if kind == "multipart" {
			body = map[string]any{"kind": kind, "fields": map[string][]string{"prompt": {"boat"}, "seconds": {"5"}, "size": {"16:9"}, "metadata": {`{"resolution_name":"720p"}`}}}
		}
		value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{"model": "MiniMax H3 · 特价按秒-YX", "body": body})
		require.NoError(t, err)
		encoded, err := common.Marshal(value)
		require.NoError(t, err)
		var intent map[string]any
		require.NoError(t, common.Unmarshal(encoded, &intent))
		assert.Equal(t, "MiniMax H3 · 特价按秒-YX", intent["model"])
		ctx := yuanxiangContext(intent["model"].(string), intent["requestBody"].(map[string]any))
		assert.Equal(t, float64(5), yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["duration"])
	}
	ctx := yuanxiangContext("GT Image 2.5-YX", map[string]any{"prompt": "boat", "image_ids": []string{"uploaded"}, "image_refs": []any{map[string]any{"url": "https://media.example/reference.png"}}})
	body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, []any{"uploaded"}, body["image_ids"])
	assert.Equal(t, []any{"https://media.example/reference.png"}, body["image_urls"])
	assert.Equal(t, "low", body["quality"], "match the actually available upstream API quality")
	ctx = yuanxiangContext("MiniMax H3 · 特价按秒-YX", map[string]any{"prompt": "boat"})
	ctx["artifactKey"], ctx["upstreamTaskId"] = "video", "private/id"
	content := yuanxiangHook(t, plugin, "buildContentRequest", ctx)
	assert.Equal(t, "https://yuanxiang.example/v1/videos/private%2Fid/content", content["url"])
	assert.Equal(t, map[string]any{"Authorization": "Bearer test-key"}, content["headers"])
	ctx["upstream"] = map[string]any{"kind": "new_api"}
	ctx["data"] = map[string]any{"video_url": "https://gateway.example/signed/content"}
	content = yuanxiangHook(t, plugin, "buildContentRequest", ctx)
	assert.Equal(t, "https://gateway.example/signed/content", content["url"])
	assert.Equal(t, true, content["credentialless"])
	assert.NotContains(t, content, "headers")
}

func TestYuanxiangSoraMultipartUploadsBeforeVideoSubmission(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	var uploaded int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploaded++
		assert.Equal(t, "/v1/files", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, common.DecodeJson(r.Body, &body))
		assert.Equal(t, map[string]any{"data_b64": "aW1hZ2UtYnl0ZXM="}, body)
		_, err := w.Write([]byte(`{"object":"file","file_id":"uploaded-image","kind":"image","mime_type":"image/png","bytes":11}`))
		require.NoError(t, err)
	}))
	defer server.Close()
	var input bytes.Buffer
	writer := multipart.NewWriter(&input)
	for _, file := range []struct{ field, name, content string }{{"input_reference", "image.png", "image-bytes"}, {"videos", "video.mp4", "video-bytes"}, {"audios", "audio.mp3", "audio-bytes"}} {
		part, err := writer.CreateFormFile(file.field, file.name)
		require.NoError(t, err)
		_, err = part.Write([]byte(file.content))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &input)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("task_request", map[string]any{"model": "MiniMax H3 · 特价按秒-YX", "prompt": "boat", "seconds": "5", "resolution_name": "720p", "size": "16:9"})
	info := &relaycommon.RelayInfo{OriginModelName: "MiniMax H3 · 特价按秒-YX", TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeTaskPlugin, UpstreamModelName: "MiniMax H3 · 特价按秒-YX", ChannelBaseUrl: server.URL, ApiKey: "test-key"}}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	reader, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, common.DecodeJson(reader, &body))
	assert.Equal(t, "MiniMax H3 · 特价按秒", body["model"])
	assert.Equal(t, []any{"uploaded-image"}, body["image_ids"])
	assert.Equal(t, []any{"dmlkZW8tYnl0ZXM="}, body["video_b64s"])
	assert.Equal(t, []any{"YXVkaW8tYnl0ZXM="}, body["audio_b64s"])
	assert.Equal(t, float64(5), body["duration"])
	assert.Equal(t, "720p", body["resolution"])
	assert.Equal(t, "16:9", body["ratio"])
	assert.Equal(t, 1, uploaded)
}

func TestYuanxiangDurationBranchesIndependentlyForEveryVideoModel(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	const durationExpr = "u(\"duration_tier\") == \"10\" ? tier(\"10s\", u(\"requests\") * 2) : u(\"duration_tier\") == \"20\" ? tier(\"20s\", u(\"requests\") * 3) : tier(\"other\", u(\"requests\") * 1)"
	const resolutionExpr = "u(\"resolution\") == \"720p\" ? tier(\"720p\", u(\"requests\") * 0.2) : tier(\"other\", u(\"requests\") * 0.4)"
	var videos int
	for _, modelName := range plugin.Meta.Models {
		schema, _ := plugin.Meta.UsageForModels(modelName)
		duration, video := schema["duration_tier"]
		if !video {
			assert.Contains(t, schema, "image_count")
			continue
		}
		videos++
		t.Run(modelName, func(t *testing.T) {
			assert.True(t, duration.AllowCustom)
			assert.Equal(t, "10s", duration.EnumLabels["10"]["en"])
			assert.Equal(t, "20秒", duration.EnumLabels["20"]["zh"])
			require.NoError(t, billing_setting.SmokeTestTaskExpr(durationExpr, schema))
			require.NoError(t, billing_setting.SmokeTestTaskExpr(resolutionExpr, schema))
			for _, tc := range []struct {
				seconds int
				cost    float64
				tier    string
			}{{10, 2, "10s"}, {20, 3, "20s"}, {12, 1, "other"}} {
				for _, resolution := range []string{"720p", "1080p"} {
					ctx := yuanxiangContext(modelName, map[string]any{
						"prompt":   "boat",
						"metadata": map[string]any{"duration": tc.seconds, "resolution_name": strings.ToUpper(resolution)},
					})
					ctx["model"] = "local-alias"
					facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
					body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
					assert.Equal(t, float64(tc.seconds), body["duration"])
					assert.Equal(t, float64(tc.seconds), facts["seconds"])
					assert.Equal(t, fmt.Sprint(tc.seconds), facts["duration_tier"])
					assert.Equal(t, resolution, facts["resolution"])
					cost, trace, err := billingexpr.RunExprWithRequest(durationExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
					require.NoError(t, err)
					assert.Equal(t, tc.cost, cost, "duration tier is a per-request total, independent of resolution")
					assert.Equal(t, tc.tier, trace.MatchedTier)
					cost, _, err = billingexpr.RunExprWithRequest(resolutionExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
					require.NoError(t, err)
					want := 0.2
					if resolution == "1080p" {
						want = 0.4
					}
					assert.Equal(t, want, cost, "resolution pricing is independent of the duration tier")
					completion := yuanxiangHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"duration": 1, "resolution": "480p"})
					assert.Empty(t, completion, "upstream echoes must preserve the frozen submission tier")
				}
			}
		})
	}
	assert.Equal(t, 9, videos)
	t.Run("Muse continuation normalizes to 20 seconds before billing", func(t *testing.T) {
		ctx := yuanxiangContext("Muse · 特价按次 ·【原生】-YX", map[string]any{
			"prompt": "boat", "seconds": 10, "video_b64s": []string{"unmeasured-reference"},
		})
		facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
		body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
		assert.Equal(t, float64(20), body["duration"])
		assert.Equal(t, "20", facts["duration_tier"])
		cost, _, err := billingexpr.RunExprWithRequest(durationExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
		require.NoError(t, err)
		assert.Equal(t, float64(3), cost)
	})
}

func TestYuanxiangTokenEstimatesMatchDoubao(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	_, doubao := newDoubaoPlugin(t)
	const expression = "tier(\"tokens\", u(\"tokens\") * 1 / 1000000)"
	var videos int
	for _, modelName := range plugin.Meta.Models {
		schema, examples := plugin.Meta.UsageForModels(modelName)
		if _, image := schema["image_count"]; image {
			assert.NotContains(t, schema, "tokens")
			continue
		}
		videos++
		t.Run(modelName, func(t *testing.T) {
			assert.Equal(t, "token", schema["tokens"].Unit)
			assert.True(t, schema["tokens"].Optional)
			require.NotEmpty(t, examples)
			require.NoError(t, billing_setting.SmokeTestTaskExpr(expression, schema))
			for _, tc := range []struct {
				resolution string
				seconds    int
				tokens     float64
			}{
				{"480p", 5, 48037.5},
				{"720p", 10, 216000},
				{"1080p", 20, 972000},
				{"4k", 5, 972000},
			} {
				ctx := yuanxiangContext(modelName, map[string]any{
					"prompt": "boat", "metadata": map[string]any{"duration": tc.seconds, "resolution_name": strings.ToUpper(tc.resolution)},
					"tokens": 0, "usage": map[string]any{"completion_tokens": 0},
				})
				ctx["model"] = "mapped-client-name"
				facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
				assert.Equal(t, tc.tokens, facts["tokens"], "client-supplied usage cannot replace metered facts")
				other := yuanxiangHook(t, doubao, "extractUsage", map[string]any{
					"model":       "doubao-seedance-2-0-260128",
					"requestBody": map[string]any{"seconds": tc.seconds, "metadata": map[string]any{"resolution": tc.resolution}},
				})
				assert.Equal(t, other["tokens"], facts["tokens"], "shared resolution tiers use the existing Doubao formula")
			}
		})
	}
	assert.Equal(t, 9, videos)
	for _, tc := range []struct {
		resolution string
		seconds    int
		tokens     float64
	}{
		{"768p", 5, 122940},
		{"8k", 60, 46656000},
	} {
		facts := yuanxiangHook(t, plugin, "extractUsage", yuanxiangContext("MiniMax H3 · 特价按秒-YX", map[string]any{
			"prompt": "boat", "seconds": tc.seconds, "resolution": tc.resolution,
		}))
		assert.Equal(t, tc.tokens, facts["tokens"])
	}
	facts := yuanxiangHook(t, plugin, "extractUsage", yuanxiangContext("Muse · 特价按次 ·【原生】-YX", map[string]any{
		"prompt": "boat", "seconds": 10, "resolution": "720p", "video_b64s": []string{"unmeasured-reference"},
	}))
	assert.Equal(t, float64(432000), facts["tokens"], "Muse's actual submitted duration is normalized before token reservation")
	facts = yuanxiangHook(t, plugin, "extractUsage", yuanxiangContext("MiniMax H3 · 特价按秒-YX", map[string]any{
		"prompt": "boat", "seconds": 60, "resolution": "9999p",
	}))
	tokens := facts["tokens"].(float64)
	assert.Positive(t, tokens)
	assert.Less(t, tokens, float64(2147483647), "every accepted resolution and duration stays below the host token ceiling")
}

func TestYuanxiangActualTokenSettlementAndEstimatedFallback(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	const modelName = "SD 2.0 满血版 按秒-YX"
	const expression = "tier(\"tokens\", u(\"tokens\") * 1 / 1000000)"
	frozen := yuanxiangHook(t, plugin, "extractUsage", yuanxiangContext(modelName, map[string]any{
		"prompt": "boat", "seconds": 10, "resolution": "720p",
	}))
	assert.Equal(t, float64(216000), frozen["tokens"])
	initial, _, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: frozen})
	require.NoError(t, err)
	assert.Equal(t, 0.216, initial, "token-only pricing reserves a positive request-derived amount")
	snap := &billingexpr.BillingSnapshot{
		ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
		ExprVersion: 1, QuotaPerUnit: common.QuotaPerUnit, GroupRatio: 1,
		TaskUsageBilling: true, UsageFacts: frozen,
	}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(&relaycommon.RelayInfo{
		OriginModelName: modelName,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "test-key", ChannelBaseUrl: "https://yuanxiang.example",
			ChannelType: constant.ChannelTypeTaskPlugin, UpstreamModelName: modelName,
		},
	})
	task := &model.Task{
		TaskID: "task_public", Properties: model.Properties{OriginModelName: modelName, UpstreamModelName: modelName},
	}
	for _, tc := range []struct {
		name  string
		usage any
		want  float64
	}{
		{"completion takes precedence", map[string]any{"completion_tokens": 300000, "total_tokens": 400000}, 300000},
		{"total fallback", map[string]any{"total_tokens": 400000}, 400000},
		{"output fallback", map[string]any{"output_tokens": 250000}, 250000},
		{"decimal integer text", map[string]any{"completion_tokens": "300000"}, 300000},
		{"explicit zero", map[string]any{"completion_tokens": 0, "total_tokens": 400000}, 0},
		{"invalid completion falls back to total", map[string]any{"completion_tokens": -1, "total_tokens": 400000}, 400000},
		{"missing usage keeps estimate", nil, 216000},
		{"malformed usage keeps estimate", "invalid", 216000},
		{"array usage keeps estimate", []any{}, 216000},
		{"negative usage keeps estimate", map[string]any{"completion_tokens": -1}, 216000},
		{"fractional usage keeps estimate", map[string]any{"completion_tokens": 0.5}, 216000},
		{"boolean usage keeps estimate", map[string]any{"completion_tokens": false}, 216000},
		{"null usage keeps estimate", map[string]any{"completion_tokens": nil}, 216000},
		{"nonfinite text keeps estimate", map[string]any{"completion_tokens": "Infinity"}, 216000},
		{"unsafe integer keeps estimate", map[string]any{"completion_tokens": 9007199254740992.0}, 216000},
		{"oversized count saturates in host", map[string]any{"completion_tokens": 3000000000}, 2147483647},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := common.Marshal(map[string]any{
				"status": "succeeded", "video_url": "https://cdn.example/video.mp4", "usage": tc.usage,
				"duration": 1, "resolution": "480p",
			})
			require.NoError(t, err)
			result, err := adaptor.ParseTaskResult(task, &http.Response{StatusCode: http.StatusOK, Header: make(http.Header)}, body)
			require.NoError(t, err)
			require.Equal(t, "SUCCESS", result.Status)
			settlement, actual, err := service.EvaluateTaskCompletionUsage(snap, result.UsageFacts)
			require.NoError(t, err)
			assert.Equal(t, tc.want, actual["tokens"])
			assert.Equal(t, float64(10), actual["seconds"], "token settlement cannot alter per-second or duration-tier charges")
			assert.Equal(t, "720p", actual["resolution"])
			assert.Equal(t, common.QuotaRound(tc.want/1000000*common.QuotaPerUnit), settlement.ActualQuotaAfterGroup)
			assert.Equal(t, float64(216000), frozen["tokens"], "completion must not mutate the frozen reservation")
		})
	}
}

func TestYuanxiangReferenceVideoPricingConditions(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	const modelName = "SD 2.0 满血版 按秒-YX"
	const expression = `
	u("resolution") == "720p" && u("video_input") == "video" ? tier("reference", u("requests") * 2) :
	u("resolution") == "720p" && u("video_input") == "none" ? tier("text", u("requests") * 1) :
	u("resolution") == "1080p" && u("video_input") == "video" ? tier("fhd-reference", u("requests") * 4) :
	u("resolution") == "1080p" && u("video_input") == "none" ? tier("fhd-text", u("requests") * 3) :
	u("resolution") == "4k" && u("video_input") == "video" ? tier("uhd-reference", u("requests") * 6) :
	u("resolution") == "4k" && u("video_input") == "none" ? tier("uhd-text", u("requests") * 5) :
	tier("other", u("requests") * 7)`
	schema, _ := plugin.Meta.UsageForModels(modelName)
	require.NoError(t, billing_setting.SmokeTestTaskExpr(expression, schema))
	for _, tc := range []struct {
		name       string
		body       map[string]any
		files      []any
		videoInput string
		cost       float64
	}{
		{name: "text only", videoInput: "none", cost: 1},
		{name: "image references", body: map[string]any{"image_urls": []string{"https://cdn.example/ref.png"}}, videoInput: "none", cost: 1},
		{name: "audio references", body: map[string]any{"audio_b64s": []string{"unmeasured-audio"}}, videoInput: "none", cost: 1},
		{name: "video references", body: map[string]any{"video_b64s": []string{"unmeasured-video"}}, videoInput: "video", cost: 2},
		{name: "nested video references", body: map[string]any{"metadata": map[string]any{"parameters": map[string]any{"videos": []string{"data:video/mp4;base64,dmlkZW8="}}}}, videoInput: "video", cost: 2},
		{name: "uploaded video", files: []any{map[string]any{"ref": "request_file:video", "field": "videos[]", "mimeType": "video/mp4", "size": 100}}, videoInput: "video", cost: 2},
		{name: "uploaded input reference", files: []any{map[string]any{"ref": "request_file:video", "field": "input_reference", "mimeType": "video/mp4", "size": 100}}, videoInput: "video", cost: 2},
		{name: "1080p without reference", body: map[string]any{"resolution": "1080p"}, videoInput: "none", cost: 3},
		{name: "1080p with reference", body: map[string]any{"resolution": "1080p", "video_b64s": []string{"unmeasured-video"}}, videoInput: "video", cost: 4},
		{name: "4k without reference", body: map[string]any{"resolution": "4k"}, videoInput: "none", cost: 5},
		{name: "4k with reference", body: map[string]any{"resolution": "4k", "video_b64s": []string{"unmeasured-video"}}, videoInput: "video", cost: 6},
		{name: "other resolution", body: map[string]any{"resolution": "8k", "video_b64s": []string{"unmeasured-video"}}, videoInput: "video", cost: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"prompt": "boat", "resolution": "720p", "seconds": 10}
			maps.Copy(body, tc.body)
			ctx := yuanxiangContext(modelName, body)
			ctx["files"] = tc.files
			ctx["usageKeys"] = []string{"requests", "resolution", "video_input"}
			facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, tc.videoInput, facts["video_input"])
			assert.Zero(t, facts["reference_seconds"], "a presence condition must not probe reference duration")
			cost, _, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.Equal(t, tc.cost, cost)
			completion := yuanxiangHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"usage": map[string]any{"completion_tokens": 123}, "video_input": "none"})
			assert.NotContains(t, completion, "video_input", "completion usage must retain the submitted reference condition")
		})
	}
}
