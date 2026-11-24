package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	asyncImageAction              = "image_generation"
	maxAsyncImageTaskRequestBytes = 1 << 20
)

var errAsyncImageBase64StorageRequired = errors.New("base64 image results require task artifact object storage")

type asyncImageTaskContextKey struct{}
type asyncImageResultContextKey struct{}

type asyncImageRunResult struct {
	ChannelID int
	Model     string
	Quota     int
}

type asyncImageTaskResponse struct {
	ID        string               `json:"id"`
	Object    string               `json:"object"`
	Status    string               `json:"status"`
	CreatedAt int64                `json:"created_at"`
	Progress  string               `json:"progress,omitempty"`
	Error     *asyncImageTaskError `json:"error,omitempty"`
	Data      []asyncImageTaskData `json:"data,omitempty"`
}

type asyncImageTaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type asyncImageTaskData struct {
	URL string `json:"url"`
}

func normalizeAsyncImageRequest(body []byte, objectStoreEnabled bool) ([]byte, error) {
	var request map[string]json.RawMessage
	if err := common.Unmarshal(body, &request); err != nil {
		return nil, err
	}
	var responseFormat string
	if raw := request["response_format"]; len(raw) > 0 {
		if err := common.Unmarshal(raw, &responseFormat); err != nil {
			return nil, err
		}
	}
	switch responseFormat {
	case "", "url", "b64_json":
	default:
		return nil, errors.New("response_format must be url or b64_json")
	}
	if !objectStoreEnabled && responseFormat == "b64_json" {
		return nil, errAsyncImageBase64StorageRequired
	}
	if !objectStoreEnabled && responseFormat == "" {
		request["response_format"] = json.RawMessage(`"url"`)
	}
	return common.Marshal(request)
}

func CreateAsyncImageTask(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if !constant.UpdateTask {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "task_processing_disabled", "message": "Image task processing is disabled"}})
		return
	}
	service.CaptureTaskClientRequest(c, nil, nil)
	request, err := helper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Invalid image generation request"}})
		return
	}
	if request.Stream != nil && *request.Stream {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Streaming is not supported for image tasks"}})
		return
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Invalid image generation request"}})
		return
	}
	body, err := storage.Bytes()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Invalid image generation request"}})
		return
	}
	if len(body) > maxAsyncImageTaskRequestBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": gin.H{"code": "request_too_large", "message": "Image task request is too large"}})
		return
	}
	body, err = normalizeAsyncImageRequest(body, service.GetTaskArtifactStore().Enabled())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}
	if _, err := request.ImageCount(false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "Invalid image count"}})
		return
	}
	userID := c.GetInt("id")
	tokenID := common.GetContextKeyInt(c, constant.ContextKeyTokenId)
	if userID <= 0 || tokenID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "Authentication required"}})
		return
	}
	now := time.Now().Unix()
	group := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
	if group == "" {
		group = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	}
	task := &model.Task{
		TaskID:     model.GenerateTaskID(),
		CreatedAt:  now,
		UpdatedAt:  now,
		UserId:     userID,
		Group:      group,
		Platform:   constant.TaskPlatformAsyncImage,
		Action:     asyncImageAction,
		Status:     model.TaskStatusNotStart,
		Progress:   "0%",
		SubmitTime: now,
		Properties: model.Properties{OriginModelName: request.Model},
		Data:       json.RawMessage(body),
		PrivateData: model.TaskPrivateData{
			TokenId:          tokenID,
			ImageRequestMode: "asynchronous",
		},
	}
	if snapshot, ok := c.Get(service.TaskClientRequestContextKey); ok {
		task.PrivateData.ClientRequest, _ = snapshot.(*model.TaskClientRequestSnapshot)
	}
	if err := task.InsertWithContext(c.Request.Context()); err != nil {
		logger.LogError(c, common.LogText("failed to persist asynchronous image task: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "task_create_failed", "message": "Failed to create image task"}})
		return
	}
	c.JSON(http.StatusAccepted, asyncImageTaskResponse{
		ID:        task.TaskID,
		Object:    "image.task",
		Status:    "queued",
		CreatedAt: task.CreatedAt,
		Progress:  task.Progress,
	})
}

func GetAsyncImageTask(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	task, exists, err := model.GetByTaskId(c.GetInt("id"), c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "task_query_failed", "message": "Failed to query image task"}})
		return
	}
	if !exists || task.Platform != constant.TaskPlatformAsyncImage || task.Action != asyncImageAction {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "task_not_found", "message": "Image task not found"}})
		return
	}
	responseTask := task
	if task.PrivateData.AsyncImageSourceTaskID != "" {
		sourceTask, sourceExists, sourceErr := model.GetByTaskId(task.UserId, task.PrivateData.AsyncImageSourceTaskID)
		if sourceErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "task_query_failed", "message": "Failed to query image task"}})
			return
		}
		if sourceExists {
			responseTask = sourceTask
		}
	}
	response := asyncImageTaskResponse{
		ID:        task.TaskID,
		Object:    "image.task",
		Status:    asyncImageStatus(responseTask.Status),
		CreatedAt: task.CreatedAt,
		Progress:  responseTask.Progress,
	}
	if responseTask.Status == model.TaskStatusFailure {
		response.Error = &asyncImageTaskError{Code: "image_generation_failed", Message: "Image generation failed"}
	}
	if responseTask.Status == model.TaskStatusSuccess {
		response.Data = make([]asyncImageTaskData, 0, len(responseTask.PrivateData.ImageArtifacts))
		for _, artifact := range responseTask.PrivateData.ImageArtifacts {
			if !artifact.Stored && artifact.SourceURL == "" {
				continue
			}
			contentURL, err := service.BuildTaskArtifactContentURL(responseTask.TaskID, artifact.Key)
			if err != nil {
				logger.LogTaskArtifactFailure(c.Request.Context(), task.TaskID, "artifact_url", 0)
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "image_delivery_failed", "message": "Image delivery failed"}})
				return
			}
			response.Data = append(response.Data, asyncImageTaskData{URL: contentURL})
		}
	}
	c.JSON(http.StatusOK, response)
}

func asyncImageStatus(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusNotStart, model.TaskStatusQueued, model.TaskStatusSubmitted:
		return "queued"
	case model.TaskStatusInProgress:
		return "in_progress"
	case model.TaskStatusSuccess:
		return "completed"
	case model.TaskStatusFailure:
		return "failed"
	default:
		return "unknown"
	}
}

func ProcessPendingAsyncImageTasks(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	tasks, err := model.GetPendingAsyncImageTasks(4)
	if err != nil {
		logger.LogWarn(ctx, common.LogText("async image task query failed: %v", err))
		return
	}
	var workers sync.WaitGroup
	for _, candidate := range tasks {
		if ctx.Err() != nil {
			break
		}
		candidate.Status = model.TaskStatusInProgress
		candidate.Progress = "1%"
		candidate.StartTime = time.Now().Unix()
		candidate.UpdatedAt = candidate.StartTime
		claimed, claimErr := candidate.UpdateWithStatus(model.TaskStatusNotStart)
		if claimErr != nil {
			logger.LogWarn(ctx, common.LogText("async image task claim failed task_id=%s: %v", candidate.TaskID, claimErr))
			continue
		}
		if !claimed {
			continue
		}
		workers.Go(func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.LogError(ctx, common.LogText("async image task worker panic task_id=%s", candidate.TaskID))
					failAsyncImageTask(candidate, model.TaskStatusInProgress)
				}
			}()
			processAsyncImageTask(ctx, candidate)
		})
	}
	workers.Wait()
}

func processAsyncImageTask(ctx context.Context, task *model.Task) {
	processAsyncImageTaskWithHandler(ctx, task, asyncImageRelayHandler())
}

func processAsyncImageTaskWithHandler(ctx context.Context, task *model.Task, handler http.Handler) {
	token, err := model.GetTokenById(task.PrivateData.TokenId)
	if err != nil || token.UserId != task.UserId || token.Key == "" {
		failAsyncImageTask(task, model.TaskStatusInProgress)
		return
	}
	result := &asyncImageRunResult{}
	request := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(task.Data))
	request.Header.Set("Authorization", "Bearer "+token.Key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(common.RequestIdKey, task.TaskID)
	request = request.WithContext(context.WithValue(ctx, asyncImageTaskContextKey{}, task))
	request = request.WithContext(context.WithValue(request.Context(), asyncImageResultContextKey{}, result))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if ctx.Err() != nil {
		return
	}
	if recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices {
		logger.LogWarn(ctx, common.LogText("async image task relay failed task_id=%s status=%d", task.TaskID, recorder.Code))
		failAsyncImageTask(task, model.TaskStatusInProgress)
		return
	}
	current, exists, loadErr := model.GetByTaskId(task.UserId, task.TaskID)
	if loadErr == nil && exists {
		task = current
	}
	artifacts := task.PrivateData.ImageArtifacts
	archived := task.PrivateData.ArtifactArchive != nil && task.PrivateData.ArtifactArchive.Status == model.TaskArchiveCompleted && len(artifacts) > 0
	if !archived {
		artifacts = service.ExtractImageResponseArtifacts(recorder.Body.Bytes(), true)
		if len(artifacts) == 0 {
			failAsyncImageTask(task, model.TaskStatusInProgress)
			return
		}
		if err := archiveAsyncImageArtifacts(ctx, task, artifacts); err != nil {
			logger.LogWarn(ctx, common.LogText("async image task artifact persistence failed task_id=%s: %v", task.TaskID, err))
			failAsyncImageTask(task, model.TaskStatusInProgress)
			return
		}
		task.PrivateData.ImageArtifacts = artifacts
	}
	task.ChannelId = result.ChannelID
	if result.Model != "" {
		task.Properties.UpstreamModelName = result.Model
	}
	if result.Quota > 0 {
		task.Quota = result.Quota
	}
	task.Data = nil
	task.Status = model.TaskStatusSuccess
	task.Progress = "100%"
	task.FinishTime = time.Now().Unix()
	task.UpdatedAt = task.FinishTime
	if updated, err := task.UpdateWithStatus(model.TaskStatusInProgress); err != nil || !updated {
		logger.LogWarn(ctx, common.LogText("async image task completion update failed task_id=%s", task.TaskID))
		return
	}
	service.ScheduleTaskArtifactArchive(task)
}

func persistAsyncImageBillingReservation(task *model.Task, info *relaycommon.RelayInfo) error {
	if task == nil || info == nil {
		return errors.New("async image billing reservation is missing task context")
	}
	task.Quota = info.FinalPreConsumedQuota
	task.ChannelId = info.ChannelId
	task.Group = info.UsingGroup
	task.Properties.OriginModelName = info.OriginModelName
	task.Properties.UpstreamModelName = info.UpstreamModelName
	task.PrivateData.BillingSource = info.BillingSource
	task.PrivateData.SubscriptionId = info.SubscriptionId
	task.PrivateData.TokenId = info.TokenId
	task.PrivateData.NodeName = common.NodeName
	task.PrivateData.BillingContext = &model.TaskBillingContext{
		ModelPrice:      info.PriceData.ModelPrice,
		GroupRatio:      info.PriceData.GroupRatioInfo.GroupRatio,
		ModelRatio:      info.PriceData.ModelRatio,
		OtherRatios:     info.PriceData.OtherRatios(),
		OriginModelName: info.OriginModelName,
		PerCallBilling:  common.StringsContains(constant.TaskPricePatches, info.OriginModelName) || info.PriceData.UsePrice,
		TieredSnapshot:  info.TieredBillingSnapshot,
	}
	updated, err := model.PersistAsyncImageTaskBillingReservation(task)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("async image task is no longer in progress")
	}
	return nil
}

func archiveAsyncImageArtifacts(ctx context.Context, task *model.Task, artifacts []model.TaskImageArtifact) error {
	store := service.GetTaskArtifactStore()
	if !store.Enabled() {
		for _, artifact := range artifacts {
			if strings.HasPrefix(artifact.SourceURL, "data:image/") {
				return errAsyncImageBase64StorageRequired
			}
		}
		return nil
	}
	if service.ArchiveTaskArtifacts == nil {
		return errAsyncImageBase64StorageRequired
	}
	task.PrivateData.ImageArtifacts = artifacts
	task.PrivateData.ArtifactArchive = &model.TaskArtifactArchive{Status: model.TaskArchiveRunning, UpdatedAt: time.Now().Unix()}
	if err := service.ArchiveTaskArtifacts(ctx, task, store); err != nil {
		for i := range task.PrivateData.ImageArtifacts {
			task.PrivateData.ImageArtifacts[i].SourceURL = ""
		}
		return err
	}
	archive := *task.PrivateData.ArtifactArchive
	archive.Status = model.TaskArchiveCompleted
	archive.UpdatedAt = time.Now().Unix()
	task.PrivateData.ArtifactArchive = &archive
	for i := range task.PrivateData.ImageArtifacts {
		task.PrivateData.ImageArtifacts[i].SourceURL = ""
	}
	return nil
}

func failAsyncImageTask(task *model.Task, fromStatus model.TaskStatus) {
	task.Data = nil
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = time.Now().Unix()
	task.UpdatedAt = task.FinishTime
	task.FailReason = "Image generation failed"
	updated, err := task.UpdateWithStatus(fromStatus)
	if err != nil || !updated {
		logger.LogWarn(context.Background(), common.LogText("async image task failure update failed task_id=%s", task.TaskID))
		return
	}
	if task.Quota != 0 {
		service.RefundTaskQuota(context.Background(), task, task.FailReason)
	}
}

var (
	asyncImageRelayOnce sync.Once
	asyncImageRelay     http.Handler
)

func asyncImageRelayHandler() http.Handler {
	asyncImageRelayOnce.Do(func() {
		engine := gin.New()
		engine.POST("/v1/images/generations",
			middleware.RouteTag("relay"),
			middleware.TokenAuth(),
			middleware.RouteTaskPluginEndpoint(),
			middleware.PrepareTaskPluginEndpoint(),
			middleware.Distribute(),
			func(c *gin.Context) {
				task, _ := c.Request.Context().Value(asyncImageTaskContextKey{}).(*model.Task)
				result, _ := c.Request.Context().Value(asyncImageResultContextKey{}).(*asyncImageRunResult)
				if task == nil || result == nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Image task execution failed"}})
					return
				}
				c.Set(common.RequestIdKey, task.TaskID)
				c.Header(common.RequestIdKey, task.TaskID)
				controllerTask := task
				c.Set(imageLogTaskContextKey, controllerTask)
				c.Set(service.ImageBillingReservationHookContextKey, service.ImageBillingReservationHook(func(info *relaycommon.RelayInfo) error {
					return persistAsyncImageBillingReservation(controllerTask, info)
				}))
				RelayTaskPluginEndpoint(c, func(c *gin.Context) {
					c.Set(imageLogTaskContextKey, controllerTask)
					Relay(c, types.RelayFormatOpenAIImage)
				})
				result.ChannelID = common.GetContextKeyInt(c, constant.ContextKeyChannelId)
				result.Quota = c.GetInt(service.SettledQuotaContextKey)
				if value, ok := c.Get(imageLogRelayInfoContextKey); ok {
					if info, ok := value.(*relaycommon.RelayInfo); ok && info != nil {
						result.Model = info.UpstreamModelName
					}
				}
				if value, ok := c.Get(imageLogTaskContextKey); ok {
					if innerTask, ok := value.(*model.Task); ok && innerTask != nil && innerTask.ID != task.ID {
						result.ChannelID = innerTask.ChannelId
						result.Model = innerTask.Properties.UpstreamModelName
						result.Quota = 0
					}
				}
			},
		)
	})
	return asyncImageRelay
}

type asyncImageTaskSystemHandler struct{}

func (asyncImageTaskSystemHandler) Type() string { return model.SystemTaskTypeAsyncImage }
func (asyncImageTaskSystemHandler) Enabled() bool {
	return constant.UpdateTask && model.HasPendingAsyncImageTasks()
}
func (asyncImageTaskSystemHandler) Interval() time.Duration { return 15 * time.Second }
func (asyncImageTaskSystemHandler) NewPayload() any         { return nil }
func (asyncImageTaskSystemHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	ProcessPendingAsyncImageTasks(ctx)
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, nil, nil)
}
