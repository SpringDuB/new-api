package controller

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	kittypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageLogMultipartRequestCapturesTextAndMetadataWithoutConsumingUpload(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	require.NoError(t, form.WriteField("model", "client-image-model"))
	require.NoError(t, form.WriteField("seed", "0"))
	require.NoError(t, form.WriteField("metadata", `{"enabled":false,"api_key":"private-body-key"}`))
	file, err := form.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	_, err = file.Write([]byte("binary-must-not-be-logged"))
	require.NoError(t, err)
	require.NoError(t, form.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", form.FormDataContentType())
	service.CaptureTaskClientRequest(c, nil, nil)
	value, exists := c.Get(service.TaskClientRequestContextKey)
	require.True(t, exists)
	snapshot, ok := value.(*model.TaskClientRequestSnapshot)
	require.True(t, ok)
	assert.Equal(t, "multipart", snapshot.BodyKind)
	assert.True(t, snapshot.FilesOmitted)
	assert.True(t, snapshot.Redacted)
	assert.Contains(t, string(snapshot.Body), "client-image-model")
	assert.Contains(t, string(snapshot.Body), `"seed":["0"]`)
	assert.Contains(t, string(snapshot.Body), "input.png")
	assert.Contains(t, string(snapshot.Body), "[REDACTED]")
	assert.NotContains(t, string(snapshot.Body), "private-body-key")
	assert.NotContains(t, string(snapshot.Body), "binary-must-not-be-logged")
	storage, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	defer storage.Close()
	reader, err := storage.NewReader()
	require.NoError(t, err)
	defer reader.Close()
	replayed, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, body.Bytes(), replayed)
}

func TestImageLogSynchronousResponsePreservesClientInputAndWireResponse(t *testing.T) {
	setupGenericTaskTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"client-image","prompt":"client prompt","seed":0,"enhance":false,"access_token":"body-secret"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer request-secret")
	c.Set("id", 7)
	c.Set("role", common.RoleCommonUser)
	c.Set(common.RequestIdKey, "request-image-1")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "client-image")
	finish := beginImageLogCapture(c)
	info := &relaycommon.RelayInfo{UserId: 7, ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 1, UpstreamModelName: "mapped-image"}, OriginModelName: "client-image", UsingGroup: "default"}
	c.Set(imageLogRelayInfoContextKey, info)
	c.Set(service.SettledQuotaContextKey, 123)
	// The upstream request was converted after the immutable client capture.
	c.Request.Body = io.NopCloser(strings.NewReader(`{"model":"mapped-image","prompt":"rewritten"}`))
	response := `{"created":42,"data":[{"url":"https://cdn.example/image.png?signature=media-secret","revised_prompt":"upstream prompt"}],"token":"response-secret"}`
	c.Data(http.StatusOK, "application/json", []byte(response))
	finish()
	assert.Equal(t, response, recorder.Body.String())
	var tasks []model.Task
	require.NoError(t, model.DB.Where("platform = ?", "image").Find(&tasks).Error)
	require.Len(t, tasks, 1)
	task := tasks[0]
	assert.Equal(t, model.TaskStatus(model.TaskStatusSuccess), task.Status)
	assert.Equal(t, 123, task.Quota)
	assert.Equal(t, "synchronous", task.PrivateData.ImageRequestMode)
	assert.Equal(t, "mapped-image", task.Properties.UpstreamModelName)
	require.NotNil(t, task.PrivateData.ClientRequest)
	assert.JSONEq(t, `{"model":"client-image","prompt":"client prompt","seed":0,"enhance":false,"access_token":"[REDACTED]"}`, string(task.PrivateData.ClientRequest.Body))
	require.NotNil(t, task.PrivateData.Response)
	assert.Contains(t, string(task.PrivateData.Response.Body), "upstream prompt")
	assert.NotContains(t, string(task.PrivateData.Response.Body), "response-secret")
	assert.NotContains(t, string(task.PrivateData.Response.Body), "media-secret")
	assert.Empty(t, task.Data, "large response bodies stay private and load on demand")
	assert.Empty(t, model.GetAllUnFinishSyncTasks(100), "image logs never enter polling or refunds")
	// Extending private JSON needs no schema migration; existing rows and
	// snapshots survive repeated startup on every supported dialect.
	require.NoError(t, model.DB.AutoMigrate(&model.Task{}))
	require.NoError(t, model.DB.AutoMigrate(&model.Task{}))
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	require.NotNil(t, reloaded.PrivateData.Response)
	assert.JSONEq(t, string(task.PrivateData.Response.Body), string(reloaded.PrivateData.Response.Body))
}

func TestImageLogRecognizesGeneratedImagesAcrossFormatsWithoutLoggingVisionInput(t *testing.T) {
	setupGenericTaskTest(t)
	for _, tc := range []struct {
		name, path, request, response, contentType string
		images                                     int
	}{
		{"images", "/v1/images/generations", `{"model":"m"}`, `{"data":{"url":"https://cdn.example/output.png"}}`, "application/json", 1},
		{"gemini", "/v1beta/models/gemini:generateContent", `{"contents":[{"parts":[{"text":"draw"}]}]}`, `{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"aW1hZ2U="}}]}}]}`, "application/json", 1},
		{"responses", "/v1/responses", `{"model":"m","tools":[{"type":"image_generation"}]}`, `{"output":[{"type":"image_generation_call","result":"aW1hZ2U="}]}`, "application/json", 1},
		{"chat", "/v1/chat/completions", `{"model":"m","modalities":["text","image"]}`, `{"choices":[{"message":{"images":[{"image_url":{"url":"https://cdn.example/output.png"}}]}}]}`, "application/json", 1},
		{"chat stream", "/v1/chat/completions", `{"model":"m","stream":true}`, "data: {\"choices\":[{\"delta\":{\"images\":[{\"image_url\":{\"url\":\"https://cdn.example/output.png\"}}]}}]}\n\ndata: [DONE]\n\n", "text/event-stream", 1},
		{"stream", "/v1/images/generations", `{"model":"m","stream":true}`, "data: {\"type\":\"image_generation.partial_image\",\"b64_json\":\"cGFydGlhbA==\"}\n\ndata: {\"type\":\"image_generation.completed\",\"b64_json\":\"aW1hZ2U=\"}\n\ndata: [DONE]\n\n", "text/event-stream", 1},
		{"completed image URL stream", "/v1/images/generations", `{"model":"m","stream":true}`, "data: {\"type\":\"image_generation.completed\",\"url\":\"https://cdn.example/output.png\"}\n\n", "text/event-stream", 1},
		{"completed edit URL stream", "/v1/images/edits", `{"model":"m","stream":true}`, "data: {\"type\":\"image_edit.completed\",\"url\":\"https://cdn.example/output.png\"}\n\n", "text/event-stream", 1},
		{"responses after text completion events", "/v1/responses", `{"model":"m","tools":[{"type":"image_generation"}],"stream":true}`, strings.Repeat("data: {\"type\":\"response.output_text.done\",\"text\":\"finished text\"}\n\n", 64) + "data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"image_generation_call\",\"result\":\"aW1hZ2U=\"}}\n\ndata: [DONE]\n\n", "text/event-stream", 1},
		{"multiline image event", "/v1/images/generations", `{"model":"m","stream":true}`, "data: {\"type\":\"image_generation.completed\",\n" + "data: \"b64_json\":\"aW1hZ2U=\"}\n\n", "text/event-stream", 1},
		{"seedream full image", "/v1/images/generations", `{"model":"m","stream":true}`, "data: {\"type\":\"image_generation.partial_succeeded\",\"url\":\"https://cdn.example/output.png\"}\n\ndata: {\"type\":\"image_generation.completed\",\"usage\":{\"generated_images\":1}}\n\n", "text/event-stream", 1},
		{"seedream base64 image", "/v1/images/generations", `{"model":"m","stream":true}`, "data: {\"type\":\"image_generation.partial_succeeded\",\"b64_json\":\"aW1hZ2U=\"}\n\n", "text/event-stream", 1},
		{"alibaba native image", "/v1/images/generations", `{"model":"m"}`, `{"output":{"choices":[{"message":{"content":[{"text":"output"},{"image":"https://cdn.example/output.png"}]}}]}}`, "application/json", 1},
		{"alibaba native stream", "/v1/images/generations", `{"model":"m","stream":true}`, "data: {\"output\":{\"choices\":[{\"message\":{\"content\":[{\"image\":\"https://cdn.example/output.png\"}]}}]}}\n\n", "text/event-stream", 1},
		{"vision only", "/v1/chat/completions", `{"model":"m","messages":[{"content":[{"type":"image_url","image_url":{"url":"https://input.example/input.png"}}]}]}`, `{"choices":[{"message":{"content":"a cat"}}]}`, "application/json", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before int64
			require.NoError(t, model.DB.Model(&model.Task{}).Count(&before).Error)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.request))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("id", 7)
			finish := beginImageLogCapture(c)
			c.Set(imageLogRelayInfoContextKey, &relaycommon.RelayInfo{UserId: 7, ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 1}, OriginModelName: "m"})
			c.Data(http.StatusOK, tc.contentType, []byte(tc.response))
			finish()
			var after int64
			require.NoError(t, model.DB.Model(&model.Task{}).Count(&after).Error)
			assert.Equal(t, int64(tc.images), after-before)
			if tc.images == 1 {
				var task model.Task
				require.NoError(t, model.DB.Order("id desc").First(&task).Error)
				require.NotNil(t, task.PrivateData.Response)
				assert.Len(t, task.PrivateData.ImageArtifacts, 1)
				assert.NotContains(t, string(task.PrivateData.Response.Body), "aW1hZ2U=")
				assert.NotContains(t, string(task.PrivateData.Response.Body), "cGFydGlhbA==")
			}
		})
	}
}

func TestImageLogOversizedRequestsKeepImageIntentOnUpstreamFailure(t *testing.T) {
	setupGenericTaskTest(t)
	input, err := common.Marshal(strings.Repeat("x", service.MaxTaskClientRequestSnapshotBytes))
	require.NoError(t, err)
	for _, tc := range []struct {
		name, path, body string
		format           kittypes.RelayFormat
	}{
		{"responses", "/v1/responses", `{"model":"client-alias","tools":[{"type":"image_generation"}],"input":` + string(input) + `}`, kittypes.RelayFormatOpenAIResponses},
		{"chat", "/v1/chat/completions", `{"model":"client-alias","modalities":["text","image"],"messages":[{"role":"user","content":` + string(input) + `}]}`, kittypes.RelayFormatOpenAI},
		{"gemini", "/v1beta/models/client-alias:generateContent", `{"contents":[{"parts":[{"text":` + string(input) + `}]}],"generationConfig":{"responseModalities":["TEXT","IMAGE"]}}`, kittypes.RelayFormatGemini},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("id", 7)
			finish := beginImageLogCapture(c)
			request, err := helper.GetAndValidateRequest(c, tc.format)
			require.NoError(t, err)
			captureImageLogRequestIntent(c, request)
			c.Data(http.StatusBadRequest, "application/json", []byte(`{"error":{"message":"upstream rejected image request"}}`))
			finish()
			var task model.Task
			require.NoError(t, model.DB.Order("id DESC").First(&task).Error)
			assert.Equal(t, "image", string(task.Platform))
			assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), task.Status)
			require.NotNil(t, task.PrivateData.ClientRequest)
			assert.Empty(t, task.PrivateData.ClientRequest.Body)
			assert.Equal(t, "size_limit", task.PrivateData.ClientRequest.BodyOmitted)
		})
	}
}

func TestImageLogsMergeAsyncSyncAndMidjourneyWithGlobalPaginationAndOwnership(t *testing.T) {
	setupGenericTaskTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Midjourney{}))
	t.Cleanup(func() { require.NoError(t, model.DB.Migrator().DropTable(&model.Midjourney{})) })
	for _, task := range []model.Task{
		{TaskID: "image-sync", Platform: "image", Action: "image_generation", UserId: 7, SubmitTime: 300, Status: model.TaskStatusSuccess, PrivateData: model.TaskPrivateData{ImageRequestMode: "synchronous", Response: &model.TaskResponseSnapshot{Body: []byte(`{"private":"response-body"}`)}}},
		{TaskID: "image-async", Platform: "lecapi", Action: "imagine", UserId: 7, SubmitTime: 100, Status: model.TaskStatusSubmitted},
		{TaskID: "foreign-image", Platform: "image", Action: "image_generation", UserId: 8, SubmitTime: 400, Status: model.TaskStatusSuccess},
		{TaskID: "video", Platform: "lecapi", Action: "text_to_video", UserId: 7, SubmitTime: 500, Status: model.TaskStatusSuccess},
	} {
		require.NoError(t, model.DB.Create(&task).Error)
	}
	require.NoError(t, model.DB.Create(&model.Midjourney{MjId: "mj-history", UserId: 7, SubmitTime: 200000, Status: "SUCCESS", Prompt: "legacy prompt"}).Error)
	for _, page := range []struct {
		number int
		taskID string
	}{{1, "image-sync"}, {2, "mj-history"}, {3, "image-async"}} {
		c, recorder := imageLogReadContext("/api/image-log/self?p="+strconv.Itoa(page.number)+"&page_size=1", "", 7, common.RoleCommonUser)
		GetUserImageLogs(c)
		var response struct {
			Data struct {
				Total int
				Items []dto.TaskDto
			}
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.Equal(t, 3, response.Data.Total)
		require.Len(t, response.Data.Items, 1)
		assert.Equal(t, page.taskID, response.Data.Items[0].TaskID)
		assert.NotContains(t, recorder.Body.String(), "response-body")
		assert.NotContains(t, recorder.Body.String(), "private_data")
	}
	// User-supplied user_id cannot widen self queries.
	c, recorder := imageLogReadContext("/api/image-log/self?user_id=8", "", 7, common.RoleCommonUser)
	GetUserImageLogs(c)
	assert.NotContains(t, recorder.Body.String(), "foreign-image")
	assert.NotContains(t, recorder.Body.String(), `"task_id":"video"`)
}

func TestImageLogDoesNotMistakeVideoThumbnailsForImageGeneration(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Action = "text_to_video"
	require.NoError(t, model.DB.Save(task).Error)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/vendor/videos", strings.NewReader(`{"model":"video"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	finish := beginImageLogCapture(c)
	c.Set(imageLogTaskContextKey, task)
	c.Data(200, "application/json", []byte(`{"video_url":"https://cdn.example/video.mp4","image_url":"https://cdn.example/cover.png"}`))
	finish()
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Nil(t, reloaded.PrivateData.Response)
	assert.Empty(t, reloaded.PrivateData.ImageArtifacts)
}

func TestImageLogReusesDurableImageTasksWithoutDuplicateRecordsOrSettlement(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Action = "imagine"
	task.Quota = 456
	task.PrivateData.ClientRequest = &model.TaskClientRequestSnapshot{Method: "POST", Path: "/native/images", BodyKind: "json", Body: []byte(`{"model":"client","seed":0}`)}
	task.PrivateData.PluginState = []byte(`{"queryable":false}`)
	task.SetData(map[string]any{"model": "mapped", "data": []any{map[string]any{"url": "https://cdn.example/retained.png"}}})
	require.NoError(t, model.DB.Save(task).Error)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/native/images", strings.NewReader(`{"model":"converted"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	finish := beginImageLogCapture(c)
	c.Set(imageLogTaskContextKey, task)
	events := []string{}
	info := taskSubmissionRelayInfo(&taskSubmissionTestBilling{events: &events})
	c.Set(imageLogRelayInfoContextKey, info)
	c.Data(200, "application/json", []byte(`{"data":[{"url":"https://cdn.example/retained.png"}]}`))
	finish()
	var count int64
	require.NoError(t, model.DB.Model(&model.Task{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	assert.Empty(t, events, "image auditing must never call reserve, settle or refund")
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, 456, reloaded.Quota)
	assert.Equal(t, "synchronous", reloaded.PrivateData.ImageRequestMode)
	require.NotNil(t, reloaded.PrivateData.Response)
	assert.JSONEq(t, `{"model":"client","seed":0}`, string(reloaded.PrivateData.ClientRequest.Body))
	assert.JSONEq(t, `{"queryable":false}`, string(reloaded.PrivateData.PluginState))
	assert.JSONEq(t, string(task.Data), string(reloaded.Data))
}

func TestImageLogAuditPreservesNewerPersistedPluginState(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Action = "imagine"
	task.PrivateData.PluginState = []byte(`{"revision":"stale"}`)
	require.NoError(t, model.DB.Save(task).Error)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"client"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	finish := beginImageLogCapture(c)
	c.Set(imageLogTaskContextKey, task)
	c.Data(http.StatusOK, "application/json", []byte(`{"data":[{"url":"https://cdn.example/result.png"}]}`))
	// A concurrent poller won the terminal CAS. The protocol observer may
	// still hold its losing in-memory terminal task while rendering the reply.
	winner := *task
	winner.PrivateData.PluginState = []byte(`{"revision":"winner"}`)
	winner.PrivateData.ResultURL = "https://cdn.example/winner.png"
	winner.PrivateData.ClientRequest = &model.TaskClientRequestSnapshot{BodyKind: "json", Body: []byte(`{"model":"original","seed":0}`)}
	require.NoError(t, model.DB.Save(&winner).Error)
	finish()
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.JSONEq(t, `{"revision":"winner"}`, string(reloaded.PrivateData.PluginState))
	assert.Equal(t, winner.PrivateData.ResultURL, reloaded.PrivateData.ResultURL)
	require.NotNil(t, reloaded.PrivateData.ClientRequest)
	assert.JSONEq(t, `{"model":"original","seed":0}`, string(reloaded.PrivateData.ClientRequest.Body))
	require.NotNil(t, reloaded.PrivateData.Response)
	assert.Equal(t, http.StatusOK, reloaded.PrivateData.Response.StatusCode)
}

func TestImageLogPreservesFailedAndAbortedResponsesWithoutPolling(t *testing.T) {
	setupGenericTaskTest(t)
	for _, tc := range []struct {
		name, response, contentType string
		status                      int
		aborted                     bool
	}{
		{"upstream failure", `{"error":{"message":"provider rejected image","api_key":"private-key"}}`, "application/json", 400, false},
		{"stream error", "data: {\"type\":\"error\",\"error\":{\"message\":\"failed\"}}\n\n", "text/event-stream", 200, false},
		{"client abort", "data: {\"type\":\"image_generation.partial_image\",\"b64_json\":\"cGFydGlhbA==\"}\n\n", "text/event-stream", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"m"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("id", 7)
			finish := beginImageLogCapture(c)
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 1}, OriginModelName: "m"}
			if tc.aborted {
				info.StreamStatus = relaycommon.NewStreamStatus()
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, nil)
			}
			c.Set(imageLogRelayInfoContextKey, info)
			c.Data(tc.status, tc.contentType, []byte(tc.response))
			finish()
			var task model.Task
			require.NoError(t, model.DB.Order("id desc").First(&task).Error)
			assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), task.Status)
			assert.Equal(t, tc.status, task.PrivateData.Response.StatusCode)
			assert.NotContains(t, string(task.PrivateData.Response.Body), "private-key")
		})
	}
	assert.Empty(t, model.GetAllUnFinishSyncTasks(100))
}

func TestImageLogResponseSnapshotSizeLimitAndAsyncResultRedaction(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Action = "imagine"
	task.PrivateData.Key = "private-channel-canary"
	task.SetData(map[string]any{"message": "private-channel-canary", "data": []any{map[string]any{"url": "https://cdn.example/image?access=private-signature"}}, "authorization": "private-value"})
	require.NoError(t, model.DB.Save(task).Error)
	c, recorder := imageLogReadContext("/api/task/"+task.TaskID+"/response", task.TaskID, 7, common.RoleCommonUser)
	GetDashboardTaskResponse(c)
	assert.Contains(t, recorder.Body.String(), `"source":"upstream_result"`)
	assert.NotContains(t, recorder.Body.String(), "private-channel-canary")
	assert.NotContains(t, recorder.Body.String(), "private-signature")
	assert.NotContains(t, recorder.Body.String(), "private-value")
	large := `{"data":[{"url":"https://cdn.example/image"}],"description":"` + strings.Repeat("x", service.MaxTaskClientRequestSnapshotBytes) + `"}`
	snapshot := service.BuildTaskResponseSnapshot(c, []byte(large), 200, "application/json", "json")
	assert.Equal(t, "size_limit", snapshot.BodyOmitted)
	assert.Empty(t, snapshot.Body)
	encoded, err := common.Marshal(snapshot)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(encoded), service.MaxTaskClientRequestSnapshotBytes)
}

func imageLogReadContext(path, taskID string, userID, role int) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	c.Params = gin.Params{{Key: "task_id", Value: taskID}}
	c.Set("id", userID)
	c.Set("role", role)
	return c, recorder
}

func TestImageLogResponseIsOwnerAdminOnlyAndNeverExposesPrivateState(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Action = "image_generation"
	task.PrivateData.Response = &model.TaskResponseSnapshot{StatusCode: 200, BodyKind: "json", Body: []byte(`{"data":[],"seed":0}`)}
	task.PrivateData.Key = "channel-key-secret"
	task.PrivateData.PluginState = []byte(`{"token":"upstream-state-secret"}`)
	require.NoError(t, model.DB.Save(task).Error)
	for _, viewer := range []struct {
		name               string
		user, role, status int
		access             bool
	}{
		{"owner", 7, common.RoleCommonUser, 200, false}, {"admin", 8, common.RoleAdminUser, 200, false}, {"other user", 8, common.RoleCommonUser, 404, false}, {"artifact capability", 7, common.RoleCommonUser, 404, true},
	} {
		t.Run(viewer.name, func(t *testing.T) {
			c, recorder := imageLogReadContext("/api/task/"+task.TaskID+"/response", task.TaskID, viewer.user, viewer.role)
			if viewer.access {
				c.Set(middleware.TaskArtifactAccessContextKey, true)
			}
			GetDashboardTaskResponse(c)
			assert.Equal(t, viewer.status, recorder.Code)
			assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
			assert.NotContains(t, recorder.Body.String(), "channel-key-secret")
			assert.NotContains(t, recorder.Body.String(), "upstream-state-secret")
			if viewer.status == 200 {
				assert.Contains(t, recorder.Body.String(), `"seed":0`)
			}
		})
	}
}
