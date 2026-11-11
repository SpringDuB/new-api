package plugins_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	"github.com/QuantumNous/new-api/relay/channel"
	taskplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/abema/go-mp4"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestYueqiVideoModelsAndMetering(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	assert.Empty(t, plugin.Meta.ChannelTypes)
	assert.True(t, plugin.Meta.SupportsUpstream(jsplugin.UpstreamKindNewAPI))
	assert.Contains(t, plugin.Meta.RequiredCapabilities, "submit-no-retry@1")
	models := map[string]struct {
		defaultResolution string
	}{
		"BD Seedance2.5":                {"720p"},
		"MINIMAXH3-2.0采样":               {"720p"},
		"kling-4.0":                     {"720p"},
		"mi-seedance2.5":                {"720p"},
		"sd2.5新":                        {"720p"},
		"seedance-2.0":                  {"720p"},
		"seedance-2.0-fast-通用":          {"720p"},
		"seedance-2.5-standard-seconds": {"720p"},
		"seedance-2.5-standard-video":   {"720p"},
		"seedance2.5无限并发版":              {"720p"},
		"shot-enhanced-2":               {"720p"},
		"vidu-q4":                       {"720p"},
		"vidu-q4-nopeak":                {"720p"},
		"w3-c1":                         {"720p"},
		"wan-3.0":                       {"720p"},
		"wan-3.0-pro":                   {"720p"},
		"wan3.0-video-prime-deal":       {"720p"},
		"不卡脸seedance-2.0":               {"720p"},
		"新seedance2.0-933":              {"720p"},
		"新满-seedance-933":               {"720p"},
		"特惠版2.5 满血 480p":                {"720p"},
	}
	for modelName := range models {
		assert.Contains(t, plugin.Meta.Models, modelName)
	}
	for modelName, profile := range models {
		t.Run(modelName, func(t *testing.T) {
			ctx := yueqiContext(modelName, map[string]any{"prompt": "A red paper boat floats on calm water", "seconds": "6", "seed": 0, "generate_audio": false})
			submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
			assert.Equal(t, "https://yueqi.example/v1/videos", submit["url"])
			assert.Equal(t, "POST", submit["method"])
			assert.Equal(t, map[string]any{"Authorization": "Bearer test-key", "Content-Type": "application/json"}, submit["headers"])
			body := submit["body"].(map[string]any)
			assert.Equal(t, modelName, body["model"])
			assert.Equal(t, float64(6), body["seconds"])
			assert.Equal(t, profile.defaultResolution, body["resolution"])
			assert.Equal(t, float64(0), body["seed"])
			assert.Equal(t, false, body["generate_audio"])
			facts := yueqiHook(t, plugin, "extractUsage", ctx)
			assert.Equal(t, profile.defaultResolution, facts["resolution"])
			schema, _ := plugin.Meta.UsageForModels(modelName)
			assert.True(t, schema["resolution"].AllowCustom, "administrators choose resolution branches for every model")
			assert.Equal(t, "second", string(schema["seconds"].Unit))
			assert.Equal(t, "count", string(schema["requests"].Unit))
			assert.Equal(t, float64(6), facts["seconds"])
			assert.Equal(t, float64(1), facts["requests"])
			for expression, want := range map[string]float64{
				`tier("request", u("requests") * 0.3)`: 0.3,
				`tier("second", u("seconds") * 0.1)`:   0.6,
			} {
				cost, _, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
				require.NoError(t, err)
				assert.InDelta(t, want, cost, 1e-12)
			}
			ctx["usagePurpose"] = "billing_ratios"
			assert.Equal(t, map[string]any{"requests": float64(1)}, yueqiHook(t, plugin, "extractUsage", ctx), "legacy per-call pricing must not multiply both facts")
			delete(ctx, "usagePurpose")
			for _, resolution := range []string{"480p", "720p", "1080p", "1440p", "2160p"} {
				t.Run(resolution, func(t *testing.T) {
					for _, req := range []map[string]any{
						{"prompt": "boat", "resolution": resolution},
						{"prompt": "boat", "metadata": map[string]any{"size": resolution}},
					} {
						ctx := yueqiContext(modelName, req)

						submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
						assert.Equal(t, resolution, submit["body"].(map[string]any)["resolution"])
						assert.Equal(t, resolution, yueqiHook(t, plugin, "extractUsage", ctx)["resolution"])
					}
				})
			}
		})
	}

	for _, modelName := range []string{"wan3.0-vedio-deal", "wan3.0-video-nsfw", "wan3.0-vedio-deal-per"} {
		assert.NotContains(t, plugin.Meta.Models, modelName)
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, yueqiContext(modelName, map[string]any{"prompt": "boat"}))
			require.ErrorContains(t, err, "unsupported Yueqi video model", "unlisted models must fail before billable submission")
		}
	}

	ctx := yueqiContext("mi-seedance2.5", map[string]any{"prompt": "boat"})
	ctx["model"] = "local-seedance-alias"
	submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "mi-seedance2.5", submit["body"].(map[string]any)["model"])
	assert.Equal(t, map[string]any{"requests": float64(1), "seconds": float64(5), "resolution": "720p", "reference_seconds": float64(0)}, yueqiHook(t, plugin, "extractUsage", ctx))
	assert.NotContains(t, submit, "model", "the driver must not rewrite the host's pinned alias")
	ctx["requestBody"] = map[string]any{"prompt": "boat", "resolution": "480p"}
	assert.Equal(t, "480p", yueqiHook(t, plugin, "buildSubmitRequest", ctx)["body"].(map[string]any)["resolution"], "upstream capabilities no longer restrict administrator pricing branches")
}

func TestYueqiPublicModelNamesBindVideoProtocolAndPreserveBillingIdentity(t *testing.T) {
	source, err := builtinplugins.Source("yueqi")
	require.NoError(t, err)
	registry := jsplugin.NewRegistry()
	plugin, err := registry.RegisterFactory(source, jsplugin.Options{Key: "yueqi"})
	require.NoError(t, err)
	aliases := map[string]string{
		"seedance2.5-per":       "BD Seedance2.5",
		"seedance-2.0-fast-per": "seedance-2.0-fast-通用",
		"seedance2.5":           "seedance2.5无限并发版",
		"seedance2.0-9:16-per":  "新seedance2.0-933",
		"vidu-q4-per":           "vidu-q4",
		"wan3.0-video-prime":    "wan3.0-video-prime-deal",
		"new-seedance2.5-per":   "sd2.5新",
		"wan3-c1-per":           "w3-c1",
	}
	for alias, upstream := range aliases {
		t.Run(alias, func(t *testing.T) {
			bindings := registry.Generation().LookupEndpointCandidates(http.MethodPost, "/v1/videos", alias)
			require.Len(t, bindings, 1, "public names must bind directly, without a channel-derived alias index")
			assert.Equal(t, "yueqi", bindings[0].Plugin.Meta.Key)
			assert.Equal(t, "openai_video", bindings[0].Protocol)
			wantSchema, _ := plugin.Meta.UsageForModels(upstream)
			actualSchema, _ := plugin.Meta.UsageForModels(alias)
			assert.Equal(t, wantSchema, actualSchema, "aliases retain the upstream model's resolution tiers and pricing units")
			ctx := yueqiContext(alias, map[string]any{"prompt": "boat", "seconds": 6})
			submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
			assert.Equal(t, upstream, submit["body"].(map[string]any)["model"])
			assert.NotContains(t, submit, "model", "the gateway keeps the public model name for billing and display")
			assert.Equal(t, alias, ctx["model"])
			assert.Equal(t, float64(1), yueqiHook(t, plugin, "extractUsage", ctx)["requests"])
			assert.Equal(t, float64(6), yueqiHook(t, plugin, "extractUsage", ctx)["seconds"])
			resolution := wantSchema["resolution"].Enum[0]
			assert.Equal(t, resolution, yueqiHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"resolution": resolution})["resolution"])
		})
	}
	ctx := yueqiContext("seedance-2.0-fast-per", map[string]any{"prompt": "boat", "resolution": "480p"})
	ctx["upstreamModel"] = "mi-seedance2.5"
	submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "mi-seedance2.5", submit["body"].(map[string]any)["model"], "an explicit channel mapping overrides the plugin's alias default")
	assert.Equal(t, "480p", yueqiHook(t, plugin, "extractUsage", ctx)["resolution"])
}

func TestYueqiPreservesFrameAndReferenceSemantics(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	for _, tc := range []struct {
		name, action string
		body         map[string]any
	}{
		{"frames", "frames_to_video", map[string]any{"first_frame_url": "https://media.example/start.png", "last_frame_url": "https://media.example/end.png"}},
		{"image references", "reference_to_video", map[string]any{"images": []any{"https://media.example/character.png", "https://media.example/scene.png"}}},
		{"input reference URL", "reference_to_video", map[string]any{"input_reference": "https://media.example/frame.png"}},
		{"multimedia", "reference_to_video", map[string]any{"video_urls": []any{"https://media.example/ref.mp4"}, "audio_urls": []any{"https://media.example/ref.mp3"}, "image_references": []any{map[string]any{"url": "https://media.example/ref.png", "role": "reference"}}}},
		{"nested billing aliases", "reference_to_video", map[string]any{"metadata": map[string]any{"duration": 8, "resolution": "720p", "images": []any{"https://media.example/ref.png"}}, "seconds": 8}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := maps.Clone(tc.body)
			body["prompt"] = "A boat gently floats across the lake"
			before, err := common.Marshal(body)
			require.NoError(t, err)
			ctx := yueqiContext("BD Seedance2.5", body)
			submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
			assert.Equal(t, tc.action, submit["action"])
			output := submit["body"].(map[string]any)
			for field, value := range tc.body {
				actual, marshalErr := common.Marshal(output[field])
				require.NoError(t, marshalErr)
				want, marshalErr := common.Marshal(value)
				require.NoError(t, marshalErr)
				assert.JSONEq(t, string(want), string(actual), field)
			}
			after, err := common.Marshal(body)
			require.NoError(t, err)
			assert.JSONEq(t, string(before), string(after), "pooled hooks must not mutate their arguments")
		})
	}

	value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
		"model": "local-alias", "body": map[string]any{"kind": "multipart", "fields": map[string]any{
			"prompt": []string{"boat"}, "duration": []string{"7"}, "images": []string{`["https://media.example/a.png","https://media.example/b.png"]`},
			"metadata": []string{`{"seed":0,"generate_audio":false}`},
		}},
	})
	require.NoError(t, err)
	decoded := value.(map[string]any)
	assert.Equal(t, "local-alias", decoded["model"])
	ctx := yueqiContext("BD Seedance2.5", decoded["requestBody"].(map[string]any))
	ctx["files"] = []map[string]any{{"ref": "request_file:input_reference", "field": "input_reference", "filename": "frame.png", "size": 3}}
	submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
	assert.Equal(t, "multipart", submit["bodyType"])
	assert.Equal(t, map[string]any{"Authorization": "Bearer test-key"}, submit["headers"])
	parts := map[string]map[string]any{}
	for _, item := range submit["parts"].([]any) {
		part := item.(map[string]any)
		parts[part["name"].(string)] = part
	}
	assert.JSONEq(t, `["https://media.example/a.png","https://media.example/b.png"]`, parts["images"]["value"].(string))
	assert.JSONEq(t, `{"seed":0,"generate_audio":false}`, parts["metadata"]["value"].(string))
	assert.Equal(t, "request_file:input_reference", parts["input_reference"]["fileRef"])
	assert.Equal(t, "7", parts["seconds"]["value"])
}

func TestYueqiInputReferenceImageObjects(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	const signedURL = "https://media.example/frame.png?Expires=1794225387&Signature=a%2Bb%2Fc%3D"
	for _, tc := range []struct {
		name, reference string
		images, want    []any
	}{
		{"image URL object", `{"image_url":"https://media.example/frame.png?Expires=1794225387\u0026Signature=a%2Bb%2Fc%3D"}`, nil, []any{signedURL}},
		{"nested image URL", `{"image_url":{"url":"https://media.example/frame.png"}}`, nil, []any{"https://media.example/frame.png"}},
		{"existing references", `{"image_url":"https://media.example/frame.png"}`, []any{"https://media.example/other.png"}, []any{"https://media.example/frame.png", "https://media.example/other.png"}},
		{"duplicate reference", `{"image_url":"https://media.example/frame.png"}`, []any{"https://media.example/frame.png"}, []any{"https://media.example/frame.png"}},
	} {
		for _, kind := range []string{"json", "multipart"} {
			t.Run(tc.name+" "+kind, func(t *testing.T) {
				const modelName = "seedance-2.0-fast-per"
				var reference any
				require.NoError(t, common.UnmarshalJsonStr(tc.reference, &reference))
				input := map[string]any{
					"model": modelName, "prompt": "Animate the reference character", "input_reference": reference,
					"seconds": "5", "resolution_name": "720p", "size": "16:9",
				}
				if tc.images != nil {
					input["images"] = tc.images
				}
				before, err := common.Marshal(input)
				require.NoError(t, err)
				body := map[string]any{"kind": kind, "value": input}
				if kind == "multipart" {
					fields := map[string][]string{}
					for key, value := range input {
						if str, ok := value.(string); ok {
							fields[key] = []string{str}
							continue
						}
						encoded, err := common.Marshal(value)
						require.NoError(t, err)
						fields[key] = []string{string(encoded)}
					}
					body = map[string]any{"kind": kind, "fields": fields}
				}
				decoded, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{"model": modelName, "body": body})
				require.NoError(t, err)
				ctx := yueqiContext(modelName, decoded.(map[string]any)["requestBody"].(map[string]any))
				ctx["usageKeys"] = []string{"reference_seconds"}
				if kind == "multipart" {
					ctx["files"] = []map[string]any{{"ref": "request_file:input_reference", "field": "input_reference", "filename": "frame.png", "mimeType": "image/png", "size": 3}}
				}
				submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
				assert.Equal(t, "reference_to_video", submit["action"])
				if kind == "json" {
					upstream := submit["body"].(map[string]any)
					assert.Equal(t, "seedance-2.0-fast-通用", upstream["model"])
					assert.Equal(t, tc.want, upstream["images"])
					assert.NotContains(t, upstream, "input_reference")
				} else {
					parts := map[string]map[string]any{}
					for _, item := range submit["parts"].([]any) {
						part := item.(map[string]any)
						parts[part["name"].(string)] = part
					}
					var images []any
					require.NoError(t, common.UnmarshalJsonStr(parts["images"]["value"].(string), &images))
					assert.Equal(t, tc.want, images)
					assert.Equal(t, "request_file:input_reference", parts["input_reference"]["fileRef"])
				}
				assert.Equal(t, map[string]any{"requests": float64(1), "seconds": float64(5), "resolution": "720p", "reference_seconds": float64(0)}, yueqiHook(t, plugin, "extractUsage", ctx))
				after, err := common.Marshal(input)
				require.NoError(t, err)
				assert.JSONEq(t, string(before), string(after), "reference conversion must preserve the client input")
			})
		}
	}
	for _, reference := range []any{
		map[string]any{"image_url": ""},
		map[string]any{"image_url": 123},
		map[string]any{"image_url": nil},
		map[string]any{"image_url": map[string]any{}},
		map[string]any{"image_url": "https://media.example/frame.png extra"},
	} {
		for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
			_, err := plugin.Engine.Call(t.Context(), hook, yueqiContext("seedance-2.0-fast-per", map[string]any{"prompt": "boat", "input_reference": reference}))
			require.Error(t, err, "invalid image objects must fail before billable submission")
		}
	}
}

func TestYueqiRejectsInvalidBillingBeforeSubmission(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"zero duration", map[string]any{"seconds": 0}},
		{"negative duration", map[string]any{"duration": -1}},
		{"boolean duration", map[string]any{"seconds": true}},
		{"oversized duration", map[string]any{"seconds": "18446744073709551615"}},
		{"conflicting durations", map[string]any{"seconds": 5, "duration": 8}},
		{"nested conflict", map[string]any{"seconds": 5, "extra": map[string]any{"metadata": map[string]any{"duration": 8}}}},
		{"nested duration overflow", map[string]any{"values": map[string]any{"duration": 1e99}}},
		{"multiple videos", map[string]any{"n": 2}},
		{"nested batch", map[string]any{"metadata": map[string]any{"batch_size": "128"}}},
		{"resolution conflict", map[string]any{"size": "720p", "extra": map[string]any{"resolution": "1080p"}}},
		{"invalid resolution", map[string]any{"resolution": "99999p"}},
		{"invalid ratio", map[string]any{"aspect_ratio": "0:16"}},
		{"ratio conflict", map[string]any{"ratio": "16:9", "aspect_ratio": "9:16"}},
		{"hidden model", map[string]any{"metadata": map[string]any{"model": "other-model"}}},
		{"unsupported quality", map[string]any{"metadata": map[string]any{"quality": "ultra"}}},
		{"invalid container", map[string]any{"values": []any{map[string]any{"seconds": 9000}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := maps.Clone(tc.body)
			body["prompt"] = "boat"
			for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
				_, err := plugin.Engine.Call(t.Context(), hook, yueqiContext("seedance2.5无限并发版", body))
				require.Error(t, err, hook)
			}
		})
	}
}

func TestYueqiResolutionAndAspectRatioAliases(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	for _, modelName := range plugin.Meta.Models {
		t.Run(modelName, func(t *testing.T) {
			for _, tc := range []struct {
				name       string
				parameters map[string]any
			}{
				{"client format", map[string]any{"resolution_name": "720p", "size": "16:9"}},
				{"uppercase resolution", map[string]any{"resolution": " 720P ", "aspect_ratio": "16:9"}},
				{"uppercase size resolution", map[string]any{"size": "720P", "ratio": "16:9"}},
				{"equivalent aliases", map[string]any{"resolution_name": "720P", "resolution": "720p", "size": "16:9", "ratio": "16:9"}},
				{"nested parameters", map[string]any{"parameters": map[string]any{"extra": map[string]any{"resolution_name": "720P", "size": "16:9"}}}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					body := maps.Clone(tc.parameters)
					body["prompt"], body["seconds"] = "A paper boat floats on the lake", "15"
					body["images"] = []any{map[string]any{"url": "https://media.example/frame.png", "size": "original", "resolution_name": "reference"}}
					before, err := common.Marshal(body)
					require.NoError(t, err)
					ctx := yueqiContext(modelName, body)
					submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
					upstream := submit["body"].(map[string]any)
					assert.Equal(t, "720p", upstream["resolution"])
					assert.Equal(t, "16:9", upstream["aspect_ratio"])
					assert.Equal(t, float64(15), upstream["seconds"])
					assert.NotContains(t, upstream, "size", "ambiguous client size is replaced by resolution and aspect_ratio")
					assert.NotContains(t, upstream, "resolution_name")
					if nested, ok := upstream["parameters"].(map[string]any); ok {
						extra := nested["extra"].(map[string]any)
						assert.NotContains(t, extra, "size")
						assert.NotContains(t, extra, "resolution_name")
						assert.Equal(t, "720p", extra["resolution"])
						assert.Equal(t, "16:9", extra["aspect_ratio"])
					}
					assert.Equal(t, "original", upstream["images"].([]any)[0].(map[string]any)["size"], "media metadata must not be normalized as generation parameters")
					assert.Equal(t, "720p", yueqiHook(t, plugin, "extractUsage", ctx)["resolution"])
					assert.Equal(t, "720p", yueqiHook(t, plugin, "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"resolution_name": " 720P "})["resolution"])
					after, err := common.Marshal(body)
					require.NoError(t, err)
					assert.JSONEq(t, string(before), string(after), "normalization must not mutate pooled request input")
				})
			}
		})
	}

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"resolution conflict", map[string]any{"resolution": "720p", "resolution_name": "480P"}},
		{"nested resolution conflict", map[string]any{"resolution": "720p", "metadata": map[string]any{"resolution_name": "480P"}}},
		{"ratio conflict", map[string]any{"resolution_name": "720P", "size": "16:9", "aspect_ratio": "9:16"}},
		{"invalid ratio", map[string]any{"resolution_name": "720P", "size": "0:16"}},
		{"invalid resolution", map[string]any{"resolution_name": "ultra", "size": "16:9"}},
		{"invalid resolution type", map[string]any{"resolution_name": 720}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := maps.Clone(tc.body)
			body["prompt"] = "boat"
			for _, hook := range []string{"buildSubmitRequest", "extractUsage"} {
				for _, modelName := range []string{"seedance-2.0-fast-per", "mi-seedance2.5"} {
					_, err := plugin.Engine.Call(t.Context(), hook, yueqiContext(modelName, body))
					require.Error(t, err, "invalid/conflicting aliases must fail before sending or reserving a price tier")
				}
			}
		})
	}

	ctx480 := yueqiContext("seedance2.5", map[string]any{"prompt": "boat", "resolution_name": "480P"})
	assert.Equal(t, "480p", yueqiHook(t, plugin, "buildSubmitRequest", ctx480)["body"].(map[string]any)["resolution"])
	assert.Equal(t, "480p", yueqiHook(t, plugin, "extractUsage", ctx480)["resolution"], "resolution_name must select the requested billing tier rather than the default")
	_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", yueqiContext("seedance2.5", map[string]any{
		"prompt": "boat", "resolution": "720p", "values": map[string]any{"metadata": map[string]any{"resolution_name": "480P"}},
	}))
	require.ErrorContains(t, err, "conflicting video resolution parameters")

	for _, tc := range []struct{ model, kind string }{
		{"mi-seedance2.5", "json"},
		{"mi-seedance2.5", "multipart"},
		{"seedance-2.0-fast-per", "multipart"},
	} {
		t.Run(tc.model+" client "+tc.kind, func(t *testing.T) {
			modelName, kind := tc.model, tc.kind
			body := map[string]any{"kind": kind, "value": map[string]any{
				"model": modelName, "prompt": "boat", "resolution_name": "480p", "seconds": "5", "size": "16:9",
			}}
			if kind == "multipart" {
				body = map[string]any{"kind": kind, "fields": map[string]any{
					"model": []string{modelName}, "prompt": []string{"boat"}, "resolution_name": []string{"480p"}, "seconds": []string{"5"}, "size": []string{"16:9"},
				}}
			}
			value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
				"model": modelName, "body": body,
			})
			require.NoError(t, err)
			ctx := yueqiContext(modelName, value.(map[string]any)["requestBody"].(map[string]any))
			if kind == "multipart" {
				ctx["files"] = []map[string]any{{"ref": "request_file:input_reference", "field": "input_reference", "filename": "frame.png", "size": 3}}
			}
			submit := yueqiHook(t, plugin, "buildSubmitRequest", ctx)
			if kind == "json" {
				assert.Equal(t, map[string]any{
					"model": modelName, "prompt": "boat", "resolution": "480p", "seconds": float64(5), "aspect_ratio": "16:9",
				}, submit["body"])
			} else {
				parts := map[string]any{}
				for _, item := range submit["parts"].([]any) {
					part := item.(map[string]any)
					parts[part["name"].(string)] = part["value"]
				}
				assert.Equal(t, "480p", parts["resolution"])
				assert.Equal(t, "16:9", parts["aspect_ratio"])
				assert.Equal(t, "5", parts["seconds"])
				assert.NotContains(t, parts, "resolution_name")
				assert.NotContains(t, parts, "size")
			}
			assert.Equal(t, map[string]any{
				"requests": float64(1), "seconds": float64(5), "resolution": "480p", "reference_seconds": float64(0),
			}, yueqiHook(t, plugin, "extractUsage", ctx))
		})
	}
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

func TestYueqiMeasuresReferenceVideosBeforeBilling(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	video := yueqiReferenceVideoFixture(t, 3.5)
	var downloads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"), "gateway credentials must not accompany reference media")
		assert.Empty(t, r.Header.Get("Cookie"))
		downloads.Add(1)
		if r.URL.Path == "/oversized" {
			w.Header().Set("Content-Length", "2097152")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/broken" {
			_, _ = io.WriteString(w, "not a video")
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write(video)
	}))
	defer server.Close()
	fetch := system_setting.GetFetchSetting()
	previousFetch := *fetch
	fetch.EnableSSRFProtection = false
	service.InitHttpClient()
	settings := config.GlobalConfig.Get("billing_setting").(*billing_setting.BillingSetting)
	previousPrices := settings.PluginBillingExpr
	settings.PluginBillingExpr = maps.Clone(previousPrices)
	t.Cleanup(func() { *fetch = previousFetch; settings.PluginBillingExpr = previousPrices; service.InitHttpClient() })
	modelName := "seedance-2.0-fast-per"
	key := billing_setting.PluginBillingExprKey("yueqi", modelName)
	settings.PluginBillingExpr[key] = `u("resolution") == "1440p" ? tier("high", u("seconds") * 0.2 + u("reference_seconds") * 0.1) : tier("base", u("requests") * 1)`
	schema, _ := plugin.Meta.UsageForModels(modelName)
	require.NoError(t, billing_setting.SmokeTestTaskExpr(settings.PluginBillingExpr[key], schema))
	require.Error(t, billing_setting.SmokeTestTaskExpr(`u("resolution") == "1440p" ? tier("custom", -1) : tier("base", 0)`, schema), "custom branches must be validated even when absent from plugin suggestions")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", map[string]any{"prompt": "boat", "seconds": 15, "resolution_name": "1440P", "videos": []any{map[string]any{"url": server.URL + "/video", "duration": 0}}, "metadata": map[string]any{"reference_seconds": 0}})
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}, OriginModelName: modelName, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: modelName, ApiKey: "test-key", ChannelBaseUrl: "https://yueqi.example"}}
	adaptor := taskplugin.New(plugin)
	adaptor.Init(info)
	facts, err := adaptor.ExtractUsageFactsValidated(c, info)
	require.NoError(t, err)
	assert.Equal(t, 3.5, facts["reference_seconds"], "measure the video track and ignore client duration and longer audio")
	assert.Equal(t, "1440p", facts["resolution"])
	cost, _, err := billingexpr.RunExprWithRequest(settings.PluginBillingExpr[key], billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: facts})
	require.NoError(t, err)
	assert.InDelta(t, 3.35, cost, 1e-9)
	_, err = adaptor.ExtractUsageFactsValidated(c, info)
	require.NoError(t, err)
	assert.EqualValues(t, 1, downloads.Load(), "repeated usage hooks reuse only request-local measured facts")

	settings.PluginBillingExpr[key] = `tier("base", u("requests") * 1)`
	facts, err = adaptor.ExtractUsageFactsValidated(c, info)
	require.NoError(t, err)
	assert.Zero(t, facts["reference_seconds"], "disabled reference pricing skips media probing")
	assert.EqualValues(t, 1, downloads.Load())

	settings.PluginBillingExpr[key] = `tier("base", u("reference_seconds") * 0.1)`
	c.Set("task_request", map[string]any{"prompt": "boat", "videos": []any{server.URL + "/broken"}})
	_, err = adaptor.ExtractUsageFactsValidated(c, info)
	require.Error(t, err, "unmeasurable media must fail before reservation or billable submission")

	// Server-side fetch protection still applies to client-provided media.
	fetch.EnableSSRFProtection = true
	fetch.AllowPrivateIp = false
	blocked, _ := gin.CreateTestContext(httptest.NewRecorder())
	blocked.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	blocked.Set("task_request", map[string]any{"prompt": "boat", "videos": []any{server.URL + "/video"}})
	_, err = adaptor.ExtractUsageFactsValidated(blocked, info)
	require.Error(t, err, "private reference URLs must respect server fetch protection")
	fetch.EnableSSRFProtection = false

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("videos", "reference.mov")
	require.NoError(t, err)
	_, err = file.Write(video)
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("prompt", "boat"))
	require.NoError(t, writer.Close())
	upload, _ := gin.CreateTestContext(httptest.NewRecorder())
	upload.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &body)
	upload.Request.Header.Set("Content-Type", writer.FormDataContentType())
	upload.Set("task_request", map[string]any{"prompt": "boat"})
	facts, err = adaptor.ExtractUsageFactsValidated(upload, info)
	require.NoError(t, err)
	assert.Equal(t, 3.5, facts["reference_seconds"], "uploaded video refs are measured from actual request files")
	previousLimit := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 1
	t.Cleanup(func() { constant.MaxFileDownloadMB = previousLimit })
	oversized, _ := gin.CreateTestContext(httptest.NewRecorder())
	oversized.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	oversized.Set("task_request", map[string]any{"prompt": "boat", "videos": []any{server.URL + "/oversized"}})
	_, err = adaptor.ExtractUsageFactsValidated(oversized, info)
	require.Error(t, err, "media download byte limits must reject before billable work")

}

func TestYueqiReferenceDurationLimitsAndCompletionRetention(t *testing.T) {
	plugin := loadYueqiPlugin(t)
	for _, invalid := range []float64{0, -1, 3601} {
		ctx := yueqiContext("BD Seedance2.5", map[string]any{"prompt": "boat", "videos": []any{"https://media.example/video"}})
		ctx["usageKeys"] = []string{"reference_seconds"}
		prober := jsplugin.WithMediaProber(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
			data, err := common.Marshal(invalid)
			require.NoError(t, err)
			return jsplugin.FetchResponse{Status: 200, Body: data}, nil
		})
		_, err := plugin.Engine.Call(prober, "extractUsage", ctx)
		require.Error(t, err)
	}
	ctx := yueqiContext("BD Seedance2.5", map[string]any{"prompt": "boat", "videos": []any{"https://media.example/a", "https://media.example/b"}})
	ctx["usageKeys"] = []string{"reference_seconds"}
	prober := jsplugin.WithMediaProber(t.Context(), func(context.Context, jsplugin.FetchRequest) (jsplugin.FetchResponse, error) {
		return jsplugin.FetchResponse{Status: 200, Body: []byte("2000")}, nil
	})
	_, err := plugin.Engine.Call(prober, "extractUsage", ctx)
	require.ErrorContains(t, err, "total must not exceed")
	facts := yueqiHook(t, plugin, "extractUsageOnComplete", map[string]any{"model": "BD Seedance2.5"}, map[string]any{"status": "SUCCESS"}, map[string]any{"seconds": 15, "reference_seconds": 0})
	assert.NotContains(t, facts, "reference_seconds", "frozen measured duration survives completion without upstream echo")
	source, err := builtinplugins.Source("yueqi")
	require.NoError(t, err)
	for _, invalid := range []string{"", " leading", strings.Repeat("x", 65), "bad\nvalue"} {
		altered := strings.Replace(source, `return { requests: 1, seconds: body.seconds, resolution: body.resolution, reference_seconds: referenceSeconds };`, `return {requests:1, seconds:body.seconds, resolution: `+strconv.Quote(invalid)+`};`, 1)
		loaded, err := jsplugin.NewRegistry().RegisterFactory(altered, jsplugin.Options{Key: "yueqi"})
		require.NoError(t, err)
		adaptor := taskplugin.New(loaded)
		info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}, OriginModelName: "BD Seedance2.5", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "BD Seedance2.5"}}
		adaptor.Init(info)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
		c.Set("task_request", map[string]any{"prompt": "boat"})
		_, err = adaptor.ExtractUsageFactsValidated(c, info)
		require.Error(t, err, "custom categorical facts remain bounded")
	}
}
