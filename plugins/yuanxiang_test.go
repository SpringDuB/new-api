package plugins_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	taskplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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
			assert.Equal(t, "high", facts["quality"])
			schema, _ := plugin.Meta.UsageForModels(model + "-YX")
			assert.Contains(t, schema, "image_count")
			assert.NotContains(t, schema, "seconds")
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

func TestYuanxiangRejectsConflictingAndUnboundedParameters(t *testing.T) {
	plugin := loadYuanxiangPlugin(t)
	for _, body := range []map[string]any{
		{"seconds": -1}, {"seconds": 0}, {"seconds": "1e99"}, {"seconds": 61}, {"seconds": true},
		{"seconds": 5, "duration": 6}, {"seconds": 5, "metadata": map[string]any{"duration": 6}},
		{"resolution_name": "720p", "parameters": map[string]any{"resolution": "1080p"}},
		{"size": "16:9", "ratio": "9:16"}, {"size": "0:9"},
		{"options": map[string]any{"duration": -1}}, {"extra": map[string]any{"n": 129}},
		{"metadata": map[string]any{"max_tokens": 1e99}}, {"n": 2}, {"metadata": "bad"},
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
