package plugins_test

import (
	"context"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	require.Len(t, plugin.Meta.Models, 74)
	assert.Contains(t, plugin.Meta.RequiredCapabilities, "submit-no-retry@1")
	assert.True(t, plugin.Meta.SupportsUpstream(jsplugin.UpstreamKindNewAPI))
	priced := 0
	for _, modelName := range plugin.Meta.Models {
		assert.False(t, strings.HasPrefix(modelName, "lec-"), "vendor prefixes are private")
		defaults, found := plugin.Meta.ModelDefaults[modelName]
		require.True(t, found)
		if defaults.BillingExpr != "" {
			priced++
		}
		for _, description := range defaults.Description {
			assert.NotContains(t, description, "lec-")
		}
	}
	assert.Equal(t, 70, priced)
	for _, tc := range []struct{ name, vendor string }{
		{"aj-wan-3-0", "Wan"}, {"ms-minimax-h3", "Minimax"}, {"ac-image-2", "OpenAI"},
		{"ac-seedream-5-0-lite", "ByteDance"}, {"gt-seedance-2-5-720p", "ByteDance"},
		{"jm-seedance-2-5-flash", "ByteDance"},
		{"nano-banana-2", "Google"}, {"sl-midjourney", "Midjourney"}, {"mf-video-upscale", ""},
	} {
		assert.Equal(t, tc.vendor, plugin.Meta.ModelDefaults[tc.name].Vendor, tc.name)
	}
	schema, _ := plugin.Meta.UsageForModel("aj-wan-3-0")
	assert.ElementsMatch(t, []string{"480p", "720p", "1080p"}, schema["resolution"].Enum)
	for _, name := range []string{"image2", "nano-banana-2"} {
		facts := lecAPIHook(t, plugin, "extractUsage", lecAPIContext(name, map[string]any{"prompt": "boat"}))
		schema, _ := plugin.Meta.UsageForModel(name)
		assert.Contains(t, schema["resolution"].Enum, facts["resolution"], "omitting an optional resolution must remain valid for %s", name)
	}
	assert.Contains(t, plugin.Meta.Models, "MiniMax-H3")
	for _, spelling := range []string{"MiniMax-H3", "minimax-h3"} {
		canonical, found := registry.Generation().CanonicalModel(spelling)
		require.True(t, found)
		assert.Equal(t, "MiniMax-H3", canonical)
		binding, found := registry.Generation().LookupEndpoint("POST", "/v1/videos", canonical)
		require.True(t, found)
		assert.Equal(t, "lecapi", binding.Plugin.Meta.Key)
	}
	imageModels := []string{"ac-image-2-5-flare", "ac-image-2-5-sunburst", "ac-image-2", "nw-image-2", "nw-image-2-5", "nw-image-2-6", "nw-nano-banana-2-1", "sl-midjourney", "sl-midjourney-niji", "ac-seedream-5-0-lite", "ac-seedream-5-0-pro", "ty-seedream-5-pro", "tinysnow-banana-3-1", "tinysnow-banana-3", "tinysnow-image-2", "tinysnow-image-2-5-flare", "tinysnow-image-2-5-sunburst", "image-2", "image2", "nano-banana-2"}
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
		{"ac-image-2-5-flare", "/v1/images/generations", "openai_image"},
		{"ty-seedream-5-pro", "/v1/images/generations", "openai_image"},
		{"image2", "/v1/images/generations", "openai_image"},
		{"nano-banana-2", "/v1/images/generations", "openai_image"},
		{"vp-seedance-2-9-301010", "/v1/videos", "openai_video"},
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
		{"aj-wan-3-0", "lec-aj-wan-3-0"}, {"ac-image-2", "lec-ac-image-2"}, {"image-2", "lec-image-2"}, {"image2", "image2"}, {"nano-banana-2", "nano-banana-2"},
		{"jm-seedance-2-5-flash", "lec-jm-seedance-2-5-flash"},
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
	variant := lecAPIContext("minimax-h3", map[string]any{"model": "minimax-h3", "prompt": "boat", "images": []any{"https://media.example/a.png"}})
	assert.Equal(t, "lec-minimax-h3", lecAPIHook(t, plugin, "buildSubmitRequest", variant)["body"].(map[string]any)["model"])
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

func TestLecAPISeedanceFlashBuildsAndMetersDocumentedRequests(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	images, audios := make([]any, 10), make([]any, 5)
	for i := range images {
		images[i] = "https://media.example/reference.png"
	}
	for i := range audios {
		audios[i] = "https://media.example/reference.mp3"
	}
	ctx := lecAPIContext("jm-seedance-2-5-flash", map[string]any{"prompt": "A red boat", "metadata": map[string]any{"parameters": map[string]any{
		"seconds": "4", "resolution_name": "480P", "ratio": "21:9", "image_urls": images, "audio_urls": audios,
	}}})
	ctx["model"] = "canvas-flash-alias"
	submit := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "https://lec.example/v1/videos", submit["url"])
	body := submit["body"].(map[string]any)
	assert.Equal(t, "lec-jm-seedance-2-5-flash", body["model"])
	assert.Equal(t, float64(4), body["duration"])
	assert.Equal(t, "480p", body["resolution"])
	assert.Equal(t, "21:9", body["aspect_ratio"])
	assert.Len(t, body["images"], 10)
	assert.Len(t, body["audios"], 5)
	facts := lecAPIHook(t, plugin, "extractUsage", ctx)
	assert.Equal(t, body["duration"], facts["seconds"])
	assert.Equal(t, body["resolution"], facts["resolution"])
	parsed := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"id": "flash/id", "status": "queued"}})
	assert.Equal(t, "flash/id", parsed["taskId"])
	assert.NotContains(t, parsed, "immediate")
	query := maps.Clone(ctx)
	query["taskId"] = parsed["taskId"]
	assert.Equal(t, "https://lec.example/v1/videos/flash%2Fid", lecAPIHook(t, plugin, "buildQueryRequest", query)["url"])
	completed := lecAPIHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 30})
	assert.Equal(t, float64(30), completed["seconds"])
	for _, invalid := range []map[string]any{
		{"duration": 3}, {"duration": 31}, {"duration": 4.5}, {"parameters": map[string]any{"duration": 9007199254740992}},
		{"resolution": "1080p"}, {"aspect_ratio": "3:2"}, {"n": 2},
		{"videos": []any{"https://media.example/reference.mp4"}}, {"extra": map[string]any{"video_url": "https://media.example/reference.mp4"}},
		{"images": append(images, "https://media.example/extra.png")}, {"audios": append(audios, "https://media.example/extra.mp3")},
		{"images": []any{"http://media.example/reference.png"}},
	} {
		request := maps.Clone(invalid)
		request["prompt"] = "A red boat"
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, lecAPIContext("jm-seedance-2-5-flash", request))
			require.Error(t, err, "%s must reject unsupported Flash parameters: %v", hook, invalid)
		}
	}
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
		{"image2", "3840x2160", "4K"}, {"nano-banana-2", "3840x2160", "4K"},
		{"aj-wan-3-0", "1080p", "1080p"},
	} {
		request := map[string]any{"prompt": "boat", "size": tc.size}
		if tc.model == "image2" || tc.model == "nano-banana-2" {
			request["resolution"] = "4k"
		}
		ctx := lecAPIContext(tc.model, request)
		body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
		assert.Equal(t, tc.resolution, body["resolution"], "%s must retain its resolution rules", tc.model)
		assert.Equal(t, tc.resolution, lecAPIHook(t, plugin, "extractUsage", ctx)["resolution"])
		if tc.model == "image2" || tc.model == "nano-banana-2" {
			assert.Equal(t, "16:9", body["aspect_ratio"])
			assert.NotContains(t, body, "size")
		}
	}
	video := lecAPIHook(t, plugin, "buildSubmitRequest", lecAPIContext("omni-video-redraw", map[string]any{"prompt": "boat", "size": "1280x720"}))["body"].(map[string]any)
	assert.Equal(t, "1280x720", video["size"])
	assert.NotContains(t, video, "aspect_ratio")
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
		{"ad-minimax-h3-replace", map[string]any{"images": []any{"https://media.example/person.png"}}},
		{"mf-video-upscale", map[string]any{"prompt": "upscale", "videos": []any{"https://media.example/source.mp4"}}},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "duration": 16, "resolution": "2K"}},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "resolution": "480p", "videos": []any{"https://media.example/source.mp4"}}},
		{"sl-midjourney", map[string]any{"prompt": "boat", "action": "blend"}},
		{"sl-midjourney", map[string]any{"prompt": "boat", "version": "8.2", "draft": true}},
		{"tinysnow-image-2", map[string]any{"prompt": "boat", "n": 11}},
		{"ty-seedream-5-pro", map[string]any{"prompt": "boat", "images": []any{"https://media.example/1.png", "https://media.example/2.png", "https://media.example/3.png", "https://media.example/4.png", "https://media.example/5.png", "https://media.example/6.png", "https://media.example/7.png", "https://media.example/8.png", "https://media.example/9.png"}}},
	} {
		_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", lecAPIContext(tc.model, tc.body))
		require.Error(t, err, "%s: %v", tc.model, tc.body)
	}
	ctx := lecAPIContext("sl-midjourney", map[string]any{"prompt": "boat", "seed": 0, "stylize": 0, "style": false, "hd": false})
	body := lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)
	assert.Equal(t, float64(0), body["seed"])
	assert.Equal(t, float64(0), body["stylize"])
	assert.Equal(t, false, body["style"])
	assert.Equal(t, false, body["hd"])
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
	ctx := lecAPIContext("ty-seedream-5-pro", map[string]any{"prompt": "boat"})
	ctx["upstream"] = map[string]any{"kind": "new_api"}
	assert.Equal(t, "https://lec.example/lecapi/v1/images/tasks", lecAPIHook(t, plugin, "buildSubmitRequest", ctx)["url"])
	ctx = lecAPIContext("ac-image-2", map[string]any{"prompt": "boat"})
	parsed := lecAPIHook(t, plugin, "parseSubmitResponse", ctx, map[string]any{"body": map[string]any{"data": []any{map[string]any{"url": "https://media.example/image.png"}}}})
	content := map[string]any{"model": "ac-image-2", "upstreamModel": "ac-image-2", "baseUrl": "https://lec.example", "artifactKey": "image-1", "upstreamTaskId": parsed["taskId"], "state": parsed["state"], "data": parsed["taskData"]}
	descriptor := lecAPIHook(t, plugin, "buildContentRequest", content)
	assert.Equal(t, "https://media.example/image.png", descriptor["url"], "a synchronous request UUID is not a queryable upstream job")
}

func TestLecAPICompletionDurationsStayWithinTheModelContract(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model   string
		seconds any
	}{
		{"aj-wan-3-0", 31}, {"aj-wan-3-0", 5.5}, {"ms-minimax-h3", 31}, {"mf-video-upscale", 2},
		{"jm-seedance-2-5-flash", 3}, {"jm-seedance-2-5-flash", 31}, {"jm-seedance-2-5-flash", 4.5},
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
}

func TestLecAPIChargesThePublishedBillingUnit(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	for _, tc := range []struct {
		model string
		body  map[string]any
		want  float64
	}{
		{"wan-3-0-h2", map[string]any{"prompt": "boat", "duration": 12, "resolution": "720p"}, 3.6},
		{"aj-wan-3-0", map[string]any{"prompt": "boat", "duration": 7, "resolution": "1080p"}, 5.124},
		{"jm-seedance-2-5-flash", map[string]any{"prompt": "boat"}, 1.2},
		{"jm-seedance-2-5-flash", map[string]any{"prompt": "boat", "duration": 4, "resolution": "480p"}, 0.72},
		{"jm-seedance-2-5-flash", map[string]any{"prompt": "boat", "duration": 30, "resolution": "720p"}, 7.2},
		{"tinysnow-image-2", map[string]any{"prompt": "boat", "n": 3}, 0.18},
		{"ac-image-2-5-flare", map[string]any{"prompt": "boat", "size": "3840x2160", "resolution": "4k"}, 0.12},
		{"ac-image-2-5-flare", map[string]any{"prompt": "boat", "size": "3840x2160"}, 0.06},
		{"nw-image-2", map[string]any{"prompt": "boat", "size": "1920x1080", "resolution": "2k", "quality": "high"}, 0.036},
		{"vp-seedance-2-5-301010-49", map[string]any{"prompt": "boat", "duration": 7}, 6.6},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "duration": 16, "resolution": "480p"}, 2.688},
		{"ms-minimax-h3", map[string]any{"prompt": "boat", "duration": 16, "resolution": "768P"}, 3.84},
	} {
		facts := lecAPIHook(t, plugin, "extractUsage", lecAPIContext(tc.model, tc.body))
		defaults, ok := plugin.Meta.DefaultsForModel(tc.model)
		require.True(t, ok)
		require.NotEmpty(t, defaults.BillingExpr)
		cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
		require.NoError(t, err)
		assert.InDelta(t, tc.want, cost, 1e-12)
	}
	for _, modelName := range []string{"ac-seedance-2-0-fast-2-720p", "image2", "nano-banana-2", "vp-seedance-2-9-301010"} {
		defaults, ok := plugin.Meta.DefaultsForModel(modelName)
		require.True(t, ok)
		assert.Empty(t, defaults.BillingExpr, "unpublished prices must not be guessed: %s", modelName)
	}
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
	ctx = lecAPIContext("mf-seedance-2-5-30ref", map[string]any{"prompt": "boat", "images": strings.Split(strings.Repeat("https://media.example/a.png,", 30), ",")[:30], "audios": []any{"https://media.example/a.mp3"}})
	_, err = plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
	require.ErrorContains(t, err, "combined reference")
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

func TestLecAPISourceVideoBillingUsesMeasuredDuration(t *testing.T) {
	plugin := loadLecAPIPlugin(t)
	video := yueqiReferenceVideoFixture(t, 3.5)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write(video)
	}))
	defer server.Close()
	fetch := system_setting.GetFetchSetting()
	previous := *fetch
	fetch.EnableSSRFProtection = false
	service.InitHttpClient()
	t.Cleanup(func() { *fetch = previous; service.InitHttpClient() })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", map[string]any{"resolution": "1080p", "videos": []any{server.URL + "/source.mp4"}})
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}, OriginModelName: "mf-video-upscale", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "mf-video-upscale", ApiKey: "test-key", ChannelBaseUrl: "https://lec.example"}}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(info)
	facts, err := adaptor.ExtractUsageFactsValidated(c, info)
	require.NoError(t, err)
	assert.Equal(t, float64(4), facts["seconds"], "source duration rounds up; the longer audio track is irrelevant")
	defaults, ok := plugin.Meta.DefaultsForModel("mf-video-upscale")
	require.True(t, ok)
	cost, _, err := billingexpr.RunExprWithRequest(defaults.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
	require.NoError(t, err)
	assert.InDelta(t, 0.288, cost, 1e-12)
}

type lecAPIFixtureTransport func(*http.Request) (*http.Response, error)

func (transport lecAPIFixtureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
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
				updated := strings.Replace(source, "price * 1.2", "price * 2.4", 1)
				updated = strings.Replace(updated, `version: "1.0.6"`, `version: "1.0.7"`, 1)
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
}
