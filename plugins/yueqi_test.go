package plugins_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	"github.com/QuantumNous/new-api/relay/channel"
	taskplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/abema/go-mp4"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const yueqiPublicModel = "seedance-2.5-超低价"

func loadYueqiPlugin(t *testing.T) *jsplugin.LoadedPlugin {
	t.Helper()
	source, err := builtinplugins.Source("yueqi")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "yueqi"})
	require.NoError(t, err)
	return plugin
}

func yueqiContext(modelName string, body map[string]any) map[string]any {
	return map[string]any{
		"model": modelName, "upstreamModel": modelName, "requestBody": body,
		"baseUrl": "https://yueqi.example/", "apiKey": "test-key",
	}
}

func yueqiHook(t *testing.T, plugin *jsplugin.LoadedPlugin, hook string, args ...any) map[string]any {
	t.Helper()
	value, err := plugin.Engine.Call(t.Context(), hook, args...)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, common.Unmarshal(encoded, &result))
	return result
}

func TestYueqiPublicModelAndDefaultBilling(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	assert.Equal(t, []string{yueqiPublicModel}, plugin.Meta.Models)
	defaults, found := plugin.Meta.DefaultsForModel(yueqiPublicModel)
	require.True(t, found)
	assert.Equal(t, yueqiPublicModel, defaults.DisplayName)
	assert.Contains(t, defaults.Description["zh"], "文生视频")
	assert.NotEmpty(t, defaults.Description["en"])

	generation := jsplugin.DefaultRegistry.Generation()
	bindings := generation.LookupEndpointCandidates(http.MethodPost, "/v1/videos", yueqiPublicModel)
	require.Len(t, bindings, 1)
	assert.Equal(t, "yueqi", bindings[0].Plugin.Meta.Key)
	assert.Equal(t, "openai_video", bindings[0].Protocol)
	assert.Empty(t, generation.LookupEndpointCandidates(http.MethodPost, "/v1/videos", "kk-sd25"))
	assert.Empty(t, generation.LookupEndpointCandidates(http.MethodPost, "/v1/videos", "BD Seedance2.5"))

	adaptor := taskplugin.New(plugin)
	info := &relaycommon.RelayInfo{
		OriginModelName: yueqiPublicModel, TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kk-sd25"},
	}
	adaptor.Init(info)
	assert.True(t, adaptor.DisableSubmitRetry())
	expr, found := adaptor.ResolveBillingExpression(info.OriginModelName, info.UpstreamModelName)
	require.True(t, found)
	assert.Equal(t, defaults.BillingExpr, expr)
	for _, seconds := range []int{5, 15} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
		c.Set("task_request", map[string]any{"prompt": "A paper boat", "seconds": seconds})
		facts, err := adaptor.ExtractUsageFactsValidated(c, info)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"requests": float64(1)}, facts)
		cost, _, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
		require.NoError(t, err)
		assert.Equal(t, float64(3), cost)
		completion := yueqiHook(t, plugin, "extractUsageOnComplete", map[string]any{"model": yueqiPublicModel, "upstreamModel": "kk-sd25"},
			map[string]any{"status": "SUCCESS"}, map[string]any{"duration": 12, "resolution": "720p"})
		settled, _, err := service.EvaluateTaskCompletionUsage(&billingexpr.BillingSnapshot{
			ExprString: expr, TaskUsageBilling: true, UsageFacts: facts, GroupRatio: 1, QuotaPerUnit: common.QuotaPerUnit,
		}, completion)
		require.NoError(t, err)
		assert.Equal(t, common.QuotaRound(3*common.QuotaPerUnit), settled.ActualQuotaAfterGroup)
	}
}

func TestYueqiTextToVideoSubmission(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	for _, tc := range []struct {
		name, wantModel string
		request         map[string]any
		upstream        map[string]any
	}{
		{"vendor JSON", "kk-sd25", map[string]any{"kind": "json", "value": map[string]any{
			"model": yueqiPublicModel, "prompt": "A paper boat", "seconds": "6",
			"metadata": map[string]any{"duration": 6, "resolution_name": "720P", "size": "16:9"},
		}}, nil},
		{"vendor multipart", "kk-sd25", map[string]any{"kind": "multipart", "fields": map[string][]string{
			"model": {yueqiPublicModel}, "prompt": {"A paper boat"}, "seconds": {"6"},
			"parameters": {`{"aspect_ratio":"16:9"}`},
		}}, nil},
		{"chained gateway", yueqiPublicModel, map[string]any{"kind": "json", "value": map[string]any{
			"model": yueqiPublicModel, "prompt": "A paper boat", "duration": 6, "aspect_ratio": "16:9",
		}}, map[string]any{"kind": "new_api"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
				"model": yueqiPublicModel, "body": tc.request,
			})
			require.NoError(t, err)
			intent := decoded.(map[string]any)
			assert.Equal(t, yueqiPublicModel, intent["model"])
			ctx := yueqiContext(yueqiPublicModel, intent["requestBody"].(map[string]any))
			if tc.upstream != nil {
				ctx["upstream"] = tc.upstream
			}
			submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
			assert.Equal(t, "https://yueqi.example/v1/videos", submit["url"])
			assert.Equal(t, http.MethodPost, submit["method"])
			assert.Equal(t, "text_to_video", submit["action"])
			assert.Equal(t, map[string]any{"Authorization": "Bearer test-key", "Content-Type": "application/json"}, submit["headers"])
			assert.Equal(t, map[string]any{
				"model": tc.wantModel, "prompt": "A paper boat", "duration": float64(6), "resolution": "720p", "aspect_ratio": "16:9",
			}, submit["body"])
		})
	}
}

func TestYueqiRejectsUnsupportedVideoRequests(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	for _, tc := range []struct {
		name, field, wantError string
		value                  any
	}{
		{"blank prompt", "prompt", "prompt is required", " "},
		{"wrong resolution", "size", "only supports 720p", "1080p"},
		{"invalid ratio", "size", "positive width:height", "0:16"},
		{"image reference", "input_reference", "input_reference is not supported", "https://example.test/frame.png"},
		{"nested video reference", "metadata", "videos is not supported", map[string]any{"videos": []string{"https://example.test/video.mp4"}}},
		{"nested model override", "metadata", "nested model must match", map[string]any{"model": "wan-3.0"}},
		{"nested batch count", "metadata", "exactly one video", map[string]any{"n": 2}},
		{"conflicting duration", "metadata", "conflicting duration", map[string]any{"seconds": 6}},
		{"unbounded duration", "duration", "between 1 and 60", 3601},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"prompt": "A paper boat", "duration": 5}
			body[tc.field] = tc.value
			_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", yueqiContext(yueqiPublicModel, body))
			require.ErrorContains(t, err, tc.wantError)
		})
	}
	_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", yueqiContext("BD Seedance2.5", map[string]any{"prompt": "A paper boat"}))
	require.ErrorContains(t, err, "unsupported Yueqi video model")

	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	file, err := writer.CreateFormFile("input_reference", "frame.png")
	require.NoError(t, err)
	_, err = file.Write([]byte("frame"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("model", yueqiPublicModel))
	require.NoError(t, writer.WriteField("prompt", "A paper boat"))
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &multipartBody)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("task_request", map[string]any{"prompt": "A paper boat"})
	info := &relaycommon.RelayInfo{
		OriginModelName: yueqiPublicModel, TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: yueqiPublicModel, ChannelBaseUrl: "https://yueqi.example"},
	}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(c, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
}

func TestYueqiTaskLifecycleAndCompletionUsage(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	ctx := map[string]any{"taskId": "private", "upstreamModel": "seedance2.5无限并发版"}
	for _, tc := range []struct {
		name, status, progress string
		body                   map[string]any
	}{
		{"queued", "QUEUED", "0%", map[string]any{"status": "queued"}},
		{"running", "IN_PROGRESS", "42%", map[string]any{"status": "in_progress", "progress": "42%"}},
		{"storing", "IN_PROGRESS", "99%", map[string]any{"status": "completed", "storage_status": "pending", "progress": 100}},
		{"stored without URL", "IN_PROGRESS", "99%", map[string]any{"status": "completed", "storage_status": "stored"}},
		{"stored", "SUCCESS", "100%", map[string]any{"status": "completed", "storage_status": "stored", "url": "https://cdn.example/video.mp4"}},
		{"signed result", "SUCCESS", "100%", map[string]any{"status": "completed", "metadata": map[string]any{"url": "https://cdn.example/video.mp4"}}},
		{"storage failed", "FAILURE", "", map[string]any{"status": "completed", "storage_status": "failed"}},
		{"generation failed", "FAILURE", "", map[string]any{"status": "failed", "error": map[string]any{"message": "generation rejected"}}},
		{"future state", "UNKNOWN", "", map[string]any{"status": "future_state"}},
		{"wrong id", "UNKNOWN", "", map[string]any{"id": "another-task", "status": "completed", "url": "https://cdn.example/video.mp4"}},
		{"wrapped response", "IN_PROGRESS", "23%", map[string]any{"code": "success", "data": map[string]any{"task_id": "private", "status": "IN_PROGRESS", "progress": "23%"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := yueqiHook(t, plugin, "parseTaskResult", ctx, tc.body)
			assert.Equal(t, tc.status, result["status"])
			if tc.progress != "" {
				assert.Equal(t, tc.progress, result["progress"])
			}
		})
	}
	assert.Equal(t, map[string]any{"seconds": float64(8), "resolution": "720p"}, yueqiHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": "8", "resolution": "720p"}))
	assert.Equal(t, map[string]any{"seconds": float64(8), "resolution": "1080p"}, yueqiHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": "8", "resolution": "1080p"}), "completion uses administrator-defined resolution branches")
	assert.Empty(t, yueqiHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"usage": map[string]any{"generation_seconds": 1000}}))
	_, err := plugin.Engine.Call(t.Context(), "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 1e99})
	require.Error(t, err, "untrusted output duration must not become an unbounded billing quantity")
	ctx["upstreamModel"] = "BD Seedance2.5"
	assert.Equal(t, map[string]any{"seconds": float64(30)}, yueqiHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 30}))

	for _, body := range []any{`<html>error</html>`, map[string]any{"id": 3}, map[string]any{"code": "failure", "data": map[string]any{"id": "private"}}} {
		_, err := plugin.Engine.Call(t.Context(), "parseSubmitResponse", ctx, map[string]any{"body": body})
		require.Error(t, err)
	}
	assert.Equal(t, "private", yueqiHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"statusCode": 201, "body": map[string]any{"id": "private", "status": "queued"}})["taskId"])
}

func TestYueqiContentRefreshAndGatewayVideoIdentity(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	adaptor := taskplugin.New(plugin)
	adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "test-key", ChannelBaseUrl: "https://yueqi.example"}})
	task := &model.Task{
		TaskID: "task_public", Status: model.TaskStatusSuccess, Properties: model.Properties{OriginModelName: "local-alias"},
		PrivateData: model.TaskPrivateData{UpstreamTaskID: "private/id"},
		Data:        []byte(`{"id":"private/id","status":"completed","url":"https://cdn.example/expired.mp4"}`),
	}
	ctx := jsplugin.WithFetcher(t.Context(), func(_ context.Context, req jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		assert.Equal(t, "https://yueqi.example/v1/videos/private%2Fid", req.URL)
		assert.Equal(t, "Bearer test-key", req.Headers["Authorization"])
		return jsplugin.FetchResponse{Status: http.StatusOK, Body: []byte(`{"id":"private/id","status":"completed","url":"https://cdn.example/fresh.mp4?signature=current"}`)}, nil
	})
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		value, err := plugin.Engine.Call(ctx, "buildContentRequest", map[string]any{
			"artifactKey": "video", "baseUrl": "https://yueqi.example", "apiKey": "test-key",
			"upstreamTaskId": "private/id", "clientRequest": map[string]any{"method": method},
		})
		require.NoError(t, err)
		descriptor := value.(map[string]any)
		assert.Equal(t, "https://cdn.example/fresh.mp4?signature=current", descriptor["url"])
		assert.Equal(t, true, descriptor["credentialless"])
		assert.NotContains(t, descriptor, "headers", "an object-storage host must never receive the channel key")
		assert.Equal(t, http.MethodGet, descriptor["method"])
	}
	for _, body := range []string{
		`{"id":"different-task","status":"completed","url":"https://cdn.example/wrong.mp4"}`,
		`{"id":"private/id","status":"completed","storage_status":"pending"}`,
	} {
		rejectedCtx := jsplugin.WithFetcher(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
			return jsplugin.FetchResponse{Status: http.StatusOK, Body: []byte(body)}, nil
		})
		_, err := plugin.Engine.Call(rejectedCtx, "buildContentRequest", map[string]any{
			"artifactKey": "video", "baseUrl": "https://yueqi.example", "apiKey": "test-key", "upstreamTaskId": "private/id",
		})
		require.Error(t, err, "content refresh must require the same task's deliverable")
	}
	artifacts, err := adaptor.ListArtifacts(task)
	require.NoError(t, err)
	assert.Equal(t, []channel.TaskArtifact{{Key: "video", Type: "video", MimeType: "video/mp4"}}, artifacts)
	rendered, err := adaptor.ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var output map[string]any
	require.NoError(t, common.Unmarshal(rendered, &output))
	assert.Equal(t, "task_public", output["id"])
	assert.Equal(t, "local-alias", output["model"])
	assert.Equal(t, "completed", output["status"])
}

func yueqiMP4Box(name string, payload []byte) []byte {
	box := make([]byte, 8, len(payload)+8)
	binary.BigEndian.PutUint32(box, uint32(len(payload)+8))
	copy(box[4:], name)
	return append(box, payload...)
}

func yueqiReferenceVideoFixture(t *testing.T, seconds float64) []byte {
	t.Helper()
	var tracks []byte
	for _, track := range []struct {
		handler string
		seconds float64
	}{{"vide", seconds}, {"soun", 90}} {
		var duration, handler bytes.Buffer
		_, err := mp4.Marshal(&duration, &mp4.Mdhd{Timescale: 1000, DurationV0: uint32(track.seconds * 1000)}, mp4.Context{})
		require.NoError(t, err)
		_, err = mp4.Marshal(&handler, &mp4.Hdlr{HandlerType: [4]byte([]byte(track.handler))}, mp4.Context{})
		require.NoError(t, err)
		media := append(yueqiMP4Box("mdhd", duration.Bytes()), yueqiMP4Box("hdlr", handler.Bytes())...)
		tracks = append(tracks, yueqiMP4Box("trak", yueqiMP4Box("mdia", media))...)
	}
	return yueqiMP4Box("moov", tracks)
}
