package plugins_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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

func loadRunningHubH3(t *testing.T) *jsplugin.LoadedPlugin {
	t.Helper()
	source, err := builtinplugins.Source("runninghub-h3")
	require.NoError(t, err)
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "runninghub-h3"})
	require.NoError(t, err)
	return plugin
}

func runningHubContext(request map[string]any) map[string]any {
	return map[string]any{
		"model": "runninghub-minimax-h3", "upstreamModel": "runninghub-minimax-h3",
		"baseUrl": "https://www.runninghub.cn", "apiKey": "test-key", "requestBody": request,
	}
}

func runningHubHook(t *testing.T, plugin *jsplugin.LoadedPlugin, hook string, args ...any) map[string]any {
	t.Helper()
	value, err := plugin.Engine.Call(t.Context(), hook, args...)
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, common.Unmarshal(encoded, &result))
	return result
}

// Wrong node ids, stale example references, or a missing clip cap would change
// the generated content or charge less than the reference video actually used.
func TestRunningHubH3SubmissionAndUsage(t *testing.T) {
	plugin := loadRunningHubH3(t)
	request := map[string]any{
		"prompt": "waves", "seconds": 8, "resolution": "4k", "aspect_ratio": "2:3",
		"images": []any{"openapi/one.png", "openapi/two.png", "openapi/three.png", "openapi/four.png", "openapi/five.png", "openapi/six.png"},
		"audios": []any{"openapi/speech.wav"},
		"videos": []any{map[string]any{"url": "openapi/clip.mp4", "seconds": 2.5}},
	}
	ctx := runningHubContext(request)
	submit := runningHubHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "https://www.runninghub.cn/openapi/v2/run/workflow/2111027801062203393", submit["url"])
	assert.Equal(t, "POST", submit["method"])
	assert.Equal(t, "Bearer test-key", submit["headers"].(map[string]any)["Authorization"])
	body := submit["body"].(map[string]any)
	assert.Equal(t, "plus", body["instanceType"])
	assert.Equal(t, false, body["usePersonalQueue"])
	nodes := map[string]any{}
	for _, entry := range body["nodeInfoList"].([]any) {
		node := entry.(map[string]any)
		key := node["nodeId"].(string) + "." + node["fieldName"].(string)
		_, duplicate := nodes[key]
		require.False(t, duplicate, key)
		nodes[key] = node["fieldValue"]
	}
	assert.Equal(t, float64(1), nodes["252.megapixels"])
	assert.Equal(t, "2:3 (Portrait Photo)", nodes["252.aspect_ratio"])
	assert.Equal(t, float64(8), nodes["259.value"])
	assert.Equal(t, "waves", nodes["263.text"])
	assert.Equal(t, "openapi/one.png", nodes["408.image"])
	assert.Equal(t, "openapi/six.png", nodes["413.image"])
	assert.Equal(t, "openapi/speech.wav", nodes["14.audio"])
	assert.Equal(t, "openapi/clip.mp4", nodes["25.video"])
	assert.Equal(t, float64(24), nodes["25.force_rate"])
	assert.Equal(t, float64(60), nodes["25.frame_load_cap"])
	for _, field := range []string{"ref_images.ref_image_6", "ref_images.ref_image_7", "ref_images.ref_image_8", "ref_audios.ref_audio_1", "ref_audios.ref_audio_2", "ref_videos.ref_video_1", "ref_videos.ref_video_2"} {
		value, exists := nodes["265."+field]
		assert.True(t, exists, field)
		assert.Nil(t, value, field)
	}
	facts := runningHubHook(t, plugin, "extractUsage", ctx)
	assert.Equal(t, map[string]any{"seconds": float64(8), "extra_images": float64(1), "input_audios": float64(1), "input_video_seconds": 2.5}, facts)
	// Prices remain an administrator expression, independently configurable.
	cost, _, err := billingexpr.RunExprWithRequest(`tier("base", u("seconds") * 0.2 + u("extra_images") * 0.01 + u("input_audios") * 0.01 + u("input_video_seconds") * 0.1)`, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
	require.NoError(t, err)
	assert.InDelta(t, 1.87, cost, 1e-12)

	withoutVideo := maps.Clone(request)
	delete(withoutVideo, "videos")
	assert.Equal(t, "default", runningHubHook(t, plugin, "buildSubmitRequest", runningHubContext(withoutVideo))["body"].(map[string]any)["instanceType"])
	for _, count := range []int{0, 5, 9} {
		images := make([]any, count)
		for i := range images {
			images[i] = fmt.Sprintf("openapi/%d.png", i)
		}
		usage := runningHubHook(t, plugin, "extractUsage", runningHubContext(map[string]any{"images": images}))
		assert.Equal(t, float64(max(0, count-5)), usage["extra_images"])
		assert.Equal(t, float64(10), usage["seconds"])
	}
}

func TestRunningHubH3AspectRatiosAndDurationBounds(t *testing.T) {
	plugin := loadRunningHubH3(t)
	for ratio, want := range map[string]string{
		"1:1": "1:1 (Square)", "2:3": "2:3 (Portrait Photo)", "3:2": "3:2 (Photo)",
		"3:4": "3:4 (Portrait Standard)", "4:3": "4:3 (Standard)", "16:9": "16:9 (Widescreen)",
		"9:16": "9:16 (Portrait Widescreen)", "21:9": "21:9 (Ultrawide)",
	} {
		for _, seconds := range []any{3, "15", 7.5} {
			submit := runningHubHook(t, plugin, "buildSubmitRequest", runningHubContext(map[string]any{"aspect_ratio": ratio, "seconds": seconds}))
			found := false
			for _, raw := range submit["body"].(map[string]any)["nodeInfoList"].([]any) {
				node := raw.(map[string]any)
				if node["nodeId"] == "252" && node["fieldName"] == "aspect_ratio" {
					assert.Equal(t, want, node["fieldValue"])
					found = true
				}
			}
			assert.True(t, found)
		}
	}
}

func TestRunningHubH3RejectsBillingBypasses(t *testing.T) {
	plugin := loadRunningHubH3(t)
	for _, request := range []map[string]any{
		{"seconds": 0}, {"seconds": -1}, {"seconds": 2.99}, {"seconds": 15.01}, {"seconds": "1e100"}, {"seconds": true},
		{"seconds": nil}, {"seconds": 3, "duration": 15}, {"aspect_ratio": "5:4"},
		{"images": []any{""}}, {"images": []any{nil}}, {"images": "openapi/image.png"},
		{"images": make([]any, 10)}, {"audios": make([]any, 4)}, {"videos": make([]any, 4)},
		{"videos": []any{"openapi/video.mp4"}},
		{"videos": []any{map[string]any{"url": "openapi/video.mp4", "seconds": 0}}},
		{"videos": []any{map[string]any{"url": "openapi/video.mp4", "seconds": 15.01}}},
		{"videos": []any{map[string]any{"url": "openapi/video.mp4", "seconds": "0.001"}}},
		{"metadata": map[string]any{"seconds": 100}}, {"nodeInfoList": []any{}}, {"instanceType": "ultra"},
		{"workflow": "override"}, {"workflowId": "other"}, {"retainSeconds": 180}, {"webhookUrl": "https://example.com"},
	} {
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, runningHubContext(request))
			assert.Error(t, err, "%s: %#v", hook, request)
		}
	}
}

func TestRunningHubH3DecodersAndTaskLifecycle(t *testing.T) {
	plugin := loadRunningHubH3(t)
	for _, surface := range []struct{ root, hook string }{{"native", "createVideo"}, {"protocols", "openai_video"}} {
		path := []string{surface.hook}
		if surface.root == "protocols" {
			path = append(path, "decodeRequest")
		}
		value, err := plugin.Engine.CallPath(t.Context(), surface.root, path, map[string]any{
			"model": "my-h3-alias", "body": map[string]any{"kind": "json", "value": map[string]any{"model": "my-h3-alias", "seconds": "3"}},
		})
		require.NoError(t, err)
		intent := alibabaObject(t, value)
		assert.Equal(t, "my-h3-alias", intent["model"])
		ctx := runningHubContext(intent["requestBody"].(map[string]any))
		ctx["model"] = "my-h3-alias"
		assert.Equal(t, float64(3), runningHubHook(t, plugin, "extractUsage", ctx)["seconds"])
	}
	ctx := runningHubContext(map[string]any{"seconds": 3})
	submitted := runningHubHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"taskId": "2111027801062203394", "status": "RUNNING", "errorCode": ""}})
	assert.Equal(t, "2111027801062203394", submitted["taskId"])
	query := runningHubHook(t, plugin, "buildQueryRequest", map[string]any{"taskId": "private-id", "baseUrl": "https://www.runninghub.cn", "apiKey": "test-key"})
	assert.Equal(t, "https://www.runninghub.cn/openapi/v2/query", query["url"])
	assert.Equal(t, map[string]any{"taskId": "private-id"}, query["body"])
	for status, want := range map[string]string{"QUEUED": "QUEUED", "RUNNING": "IN_PROGRESS", "SUCCESS": "SUCCESS", "FAILED": "FAILURE", "FUTURE_STATUS": "UNKNOWN", "": "UNKNOWN"} {
		data := map[string]any{"taskId": "private-id", "status": status, "errorMessage": "provider failure", "results": []any{map[string]any{"url": "https://media.example/video.mp4", "nodeId": "418", "outputType": "mp4"}}}
		result := runningHubHook(t, plugin, "parseTaskResult", map[string]any{"taskId": "private-id"}, data)
		assert.Equal(t, want, result["status"], status)
		if want == "FAILURE" {
			assert.Equal(t, "provider failure", result["reason"])
		}
	}
	result := runningHubHook(t, plugin, "parseTaskResult", map[string]any{}, map[string]any{"status": "SUCCESS", "results": []any{}})
	assert.Equal(t, "UNKNOWN", result["status"], "missing video must not settle as a successful generation")
	_, err := plugin.Engine.Call(t.Context(), "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"taskId": "id", "errorCode": "BAD_REQUEST", "errorMessage": "rejected"}})
	assert.ErrorContains(t, err, "rejected")
	_, err = plugin.Engine.Call(t.Context(), "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"status": "RUNNING"}})
	assert.Error(t, err)
	// RunningHub taskCostTime is compute time, never output video duration.
	value, err := plugin.Engine.Call(t.Context(), "extractUsageOnComplete", map[string]any{}, map[string]any{"status": "SUCCESS"}, map[string]any{"usage": map[string]any{"taskCostTime": "800"}})
	require.NoError(t, err)
	assert.Nil(t, value)
}

// Multipart URL fields must produce the same nodes and billing quantities as
// JSON, preserving signed OSS query strings and the order of reference media.
func TestRunningHubH3MultipartURLReferences(t *testing.T) {
	plugin := loadRunningHubH3(t)
	const imageURL = "https://assets.oss-cn-hangzhou.aliyuncs.com/image.png?Signature=a%2Bb%3D&Expires=9999999999"
	for _, surface := range []struct {
		name, root string
		path       []string
	}{
		{"native", "native", []string{"createVideo"}},
		{"OpenAI video", "protocols", []string{"openai_video", "decodeRequest"}},
	} {
		for _, tc := range []struct {
			name    string
			fields  map[string][]string
			request map[string]any
			facts   map[string]any
		}{
			{
				name: "single signed OSS URL",
				fields: map[string][]string{
					"model": {"my-h3-alias"}, "prompt": {"waves"}, "seconds": {"5"}, "input_reference": {imageURL},
				},
				request: map[string]any{"model": "my-h3-alias", "prompt": "waves", "seconds": "5", "input_reference": imageURL},
				facts:   map[string]any{"seconds": float64(5), "extra_images": float64(0), "input_audios": float64(0), "input_video_seconds": float64(0)},
			},
			{
				name: "repeated references",
				fields: map[string][]string{
					"seconds": {"8"}, "images": {imageURL, "openapi/2.png", "openapi/3.png", "openapi/4.png", "openapi/5.png", "openapi/6.png"},
					"audios": {"https://assets.oss-cn-hangzhou.aliyuncs.com/audio.wav", "openapi/second.wav"},
					"videos": {`{"url":"openapi/first.mp4","seconds":2.5}`, `{"url":"openapi/second.mp4","seconds":3}`},
				},
				request: map[string]any{
					"seconds": "8", "images": []any{imageURL, "openapi/2.png", "openapi/3.png", "openapi/4.png", "openapi/5.png", "openapi/6.png"},
					"audios": []any{"https://assets.oss-cn-hangzhou.aliyuncs.com/audio.wav", "openapi/second.wav"},
					"videos": []any{map[string]any{"url": "openapi/first.mp4", "seconds": float64(2.5)}, map[string]any{"url": "openapi/second.mp4", "seconds": float64(3)}},
				},
				facts: map[string]any{"seconds": float64(8), "extra_images": float64(1), "input_audios": float64(2), "input_video_seconds": 5.5},
			},
			{
				name: "JSON arrays in text fields",
				fields: map[string][]string{
					"duration": {"7.5"}, "aspect_ratio": {"9:16"}, "resolution": {"1080p"},
					"images": {`["openapi/1.png","openapi/2.png"]`}, "audios": {`[]`},
					"videos": {`[{"url":"https://assets.oss-cn-hangzhou.aliyuncs.com/video.mp4?Signature=a%2Bb","seconds":4}]`},
				},
				request: map[string]any{
					"duration": "7.5", "aspect_ratio": "9:16", "resolution": "1080p", "images": []any{"openapi/1.png", "openapi/2.png"}, "audios": []any{},
					"videos": []any{map[string]any{"url": "https://assets.oss-cn-hangzhou.aliyuncs.com/video.mp4?Signature=a%2Bb", "seconds": float64(4)}},
				},
				facts: map[string]any{"seconds": 7.5, "extra_images": float64(0), "input_audios": float64(0), "input_video_seconds": float64(4)},
			},
		} {
			t.Run(surface.name+"/"+tc.name, func(t *testing.T) {
				value, err := plugin.Engine.CallPath(t.Context(), surface.root, surface.path, map[string]any{
					"model": "my-h3-alias", "body": map[string]any{"kind": "multipart", "fields": tc.fields, "files": []any{}},
				})
				require.NoError(t, err)
				intent := alibabaObject(t, value)
				assert.Equal(t, "my-h3-alias", intent["model"])
				request := intent["requestBody"].(map[string]any)
				assert.Equal(t, tc.request, request)
				ctx := runningHubContext(request)
				assert.Equal(t, tc.facts, runningHubHook(t, plugin, "extractUsage", ctx))
				assert.Equal(t,
					runningHubHook(t, plugin, "buildSubmitRequest", runningHubContext(tc.request)),
					runningHubHook(t, plugin, "buildSubmitRequest", ctx),
				)
			})
		}
	}
}

func TestRunningHubH3MultipartRejectsInvalidFields(t *testing.T) {
	plugin := loadRunningHubH3(t)
	for _, tc := range []struct {
		name   string
		fields map[string][]string
		files  []any
		want   string
	}{
		{name: "duplicate scalar", fields: map[string][]string{"seconds": {"3", "15"}}, want: "seconds must be provided once"},
		{name: "duplicate input reference", fields: map[string][]string{"input_reference": {"openapi/1.png", "openapi/2.png"}}, want: "input_reference must be provided once"},
		{name: "malformed image array", fields: map[string][]string{"images": {`["broken"`}}, want: "images must contain valid JSON"},
		{name: "malformed video JSON", fields: map[string][]string{"videos": {`{"url":`}}, want: "videos must contain valid JSON"},
		{name: "bare video URL", fields: map[string][]string{"videos": {"https://assets.oss-cn-hangzhou.aliyuncs.com/video.mp4"}}, want: "videos must contain valid JSON"},
		{name: "unexpected binary field", files: []any{map[string]any{"ref": "request_file:other", "field": "other", "filename": "image.png", "mimeType": "image/png", "size": 3}}, want: "unsupported file field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
				"model": "runninghub-minimax-h3", "body": map[string]any{"kind": "multipart", "fields": tc.fields, "files": tc.files},
			})
			require.ErrorContains(t, err, tc.want)
		})
	}
	for _, fields := range []map[string][]string{
		{"seconds": {"0"}}, {"seconds": {"16"}}, {"seconds": {"1e100"}},
		{"seconds": {"3"}, "duration": {"15"}}, {"images": {`["",null]`}},
		{"images": {`["1","2","3","4","5","6","7","8","9","10"]`}},
		{"audios": {`["1","2","3","4"]`}},
		{"videos": {`[{"url":"openapi/video.mp4","seconds":0}]`}},
		{"videos": {`[{"url":"openapi/video.mp4","seconds":1000000000000}]`}},
		{"metadata": {`{"seconds":100}`}}, {"nodeInfoList": {`[]`}}, {"__proto__": {`ignored`}},
	} {
		value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
			"model": "runninghub-minimax-h3", "body": map[string]any{"kind": "multipart", "fields": fields, "files": []any{}},
		})
		require.NoError(t, err)
		intent := alibabaObject(t, value)
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, runningHubContext(intent["requestBody"].(map[string]any)))
			assert.Error(t, err, "%s: %#v", hook, fields)
		}
	}
}

// The client's input.media/parameters envelope must control the same nodes and
// bounded billing quantities as the original top-level request format.
func TestRunningHubH3NestedVideoRequest(t *testing.T) {
	plugin := loadRunningHubH3(t)
	const imageURL = "https://assets.oss-cn-shanghai.aliyuncs.com/image.png?Expires=9999999999&Signature=a%2Bb%3D"
	const prompt = "以图片1作为店铺外观的视觉依据，生成水果店购物视频"
	request := map[string]any{
		"model": "runninghub-minimax-h3",
		"input": map[string]any{
			"prompt": prompt,
			"media":  []any{map[string]any{"type": "first_frame", "url": imageURL}},
		},
		"parameters": map[string]any{"duration": 8, "ratio": "16:9", "resolution": "768p", "prompt_extend": false, "watermark": false},
	}
	inputText, err := common.Marshal(request["input"])
	require.NoError(t, err)
	parametersText, err := common.Marshal(request["parameters"])
	require.NoError(t, err)
	for _, surface := range []struct {
		name, root string
		path       []string
	}{
		{"native", "native", []string{"createVideo"}},
		{"OpenAI video", "protocols", []string{"openai_video", "decodeRequest"}},
	} {
		for kind, body := range map[string]any{
			"JSON": map[string]any{"kind": "json", "value": request},
			"multipart": map[string]any{"kind": "multipart", "files": []any{}, "fields": map[string][]string{
				"model": {"runninghub-minimax-h3"}, "input": {string(inputText)}, "parameters": {string(parametersText)},
			}},
		} {
			t.Run(surface.name+"/"+kind, func(t *testing.T) {
				value, err := plugin.Engine.CallPath(t.Context(), surface.root, surface.path, map[string]any{"model": "runninghub-minimax-h3", "body": body})
				require.NoError(t, err)
				intent := alibabaObject(t, value)
				ctx := runningHubContext(intent["requestBody"].(map[string]any))
				submit := runningHubHook(t, plugin, "buildSubmitRequest", ctx)
				upstream := submit["body"].(map[string]any)
				assert.Equal(t, "default", upstream["instanceType"])
				nodes := map[string]any{}
				for _, raw := range upstream["nodeInfoList"].([]any) {
					node := raw.(map[string]any)
					nodes[node["nodeId"].(string)+"."+node["fieldName"].(string)] = node["fieldValue"]
				}
				assert.Equal(t, imageURL, nodes["408.image"])
				assert.Equal(t, prompt, nodes["263.text"])
				assert.Equal(t, float64(8), nodes["259.value"])
				assert.Equal(t, float64(1), nodes["252.megapixels"])
				assert.Equal(t, "16:9 (Widescreen)", nodes["252.aspect_ratio"])
				assert.Equal(t, map[string]any{"seconds": float64(8), "extra_images": float64(0), "input_audios": float64(0), "input_video_seconds": float64(0)}, runningHubHook(t, plugin, "extractUsage", ctx))
			})
		}
	}
	media := []any{
		map[string]any{"type": "reference_image", "url": "openapi/2.png"},
		map[string]any{"type": "first_frame", "url": imageURL},
		map[string]any{"type": "reference_image", "url": "openapi/3.png"},
		map[string]any{"type": "reference_image", "url": "openapi/4.png"},
		map[string]any{"type": "image", "url": "openapi/5.png"},
		map[string]any{"type": "image", "url": "openapi/6.png"},
		map[string]any{"type": "reference_audio", "url": "openapi/audio.wav"},
		map[string]any{"type": "reference_video", "url": "openapi/video.mp4", "duration": 3.25},
	}
	ctx := runningHubContext(map[string]any{"input": map[string]any{"media": media}, "parameters": map[string]any{"seconds": 12, "ratio": "9:16", "resolution": "4k"}})
	submit := runningHubHook(t, plugin, "buildSubmitRequest", ctx)
	body := submit["body"].(map[string]any)
	assert.Equal(t, "plus", body["instanceType"])
	for _, raw := range body["nodeInfoList"].([]any) {
		node := raw.(map[string]any)
		if node["nodeId"] == "408" && node["fieldName"] == "image" {
			assert.Equal(t, imageURL, node["fieldValue"], "first_frame is the workflow's first image reference")
		}
		if node["nodeId"] == "25" && node["fieldName"] == "frame_load_cap" {
			assert.Equal(t, float64(78), node["fieldValue"])
		}
	}
	assert.Equal(t, map[string]any{"seconds": float64(12), "extra_images": float64(1), "input_audios": float64(1), "input_video_seconds": 3.25}, runningHubHook(t, plugin, "extractUsage", ctx))
}

func TestRunningHubH3NestedRequestRejectsInvalidOverrides(t *testing.T) {
	plugin := loadRunningHubH3(t)
	for _, request := range []map[string]any{
		{"input": nil}, {"input": []any{}}, {"input": `{"prompt":`},
		{"parameters": nil}, {"parameters": "[]"},
		{"parameters": map[string]any{"duration": 0}}, {"parameters": map[string]any{"duration": 16}},
		{"seconds": 3, "parameters": map[string]any{"duration": 15}},
		{"duration": 3, "parameters": map[string]any{"duration": 8}},
		{"aspect_ratio": "9:16", "parameters": map[string]any{"ratio": "16:9"}},
		{"prompt": "top", "input": map[string]any{"prompt": "nested"}},
		{"images": []any{"openapi/1.png"}, "input": map[string]any{"media": []any{map[string]any{"type": "image", "url": "openapi/2.png"}}}},
		{"input": map[string]any{"media": "openapi/1.png"}},
		{"input": map[string]any{"media": []any{nil}}},
		{"input": map[string]any{"media": []any{map[string]any{"type": "unknown", "url": "openapi/1.png"}}}},
		{"input": map[string]any{"media": []any{map[string]any{"type": "image", "url": ""}}}},
		{"input": map[string]any{"media": []any{map[string]any{"type": "video", "url": "openapi/1.mp4", "seconds": 16}}}},
		{"input": map[string]any{"media": []any{map[string]any{"type": "video", "url": "openapi/1.mp4", "seconds": 3, "duration": 15}}}},
		{"input": map[string]any{"media": []any{map[string]any{"type": "first_frame", "url": "openapi/1.png"}, map[string]any{"type": "first_frame", "url": "openapi/2.png"}}}},
		{"parameters": map[string]any{"prompt_extend": true}}, {"parameters": map[string]any{"watermark": true}},
		{"parameters": map[string]any{"prompt_extend": "false"}},
		{"parameters": map[string]any{"nodeInfoList": []any{}}}, {"input": map[string]any{"workflowId": "other"}},
	} {
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, runningHubContext(request))
			assert.Error(t, err, "%s: %#v", hook, request)
		}
	}
	for _, tc := range []struct {
		mediaType string
		count     int
	}{{"reference_image", 10}, {"reference_audio", 4}, {"reference_video", 4}} {
		media := make([]any, tc.count)
		for i := range media {
			entry := map[string]any{"type": tc.mediaType, "url": fmt.Sprintf("openapi/%d", i)}
			if tc.mediaType == "reference_video" {
				entry["seconds"] = 3
			}
			media[i] = entry
		}
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, runningHubContext(map[string]any{"input": map[string]any{"media": media}}))
			assert.Error(t, err, "%s: %s", hook, tc.mediaType)
		}
	}
}

// The OpenAI client downloads an OSS reference and sends real multipart file
// bytes. Upload those once and submit RunningHub's fileName, not a data URI.
func TestRunningHubH3OpenAIFileUpload(t *testing.T) {
	plugin := loadRunningHubH3(t)
	image := []byte("\x89PNG\r\n\x1a\nreference image")
	uploads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploads++
		assert.Equal(t, "/openapi/v2/media/upload/binary", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		require.NoError(t, r.ParseMultipartForm(1<<20))
		defer r.MultipartForm.RemoveAll()
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, "input-reference.png", header.Filename)
		assert.Equal(t, "image/png", header.Header.Get("Content-Type"))
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, image, data)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"code":0,"message":"success","data":{"fileName":"openapi/uploaded.png","download_url":"https://cdn.example/uploaded.png","type":"image","size":"23"}}`)
		assert.NoError(t, err)
	}))
	defer server.Close()
	fields := map[string][]string{"model": {"runninghub-minimax-h3"}, "prompt": {"水果店实拍"}, "seconds": {"8"}, "resolution_name": {"768p"}, "size": {"16:9"}}
	files := []any{map[string]any{"ref": "request_file:input_reference", "field": "input_reference", "filename": "input-reference.png", "mimeType": "image/png", "size": len(image)}}
	value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
		"model": "runninghub-minimax-h3", "body": map[string]any{"kind": "multipart", "fields": fields, "files": files},
	})
	require.NoError(t, err)
	intent := alibabaObject(t, value)
	var input bytes.Buffer
	writer := multipart.NewWriter(&input)
	for name, values := range fields {
		for _, text := range values {
			require.NoError(t, writer.WriteField(name, text))
		}
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="input_reference"; filename="input-reference.png"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(image)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &input)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("task_request", intent["requestBody"])
	info := &relaycommon.RelayInfo{
		OriginModelName: "runninghub-minimax-h3", TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeTaskPlugin, UpstreamModelName: "runninghub-minimax-h3", ChannelBaseUrl: server.URL, ApiKey: "test-key"},
	}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	reader, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	var upstream struct {
		Nodes []map[string]any `json:"nodeInfoList"`
	}
	require.NoError(t, common.DecodeJson(reader, &upstream))
	found := false
	for _, node := range upstream.Nodes {
		if node["nodeId"] == "408" && node["fieldName"] == "image" {
			assert.Equal(t, "openapi/uploaded.png", node["fieldValue"])
			found = true
		}
	}
	assert.True(t, found)
	facts, err := adaptor.ExtractUsageFactsValidated(c, info)
	require.NoError(t, err)
	assert.Equal(t, float64(8), facts["seconds"])
	assert.Equal(t, float64(0), facts["extra_images"])
	assert.Equal(t, 1, uploads, "usage extraction and request serialization must not upload the reference again")
}

func TestTaskPluginFileUploadHookBoundary(t *testing.T) {
	engine, err := jsplugin.Compile(`
export function buildSubmitRequest(request) { return utils.uploadFile(request); }
export function extractUsage(request) { return utils.uploadFile(request); }
export function read(request) { return utils.fetch(request); }
`, jsplugin.Options{Key: "upload-boundary"})
	require.NoError(t, err)
	request := map[string]any{"url": "https://api.example/upload", "fileRef": "request_file:input_reference", "headers": map[string]any{"Authorization": "Bearer example"}}
	_, err = engine.Call(t.Context(), "buildSubmitRequest", request)
	require.ErrorContains(t, err, "not available")
	ctx := jsplugin.WithFileUploader(t.Context(), func(_ context.Context, request jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "request_file:input_reference", request.FileRef)
		assert.Equal(t, "Bearer example", request.Headers["Authorization"])
		return jsplugin.FetchResponse{Status: http.StatusCreated, Body: []byte(`{"fileName":"openapi/image.png"}`)}, nil
	})
	value, err := engine.Call(ctx, "buildSubmitRequest", request)
	require.NoError(t, err)
	assert.Equal(t, float64(201), alibabaObject(t, value)["status"])
	_, err = engine.Call(ctx, "extractUsage", request)
	require.ErrorContains(t, err, "not available in extractUsage")
	_, err = engine.Call(jsplugin.WithFetcher(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		t.Fatal("read-only fetch must never issue POST")
		return jsplugin.FetchResponse{}, nil
	}), "read", map[string]any{"url": "https://api.example/upload", "method": "POST"})
	require.ErrorContains(t, err, "only GET and HEAD")
	for _, invalid := range []map[string]any{
		{"url": "https://api.example/upload", "fileRef": "C:/private/file.png"},
		{"url": "https://api.example/upload", "fileRef": "request_file:x", "body": "unexpected"},
		{"url": "file:///private/file.png", "fileRef": "request_file:x"},
		{"url": "https://api.example/upload", "fileRef": "request_file:x", "headers": map[string]any{"Host": "other.example"}},
	} {
		_, err := engine.Call(jsplugin.WithFileUploader(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
			t.Fatal("invalid upload reached the transport")
			return jsplugin.FetchResponse{}, nil
		}), "buildSubmitRequest", invalid)
		require.Error(t, err)
	}
	oversized := jsplugin.WithFileUploader(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		return jsplugin.FetchResponse{Status: 200, Body: bytes.Repeat([]byte("x"), jsplugin.MaxFetchBodyBytes+1)}, nil
	})
	_, err = engine.Call(oversized, "buildSubmitRequest", request)
	require.ErrorContains(t, err, "exceeds 1048576 bytes")
}

func TestRunningHubH3UploadRejectsInvalidFiles(t *testing.T) {
	plugin := loadRunningHubH3(t)
	for _, tc := range []struct {
		name, mimeType string
		size           int
		files          int
	}{
		{"empty", "image/png", 0, 1},
		{"oversized", "image/png", (64 << 20) + 1, 1},
		{"wrong media type", "video/mp4", 10, 1},
		{"duplicate image", "image/png", 10, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := make([]any, tc.files)
			for i := range files {
				files[i] = map[string]any{"field": "input_reference", "ref": jsplugin.FileReference("input_reference", i), "filename": "image.png", "mimeType": tc.mimeType, "size": tc.size}
			}
			_, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
				"model": "runninghub-minimax-h3", "body": map[string]any{"kind": "multipart", "fields": map[string][]string{}, "files": files},
			})
			require.Error(t, err)
		})
	}
	for _, request := range []map[string]any{
		{"input_reference": map[string]any{"__fileRef": "request_file:input_reference", "encoding": "dataUrl"}},
		{"input_reference": map[string]any{"__fileRef": "C:/private/file.png", "encoding": "dataUrl"}},
	} {
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", runningHubContext(request))
		require.ErrorContains(t, err, "invalid uploaded file reference")
	}
	// A provider upload failure never proceeds to billable workflow creation.
	uploadContext := runningHubContext(map[string]any{"input_reference": map[string]any{"__fileRef": "request_file:input_reference", "encoding": "dataUrl"}})
	uploadContext["files"] = []any{map[string]any{"field": "input_reference", "ref": "request_file:input_reference", "filename": "image.png", "mimeType": "image/png", "size": 3}}
	for _, response := range []jsplugin.FetchResponse{
		{Status: 403, Body: []byte(`{"code":401,"message":"denied"}`)},
		{Status: 200, Body: []byte(`{"code":0,"data":{"download_url":"https://cdn.example/image.png"}}`)},
		{Status: 200, Body: []byte(`{"code":0,"data":{"fileName":""}}`)},
	} {
		ctx := jsplugin.WithFileUploader(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) { return response, nil })
		_, err := plugin.Engine.Call(ctx, "buildSubmitRequest", uploadContext)
		require.ErrorContains(t, err, "RunningHub media upload failed")
	}
	// Keep parsing errors free of signed OSS URLs and credential-bearing values.
	value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
		"model": "runninghub-minimax-h3", "body": map[string]any{"kind": "multipart", "fields": map[string][]string{"input": {strings.Repeat("private", 10)}}, "files": []any{}},
	})
	assert.Nil(t, value)
	require.ErrorContains(t, err, "input must contain a JSON object")
	assert.NotContains(t, err.Error(), "private")
}

func TestTaskPluginFileUploadTransportBoundary(t *testing.T) {
	const source = `
export const meta = {apiVersion:1,key:"upload-probe",name:"Upload probe",version:"1.0.0",author:{name:"Test"},models:["upload-probe"],fetchMode:"per_task"};
export function buildSubmitRequest(ctx) {
  const response = utils.uploadFile({url:ctx.requestBody.target || ctx.baseUrl+"/upload",fileRef:ctx.requestBody.ref || ctx.files[0].ref,headers:{Authorization:"Bearer upload-key"}});
  return {url:ctx.baseUrl+"/create",method:"POST",body:{uploadStatus:response.status}};
}
export function parseSubmitResponse(){return {taskId:"private"};}
export function buildQueryRequest(ctx){return {url:ctx.baseUrl+"/query"};}
export function parseTaskResult(){return {status:"QUEUED"};}
`
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: "upload-probe"})
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		size     int
		request  map[string]any
		redirect bool
		wantErr  string
	}{
		{name: "unknown reference", size: 3, request: map[string]any{"ref": "request_file:missing"}, wantErr: "unknown uploaded file reference"},
		{name: "oversized file", size: (1 << 20) + 1, wantErr: "byte limit"},
		{name: "untrusted target", size: 3, request: map[string]any{"target": "https://untrusted.example/upload"}, wantErr: "not allowed"},
		{name: "redirect is returned without replay", size: 3, redirect: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previous := constant.MaxFileDownloadMB
			constant.MaxFileDownloadMB = 1
			t.Cleanup(func() { constant.MaxFileDownloadMB = previous })
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Equal(t, "/upload", r.URL.Path)
				if tc.redirect {
					w.Header().Set("Location", "/unexpected")
					w.WriteHeader(http.StatusTemporaryRedirect)
				}
			}))
			defer server.Close()
			var input bytes.Buffer
			writer := multipart.NewWriter(&input)
			part, err := writer.CreateFormFile("input_reference", "image.png")
			require.NoError(t, err)
			_, err = part.Write(bytes.Repeat([]byte("x"), tc.size))
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &input)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())
			request := tc.request
			if request == nil {
				request = map[string]any{}
			}
			c.Set("task_request", request)
			info := &relaycommon.RelayInfo{
				OriginModelName: "upload-probe", TaskRelayInfo: &relaycommon.TaskRelayInfo{},
				ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeTaskPlugin, UpstreamModelName: "upload-probe", ChannelBaseUrl: server.URL},
			}
			adaptor := taskplugin.New(plugin)
			adaptor.Init(info)
			taskErr := adaptor.ValidateRequestAndSetAction(c, info)
			if tc.wantErr != "" {
				require.NotNil(t, taskErr)
				assert.ErrorContains(t, taskErr.Error, tc.wantErr)
				assert.Zero(t, calls, "rejected uploads must not reach a network target")
				return
			}
			require.Nil(t, taskErr)
			assert.Equal(t, 1, calls)
			reader, err := adaptor.BuildRequestBody(c, info)
			require.NoError(t, err)
			var body map[string]any
			require.NoError(t, common.DecodeJson(reader, &body))
			assert.Equal(t, float64(307), body["uploadStatus"])
		})
	}
}

func TestRunningHubH3Artifacts(t *testing.T) {
	plugin := loadRunningHubH3(t)
	data := map[string]any{"status": "SUCCESS", "results": []any{
		map[string]any{"url": "https://media.example/preview.png", "nodeId": "1", "outputType": "png"},
		map[string]any{"url": "https://media.example/video.mp4", "nodeId": "418", "outputType": "mp4"},
	}}
	value, err := plugin.Engine.Call(t.Context(), "listArtifacts", map[string]any{"status": "SUCCESS", "data": data})
	require.NoError(t, err)
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"key":"video","type":"video","mimeType":"video/mp4"}]`, string(encoded))
	content := runningHubHook(t, plugin, "buildContentRequest", map[string]any{"artifactKey": "video", "data": data, "clientRequest": map[string]any{"method": "HEAD"}})
	assert.Equal(t, map[string]any{"url": "https://media.example/video.mp4", "method": "HEAD", "credentialless": true}, content)
	_, err = plugin.Engine.Call(t.Context(), "buildContentRequest", map[string]any{"artifactKey": "other", "data": data})
	assert.ErrorContains(t, err, "artifact_not_found")
	data["results"] = []any{map[string]any{"url": "https://media.example/video.webm", "nodeId": "418", "outputType": "webm"}}
	value, err = plugin.Engine.Call(t.Context(), "listArtifacts", map[string]any{"status": "SUCCESS", "data": data})
	require.NoError(t, err)
	encoded, err = common.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"key":"video","type":"video","mimeType":"video/webm"}]`, string(encoded))
}

func TestRunningHubH3NativeTaskIsQueryable(t *testing.T) {
	plugin := loadRunningHubH3(t)
	for status, want := range map[string]string{"SUBMITTED": "queued", "QUEUED": "queued", "IN_PROGRESS": "in_progress", "SUCCESS": "completed", "FAILURE": "failed"} {
		value, err := plugin.Engine.CallPath(t.Context(), "native", []string{"video"}, map[string]any{}, map[string]any{
			"task_id": "task_public", "status": status, "progress": "50%", "created_at": 100, "updated_at": 200,
			"properties": map[string]any{"origin_model_name": "my-h3-alias"},
			"data":       map[string]any{"taskId": "private-id", "results": []any{map[string]any{"nodeId": "418", "outputType": "mp4", "url": "https://media.example/video.mp4"}}},
		})
		require.NoError(t, err)
		video := alibabaObject(t, value)
		assert.Equal(t, "task_public", video["id"], "the client needs this id to poll its task")
		assert.Equal(t, want, video["status"])
		assert.Equal(t, "my-h3-alias", video["model"])
		assert.Equal(t, "768p", video["resolution"])
		if status == "SUCCESS" {
			assert.Equal(t, "https://media.example/video.mp4", video["url"])
		} else {
			assert.NotContains(t, video, "url")
			assert.NotContains(t, video, "completed_at")
		}
	}
}

// Null slot values must survive the actual host serializer, not merely the JS
// hook result, or the workflow can keep loading its saved example references.
func TestRunningHubH3HostSerializesDisconnectedReferences(t *testing.T) {
	plugin := loadRunningHubH3(t)
	adaptor := taskplugin.New(plugin)
	info := &relaycommon.RelayInfo{
		OriginModelName: "my-h3-alias",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeTaskPlugin, UpstreamModelName: "runninghub-minimax-h3",
			ChannelBaseUrl: "https://www.runninghub.cn", ApiKey: "test-key",
		},
	}
	adaptor.Init(info)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", map[string]any{"model": "my-h3-alias", "seconds": 5, "images": []any{"openapi/image.png"}})
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	reader, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	var body struct {
		Nodes []map[string]any `json:"nodeInfoList"`
	}
	require.NoError(t, common.DecodeJson(reader, &body))
	var disconnected int
	for _, node := range body.Nodes {
		if node["nodeId"] == "265" {
			assert.Contains(t, node, "fieldValue")
			assert.Nil(t, node["fieldValue"])
			disconnected++
		}
	}
	assert.Equal(t, 14, disconnected)
	facts, err := adaptor.ExtractUsageFactsValidated(c, info)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"extra_images": float64(0), "input_audios": float64(0), "input_video_seconds": float64(0), "seconds": float64(5)}, facts)
	assert.True(t, adaptor.DisableSubmitRetry(), "accepted RunningHub submissions must not be duplicated on retries")
}
