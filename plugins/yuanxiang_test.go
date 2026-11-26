package plugins_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
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

func TestYuanxiangInlineGatewayImagesRemainAvailableForStorage(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	ctx := map[string]any{"status": "SUCCESS", "data": map[string]any{"object": "image", "data": []any{map[string]any{"b64_json": "aW1hZ2U="}}}, "upstream": map[string]any{"kind": "new_api"}, "artifactKey": "image-b64-1"}
	artifacts, err := plugin.Engine.Call(t.Context(), "listArtifacts", ctx)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	descriptor := yuanxiangHook(t, plugin, "buildContentRequest", ctx)
	assert.Equal(t, "data:image/png;base64,aW1hZ2U=", descriptor["url"])
	assert.Equal(t, true, descriptor["credentialless"])
	assert.NotContains(t, descriptor, "headers")
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
	ctx := yuanxiangContext("seedance-2-5-per-request", map[string]any{"model": "seedance-2-5-per-request", "prompt": "A paper boat", "resolution_name": "720p", "seconds": "5", "size": "16:9"})
	submit := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "https://yuanxiang.example/v1/videos", submit["url"])
	assert.NotContains(t, submit, "model", "the submit hook keeps the pinned billing identity")
	body := submit["body"].(map[string]any)
	assert.Equal(t, "SD 2.5 · 特价按次", body["model"])
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
	ctx["upstreamModel"] = "seedance-2-5-per-request"
	ctx["model"] = "canvas-custom-YX"
	assert.Equal(t, "SD 2.5 · 特价按次", yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["model"])
	ctx["upstream"] = map[string]any{"kind": "new_api"}
	gateway := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "https://yuanxiang.example/yuanxiang/v1/videos", gateway["url"])
	assert.Equal(t, "seedance-2-5-per-request", gateway["body"].(map[string]any)["model"])
}

func TestYuanxiangHistoricalModelsCanOnlyBeQueried(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	for _, tc := range []struct {
		model, path string
	}{
		{"SD 2.5 · 官方满血按秒", "videos"},
		{"SD 2.0 · 官转满血68折（返官链）", "videos"},
		{"SD 2.5 · 特价按次", "videos"},
		{"Sd 2.0 · 特价按次 · 漫剧短剧专线", "videos"},
		{"SD 2.0 · 特价按次 · 电商专线", "videos"},
		{"SD2.0 · 即梦满血版", "videos"},
		{"SD 2.0 FAST · 特价按秒 · 稳定冲量", "videos"},
		{"Sd 2.0 · 即梦满血按秒专线", "videos"},
		{"Sd 2.5 · 即梦满血按秒专线", "videos"},
		{"GK-video-3.5", "videos"},
		{"Muse · 特价按次 ·【原生】", "videos"},
		{"MiniMax H3 · 特价按秒", "videos"},
		{"MiniMax H3 · 特价按次（慢线）", "videos"},
		{"Wan 3.0 ·  官方直连 · 特价按秒", "videos"},
		{"GT Image 2.5", "images"},
		{"GT Image 2.5 · 特价按张", "images"},
		{"GT Image 2.6 Flare", "images"},
		{"GT Image 2 · OpenAI", "images"},
		{"Banana 2", "images"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			for _, id := range []string{tc.model, tc.model + "-YX"} {
				ctx := yuanxiangContext(id, map[string]any{"model": id, "prompt": "boat"})
				for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
					_, err := plugin.Engine.Call(t.Context(), hook, ctx)
					require.ErrorContains(t, err, "unsupported Yuanxiang model for new tasks")
				}
				ctx["taskId"] = "historical/id"
				query := yuanxiangHook(t, plugin, "buildQueryRequest", ctx)
				assert.Equal(t, "https://yuanxiang.example/v1/"+tc.path+"/historical%2Fid", query["url"])
				ctx["upstream"] = map[string]any{"kind": "new_api"}
				query = yuanxiangHook(t, plugin, "buildQueryRequest", ctx)
				assert.Equal(t, "https://yuanxiang.example/yuanxiang/v1/"+tc.path+"/historical%2Fid", query["url"])
			}
		})
	}
	value, err := plugin.Engine.CallPath(t.Context(), "native", []string{"task"}, map[string]any{}, map[string]any{
		"task_id": "old-image", "status": "SUCCESS", "data": map[string]any{"data": []any{map[string]any{"url": "https://cdn.example/image.png"}}},
	})
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"old-image","status":"succeeded","data":[{"url":"https://cdn.example/image.png"}]}`, string(encoded))
}

func TestYuanxiangSD25PerItemStableRoute(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	const model = "seedance-2-5-per-request"
	ctx := yuanxiangContext(model, map[string]any{"prompt": "boat", "seconds": "30", "image_ids": slices.Repeat([]string{"uploaded-image"}, 30)})
	body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, "line:stable-longsupply-item", body["channel"])
	assert.Equal(t, float64(30), body["duration"])
	assert.Len(t, body["image_ids"], 30)
	facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
	assert.Equal(t, "30", facts["duration_tier"])
	assert.Equal(t, "720p", facts["resolution"])
	assert.Equal(t, float64(1), facts["requests"])

	for _, tc := range []map[string]any{
		{"duration": 31}, {"resolution": "1080p"}, {"image_ids": slices.Repeat([]string{"uploaded-image"}, 31)},
		{"metadata": map[string]any{"channel": "route:tejia-y"}},
	} {
		tc["prompt"] = "boat"
		ctx := yuanxiangContext(model, tc)
		for _, hook := range []string{"extractUsage", "buildSubmitRequest"} {
			_, err := plugin.Engine.Call(t.Context(), hook, ctx)
			require.Error(t, err, "%s: %v", hook, tc)
		}
	}
}

func TestYuanxiangSharesSeedancePerRequestModelsWithLecAPI(t *testing.T) {
	registry := jsplugin.NewRegistry()
	for _, key := range []string{"lecapi", "yuanxiang"} {
		source, err := builtinplugins.Source(key)
		require.NoError(t, err)
		_, err = registry.RegisterFactory(source, jsplugin.Options{Key: key})
		require.NoError(t, err)
	}
	const sd20, sd25 = "seedance-2-0-per-request", "seedance-2-5-per-request"
	const modelPath = "/v1/videos"
	generation := registry.Generation()
	yuanxiang, ok := generation.Get("yuanxiang")
	require.True(t, ok)
	assert.ElementsMatch(t, []string{sd20, sd25}, yuanxiang.Meta.Models)
	assert.Empty(t, generation.LookupEndpointCandidates("POST", "/v1/images/generations", "GT Image 2.5-YX"))
	for _, model := range []string{sd20, sd25} {
		bindings := generation.LookupEndpointCandidates("POST", modelPath, model)
		require.Len(t, bindings, 2)
		assert.Equal(t, "lecapi", bindings[0].Plugin.Meta.Key)
		assert.Equal(t, "yuanxiang", bindings[1].Plugin.Meta.Key)
	}
	for _, tc := range []struct {
		public, wire, line, ratio, quality string
		seconds                            int
		price                              float64
	}{
		{sd20, "Sd 2.0 · 特价按次 · 漫剧短剧专线", "", "4:3", "", 10, 2.9},
		{sd20, "SD 2.0 · 特价按次 · 电商专线", "", "9:16", "真人模特", 15, 2.9},
		{sd20, "SD2.0 · 即梦满血版", "", "16:9", "", 12, 2.9},
		{sd25, "SD 2.5 · 特价按次", "line:stable-longsupply-item", "1:1", "", 30, 6},
		{sd25, "SD 2.5 · 特价按次", "line:economy-sd25", "21:9", "", 4, 6},
	} {
		t.Run(tc.wire+"/"+tc.line, func(t *testing.T) {
			input := map[string]any{"model": tc.public, "prompt": "boat", "duration": tc.seconds, "aspect_ratio": tc.ratio}
			if tc.quality != "" {
				input["quality"] = tc.quality
			}
			ctx := yuanxiangContext(tc.public, input)
			request := yuanxiangHook(t, yuanxiang, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, tc.wire, request["model"])
			assert.Equal(t, float64(tc.seconds), request["duration"])
			if tc.line != "" {
				assert.Equal(t, tc.line, request["channel"])
			}
			facts := yuanxiangHook(t, yuanxiang, "extractUsage", ctx)
			defaults, found := yuanxiang.Meta.DefaultsForModel(tc.public)
			require.True(t, found)
			cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.price, cost, 1e-12)
			ctx["upstream"] = map[string]any{"kind": "new_api"}
			chained := yuanxiangHook(t, yuanxiang, "buildSubmitRequest", ctx)["body"].(map[string]any)
			assert.Equal(t, tc.public, chained["model"])
			assert.NotContains(t, chained, "channel")
			_, err = yuanxiang.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
				"model": tc.public, "body": map[string]any{"kind": "json", "value": chained},
			})
			require.NoError(t, err)
		})
	}
	lecapi, ok := generation.Get("lecapi")
	require.True(t, ok)
	for _, tc := range []struct {
		model  string
		fields map[string]any
		accept *jsplugin.LoadedPlugin
		reject *jsplugin.LoadedPlugin
	}{
		{sd25, map[string]any{"resolution": "480p"}, nil, lecapi},
		{sd25, map[string]any{"resolution": "480p"}, nil, yuanxiang},
		{sd20, map[string]any{"quality": "真人模特", "duration": 10}, yuanxiang, lecapi},
		{sd20, map[string]any{"videos": []string{"https://media.example/ref.mp4"}, "images": []string{"https://media.example/ref.png"}, "duration": 5}, lecapi, yuanxiang},
		{sd25, map[string]any{"channel": "arbitrary"}, nil, yuanxiang},
		{sd20, map[string]any{"options": map[string]any{"quality": "满血9图"}}, nil, yuanxiang},
	} {
		input := map[string]any{"model": tc.model, "prompt": "boat"}
		maps.Copy(input, tc.fields)
		ctx := map[string]any{"model": tc.model, "body": map[string]any{"kind": "json", "value": input}}
		_, err := tc.reject.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, ctx)
		require.Error(t, err)
		if tc.accept != nil {
			_, err = tc.accept.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, ctx)
			require.NoError(t, err)
		}
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
			_, err := plugin.Engine.Call(t.Context(), hook, yuanxiangContext("seedance-2-5-per-request", body))
			require.Error(t, err, "%s: %v", hook, body)
		}
	}
	ctx := yuanxiangContext("seedance-2-5-per-request", map[string]any{"prompt": "boat", "metadata": map[string]any{"seconds": "8", "size": "9:16", "resolution_name": "720P"}})
	body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, float64(8), body["duration"])
	assert.Equal(t, "9:16", body["ratio"])
	assert.Equal(t, "720p", body["resolution"])
}

func TestYuanxiangReferenceBillingUsesMediaAndSkipsDisabledMeasurements(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	video := base64.StdEncoding.EncodeToString(yueqiReferenceVideoFixture(t, 3.5))
	ctx := yuanxiangContext("seedance-2-5-per-request", map[string]any{"prompt": "boat", "video_b64s": []string{video}})
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
			"model": "seedance-2-5-per-request",
			"body":  map[string]any{"kind": "json", "value": map[string]any{"model": "seedance-2-5-per-request", "prompt": "boat", "seconds": "5", "resolution_name": "720p", "size": "16:9", "input_reference": reference}},
		})
		require.NoError(t, err)
		encoded, err := common.Marshal(value)
		require.NoError(t, err)
		var intent map[string]any
		require.NoError(t, common.Unmarshal(encoded, &intent))
		ctx := yuanxiangContext("seedance-2-5-per-request", intent["requestBody"].(map[string]any))
		submit := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)
		assert.Equal(t, []any{signedURL}, submit["body"].(map[string]any)["image_urls"], "preserve encoded signatures and query parameters byte for byte")
	}
	for _, kind := range []string{"json", "multipart"} {
		body := map[string]any{"kind": kind, "value": map[string]any{"model": "seedance-2-5-per-request", "prompt": "boat", "seconds": "5", "size": "16:9"}}
		if kind == "multipart" {
			body = map[string]any{"kind": kind, "fields": map[string][]string{"prompt": {"boat"}, "seconds": {"5"}, "size": {"16:9"}, "metadata": {`{"resolution_name":"720p"}`}}}
		}
		value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{"model": "seedance-2-5-per-request", "body": body})
		require.NoError(t, err)
		encoded, err := common.Marshal(value)
		require.NoError(t, err)
		var intent map[string]any
		require.NoError(t, common.Unmarshal(encoded, &intent))
		assert.Equal(t, "seedance-2-5-per-request", intent["model"])
		ctx := yuanxiangContext(intent["model"].(string), intent["requestBody"].(map[string]any))
		assert.Equal(t, float64(5), yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["duration"])
	}
	ctx := yuanxiangContext("seedance-2-5-per-request", map[string]any{"prompt": "boat", "image_ids": []string{"uploaded"}, "image_refs": []any{map[string]any{"url": "https://media.example/reference.png"}}})
	body := yuanxiangHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, []any{"uploaded"}, body["image_ids"])
	assert.Equal(t, []any{"https://media.example/reference.png"}, body["image_urls"])
	ctx = yuanxiangContext("MiniMax H3 · 特价按秒-YX", map[string]any{"prompt": "boat"})
	ctx["artifactKey"], ctx["upstreamTaskId"] = "video", "private/id"
	callCtx := jsplugin.WithFetcher(t.Context(), func(_ context.Context, request jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		assert.Equal(t, "https://yuanxiang.example/v1/videos/private%2Fid", request.URL)
		assert.Equal(t, "Bearer test-key", request.Headers["Authorization"])
		return jsplugin.FetchResponse{Status: 200, Body: []byte(`{"id":"private/id","status":"succeeded","video_url":"https://media.example/fresh.mp4?signature=fresh"}`)}, nil
	})
	value, err := plugin.Engine.Call(callCtx, "buildContentRequest", ctx)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `{"url":"https://media.example/fresh.mp4?signature=fresh","method":"GET","credentialless":true}`, string(encoded))
	ctx["upstream"] = map[string]any{"kind": "new_api"}
	ctx["data"] = map[string]any{"video_url": "https://gateway.example/signed/content"}
	content := yuanxiangHook(t, plugin, "buildContentRequest", ctx)
	assert.Equal(t, "https://gateway.example/signed/content", content["url"])
	assert.Equal(t, true, content["credentialless"])
	assert.NotContains(t, content, "headers")
}

func TestYuanxiangContentRefreshValidatesTaskAndResult(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	for _, tc := range []struct {
		name, model, key, body, wantURL string
		status                          int
	}{
		{name: "image", model: "GT Image 2.5-YX", key: "image-2", status: 200, body: `{"id":"vendor-id","status":"succeeded","object":"image","data":[{"url":"https://cdn.example/one.png"},{"url":"https://cdn.example/two.png"}]}`, wantURL: "https://cdn.example/two.png"},
		{name: "query failed", status: 403, body: `{"message":"secret-key https://cdn.example/?signature=secret"}`},
		{name: "wrong task", status: 200, body: `{"id":"another-id","status":"succeeded","video_url":"https://cdn.example/video.mp4"}`},
		{name: "missing task id", status: 200, body: `{"status":"succeeded","video_url":"https://cdn.example/video.mp4"}`},
		{name: "pending", status: 200, body: `{"id":"vendor-id","status":"running","video_url":"https://cdn.example/video.mp4"}`},
		{name: "invalid URL", status: 200, body: `{"id":"vendor-id","status":"succeeded","video_url":"https://user:secret@cdn.example/video.mp4"}`},
		{name: "no result", status: 200, body: `{"id":"vendor-id","status":"succeeded"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modelName, key := tc.model, tc.key
			if modelName == "" {
				modelName = "MiniMax H3 · 特价按秒-YX"
			}
			if key == "" {
				key = "video"
			}
			ctx := yuanxiangContext(modelName, nil)
			ctx["artifactKey"], ctx["upstreamTaskId"] = key, "vendor-id"
			callCtx := jsplugin.WithFetcher(t.Context(), func(_ context.Context, request jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
				if tc.model != "" {
					assert.Equal(t, "https://yuanxiang.example/v1/images/vendor-id", request.URL)
				}
				return jsplugin.FetchResponse{Status: tc.status, Body: []byte(tc.body)}, nil
			})
			value, err := plugin.Engine.Call(callCtx, "buildContentRequest", ctx)
			if tc.wantURL == "" {
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "secret")
				assert.NotContains(t, err.Error(), "https://")
				return
			}
			require.NoError(t, err)
			encoded, err := common.Marshal(value)
			require.NoError(t, err)
			var result map[string]any
			require.NoError(t, common.Unmarshal(encoded, &result))
			assert.Equal(t, tc.wantURL, result["url"])
			assert.Equal(t, true, result["credentialless"])
			assert.NotContains(t, result, "headers")
		})
	}
}

func TestYuanxiangMultipartUploadsBeforeVideoSubmission(t *testing.T) {
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
	c.Set("task_request", map[string]any{"model": "seedance-2-5-per-request", "prompt": "boat", "seconds": "5", "resolution_name": "720p", "size": "16:9"})
	info := &relaycommon.RelayInfo{OriginModelName: "seedance-2-5-per-request", TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeTaskPlugin, UpstreamModelName: "seedance-2-5-per-request", ChannelBaseUrl: server.URL, ApiKey: "test-key"}}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	reader, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, common.DecodeJson(reader, &body))
	assert.Equal(t, "SD 2.5 · 特价按次", body["model"])
	assert.Equal(t, []any{"uploaded-image"}, body["image_ids"])
	assert.Equal(t, []any{"dmlkZW8tYnl0ZXM="}, body["video_b64s"])
	assert.Equal(t, []any{"YXVkaW8tYnl0ZXM="}, body["audio_b64s"])
	assert.Equal(t, float64(5), body["duration"])
	assert.Equal(t, "720p", body["resolution"])
	assert.Equal(t, "16:9", body["ratio"])
	assert.Equal(t, 1, uploaded)
}

func TestYuanxiangDurationBranchesUseSubmittedSeconds(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	const expression = `u("duration_tier") == "10" ? tier("10s", u("requests") * 2) : u("duration_tier") == "20" ? tier("20s", u("requests") * 3) : tier("other", u("requests") * 1)`
	for _, model := range plugin.Meta.Models {
		schema, _ := plugin.Meta.UsageForModels(model)
		assert.Equal(t, "10s", schema["duration_tier"].EnumLabels["10"]["en"])
		assert.Equal(t, "20秒", schema["duration_tier"].EnumLabels["20"]["zh"])
		require.NoError(t, billing_setting.SmokeTestTaskExpr(expression, schema))
		for _, tc := range []struct {
			seconds string
			price   float64
			tier    string
		}{{"10", 2, "10s"}, {"12", 1, "other"}} {
			ctx := yuanxiangContext(model, map[string]any{"prompt": "boat", "metadata": map[string]any{"duration": tc.seconds, "resolution_name": "720P"}})
			facts := yuanxiangHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, tc.seconds, facts["duration_tier"])
			cost, trace, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
			require.NoError(t, err)
			assert.Equal(t, tc.price, cost)
			assert.Equal(t, tc.tier, trace.MatchedTier)
		}
	}
}

func TestYuanxiangTokenEstimatesMatchDoubao(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	_, doubao := newDoubaoPlugin(t)
	const expression = `tier("tokens", u("tokens") * 1 / 1000000)`
	for _, model := range plugin.Meta.Models {
		schema, examples := plugin.Meta.UsageForModels(model)
		assert.Equal(t, "token", schema["tokens"].Unit)
		assert.True(t, schema["tokens"].Optional)
		require.NotEmpty(t, examples)
		require.NoError(t, billing_setting.SmokeTestTaskExpr(expression, schema))
		request := map[string]any{"prompt": "boat", "seconds": 10, "resolution": "720p"}
		facts := yuanxiangHook(t, plugin, "extractUsage", yuanxiangContext(model, request))
		assert.Equal(t, float64(216000), facts["tokens"])
		other := yuanxiangHook(t, doubao, "extractUsage", map[string]any{
			"model":       "doubao-seedance-2-0-260128",
			"requestBody": map[string]any{"seconds": 10, "metadata": map[string]any{"resolution": "720p"}},
		})
		assert.Equal(t, other["tokens"], facts["tokens"])
	}
}

func TestYuanxiangActualTokenSettlementAndEstimatedFallback(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	const modelName = "SD 2.5 · 官方满血按秒-YX"
	const expression = "tier(\"tokens\", u(\"tokens\") * 1 / 1000000)"
	frozen := yuanxiangHook(t, plugin, "extractUsage", yuanxiangContext("seedance-2-5-per-request", map[string]any{
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
	const modelName = "seedance-2-5-per-request"
	const expression = `
	u("resolution") == "720p" && u("video_input") == "video" ? tier("reference", u("requests") * 2) :
	u("resolution") == "720p" && u("video_input") == "none" ? tier("text", u("requests") * 1) :
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
