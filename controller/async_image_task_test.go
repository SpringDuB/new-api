package controller

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAsyncImageRequest(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		objectStore  bool
		wantFormat   string
		wantError    bool
		wantPreserve string
	}{
		{
			name:         "defaults to URL delivery without object storage",
			body:         `{"model":"image-model","prompt":"a cat","extra_option":"keep"}`,
			wantFormat:   "url",
			wantPreserve: "keep",
		},
		{
			name:      "rejects base64 delivery without object storage",
			body:      `{"model":"image-model","prompt":"a cat","response_format":"b64_json"}`,
			wantError: true,
		},
		{
			name:         "preserves explicit URL format",
			body:         `{"model":"image-model","prompt":"a cat","response_format":"url"}`,
			wantFormat:   "url",
			wantPreserve: "",
		},
		{
			name:        "allows explicit base64 format with object storage",
			body:        `{"model":"image-model","prompt":"a cat","response_format":"b64_json"}`,
			objectStore: true,
			wantFormat:  "b64_json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeAsyncImageRequest([]byte(test.body), test.objectStore)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			var result map[string]any
			require.NoError(t, common.Unmarshal(got, &result))
			assert.Equal(t, test.wantFormat, result["response_format"])
			if test.wantPreserve != "" {
				assert.Equal(t, test.wantPreserve, result["extra_option"])
			}
		})
	}
}

type inlineImageArtifactStore struct {
	content []byte
}

func (*inlineImageArtifactStore) Enabled() bool { return true }
func (*inlineImageArtifactStore) Resolve(*model.Task, string) (*service.StoredArtifactRef, error) {
	return nil, nil
}
func (s *inlineImageArtifactStore) Persist(_ context.Context, _ *model.Task, _ hosttypes.TaskArtifact, content io.Reader) (*service.StoredArtifactRef, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}
	s.content = data
	return &service.StoredArtifactRef{Backend: "test", ArtifactKey: "image-1", Size: int64(len(data))}, nil
}
func (*inlineImageArtifactStore) Serve(_ *gin.Context, _ *model.Task, _ *service.StoredArtifactRef) error {
	return nil
}

func TestArchiveTaskArtifactPersistsInlineImageDataURL(t *testing.T) {
	store := &inlineImageArtifactStore{}
	task := &model.Task{
		ID:     1,
		TaskID: "task_inline_image",
		UserId: 7,
		PrivateData: model.TaskPrivateData{ImageArtifacts: []model.TaskImageArtifact{{
			Key: "image-1", MimeType: "image/png", SourceURL: "data:image/png;base64,aW1hZ2UtYnl0ZXM=",
		}}},
	}
	ref, err := archiveTaskArtifact(t.Context(), task, channel.TaskArtifact{Key: "image-1", Type: "image", MimeType: "image/png"}, store)
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, []byte("image-bytes"), store.content)
	assert.EqualValues(t, len(store.content), ref.Size)
}

func TestCreateAsyncImageTaskPersistsValidatedQueuedRequest(t *testing.T) {
	previousUpdateTask := constant.UpdateTask
	constant.UpdateTask = true
	t.Cleanup(func() { constant.UpdateTask = previousUpdateTask })
	t.Setenv("TASK_ARTIFACT_STORE_MODE", "upstream")
	task := setupGenericTaskTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"image-model","prompt":"a cat","extra_option":"keep"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", task.UserId)
	common.SetContextKey(c, constant.ContextKeyTokenId, 11)
	common.SetContextKey(c, constant.ContextKeyTokenGroup, "default")

	CreateAsyncImageTask(c)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	var response asyncImageTaskResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "queued", response.Status)
	stored, exists, err := model.GetByTaskId(task.UserId, response.ID)
	require.NoError(t, err)
	require.True(t, exists)
	pending, err := model.GetPendingAsyncImageTasks(10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, response.ID, pending[0].TaskID)
	assert.Empty(t, model.GetAllUnFinishSyncTasks(10))
	assert.True(t, model.HasUnfinishedSyncTasks())
	assert.Equal(t, constant.TaskPlatformAsyncImage, stored.Platform)
	assert.Equal(t, asyncImageAction, stored.Action)
	assert.Equal(t, model.TaskStatusNotStart, stored.Status)
	assert.Equal(t, 11, stored.PrivateData.TokenId)
	assert.Equal(t, "asynchronous", stored.PrivateData.ImageRequestMode)
	var request map[string]any
	require.NoError(t, common.Unmarshal(stored.Data, &request))
	assert.Equal(t, "url", request["response_format"])
	assert.Equal(t, "keep", request["extra_option"])
}

func TestCreateAsyncImageTaskRejectsBase64WithoutObjectStorage(t *testing.T) {
	previousUpdateTask := constant.UpdateTask
	constant.UpdateTask = true
	t.Cleanup(func() { constant.UpdateTask = previousUpdateTask })
	t.Setenv("TASK_ARTIFACT_STORE_MODE", "upstream")
	task := setupGenericTaskTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"image-model","prompt":"a cat","response_format":"b64_json"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", task.UserId)
	common.SetContextKey(c, constant.ContextKeyTokenId, 11)

	CreateAsyncImageTask(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "object storage")
}

func TestAsyncImageTaskStatusMapsPublicStates(t *testing.T) {
	tests := []struct {
		status model.TaskStatus
		want   string
	}{
		{model.TaskStatusNotStart, "queued"},
		{model.TaskStatusInProgress, "in_progress"},
		{model.TaskStatusSuccess, "completed"},
		{model.TaskStatusFailure, "failed"},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, asyncImageStatus(test.status))
	}
}

func TestAsyncImageTaskDeliveryArchiveUsesOuterTask(t *testing.T) {
	configureImageDeliveryTestOSS(t)
	existing := setupGenericTaskTest(t)
	existing.Platform = constant.TaskPlatformAsyncImage
	existing.Action = asyncImageAction
	existing.Status = model.TaskStatusInProgress
	existing.Progress = "30%"
	existing.PrivateData.ImageRequestMode = "asynchronous"
	require.NoError(t, model.DB.Save(existing).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"image-model","prompt":"a cat"}`))
	c.Set("id", existing.UserId)
	c.Set(imageLogTaskContextKey, existing)
	c.Set(service.SettledQuotaContextKey, 41)
	finish := beginImageLogCapture(c)
	body := []byte(`{"created":1,"data":[{"b64_json":"` + testArtifactPNG + `"}]}`)
	require.NoError(t, writeStoredImages(c, existing, service.GetTaskArtifactStore(), body, nil, imageDeliveryFormat{inlineBase64: true}))
	finish()

	stored, exists, err := model.GetByTaskId(existing.UserId, existing.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, model.TaskStatus(model.TaskStatusInProgress), stored.Status)
	require.NotNil(t, stored.PrivateData.ArtifactArchive)
	assert.Equal(t, model.TaskArchiveCompleted, stored.PrivateData.ArtifactArchive.Status)
	assert.Equal(t, 41, stored.Quota)
	require.Len(t, stored.PrivateData.ImageArtifacts, 1)
	assert.True(t, stored.PrivateData.ImageArtifacts[0].Stored)
	assert.Empty(t, stored.PrivateData.ImageArtifacts[0].SourceURL)
	privateData, err := common.Marshal(stored.PrivateData)
	require.NoError(t, err)
	assert.NotContains(t, string(privateData), testArtifactPNG)
}

func TestGetAsyncImageTaskScopesOwnerAndReturnsQueuedState(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Platform = constant.TaskPlatformAsyncImage
	task.Action = asyncImageAction
	task.Status = model.TaskStatusInProgress
	task.Progress = "1%"
	task.PrivateData.ImageRequestMode = "asynchronous"
	require.NoError(t, model.DB.Save(task).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/images/tasks/"+task.TaskID, nil)
	c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	c.Set("id", task.UserId)
	GetAsyncImageTask(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response asyncImageTaskResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, task.TaskID, response.ID)
	assert.Equal(t, "in_progress", response.Status)

	foreignRecorder := httptest.NewRecorder()
	foreign, _ := gin.CreateTestContext(foreignRecorder)
	foreign.Request = httptest.NewRequest(http.MethodGet, "/v1/images/tasks/"+task.TaskID, nil)
	foreign.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	foreign.Set("id", task.UserId+1)
	GetAsyncImageTask(foreign)
	assert.Equal(t, http.StatusNotFound, foreignRecorder.Code)
}

func TestAsyncImageTaskClaimUsesCompareAndSwap(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Platform = constant.TaskPlatformAsyncImage
	task.Action = asyncImageAction
	task.Status = model.TaskStatusNotStart
	task.Progress = "0%"
	require.NoError(t, model.DB.Save(task).Error)
	stale := *task

	task.Status = model.TaskStatusInProgress
	task.Progress = "1%"
	claimed, err := task.UpdateWithStatus(model.TaskStatusNotStart)
	require.NoError(t, err)
	require.True(t, claimed)

	stale.Status = model.TaskStatusInProgress
	stale.Progress = "1%"
	claimed, err = stale.UpdateWithStatus(model.TaskStatusNotStart)
	require.NoError(t, err)
	assert.False(t, claimed)
	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusInProgress), stored.Status)
}

func TestAsyncImageTaskArchivesRemoteImageURLsWhenObjectStorageIsEnabled(t *testing.T) {
	configureImageDeliveryTestOSS(t)
	previousArchive := service.ArchiveTaskArtifacts
	defer func() { service.ArchiveTaskArtifacts = previousArchive }()
	archiveCalls := 0
	service.ArchiveTaskArtifacts = func(_ context.Context, task *model.Task, _ service.TaskArtifactStore) error {
		archiveCalls++
		for i := range task.PrivateData.ImageArtifacts {
			task.PrivateData.ImageArtifacts[i].Stored = true
		}
		task.PrivateData.ArtifactArchive.Artifacts = []model.TaskArchivedArtifact{{Key: "image-1", Type: "image", Backend: "oss"}}
		return nil
	}
	task := &model.Task{
		TaskID:      "task_remote_image",
		PrivateData: model.TaskPrivateData{ImageRequestMode: "asynchronous"},
	}
	artifacts := []model.TaskImageArtifact{{Key: "image-1", MimeType: "image/png", SourceURL: "https://cdn.example/image.png?signature=private"}}
	require.NoError(t, archiveAsyncImageArtifacts(t.Context(), task, artifacts))
	assert.Equal(t, 1, archiveCalls)
	assert.Equal(t, model.TaskArchiveCompleted, task.PrivateData.ArtifactArchive.Status)
	require.Len(t, task.PrivateData.ImageArtifacts, 1)
	assert.True(t, task.PrivateData.ImageArtifacts[0].Stored)
	assert.Empty(t, task.PrivateData.ImageArtifacts[0].SourceURL)
}

func TestCreateAsyncImageTaskRejectsDisabledTaskProcessing(t *testing.T) {
	previous := constant.UpdateTask
	constant.UpdateTask = false
	t.Cleanup(func() { constant.UpdateTask = previous })
	task := setupGenericTaskTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"image-model","prompt":"a cat"}`))
	c.Set("id", task.UserId)
	common.SetContextKey(c, constant.ContextKeyTokenId, 11)

	CreateAsyncImageTask(c)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestCreateAsyncImageTaskRejectsStreamingRequests(t *testing.T) {
	previousUpdateTask := constant.UpdateTask
	constant.UpdateTask = true
	t.Cleanup(func() { constant.UpdateTask = previousUpdateTask })
	task := setupGenericTaskTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"image-model","prompt":"a cat","stream":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", task.UserId)
	common.SetContextKey(c, constant.ContextKeyTokenId, 11)

	CreateAsyncImageTask(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Streaming is not supported")
}

func TestAsyncImageTaskWorkerFailsAfterTokenRevocation(t *testing.T) {
	task := setupGenericTaskTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}))
	task.Platform = constant.TaskPlatformAsyncImage
	task.Action = asyncImageAction
	task.Status = model.TaskStatusNotStart
	task.Progress = "0%"
	task.PrivateData.TokenId = 99999
	task.SetData(map[string]any{"model": "image-model", "prompt": "a cat"})
	require.NoError(t, model.DB.Save(task).Error)

	ProcessPendingAsyncImageTasks(t.Context())

	stored, exists, err := model.GetByTaskId(task.UserId, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), stored.Status)
	assert.Equal(t, "Image generation failed", stored.FailReason)
	assert.Empty(t, stored.Data)
}

func TestAsyncImageTaskWorkerPersistsSynchronousImageResult(t *testing.T) {
	task := setupGenericTaskTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}))
	token := &model.Token{UserId: task.UserId, Key: "async-worker-token", Status: common.TokenStatusEnabled, ExpiredTime: -1}
	require.NoError(t, model.DB.Create(token).Error)
	task.Platform = constant.TaskPlatformAsyncImage
	task.Action = asyncImageAction
	task.Status = model.TaskStatusInProgress
	task.Progress = "1%"
	task.PrivateData.TokenId = token.Id
	task.PrivateData.ImageRequestMode = "asynchronous"
	task.SetData(map[string]any{"model": "image-model", "prompt": "a cat", "response_format": "url"})
	require.NoError(t, model.DB.Save(task).Error)

	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/images/generations", r.URL.Path)
		assert.Equal(t, "Bearer async-worker-token", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"created":10,"data":[{"url":"https://cdn.example/async-image.png"}]}`)
	})
	processAsyncImageTaskWithHandler(t.Context(), task, upstream)

	stored, exists, err := model.GetByTaskId(task.UserId, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, model.TaskStatus(model.TaskStatusSuccess), stored.Status)
	assert.Equal(t, "100%", stored.Progress)
	require.Len(t, stored.PrivateData.ImageArtifacts, 1)
	assert.Equal(t, "https://cdn.example/async-image.png", stored.PrivateData.ImageArtifacts[0].SourceURL)
	assert.Empty(t, stored.Data)
}

func TestPersistAsyncImageBillingReservation(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Platform = constant.TaskPlatformAsyncImage
	task.Action = asyncImageAction
	task.Status = model.TaskStatusInProgress
	task.Progress = "1%"
	task.PrivateData.ImageRequestMode = "asynchronous"
	require.NoError(t, model.DB.Save(task).Error)

	info := &relaycommon.RelayInfo{
		TokenId: 31, UserId: task.UserId, UsingGroup: "premium",
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelId: 37, UpstreamModelName: "provider-image-model"},
		BillingSource: service.BillingSourceSubscription, SubscriptionId: 41,
		FinalPreConsumedQuota: 2900, OriginModelName: "image-model",
		PriceData: hosttypes.PriceData{ModelPrice: 0.02, ModelRatio: 2, UsePrice: true, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1.5}},
	}
	require.NoError(t, persistAsyncImageBillingReservation(task, info))

	stored, exists, err := model.GetByTaskId(task.UserId, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, 2900, stored.Quota)
	assert.Equal(t, 37, stored.ChannelId)
	assert.Equal(t, "premium", stored.Group)
	assert.Equal(t, "provider-image-model", stored.Properties.UpstreamModelName)
	assert.Equal(t, 31, stored.PrivateData.TokenId)
	assert.Equal(t, service.BillingSourceSubscription, stored.PrivateData.BillingSource)
	assert.Equal(t, 41, stored.PrivateData.SubscriptionId)
	require.NotNil(t, stored.PrivateData.BillingContext)
	assert.True(t, stored.PrivateData.BillingContext.PerCallBilling)
	assert.Equal(t, 0.02, stored.PrivateData.BillingContext.ModelPrice)
	assert.Equal(t, 2.0, stored.PrivateData.BillingContext.ModelRatio)
	assert.Equal(t, 1.5, stored.PrivateData.BillingContext.GroupRatio)
}

func TestGetAsyncImageTaskFollowsLinkedUpstreamTask(t *testing.T) {
	setupGenericTaskTest(t)
	previousAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://gateway.example"
	t.Cleanup(func() { system_setting.ServerAddress = previousAddress })
	wrapper := &model.Task{
		TaskID: "task_async_image_wrapper_query", UserId: 7, Platform: constant.TaskPlatformAsyncImage,
		Action: asyncImageAction, Status: model.TaskStatusInProgress, Progress: "1%", CreatedAt: 100,
	}
	source := &model.Task{
		TaskID: "task_lecapi_image_query", UserId: 7, Platform: "lecapi",
		Action: asyncImageAction, Status: model.TaskStatusSuccess, Progress: "100%",
		PrivateData: model.TaskPrivateData{ImageArtifacts: []model.TaskImageArtifact{{Key: "image-1", MimeType: "image/png", Stored: true}}},
	}
	require.NoError(t, model.DB.Create(wrapper).Error)
	require.NoError(t, model.DB.Create(source).Error)
	linked, err := model.LinkAsyncImageSourceTask(wrapper.UserId, wrapper.TaskID, source.TaskID)
	require.NoError(t, err)
	require.True(t, linked)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/images/tasks/"+wrapper.TaskID, nil)
	c.Params = gin.Params{{Key: "task_id", Value: wrapper.TaskID}}
	c.Set("id", wrapper.UserId)
	GetAsyncImageTask(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response asyncImageTaskResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "completed", response.Status)
	require.Len(t, response.Data, 1)
	assert.Contains(t, response.Data[0].URL, "/v1/tasks/"+source.TaskID+"/artifacts/image-1/content")
}

func TestAsyncImageTaskBillingPersistenceDatabase(t *testing.T) {
	db, dialect := openTaskDialectDatabase(t, &model.Task{})
	previousDB := model.DB
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	model.DB = db
	common.SetDatabaseTypes(dialect, previousLog)
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetDatabaseTypes(previousMain, previousLog)
	})
	wrapper := &model.Task{
		TaskID: "task_async_image_matrix", UserId: 17, Platform: constant.TaskPlatformAsyncImage,
		Action: asyncImageAction, Status: model.TaskStatusInProgress, Quota: 1234,
		PrivateData: model.TaskPrivateData{BillingSource: service.BillingSourceWallet, TokenId: 19},
	}
	source := &model.Task{TaskID: "task_async_image_source_matrix", UserId: wrapper.UserId, Platform: "lecapi", Status: model.TaskStatusSubmitted}
	require.NoError(t, db.Create(wrapper).Error)
	require.NoError(t, db.Create(source).Error)
	updated, err := model.PersistAsyncImageTaskBillingReservation(wrapper)
	require.NoError(t, err)
	require.True(t, updated)
	linked, err := model.LinkAsyncImageSourceTask(wrapper.UserId, wrapper.TaskID, source.TaskID)
	require.NoError(t, err)
	require.True(t, linked)

	stored, exists, err := model.GetByTaskId(wrapper.UserId, wrapper.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, 1234, stored.Quota)
	assert.Equal(t, service.BillingSourceWallet, stored.PrivateData.BillingSource)
	assert.Equal(t, source.TaskID, stored.PrivateData.AsyncImageSourceTaskID)
}

func TestAsyncImageTaskFailureRefundsPersistedReservation(t *testing.T) {
	task := setupGenericTaskTest(t)
	previousLogDB := model.LOG_DB
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	model.LOG_DB = model.DB
	common.SetDatabaseTypes(previousMain, previousMain)
	t.Cleanup(func() {
		model.LOG_DB = previousLogDB
		common.SetDatabaseTypes(previousMain, previousLog)
	})
	require.NoError(t, model.DB.AutoMigrate(&model.Log{}))
	task.Platform = constant.TaskPlatformAsyncImage
	task.Action = asyncImageAction
	task.Status = model.TaskStatusInProgress
	task.Progress = "1%"
	task.Quota = 500
	task.PrivateData.BillingSource = service.BillingSourceWallet
	require.NoError(t, model.DB.Save(task).Error)

	failAsyncImageTask(task, model.TaskStatusInProgress)

	stored, exists, err := model.GetByTaskId(task.UserId, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), stored.Status)
	assert.Zero(t, stored.Quota)
	var user model.User
	require.NoError(t, model.DB.First(&user, task.UserId).Error)
	assert.Equal(t, 500, user.Quota)
}
