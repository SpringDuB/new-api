package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const imageLogRelayInfoContextKey = "image_log_relay_info"
const imageLogTaskContextKey = "image_log_task"
const imageLogRequestIntentContextKey = "image_log_request_intent"

// Capture is bounded independently of request/billing limits. SSE partial
// images are ignored; only final events are retained, up to 64 MiB/64 events.
// Persisted request/response snapshots remain bounded to 64 KiB each.
const maxImageLogResponseBytes = 64 << 20

type imageLogResponseWriter struct {
	gin.ResponseWriter
	mu          sync.Mutex
	body        bytes.Buffer
	line        bytes.Buffer
	event       bytes.Buffer
	events      []json.RawMessage
	size        int
	omitted     bool
	stream      bool
	streamError bool
}

func (w *imageLogResponseWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.ResponseWriter.Write(p)
	w.capture(p[:n])
	return n, err
}

func (w *imageLogResponseWriter) WriteString(p string) (int, error) {
	return w.Write([]byte(p))
}

func (w *imageLogResponseWriter) capture(p []byte) {
	w.stream = strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream")
	if !w.stream {
		if w.omitted {
			return
		}
		if w.body.Len()+len(p) > maxImageLogResponseBytes {
			w.body.Reset()
			w.omitted = true
			return
		}
		_, _ = w.body.Write(p)
		return
	}
	for len(p) > 0 {
		line, rest, complete := bytes.Cut(p, []byte{'\n'})
		if w.line.Len()+len(line) > maxImageLogResponseBytes {
			w.omitted = true
			w.line.Reset()
			return
		}
		_, _ = w.line.Write(line)
		if !complete {
			return
		}
		raw := bytes.TrimSuffix(w.line.Bytes(), []byte{'\r'})
		if len(raw) == 0 {
			w.captureEvent(w.event.Bytes())
			w.event.Reset()
		} else if data, ok := bytes.CutPrefix(raw, []byte("data:")); ok {
			data = bytes.TrimSpace(data)
			if w.event.Len()+len(data)+1 <= maxImageLogResponseBytes {
				_, _ = w.event.Write(data)
				_ = w.event.WriteByte('\n')
			} else {
				w.omitted = true
			}
		}
		w.line.Reset()
		p = rest
	}
}

func (w *imageLogResponseWriter) captureEvent(data []byte) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) || !gjson.ValidBytes(data) {
		return
	}
	typ := gjson.GetBytes(data, "type").String()
	failure := gjson.GetBytes(data, "error")
	eventError := typ == "error" || strings.Contains(typ, "failed") || failure.Exists() && failure.Type != gjson.Null || gjson.GetBytes(data, "response.status").String() == "failed"
	w.streamError = w.streamError || eventError
	// Ignore text/reasoning/tool completion events. Ark partial_succeeded is
	// a complete image, unlike progressive partial_image or delta events.
	relevant := eventError || typ == "image_generation.partial_succeeded" || typ == "image_generation.completed" || typ == "image_edit.completed" ||
		bytes.Contains(data, []byte(`"image_generation_call"`)) && bytes.Contains(data, []byte(`"result"`)) ||
		bytes.Contains(data, []byte(`"b64_json"`)) || bytes.Contains(data, []byte(`"image_url"`)) ||
		bytes.Contains(data, []byte(`"inlineData"`)) || bytes.Contains(data, []byte(`"inline_data"`)) ||
		bytes.Contains(data, []byte(`"bytesBase64Encoded"`)) || bytes.Contains(data, []byte(`"images"`)) || bytes.Contains(data, []byte(`"image"`))
	if !relevant || strings.Contains(typ, "partial_image") || strings.Contains(typ, "delta") {
		return
	}
	if w.size+len(data) <= maxImageLogResponseBytes && len(w.events) < 64 {
		w.events = append(w.events, bytes.Clone(data))
		w.size += len(data)
	} else {
		w.omitted = true
	}
}

// Preserve only an intent flag from the parsed client DTO before mapping.
// Large request snapshots may omit their body without losing failure logs.
func captureImageLogRequestIntent(c *gin.Context, request dto.Request) {
	var modalities []string
	switch request := request.(type) {
	case *dto.GeneralOpenAIRequest:
		for _, value := range gjson.ParseBytes(request.Modalities).Array() {
			modalities = append(modalities, value.String())
		}
	case *dto.GeminiChatRequest:
		modalities = request.GenerationConfig.ResponseModalities
	case *dto.OpenAIResponsesRequest:
		for _, tool := range gjson.ParseBytes(request.Tools).Array() {
			if tool.Get("type").String() == "image_generation" {
				c.Set(imageLogRequestIntentContextKey, true)
				return
			}
		}
	}
	for _, modality := range modalities {
		if strings.EqualFold(modality, "image") {
			c.Set(imageLogRequestIntentContextKey, true)
			return
		}
	}
}

func beginImageLogCapture(c *gin.Context) func() {
	if c.Request == nil || c.Request.Method != http.MethodPost {
		return func() {}
	}
	if _, captured := c.Writer.(*imageLogResponseWriter); captured {
		return func() {}
	}
	service.CaptureTaskClientRequest(c, nil, nil)
	writer := &imageLogResponseWriter{ResponseWriter: c.Writer}
	c.Writer = writer
	started := time.Now().Unix()
	return func() {
		c.Writer = writer.ResponseWriter
		writer.mu.Lock()
		defer writer.mu.Unlock()
		body := writer.body.Bytes()
		kind := "json"
		if writer.stream {
			kind = "events"
			body, _ = common.Marshal(map[string]any{"events": writer.events})
		}
		taskValue, _ := c.Get(imageLogTaskContextKey)
		task, _ := taskValue.(*model.Task)
		intent := imageLogRequestIntent(c) || task != nil && model.IsImageTaskAction(task.Action)
		if task != nil && !intent {
			return
		}
		artifacts := service.ExtractImageResponseArtifacts(body, intent)
		if !intent && len(artifacts) == 0 {
			return
		}
		if task != nil && task.Status != model.TaskStatusSuccess && task.Status != model.TaskStatusFailure {
			return // the task poller owns the mutable result; expose it on demand
		}
		infoValue, _ := c.Get(imageLogRelayInfoContextKey)
		info, _ := infoValue.(*relaycommon.RelayInfo)
		if task == nil {
			userID := c.GetInt("id")
			if userID <= 0 {
				return
			}
			task = &model.Task{TaskID: model.GenerateTaskID(), Platform: "image", UserId: userID,
				ChannelId: common.GetContextKeyInt(c, constant.ContextKeyChannelId),
				Action:    "image_generation", Status: model.TaskStatusSuccess, Progress: "100%",
				SubmitTime: started, StartTime: started, FinishTime: time.Now().Unix(),
				Properties: model.Properties{OriginModelName: common.GetContextKeyString(c, constant.ContextKeyOriginalModel)}}
			if strings.Contains(c.Request.URL.Path, "/edits") {
				task.Action = "image_edit"
			}
			if info != nil {
				task.Group = info.UsingGroup
				task.Properties.OriginModelName = info.OriginModelName
				if info.ChannelMeta != nil {
					task.ChannelId = info.ChannelId
					task.Properties.UpstreamModelName = info.UpstreamModelName
				}
			}
			task.Quota = c.GetInt(service.SettledQuotaContextKey)
			task.PrivateData.Execution = service.TaskExecutionSnapshotFromContext(c)
			if snapshot, ok := c.Get(service.TaskClientRequestContextKey); ok {
				task.PrivateData.ClientRequest, _ = snapshot.(*model.TaskClientRequestSnapshot)
			}
			streamFailed := false
			if info != nil && info.StreamStatus != nil {
				outcome := info.StreamStatus.OutcomeSnapshot()
				streamFailed = outcome.HasErrors || outcome.Response == relaycommon.ResponseOutcomeFailed || outcome.Response == relaycommon.ResponseOutcomeIncomplete || outcome.Response == relaycommon.ResponseOutcomeCancelled ||
					outcome.EndReason != relaycommon.StreamEndReasonNone && outcome.EndReason != relaycommon.StreamEndReasonDone && outcome.EndReason != relaycommon.StreamEndReasonEOF
			}
			failure := gjson.GetBytes(body, "error")
			if writer.Status() >= 400 || writer.streamError || streamFailed || failure.Exists() && failure.Type != gjson.Null {
				task.Status = model.TaskStatusFailure
				task.FailReason = "Image generation failed"
			}
		}
		task.PrivateData.ImageRequestMode = "synchronous"
		task.PrivateData.Response = service.BuildTaskResponseSnapshot(c, body, writer.Status(), writer.Header().Get("Content-Type"), kind)
		if writer.omitted {
			task.PrivateData.Response.BodyOmitted = "size_limit"
		}
		// Retained plugin tasks keep their refresh/query/content hooks. The
		// host owns ordinary relays and separately archived inline results.
		if !taskHasPluginExecution(task) || task.PrivateData.ResultDiscarded {
			if task.PrivateData.ResultDiscarded && taskHasPluginExecution(task) {
				if projected, err := taskStorageArtifacts(task); err == nil {
					index := 0
					for _, artifact := range projected {
						if artifact.Type == "image" && index < len(artifacts) {
							artifacts[index].Key = artifact.Key
							index++
						}
					}
				}
			}
			task.PrivateData.ImageArtifacts = artifacts
		}
		// Never put image binaries in SQL, including the initial INSERT.
		inline := make(map[string]string)
		for i := range task.PrivateData.ImageArtifacts {
			artifact := &task.PrivateData.ImageArtifacts[i]
			if strings.HasPrefix(artifact.SourceURL, "data:") {
				inline[artifact.Key] = artifact.SourceURL
				artifact.SourceURL = ""
			}
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Minute)
		defer cancel()
		if task.ID == 0 {
			if err := task.InsertWithContext(ctx); err != nil {
				logger.LogTaskArtifactFailure(ctx, task.TaskID, "image_log", 0)
				return
			}
		}
		writeCtx, stopWrite := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 10*time.Second)
		defer stopWrite()
		if err := model.UpdateTaskImageLog(writeCtx, task); err != nil {
			logger.LogTaskArtifactFailure(ctx, task.TaskID, "image_log", 0)
			return
		}
		for i := range task.PrivateData.ImageArtifacts {
			if source := inline[task.PrivateData.ImageArtifacts[i].Key]; source != "" {
				task.PrivateData.ImageArtifacts[i].SourceURL = source
			}
		}
		service.ScheduleTaskArtifactArchive(task)
	}
}

func imageLogRequestIntent(c *gin.Context) bool {
	if c.GetBool(imageLogRequestIntentContextKey) || strings.Contains(c.Request.URL.Path, "/images/") {
		return true
	}
	value, _ := c.Get(service.TaskClientRequestContextKey)
	snapshot, _ := value.(*model.TaskClientRequestSnapshot)
	if snapshot == nil {
		return false
	}
	for _, path := range []string{"modalities", "generationConfig.responseModalities", "generation_config.response_modalities"} {
		for _, item := range gjson.GetBytes(snapshot.Body, path).Array() {
			if strings.EqualFold(item.String(), "image") {
				return true
			}
		}
	}
	for _, tool := range gjson.GetBytes(snapshot.Body, "tools").Array() {
		if tool.Get("type").String() == "image_generation" {
			return true
		}
	}
	modelName := common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	return strings.Contains(modelName, "-image") || strings.HasPrefix(modelName, "imagen-") || model.IsImageTaskAction(c.GetString("task_action"))
}

func imageLogArtifact(artifact model.TaskImageArtifact) types.TaskArtifact {
	return types.TaskArtifact{Key: artifact.Key, Type: "image", MimeType: artifact.MimeType}
}

func GetDashboardTaskResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if middleware.IsTaskArtifactAccess(c) {
		writeTaskArtifactError(c, http.StatusNotFound, "artifact_not_found", "Task or artifact not found")
		return
	}
	task, exists, err := getTaskForArtifactRequest(c, c.Param("task_id"))
	if err != nil || !exists || task == nil {
		writeTaskArtifactError(c, http.StatusNotFound, "artifact_not_found", "Task or artifact not found")
		return
	}
	snapshot := task.PrivateData.Response
	source := "client_response"
	if snapshot == nil && len(task.Data) > 0 {
		source = "upstream_result"
		secrets := []string{task.PrivateData.Key}
		if channel, channelErr := model.CacheGetChannel(task.ChannelId); channelErr == nil {
			for key := range strings.SplitSeq(channel.Key, "\n") {
				if key != "" {
					secrets = append(secrets, key)
				}
			}
		}
		snapshot = service.BuildTaskResponseSnapshot(c, task.Data, 0, "application/json", "json", secrets...)
	}
	common.ApiSuccess(c, gin.H{"recorded": snapshot != nil, "snapshot": snapshot, "source": source, "max_bytes": service.MaxTaskClientRequestSnapshotBytes, "retention": "task_lifetime"})
}

func GetAllImageLogs(c *gin.Context)  { getImageLogs(c, true) }
func GetUserImageLogs(c *gin.Context) { getImageLogs(c, false) }

func getImageLogs(c *gin.Context, admin bool) {
	page := common.GetPageQuery(c)
	params := model.ImageLogQuery{TaskID: c.Query("task_id"), ChannelID: c.Query("channel_id"), StartTimestamp: c.Query("start_timestamp"), EndTimestamp: c.Query("end_timestamp")}
	if !admin {
		params.UserID = c.GetInt("id")
		params.ChannelID = ""
	}
	tasks, count, err := model.GetImageLogs(page.GetStartIdx(), page.GetPageSize(), params)
	if err != nil {
		common.ApiErrorT(c, "Failed to query image logs")
		return
	}
	role := common.RoleCommonUser
	if admin {
		role = c.GetInt("role")
	}
	items := tasksToDto(tasks, admin, role)
	for i, task := range tasks {
		if strings.HasPrefix(task.TaskID, "mjlog_") {
			items[i].DetailID = task.TaskID
			items[i].TaskID = task.PrivateData.UpstreamTaskID
		}
	}
	page.SetItems(items)
	page.SetTotal(int(count))
	common.ApiSuccess(c, page)
}
