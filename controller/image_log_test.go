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
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relay"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
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
		{"responses null error", "/v1/responses", `{"model":"m","tools":[{"type":"image_generation"}]}`, `{"status":"completed","error":null,"output":[{"type":"image_generation_call","result":"aW1hZ2U="}]}`, "application/json", 1},
		{"image null error", "/v1/images/generations", `{"model":"m"}`, `{"error":null,"data":[{"url":"https://cdn.example/output.png"}]}`, "application/json", 1},
		{"stream null error", "/v1/images/generations", `{"model":"m","stream":true}`, "data: {\"type\":\"image_generation.completed\",\"error\":null,\"b64_json\":\"aW1hZ2U=\"}\n\n", "text/event-stream", 1},
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
				assert.Equal(t, model.TaskStatus(model.TaskStatusSuccess), task.Status)
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
	// Images belong exclusively to drawing history. Task rows, totals and
	// pagination must all exclude them for both owner and admin queries.
	userTasks := model.TaskGetAllUserTask(7, 0, 10, model.SyncTaskQueryParams{})
	require.Len(t, userTasks, 2)
	assert.Equal(t, "video", userTasks[0].TaskID)
	assert.Equal(t, "task_generic", userTasks[1].TaskID)
	assert.EqualValues(t, 2, model.TaskCountAllUserTask(7, model.SyncTaskQueryParams{}))
	adminTasks := model.TaskGetAllTasks(0, 10, model.SyncTaskQueryParams{})
	require.Len(t, adminTasks, 2)
	assert.EqualValues(t, 2, model.TaskCountAllTasks(model.SyncTaskQueryParams{}))
	assert.Empty(t, model.TaskGetAllUserTask(8, 0, 10, model.SyncTaskQueryParams{}))
	assert.Zero(t, model.TaskCountAllUserTask(8, model.SyncTaskQueryParams{}))
	page := model.TaskGetAllUserTask(7, 1, 1, model.SyncTaskQueryParams{})
	require.Len(t, page, 1)
	assert.Equal(t, "task_generic", page[0].TaskID)
	assert.Empty(t, model.TaskGetAllTasks(0, 10, model.SyncTaskQueryParams{Action: "imagine"}))
	assert.Zero(t, model.TaskCountAllTasks(model.SyncTaskQueryParams{TaskID: "image-sync"}))

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
	require.NotNil(t, reloaded.PrivateData.Response)
	assert.Contains(t, string(reloaded.PrivateData.Response.Body), "video_url")
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
		{"JSON error in HTTP 200", `{"error":{"message":"provider rejected image"}}`, "application/json", 200, false},
		{"stream error", "data: {\"type\":\"error\",\"error\":{\"message\":\"failed\"}}\n\n", "text/event-stream", 200, false},
		{"image event error", "data: {\"type\":\"image_generation.completed\",\"error\":{\"message\":\"failed\"}}\n\n", "text/event-stream", 200, false},
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
	c, recorder := imageLogReadContext("/api/task/"+task.TaskID+"/response?source=upstream_result", task.TaskID, 7, common.RoleAdminUser)
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

func TestDashboardTaskResponseUsesPlatformVideoRenderer(t *testing.T) {
	task := setupGenericTaskTest(t)
	source := `export const meta={apiVersion:1,key:"response-preview",name:"Preview",version:"1.0.0",author:{name:"Test"},models:["m"],fetchMode:"per_task",protocols:["openai_video"]};
export function buildSubmitRequest(){return {}} export function parseSubmitResponse(){return {taskId:"vendor"}} export function buildQueryRequest(){return {}} export function parseTaskResult(){return {status:"SUCCESS"}}
export function listArtifacts(){return []} export function buildContentRequest(){return {}}
export const protocols={openai_video:{decodeRequest(){return {}},render(ctx,task){if(task.data.unavailable)throw new Error("renderer unavailable");return {id:"vendor",status:"provider-success",url:task.data.url,platform_field:true}}}};`
	_, err := jsplugin.DefaultRegistry.RegisterFactory(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister("response-preview") })
	task.Platform = "response-preview"
	task.Properties.OriginModelName = "m"
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{RequestPath: "/v1/videos"}
	task.SetData(map[string]any{"id": "vendor", "status": "provider-success", "url": "https://cdn.example/video.mp4"})
	require.NoError(t, model.DB.Save(task).Error)
	converter, ok := relay.GetTaskAdaptor(task.Platform).(relaychannel.OpenAIVideoConverter)
	require.True(t, ok)
	wire, err := converter.ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	c, recorder := imageLogReadContext("/api/task/"+task.TaskID+"/response", task.TaskID, 7, common.RoleCommonUser)
	GetDashboardTaskResponse(c)
	var result struct {
		Data struct {
			Source      string                      `json:"source"`
			Snapshot    *model.TaskResponseSnapshot `json:"snapshot"`
			Unavailable bool                        `json:"unavailable"`
			Submission  *model.TaskResponseSnapshot `json:"submission_snapshot"`
		}
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
	assert.Equal(t, "current_platform_result", result.Data.Source)
	require.NotNil(t, result.Data.Snapshot)
	assert.JSONEq(t, string(wire), string(result.Data.Snapshot.Body))
	assert.NotContains(t, recorder.Body.String(), "upstream_result")
	t.Run("failed projection keeps the submission available", func(t *testing.T) {
		task.SetData(map[string]any{"unavailable": true, "raw": "upstream-canary"})
		task.PrivateData.SubmissionResponse = &model.TaskResponseSnapshot{Body: []byte(`{"status":"queued"}`)}
		require.NoError(t, model.DB.Save(task).Error)
		c, recorder := imageLogReadContext("/api/task/"+task.TaskID+"/response", task.TaskID, 7, common.RoleCommonUser)
		GetDashboardTaskResponse(c)
		require.Equal(t, http.StatusOK, recorder.Code)
		result.Data.Snapshot = nil
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
		assert.True(t, result.Data.Unavailable)
		assert.Nil(t, result.Data.Snapshot)
		require.NotNil(t, result.Data.Submission)
		assert.JSONEq(t, `{"status":"queued"}`, string(result.Data.Submission.Body))
		assert.NotContains(t, recorder.Body.String(), "upstream-canary")
	})
}

func TestDashboardTaskResponseNeverSubstitutesUpstreamForMissingPlatformResponse(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.SetData(map[string]any{"raw": "upstream-canary"})
	require.NoError(t, model.DB.Save(task).Error)
	for _, test := range []struct {
		query                       string
		user, role, tokenID, status int
		capability                  bool
	}{
		{query: "", user: 7, role: common.RoleCommonUser, status: 200},
		{query: "?source=upstream_result", user: 7, role: common.RoleCommonUser, status: 403},
		{query: "?source=upstream_result", user: 8, role: common.RoleCommonUser, status: 404},
		{query: "?source=upstream_result", user: 8, role: common.RoleAdminUser, status: 200},
		{query: "?source=upstream_result", user: 7, role: common.RoleAdminUser, tokenID: 1, status: 403},
		{query: "?source=upstream_result", user: 7, role: common.RoleAdminUser, capability: true, status: 404},
	} {
		c, recorder := imageLogReadContext("/api/task/"+task.TaskID+"/response"+test.query, task.TaskID, test.user, test.role)
		c.Set("token_id", test.tokenID)
		if test.capability {
			c.Set(middleware.TaskArtifactAccessContextKey, true)
		}
		GetDashboardTaskResponse(c)
		assert.Equal(t, test.status, recorder.Code)
		if test.query == "" {
			assert.Contains(t, recorder.Body.String(), `"recorded":false`)
		}
		assert.Equal(t, test.query != "" && test.status == 200, strings.Contains(recorder.Body.String(), "upstream-canary"))
	}
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

func TestTaskSubmissionResponseSurvivesStalePollingAndKeepsUpstreamPrivate(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Status = model.TaskStatusQueued
	task.Progress = "0%"
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{RequestPath: "/v1/videos"}
	task.SetData(map[string]any{"upstream": "provider-snapshot"})
	require.NoError(t, model.DB.Save(task).Error)
	stale := *task
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(`{"model":"m"}`))
	c.Set("id", task.UserId)
	finish := beginImageLogCapture(c)
	c.Set(imageLogTaskContextKey, task)
	c.JSON(http.StatusOK, gin.H{"id": task.TaskID, "status": "queued"})
	finish()
	assert.Contains(t, recorder.Body.String(), `"status":"queued"`)
	stale.Status = model.TaskStatusInProgress
	stale.Progress = "50%"
	won, err := stale.UpdateWithStatus(model.TaskStatusQueued)
	require.NoError(t, err)
	require.True(t, won)
	reloaded, exists, err := model.GetByTaskId(task.UserId, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	require.NotNil(t, reloaded.PrivateData.SubmissionResponse)
	assert.JSONEq(t, `{"id":"task_generic","status":"queued"}`, string(reloaded.PrivateData.SubmissionResponse.Body))
	assert.Greater(t, reloaded.PrivateData.SubmissionResponse.CapturedAt, int64(0))
	assert.Equal(t, model.TaskStatus(model.TaskStatusInProgress), reloaded.Status)
	assert.JSONEq(t, `{"upstream":"provider-snapshot"}`, string(reloaded.Data))
	// A second response capture cannot replace the original submission.
	require.NoError(t, model.UpdateTaskResponseLog(c, reloaded, &model.TaskResponseSnapshot{Body: []byte(`{"status":"replaced"}`)}, true))
	reloaded, exists, err = model.GetByTaskId(task.UserId, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.JSONEq(t, `{"id":"task_generic","status":"queued"}`, string(reloaded.PrivateData.SubmissionResponse.Body))
}

func TestDashboardTaskResponseMatchesNativeAndResponsesProtocols(t *testing.T) {
	task := setupGenericTaskTest(t)
	source := `export const meta={apiVersion:1,key:"response-protocol",name:"Response",version:"1.0.0",author:{name:"Test"},models:["m"],fetchMode:"per_task",routes:[{method:"POST",path:"/vendor/jobs",type:"submit",decode:"create",render:"created"},{method:"GET",path:"/vendor/jobs/:task_id",type:"query",render:"query"}],protocols:[{name:"openai_responses",supports:["background"]}]};
 export function buildSubmitRequest(){return {}} export function parseSubmitResponse(){return {taskId:"vendor"}} export function buildQueryRequest(){return {}} export function parseTaskResult(){return {status:"SUCCESS"}}
 export const native={create(){return {kind:"submit",model:"m"}},created(ctx,task){return {id:task.task_id}},query(ctx,task){return {__status:202,__body:{id:task.task_id,state:task.status,query:ctx.params.task_id}}}};
 export const protocols={openai_responses:{decodeRequest(){return {kind:"submit",model:"m"}},renderFinal(){return {output:[{type:"message",role:"assistant",status:"completed",content:[{type:"output_text",text:"platform-result",annotations:[],logprobs:[]}]}]}}}};`
	plugin, err := jsplugin.DefaultRegistry.RegisterFactory(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister("response-protocol") })
	task.Platform = "response-protocol"
	task.Properties.OriginModelName = "m"
	for _, path := range []string{"/vendor/jobs", "/v1/responses"} {
		task.PrivateData.Execution = &model.TaskExecutionSnapshot{RequestPath: path}
		for _, status := range []model.TaskStatus{model.TaskStatusQueued, model.TaskStatusSuccess, model.TaskStatusFailure} {
			task.Status = status
			task.SetData(map[string]any{"raw": "upstream-canary"})
			require.NoError(t, model.DB.Save(task).Error)
			c, recorder := imageLogReadContext("/api/task/"+task.TaskID+"/response", task.TaskID, 7, common.RoleCommonUser)
			GetDashboardTaskResponse(c)
			require.Equal(t, http.StatusOK, recorder.Code)
			var result struct {
				Data struct {
					Source   string
					Snapshot *model.TaskResponseSnapshot
				}
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
			require.NotNil(t, result.Data.Snapshot)
			assert.Equal(t, "current_platform_result", result.Data.Source)
			if path == "/v1/responses" {
				wireContext, wire := newPluginProtocolRetrieveContext("resp_generic")
				common.SetContextKey(wireContext, constant.ContextKeyUserId, task.UserId)
				pinned := jsplugin.PinnedEndpoint{Plugin: plugin, Protocol: "openai_responses"}
				retrieveTaskPluginResponse(wireContext, pluginProtocolRetrieveDeps(pinned, task, true, nil))
				assert.JSONEq(t, wire.Body.String(), string(result.Data.Snapshot.Body))
			} else {
				assert.Equal(t, http.StatusAccepted, result.Data.Snapshot.StatusCode)
				assert.JSONEq(t, `{"id":"task_generic","state":"`+string(status)+`","query":"task_generic"}`, string(result.Data.Snapshot.Body))
			}
			assert.NotContains(t, recorder.Body.String(), "upstream-canary")
		}
	}
}

func TestTaskClientResponseCaptureDistinguishesSynchronousAndBackgroundReplies(t *testing.T) {
	for _, tc := range []struct {
		name                                                string
		terminal, responses, background, stream, submission bool
	}{
		{name: "synchronous native", terminal: true},
		{name: "synchronous Responses", responses: true},
		{name: "background Responses", responses: true, background: true, submission: true},
		{name: "streamed Responses", responses: true, background: true, stream: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task := setupGenericTaskTest(t)
			task.Status = model.TaskStatusQueued
			if tc.terminal {
				task.Status = model.TaskStatusSuccess
			}
			task.PrivateData.ResponsesBackground = tc.background
			require.NoError(t, model.DB.Save(task).Error)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))
			c.Set("id", task.UserId)
			if tc.responses {
				c.Set(jsplugin.ContextKeyProtocolRequest, jsplugin.ProtocolRequestContext{Protocol: "openai_responses"})
			}
			finish := beginImageLogCapture(c)
			c.Set(imageLogTaskContextKey, task)
			if tc.stream {
				c.Data(http.StatusOK, "text/event-stream", []byte("data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[]}}\n\n"))
			} else {
				c.JSON(http.StatusOK, gin.H{"status": "completed", "id": task.TaskID})
			}
			finish()
			stored, exists, err := model.GetByTaskId(task.UserId, task.TaskID)
			require.NoError(t, err)
			require.True(t, exists)
			if tc.submission {
				require.NotNil(t, stored.PrivateData.SubmissionResponse)
				assert.Nil(t, stored.PrivateData.Response)
			} else {
				require.NotNil(t, stored.PrivateData.Response)
				assert.Contains(t, string(stored.PrivateData.Response.Body), `"completed"`)
				assert.Nil(t, stored.PrivateData.SubmissionResponse)
			}
		})
	}
}
