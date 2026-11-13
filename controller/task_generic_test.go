package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	taskjsplugin "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskArtifactOSSArchivePreservesMediaAndServesSignedLinks(t *testing.T) {
	task := setupGenericTaskTest(t)
	allowPrivateTaskMediaTest(t)
	objects := map[string][]byte{}
	puts := 0
	var imageResponse *httptest.ResponseRecorder
	ossServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			if value, ok := objects[r.URL.Path]; ok {
				w.Header().Set("Content-Length", fmt.Sprint(len(value)))
				w.Header().Set("Content-Type", "video/mp4")
			} else {
				w.WriteHeader(404)
			}
		case http.MethodPut:
			if imageResponse != nil {
				assert.True(t, imageResponse.Flushed, "the client response must be flushed before OSS upload")
			}
			value, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Contains(t, r.Header.Get("Authorization"), "OSS4-HMAC-SHA256")
			objects[r.URL.Path] = value
			puts++
			w.Header().Set("ETag", `"test"`)
		case http.MethodGet:
			assert.Empty(t, r.Header.Get("Authorization"))
			assert.Empty(t, r.Header.Get("Cookie"))
			value := objects[r.URL.Path]
			if r.Header.Get("Range") == "bytes=0-3" && len(value) >= 4 {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-3/%d", len(value)))
				w.WriteHeader(http.StatusPartialContent)
				value = value[:4]
			}
			_, _ = w.Write(value)
		default:
			t.Errorf("unexpected OSS method %s", r.Method)
		}
	}))
	defer ossServer.Close()
	oldTransport := http.DefaultTransport
	http.DefaultTransport = ossServer.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	config := system_setting.TaskArtifactStoreConfig{Mode: "oss", OSSEndpoint: ossServer.URL, OSSBucket: "results", OSSRegion: "cn-hangzhou", OSSAccessKey: "oss-access", OSSSecretKey: "oss-secret", OSSPrefix: "tasks/v1", OSSPresignTTLSeconds: 900, OSSMaxBytes: 1024}
	encoded, err := common.Marshal(config)
	require.NoError(t, err)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	previous := common.OptionMap[system_setting.TaskArtifactStoreOptionKey]
	common.OptionMap[system_setting.TaskArtifactStoreOptionKey] = string(encoded)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[system_setting.TaskArtifactStoreOptionKey] = previous
		common.OptionMapRWMutex.Unlock()
	})
	mediaCalls := 0
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaCalls++
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Cookie"))
		assert.Empty(t, r.Header.Get("Range"))
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = io.WriteString(w, "archived-media")
	}))
	defer media.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/embedded" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"done":true,"response":{"video":"ZW1iZWRkZWQtdmlkZW8="}}`)
			return
		}
		http.Redirect(w, r, media.URL+r.URL.Path, http.StatusFound)
	}))
	defer origin.Close()
	source := `export const meta={apiVersion:1,key:"archive-regression",name:"Archive",version:"1.0.0",author:{name:"Test"},models:["m"],fetchMode:"per_task"};
export function buildSubmitRequest(){return {}} export function parseSubmitResponse(){return {taskId:"vendor"}} export function buildQueryRequest(){return {}} export function parseTaskResult(){return {status:"SUCCESS"}}
export function listArtifacts(task){return task.data.object==="image"?[{key:"image-1",type:"image",mimeType:"image/png"}]:[{key:"video",type:"video",mimeType:"video/mp4"},{key:"image-1",type:"image",mimeType:"image/png"}]}
export function buildContentRequest(ctx){if(ctx.data.object==="operation")return {url:ctx.baseUrl+"/embedded",method:"GET",credentialless:true,contentPath:["response","video"],mimeType:"video/mp4"};return {url:ctx.data.object==="image"?"data:image/png;base64,"+ctx.data.data[0].b64_json:ctx.baseUrl+"/"+ctx.artifactKey,method:"GET",credentialless:true}}`
	_, err = jsplugin.DefaultRegistry.RegisterFactory(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister("archive-regression") })
	channel, err := model.GetChannelById(task.ChannelId, true)
	require.NoError(t, err)
	channel.BaseURL = &origin.URL
	require.NoError(t, model.DB.Save(channel).Error)
	task.Platform = "archive-regression"
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{Key: "archive-regression"}}
	task.SetData(map[string]any{"video_url": "https://old.example/video.mp4"})
	require.NoError(t, model.DB.Save(task).Error)
	for _, artifact := range []relaychannel.TaskArtifact{{Key: "video", Type: "video", MimeType: "video/mp4"}, {Key: "image-1", Type: "image", MimeType: "image/png"}} {
		ref, err := archiveTaskArtifact(t.Context(), task, artifact)
		require.NoError(t, err)
		require.NotNil(t, ref)
		assert.Equal(t, int64(14), ref.Size)
		_, err = archiveTaskArtifact(t.Context(), task, artifact)
		require.NoError(t, err)
	}
	assert.Equal(t, 2, mediaCalls)
	assert.Equal(t, 2, puts)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(method, "/v1/tasks/"+task.TaskID+"/artifacts/video/content", nil)
		c.Request.Header.Set("Range", "bytes=0-3")
		c.Set("id", task.UserId)
		c.Params = gin.Params{{Key: "key", Value: task.TaskID}, {Key: "artifact_key", Value: "video"}}
		TaskArtifactContent(c)
		assert.Equal(t, 307, recorder.Code)
		assert.Contains(t, recorder.Header().Get("Location"), "x-oss-signature=")
		request, err := http.NewRequestWithContext(t.Context(), method, recorder.Header().Get("Location"), nil)
		require.NoError(t, err)
		request.Header.Set("Range", "bytes=0-3")
		response, err := ossServer.Client().Do(request)
		require.NoError(t, err)
		content, err := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, err)
		if method == http.MethodGet {
			assert.Equal(t, http.StatusPartialContent, response.StatusCode)
			assert.Equal(t, "arch", string(content))
		} else {
			assert.Equal(t, http.StatusOK, response.StatusCode)
			assert.Empty(t, content)
		}
	}
	assert.Equal(t, 2, mediaCalls, "cached reads never need the upstream again")
	view, err := service.BuildTaskPluginView(task)
	require.NoError(t, err)
	assert.Contains(t, view.Data.(map[string]any)["video_url"], "x-oss-signature=")
	assert.Contains(t, string(task.Data), "old.example", "OSS presentation does not overwrite the upstream snapshot")
	// Immediate image submissions can deliberately omit the provider result
	// from the task row, but the bytes still need archiving before responding.
	immediate := *task
	immediate.ID = 0
	immediate.TaskID += "_sync"
	immediate.PrivateData.ResultDiscarded = true
	immediate.SetData(map[string]any{"object": "image", "data": []any{map[string]any{"b64_json": "aW1hZ2UtYnl0ZXM="}}})
	require.NoError(t, model.DB.Omit("data").Create(&immediate).Error)
	service.ArchiveTaskArtifacts(t.Context(), &immediate)
	assert.Equal(t, 3, puts)
	view, err = service.BuildTaskPluginView(&immediate)
	require.NoError(t, err)
	image := view.Data.(map[string]any)["data"].([]any)[0].(map[string]any)
	assert.Contains(t, image["url"], "x-oss-signature=")
	assert.Equal(t, "aW1hZ2UtYnl0ZXM=", image["b64_json"])
	var discarded model.Task
	require.NoError(t, model.DB.First(&discarded, immediate.ID).Error)
	assert.Empty(t, discarded.Data)
	assert.False(t, discarded.ResultRetrievable())
	inlineVideo := *task
	inlineVideo.ID = 0
	inlineVideo.TaskID += "_embedded"
	inlineVideo.SetData(map[string]any{"object": "operation"})
	require.NoError(t, model.DB.Create(&inlineVideo).Error)
	ref, err := archiveTaskArtifact(t.Context(), &inlineVideo, relaychannel.TaskArtifact{Key: "video", Type: "video", MimeType: "video/mp4"})
	require.NoError(t, err)
	assert.Equal(t, int64(len("embedded-video")), ref.Size)
	assert.Equal(t, 4, puts)
	old := *task
	old.ID = 0
	old.TaskID += "_old"
	require.NoError(t, model.DB.Create(&old).Error)
	view, err = service.BuildTaskPluginView(&old)
	require.NoError(t, err)
	assert.Contains(t, view.Data.(map[string]any)["video_url"], "x-oss-signature=", "historical successful tasks archive on retrieval with their existing ID")
	assert.Equal(t, 6, puts)

	// Ordinary Images responses use the same private OSS store, without task
	// plugin credentials or task billing. Both URL and Base64 output archive.
	imageResponse = httptest.NewRecorder()
	imageContext, _ := gin.CreateTestContext(imageResponse)
	imageContext.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"ordinary-image","prompt":"draw","seed":0}`))
	imageContext.Request.Header.Set("Content-Type", "application/json")
	imageContext.Request.Header.Set("Authorization", "Bearer downstream-secret")
	imageContext.Set("id", task.UserId)
	common.SetContextKey(imageContext, constant.ContextKeyChannelId, task.ChannelId)
	finishImageLog := beginImageLogCapture(imageContext)
	imageContext.Data(http.StatusOK, "application/json", []byte(`{"data":[{"url":"`+origin.URL+`/ordinary-image"},{"b64_json":"aW1hZ2UtYnl0ZXM="}]}`))
	finishImageLog()
	assert.Equal(t, 8, puts)
	var imageLog model.Task
	require.NoError(t, model.DB.Where("platform = ?", "image").First(&imageLog).Error)
	require.Len(t, imageLog.PrivateData.ImageArtifacts, 2)
	assert.True(t, imageLog.PrivateData.ImageArtifacts[0].Stored)
	assert.True(t, imageLog.PrivateData.ImageArtifacts[1].Stored)
	privateJSON, err := common.Marshal(imageLog.PrivateData)
	require.NoError(t, err)
	assert.NotContains(t, string(privateJSON), "aW1hZ2UtYnl0ZXM=")
	assert.NotContains(t, string(privateJSON), "downstream-secret")
	assert.NotContains(t, string(privateJSON), "x-oss-signature")
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		mediaRecorder := httptest.NewRecorder()
		mediaContext, _ := gin.CreateTestContext(mediaRecorder)
		mediaContext.Request = httptest.NewRequest(method, "/content", nil)
		mediaContext.Request.Header.Set("Range", "bytes=0-3")
		mediaContext.Set("id", task.UserId)
		mediaContext.Params = gin.Params{{Key: "key", Value: imageLog.TaskID}, {Key: "artifact_key", Value: "image-1"}}
		TaskArtifactContent(mediaContext)
		require.Equal(t, http.StatusTemporaryRedirect, mediaRecorder.Code)
		request, err := http.NewRequestWithContext(t.Context(), method, mediaRecorder.Header().Get("Location"), nil)
		require.NoError(t, err)
		request.Header.Set("Range", "bytes=0-3")
		response, err := ossServer.Client().Do(request)
		require.NoError(t, err)
		mediaBytes, err := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, err)
		if method == http.MethodGet {
			assert.Equal(t, http.StatusPartialContent, response.StatusCode)
			assert.Equal(t, "arch", string(mediaBytes))
		} else {
			assert.Equal(t, http.StatusOK, response.StatusCode)
			assert.Empty(t, mediaBytes)
		}
	}
	assert.Equal(t, 8, puts, "preview never regenerates or uploads a duplicate")
	assert.Equal(t, 0, imageLog.Quota, "archival does not charge quota")
}

func TestTaskArtifactStoreSettingsPersistWithoutReturningCredentials(t *testing.T) {
	setupGenericTaskTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Option{}))
	t.Cleanup(func() { require.NoError(t, model.DB.Migrator().DropTable(&model.Option{})) })
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() { common.OptionMapRWMutex.Lock(); common.OptionMap = previous; common.OptionMapRWMutex.Unlock() })
	for _, credentials := range []string{`,"access_key":"access-canary","secret_key":"secret-canary"`, ``} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/option/task_artifact_store", strings.NewReader(`{"mode":"oss","endpoint":"https://oss-cn-hangzhou.aliyuncs.com","bucket":"task-results","region":"cn-hangzhou","prefix":"tasks/v1","presign_ttl":900,"max_mb":512`+credentials+`}`))
		UpdateTaskArtifactStoreSettings(c)
		var result struct {
			Success bool
			Data    struct {
				CredentialsConfigured bool `json:"credentials_configured"`
			}
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
		assert.True(t, result.Success)
		assert.True(t, result.Data.CredentialsConfigured)
		assert.NotContains(t, recorder.Body.String(), "access-canary")
		assert.NotContains(t, recorder.Body.String(), "secret-canary")
	}
	var stored model.Option
	require.NoError(t, model.DB.Where(&model.Option{Key: system_setting.TaskArtifactStoreOptionKey}).First(&stored).Error)
	assert.Contains(t, stored.Value, "secret-canary", "blank credential edits preserve the configured value")
	config := system_setting.LoadTaskArtifactStoreConfig()
	assert.Equal(t, "secret-canary", config.OSSSecretKey)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	GetOptions(c)
	assert.NotContains(t, recorder.Body.String(), system_setting.TaskArtifactStoreOptionKey)
	assert.NotContains(t, recorder.Body.String(), "secret-canary")
	assert.Error(t, model.UpdateOptionsBulk(map[string]string{system_setting.TaskArtifactStoreOptionKey: `{"mode":"oss","oss_endpoint":"https://attacker.example"}`}))
}

func TestTaskMediaFailuresKeepSafeInternalDiagnostics(t *testing.T) {
	task := setupGenericTaskTest(t)
	allowPrivateTaskMediaTest(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer upstream.Close()
	var logs bytes.Buffer
	previous := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logs
	t.Cleanup(func() { gin.DefaultErrorWriter = previous })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/"+task.TaskID+"/artifacts/video/content?access=private-access", nil)
	c.Set(middleware.TaskArtifactAccessContextKey, true)
	err := proxyTaskMedia(c, task, &relaychannel.TaskContentRequest{URL: upstream.URL + "/video?signature=private-signature", Method: http.MethodGet, Credentialless: true})
	require.Error(t, err)
	writeTaskMediaProxyError(c, err)
	assert.Contains(t, logs.String(), "stage=media_download")
	assert.Contains(t, logs.String(), "http=403")
	assert.Contains(t, logs.String(), "task_generic")
	assert.NotContains(t, logs.String(), "private-signature")
	assert.NotContains(t, logs.String(), "private-access")
	assert.NotContains(t, logs.String(), upstream.URL)

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/content?access=private-access", nil)
	c.Params = gin.Params{{Key: "key", Value: task.TaskID}, {Key: "artifact_key", Value: "video"}}
	middleware.TokenOrTaskArtifactAccessAuth("key", "artifact_key")(c)
	assert.Contains(t, logs.String(), "stage=access_signature")

	task.Platform = "missing-diagnostics-plugin"
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{Key: string(task.Platform)}}
	require.NoError(t, model.DB.Save(task).Error)
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/content?access=private-access", nil)
	c.Params = gin.Params{{Key: "key", Value: task.TaskID}, {Key: "artifact_key", Value: "video"}}
	c.Set("id", task.UserId)
	c.Set(middleware.TaskArtifactAccessContextKey, true)
	TaskArtifactContent(c)
	assert.Contains(t, logs.String(), "stage=projection")

	query := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer query.Close()
	source := `export const meta={apiVersion:1,key:"diagnostics-regression",name:"Diagnostics",version:"1.0.0",author:{name:"Test"},models:["m"],fetchMode:"per_task"};
export function buildSubmitRequest(){return {}} export function parseSubmitResponse(){return {taskId:"vendor"}} export function buildQueryRequest(){return {}} export function parseTaskResult(){return {status:"SUCCESS"}}
export function listArtifacts(){return [{key:"video",type:"video"}]}
export function buildContentRequest(ctx){utils.fetch({url:ctx.baseUrl+"/query?signature=private-signature",headers:{Authorization:"Bearer channel-canary"}});throw new Error(ctx.baseUrl+"/query?signature=private-signature")}`
	_, err = jsplugin.DefaultRegistry.RegisterFactory(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister("diagnostics-regression") })
	channel, err := model.GetChannelById(task.ChannelId, true)
	require.NoError(t, err)
	channel.BaseURL = &query.URL
	require.NoError(t, model.DB.Save(channel).Error)
	task.Platform = "diagnostics-regression"
	task.PrivateData.Execution.TaskPlugin.Key = string(task.Platform)
	require.NoError(t, model.DB.Save(task).Error)
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/content?access=private-access", nil)
	c.Params = gin.Params{{Key: "key", Value: task.TaskID}, {Key: "artifact_key", Value: "video"}}
	c.Set("id", task.UserId)
	c.Set(middleware.TaskArtifactAccessContextKey, true)
	TaskArtifactContent(c)
	assert.Contains(t, logs.String(), "stage=refresh_query http=502")
	assert.NotContains(t, logs.String(), "private-signature")
	assert.NotContains(t, logs.String(), "private-access")
	assert.NotContains(t, logs.String(), "channel-canary")
	assert.NotContains(t, logs.String(), query.URL)
}

func setupGenericTaskTest(t *testing.T) *model.Task {
	t.Helper()
	originalDB := model.DB
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	// The dialect harness lets the same fixture run against MySQL and
	// PostgreSQL when TEST_TASK_DB_DIALECT selects them; SQLite stays the default.
	database, _ := openTaskDialectDatabase(t, &model.Task{}, &model.Channel{}, &model.User{})
	model.DB = database
	t.Cleanup(func() {
		model.DB = originalDB
		common.RedisEnabled = previousRedisEnabled
	})

	require.NoError(t, database.Create(&model.User{
		Id: 7, Username: "artifact-owner", Status: common.UserStatusEnabled,
		Role: common.RoleCommonUser, Group: "default",
	}).Error)
	baseURL := "https://example.com"
	require.NoError(t, database.Create(&model.Channel{
		Id: 1, Name: "artifact", Key: "key", BaseURL: &baseURL, Status: common.ChannelStatusEnabled,
	}).Error)
	task := &model.Task{
		TaskID: "task_generic", Platform: "document", UserId: 7, ChannelId: 1,
		Status: model.TaskStatusSuccess, Progress: "100%", SubmitTime: 10, FinishTime: 20,
	}
	require.NoError(t, database.Create(task).Error)
	return task
}

func allowPrivateTaskMediaTest(t *testing.T) {
	t.Helper()
	originalFetchSetting := *system_setting.GetFetchSetting()
	system_setting.GetFetchSetting().EnableSSRFProtection = true
	system_setting.GetFetchSetting().AllowPrivateIp = true
	system_setting.GetFetchSetting().AllowedPorts = []string{"1-65535"}
	t.Cleanup(func() { *system_setting.GetFetchSetting() = originalFetchSetting })
	service.InitHttpClient()
}

func TestDashboardTaskClientRequestUsesOwnershipAndNeverFallsBackToResults(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.SetData(map[string]any{"prompt": "upstream-result-canary"})
	require.NoError(t, model.DB.Save(task).Error)
	for _, recorded := range []bool{false, true} {
		if recorded {
			task.PrivateData.ClientRequest = &model.TaskClientRequestSnapshot{Method: "POST", Path: "/v1/videos", ContentType: "application/json", BodyKind: "json", Body: []byte(`{"prompt":"original-client","seed":0}`)}
			task.PrivateData.Key = "channel-key-canary"
			require.NoError(t, model.DB.Save(task).Error)
		}
		for _, viewer := range []struct {
			name               string
			user, role, status int
		}{
			{"owner", 7, common.RoleCommonUser, 200}, {"other user", 8, common.RoleCommonUser, 404}, {"admin", 8, common.RoleAdminUser, 200},
		} {
			t.Run(fmt.Sprintf("recorded=%t/%s", recorded, viewer.name), func(t *testing.T) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodGet, "/api/task/"+task.TaskID+"/request", nil)
				c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
				c.Set("id", viewer.user)
				c.Set("role", viewer.role)
				GetDashboardTaskClientRequest(c)
				assert.Equal(t, viewer.status, recorder.Code)
				assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
				assert.NotContains(t, recorder.Body.String(), "channel-key-canary")
				assert.NotContains(t, recorder.Body.String(), "upstream-result-canary")
				if viewer.status == 200 {
					var result struct {
						Data struct {
							Recorded bool
							Snapshot *model.TaskClientRequestSnapshot
						}
					}
					require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
					assert.Equal(t, recorded, result.Data.Recorded)
					if recorded {
						require.NotNil(t, result.Data.Snapshot)
						assert.JSONEq(t, `{"prompt":"original-client","seed":0}`, string(result.Data.Snapshot.Body))
					} else {
						assert.Nil(t, result.Data.Snapshot)
					}
				}
			})
		}
	}
	list := tasksToDto([]*model.Task{task}, false, common.RoleRootUser)
	encoded, err := common.Marshal(list)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "original-client")
	assert.NotContains(t, string(encoded), "client_request")
}

func TestTaskClientRequestSurvivesRealPollingAndDatabaseRoundTrip(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Status = model.TaskStatusSubmitted
	task.Progress = "0%"
	task.PrivateData.ClientRequest = &model.TaskClientRequestSnapshot{Method: "POST", Path: "/v1/videos", ContentType: "application/json", BodyKind: "json", Body: []byte(`{"prompt":"client-before-conversion","seed":0}`)}
	task.PrivateData.Response = &model.TaskResponseSnapshot{StatusCode: 202, ContentType: "application/json", BodyKind: "json", Body: []byte(`{"id":"client-task","status":"queued"}`)}
	require.NoError(t, model.DB.Save(task).Error)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"running","prompt":"upstream-result"}`)
	}))
	defer server.Close()
	plugin, err := jsplugin.CompilePlugin(`export const meta={apiVersion:1,key:"snapshot-poll",name:"Snapshot",version:"1.0.0",author:{name:"Test"},models:["m"],fetchMode:"per_task"};
export function buildSubmitRequest(){return {}} export function parseSubmitResponse(){return {taskId:"vendor"}}
export function buildQueryRequest(ctx){return {url:ctx.baseUrl+"/query",method:"GET"}}
export function parseTaskResult(){return {status:"IN_PROGRESS",progress:"35%",state:{round:2}}}`, jsplugin.Options{})
	require.NoError(t, err)
	oldFactory := service.GetTaskAdaptorFunc
	service.GetTaskAdaptorFunc = func(constant.TaskPlatform) service.TaskPollingAdaptor { return taskjsplugin.New(plugin) }
	t.Cleanup(func() { service.GetTaskAdaptorFunc = oldFactory })
	channel, err := model.GetChannelById(task.ChannelId, true)
	require.NoError(t, err)
	channel.BaseURL = &server.URL
	require.NoError(t, model.DB.Save(channel).Error)
	require.NoError(t, service.UpdateVideoTasks(t.Context(), task.Platform, map[int][]string{task.ChannelId: {task.TaskID}}, map[string]*model.Task{task.TaskID: task}))
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusInProgress), reloaded.Status)
	assert.Contains(t, string(reloaded.Data), "upstream-result")
	require.NotNil(t, reloaded.PrivateData.ClientRequest)
	assert.JSONEq(t, `{"prompt":"client-before-conversion","seed":0}`, string(reloaded.PrivateData.ClientRequest.Body))
	assert.JSONEq(t, `{"round":2}`, string(reloaded.PrivateData.PluginState))
	require.NotNil(t, reloaded.PrivateData.Response)
	assert.JSONEq(t, `{"id":"client-task","status":"queued"}`, string(reloaded.PrivateData.Response.Body))
	// No SQL schema change is introduced. Both legacy rows and the additional
	// private JSON member must survive repeated startup migrations unchanged.
	require.NoError(t, model.DB.AutoMigrate(&model.Task{}))
	require.NoError(t, model.DB.AutoMigrate(&model.Task{}))
	reloaded = model.Task{}
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	require.NotNil(t, reloaded.PrivateData.ClientRequest)
	assert.JSONEq(t, `{"prompt":"client-before-conversion","seed":0}`, string(reloaded.PrivateData.ClientRequest.Body))
}

func TestGetTaskDoesNotProjectArtifacts(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.FailReason = "https://stale-upstream.invalid/video.mp4"
	task.PrivateData = model.TaskPrivateData{
		ResultURL: "https://private-upstream.invalid/video.mp4",
		Execution: &model.TaskExecutionSnapshot{
			TaskPlugin: &model.TaskPluginSnapshot{Key: "missing-plugin", Name: "Missing"},
		},
	}
	require.NoError(t, model.DB.Save(task).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", 7)
	c.Params = gin.Params{{Key: "key", Value: task.TaskID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/"+task.TaskID, nil)

	GetTask(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, task.TaskID, response["task_id"])
	assert.NotContains(t, response, "artifacts")
	assert.NotContains(t, recorder.Body.String(), "upstream.invalid")
}

func TestGetTaskArtifactsReturnsEmptyForLegacyTask(t *testing.T) {
	task := setupGenericTaskTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", task.UserId)
	c.Params = gin.Params{{Key: "key", Value: task.TaskID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/"+task.TaskID+"/artifacts", nil)

	GetTaskArtifacts(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	var response struct {
		TaskID    string                 `json:"task_id"`
		Artifacts []taskArtifactResponse `json:"artifacts"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, task.TaskID, response.TaskID)
	assert.Empty(t, response.Artifacts)
}

func TestTaskArtifactAuthorizationKeepsForeignTasksHidden(t *testing.T) {
	task := setupGenericTaskTest(t)

	commonUser, _ := gin.CreateTestContext(httptest.NewRecorder())
	commonUser.Set("id", 8)
	commonUser.Set("role", common.RoleCommonUser)
	_, exists, err := getTaskForArtifactRequest(commonUser, task.TaskID)
	require.NoError(t, err)
	assert.False(t, exists)

	admin, _ := gin.CreateTestContext(httptest.NewRecorder())
	admin.Set("id", 8)
	admin.Set("role", common.RoleAdminUser)
	found, exists, err := getTaskForArtifactRequest(admin, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, task.TaskID, found.TaskID)

	apiToken, _ := gin.CreateTestContext(httptest.NewRecorder())
	apiToken.Set("id", 8)
	apiToken.Set("role", common.RoleRootUser)
	apiToken.Set("token_id", 99)
	_, exists, err = getTaskForArtifactRequest(apiToken, task.TaskID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDashboardTaskArtifactsReturnsLegacyCapabilityWithoutUpstreamURL(t *testing.T) {
	task := setupGenericTaskTest(t)
	previousSecret := common.CryptoSecret
	previousPublicAddress := system_setting.TaskPublicAddress
	common.CryptoSecret = "controller-task-artifact-access-secret"
	system_setting.TaskPublicAddress = "https://gateway.example/prefix"
	t.Cleanup(func() {
		common.CryptoSecret = previousSecret
		system_setting.TaskPublicAddress = previousPublicAddress
	})
	task.Action = constant.TaskActionTextToVideo
	task.FailReason = "https://upstream.invalid/private-video.mp4?signature=secret"
	require.NoError(t, model.DB.Save(task).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", task.UserId)
	c.Set("role", common.RoleCommonUser)
	c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/task/"+task.TaskID+"/artifacts", nil)

	GetDashboardTaskArtifacts(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Artifacts        []taskArtifactResponse `json:"artifacts"`
			LegacyContentURL string                 `json:"legacy_content_url"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Empty(t, response.Data.Artifacts)
	contentURL, err := url.Parse(response.Data.LegacyContentURL)
	require.NoError(t, err)
	assert.Equal(t, "/prefix/v1/tasks/"+task.TaskID+"/artifacts/video/content", contentURL.Path)
	assert.True(t, service.VerifyTaskArtifactAccess(
		contentURL.Query().Get(service.TaskArtifactAccessQueryParameter),
		task.TaskID,
		"video",
	))
	assert.NotContains(t, recorder.Body.String(), "upstream.invalid")
	assert.NotContains(t, recorder.Body.String(), "signature=secret")
}

// Task lists no longer carry the persisted snapshot, so the dashboard reads a
// legacy Suno task's playable clips from the artifacts endpoint instead.
func TestDashboardTaskArtifactsProjectsLegacySunoAudioClips(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Platform = constant.TaskPlatformSuno
	task.Action = "MUSIC"
	task.Data = []byte(`[{"id":"clip-1","title":"Song","audio_url":"https://cdn.example/a.mp3","metadata":{"tags":"pop","duration":30},"lyric":"private"},{"id":"clip-2","title":"No audio"}]`)
	require.NoError(t, model.DB.Save(task).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", task.UserId)
	c.Set("role", common.RoleCommonUser)
	c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/task/"+task.TaskID+"/artifacts", nil)

	GetDashboardTaskArtifacts(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Artifacts        []taskArtifactResponse `json:"artifacts"`
			LegacyAudioClips []map[string]any       `json:"legacy_audio_clips"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Empty(t, response.Data.Artifacts)
	require.Len(t, response.Data.LegacyAudioClips, 1)
	assert.Equal(t, "https://cdn.example/a.mp3", response.Data.LegacyAudioClips[0]["audio_url"])
	assert.Equal(t, "Song", response.Data.LegacyAudioClips[0]["title"])
	assert.NotContains(t, recorder.Body.String(), "private")
}

// Task lists omit the data column; the API DTO keeps the key as null so shape
// checks survive while the payload no longer travels with every row.
func TestTaskListsOmitPersistedSnapshot(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Data = []byte(`{"data":[{"url":"https://cdn.example/a.png"}]}`)
	require.NoError(t, model.DB.Save(task).Error)

	userTasks := model.TaskGetAllUserTask(task.UserId, 0, 10, model.SyncTaskQueryParams{})
	require.Len(t, userTasks, 1)
	assert.Empty(t, userTasks[0].Data)
	adminTasks := model.TaskGetAllTasks(0, 10, model.SyncTaskQueryParams{})
	require.Len(t, adminTasks, 1)
	assert.Empty(t, adminTasks[0].Data)
	assert.Equal(t, task.TaskID, adminTasks[0].TaskID)

	encoded, err := common.Marshal(tasksToDto(adminTasks, false, common.RoleAdminUser)[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"data":null`)
	assert.NotContains(t, string(encoded), `"result_discarded"`)

	adminTasks[0].PrivateData.ResultDiscarded = true
	encoded, err = common.Marshal(tasksToDto(adminTasks, false, common.RoleAdminUser)[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"result_discarded":true`, "task lists tell the UI that an inline result was not retained")

	stored, exists, err := model.GetByTaskId(task.UserId, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.JSONEq(t, string(task.Data), string(stored.Data), "single-task lookups keep the snapshot")
}

func TestTaskArtifactAccessRequiresActiveOwner(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Action = constant.TaskActionTextToVideo
	task.FailReason = "https://upstream.invalid/private-video.mp4"
	require.NoError(t, model.DB.Save(task).Error)
	require.NoError(t, model.DB.Model(&model.User{}).
		Where("id = ?", task.UserId).
		Update("status", common.UserStatusDisabled).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(middleware.TaskArtifactAccessContextKey, true)
	c.Params = gin.Params{
		{Key: "key", Value: task.TaskID},
		{Key: "artifact_key", Value: "video"},
	}
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/tasks/"+task.TaskID+"/artifacts/video/content",
		nil,
	)

	TaskArtifactContent(c)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
}

func TestTaskArtifactAccessRejectsAmbiguousHistoricalTaskID(t *testing.T) {
	task := setupGenericTaskTest(t)
	task.Action = constant.TaskActionTextToVideo
	task.FailReason = "https://first-upstream.invalid/video.mp4"
	require.NoError(t, model.DB.Save(task).Error)
	require.NoError(t, model.DB.Create(&model.User{
		Id: 8, Username: "other-artifact-owner", Status: common.UserStatusEnabled,
		Role: common.RoleCommonUser, Group: "default", AffCode: "artifact-owner-8",
	}).Error)
	require.NoError(t, model.DB.Create(&model.Task{
		TaskID: task.TaskID, Platform: task.Platform, UserId: 8, ChannelId: task.ChannelId,
		Action: constant.TaskActionTextToVideo, Status: model.TaskStatusSuccess,
		FailReason: "https://second-upstream.invalid/video.mp4",
	}).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(middleware.TaskArtifactAccessContextKey, true)
	c.Params = gin.Params{
		{Key: "key", Value: task.TaskID},
		{Key: "artifact_key", Value: "video"},
	}
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/tasks/"+task.TaskID+"/artifacts/video/content",
		nil,
	)

	TaskArtifactContent(c)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "upstream.invalid")
}

func TestLegacyVideoArtifactContentUsesGetResultURL(t *testing.T) {
	task := setupGenericTaskTest(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "bytes=0-3", r.Header.Get("Range"))
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-3/4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("data"))
	}))
	defer upstream.Close()
	allowPrivateTaskMediaTest(t)

	task.Action = constant.TaskActionTextToVideo
	task.PrivateData.ResultURL = upstream.URL
	task.FailReason = "https://stale.invalid/legacy-fallback.mp4"
	require.NoError(t, model.DB.Save(task).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(middleware.TaskArtifactAccessContextKey, true)
	c.Params = gin.Params{
		{Key: "key", Value: task.TaskID},
		{Key: "artifact_key", Value: "video"},
	}
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/tasks/"+task.TaskID+"/artifacts/video/content",
		nil,
	)
	c.Request.Header.Set("Range", "bytes=0-3")

	TaskArtifactContent(c)

	assert.Equal(t, http.StatusPartialContent, recorder.Code)
	assert.Equal(t, "data", recorder.Body.String())
	assert.Equal(t, "bytes 0-3/4", recorder.Header().Get("Content-Range"))
}

func TestDisabledArtifactStorePreservesPluginUpstreamContent(t *testing.T) {
	task := setupGenericTaskTest(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "provider-key", r.Header.Get("x-goog-api-key"))
		assert.Equal(t, "provider-key", r.Header.Get("X-API-Key"), "channel header overrides reach credentialed downloads")
		assert.NotEqual(t, "ignored.example", r.Host)
		assert.Equal(t, "bytes=0-13", r.Header.Get("Range"))
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-13/14")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("artifact-bytes"))
	}))
	defer upstream.Close()
	allowPrivateTaskMediaTest(t)
	previousMemoryCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousMemoryCache })

	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", task.ChannelId).Updates(map[string]any{
		"type":            constant.ChannelTypeGemini,
		"key":             "provider-key",
		"base_url":        upstream.URL,
		"header_override": `{"X-API-Key":"{api_key}","Host":"ignored.example"}`,
	}).Error)
	task.Platform = constant.TaskPlatform("google")
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{
		Key: "google", Name: "Google Veo (Gemini API)", Version: "1.0.0", APIVersion: 1,
	}}
	task.SetData(map[string]any{"response": map[string]any{
		"generateVideoResponse": map[string]any{
			"generatedSamples": []any{map[string]any{"video": map[string]any{"uri": upstream.URL}}},
		},
	}})
	require.NoError(t, model.DB.Save(task).Error)
	require.False(t, service.GetTaskArtifactStore().Enabled())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(middleware.TaskArtifactAccessContextKey, true)
	c.Params = gin.Params{
		{Key: "key", Value: task.TaskID},
		{Key: "artifact_key", Value: "video"},
	}
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/"+task.TaskID+"/artifacts/video/content", nil)
	c.Request.Header.Set("Range", "bytes=0-13")

	TaskArtifactContent(c)

	assert.Equal(t, http.StatusPartialContent, recorder.Code)
	assert.Equal(t, "artifact-bytes", recorder.Body.String())
	assert.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "bytes 0-13/14", recorder.Header().Get("Content-Range"))
}

// hailuo looks the MiniMax download link up with utils.fetch; the signed OSS
// link then gets a plain GET without the channel key, even for HEAD clients.
func TestTaskArtifactContentFollowsHailuoRetrieveLink(t *testing.T) {
	task := setupGenericTaskTest(t)
	var mu sync.Mutex
	var ossRequests []string
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/files/retrieve":
			assert.Equal(t, "Bearer provider-key", r.Header.Get("Authorization"))
			if r.URL.Query().Get("file_id") == "login-fail" {
				_, _ = w.Write([]byte(`{"base_resp":{"status_code":1004,"status_msg":"login fail"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"file":{"download_url":"` + upstream.URL + `/oss/video.mp4?Expires=1&Signature=sig"},"base_resp":{"status_code":0}}`))
		case "/oss/video.mp4":
			mu.Lock()
			ossRequests = append(ossRequests, r.Method+" range="+r.Header.Get("Range")+" authorization="+r.Header.Get("Authorization"))
			mu.Unlock()
			w.Header().Set("Content-Type", "video/mp4")
			if r.Header.Get("Range") != "" {
				w.Header().Set("Content-Range", "bytes 0-3/10")
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write([]byte("mp4!"))
				return
			}
			_, _ = w.Write([]byte("mp4!video!"))
		default:
			t.Errorf("unexpected upstream request %s", r.URL.Path)
		}
	}))
	defer upstream.Close()
	allowPrivateTaskMediaTest(t)
	previousMemoryCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousMemoryCache })

	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", task.ChannelId).Updates(map[string]any{
		"type":     constant.ChannelTypeMiniMax,
		"key":      "provider-key",
		"base_url": upstream.URL,
		// Neither utils.fetch nor the credentialless OSS link gets overrides.
		"header_override": `{"Authorization":"Bearer override"}`,
	}).Error)
	task.Platform = constant.TaskPlatform("hailuo")
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{
		Key: "hailuo", Name: "Hailuo", Version: "1.2.0", APIVersion: 1,
	}}
	request := func(method, fileID string) *httptest.ResponseRecorder {
		task.SetData(map[string]any{"file_id": fileID})
		require.NoError(t, model.DB.Save(task).Error)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Set("id", 7)
		c.Params = gin.Params{{Key: "key", Value: task.TaskID}, {Key: "artifact_key", Value: "video"}}
		c.Request = httptest.NewRequest(method, "/v1/tasks/"+task.TaskID+"/artifacts/video/content", nil)
		if method == http.MethodGet {
			c.Request.Header.Set("Range", "bytes=0-3")
		}
		TaskArtifactContent(c)
		return recorder
	}

	ranged := request(http.MethodGet, "file-1")
	assert.Equal(t, http.StatusPartialContent, ranged.Code)
	assert.Equal(t, "mp4!", ranged.Body.String())
	assert.Equal(t, "bytes 0-3/10", ranged.Header().Get("Content-Range"))

	head := request(http.MethodHead, "file-1")
	assert.Equal(t, http.StatusOK, head.Code)
	assert.Empty(t, head.Body.String())
	assert.Equal(t, "video/mp4", head.Header().Get("Content-Type"))

	failed := request(http.MethodGet, "login-fail")
	assert.Equal(t, http.StatusInternalServerError, failed.Code)
	assert.Contains(t, failed.Body.String(), "artifact_plugin_error")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"GET range=bytes=0-3 authorization=", "GET range= authorization="}, ossRequests)
}

func TestProjectedTaskArtifactValidationRejectsAmbiguousIdentity(t *testing.T) {
	validated, err := validateProjectedTaskArtifacts([]relaychannel.TaskArtifact{
		{Key: "video-main", Type: "video", MimeType: "video/mp4"},
		{Key: "cover.main", Type: "image", MimeType: "image/png"},
	})
	require.NoError(t, err)
	require.Len(t, validated, 2)
	assert.Equal(t, "video-main", validated[0].Key)

	for _, artifacts := range [][]relaychannel.TaskArtifact{
		{{Key: "../video", Type: "video"}},
		{{Key: "video/0", Type: "video"}},
		{{Key: "video-main", Type: "video"}, {Key: "video-main", Type: "image"}},
		{{Key: "video-main", Type: "unknown"}},
		{{Key: "video-main", Type: "video", MimeType: "video/mp4\r\nX-Test: injected"}},
	} {
		_, err := validateProjectedTaskArtifacts(artifacts)
		assert.ErrorIs(t, err, errTaskArtifactPlugin)
	}
}

func TestProxyTaskMediaForwardsRangeAndFiltersResponseHeaders(t *testing.T) {
	task := setupGenericTaskTest(t)
	var receivedRange, receivedAuthorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRange = r.Header.Get("Range")
		receivedAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Range", "bytes 0-3/10")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Set-Cookie", "provider=secret")
		w.Header().Set("WWW-Authenticate", "Bearer provider")
		w.Header().Set("X-Provider-Secret", "hidden")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("data"))
	}))
	defer upstream.Close()

	allowPrivateTaskMediaTest(t)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/task_generic/artifacts/video-main/content", nil)
	c.Request.Header.Set("Range", "bytes=0-3")

	err := proxyTaskMedia(c, task, &relaychannel.TaskContentRequest{
		URL:     upstream.URL,
		Method:  http.MethodGet,
		Headers: map[string]string{"Authorization": "Bearer provider-secret"},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusPartialContent, recorder.Code)
	assert.Equal(t, "data", recorder.Body.String())
	assert.Equal(t, "bytes=0-3", receivedRange)
	assert.Equal(t, "Bearer provider-secret", receivedAuthorization)
	assert.Equal(t, "bytes 0-3/10", recorder.Header().Get("Content-Range"))
	assert.Equal(t, "bytes", recorder.Header().Get("Accept-Ranges"))
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "sandbox; default-src 'none'", recorder.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
	assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	assert.Empty(t, recorder.Header().Get("Set-Cookie"))
	assert.Empty(t, recorder.Header().Get("WWW-Authenticate"))
	assert.Empty(t, recorder.Header().Get("X-Provider-Secret"))
}

func TestProxyTaskMediaPassesThroughUnsatisfiedRange(t *testing.T) {
	task := setupGenericTaskTest(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Range", "bytes */10")
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
	}))
	defer upstream.Close()

	allowPrivateTaskMediaTest(t)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/content", nil)

	require.NoError(t, proxyTaskMedia(c, task, &relaychannel.TaskContentRequest{
		URL: upstream.URL, Method: http.MethodGet,
	}))
	assert.Equal(t, http.StatusRequestedRangeNotSatisfiable, recorder.Code)
	assert.Equal(t, "bytes */10", recorder.Header().Get("Content-Range"))
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
}

func TestTaskMediaResponseHeaderTimeoutDoesNotTruncateBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(75 * time.Millisecond)
		_, _ = w.Write([]byte("complete-body"))
	}))
	defer upstream.Close()

	request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	require.NoError(t, err)
	response, err := doTaskMediaRequest(upstream.Client(), request, 20*time.Millisecond)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "complete-body", string(body))
}

func TestTaskMediaResponseHeaderTimeoutCancelsBeforeHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(75 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer upstream.Close()

	request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	require.NoError(t, err)
	_, err = doTaskMediaRequest(upstream.Client(), request, 10*time.Millisecond)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// A descriptor with a content path reads the artifact out of a long-running
// operation the upstream returns again, as Vertex does for inline video.
func TestProxyTaskMediaReadsArtifactFromJSONContentPath(t *testing.T) {
	task := setupGenericTaskTest(t)
	previousLimit := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 1
	t.Cleanup(func() { constant.MaxFileDownloadMB = previousLimit })
	inline := base64.StdEncoding.EncodeToString([]byte("inline-video"))
	var replyStatus int
	var reply string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.JSONEq(t, `{"operationName":"op"}`, string(body))
		assert.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Range"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(replyStatus)
		_, _ = w.Write([]byte(reply))
	}))
	defer upstream.Close()
	allowPrivateTaskMediaTest(t)

	for _, testCase := range []struct {
		name, method, reply string
		replyStatus         int
		status              int
		code, body          string
	}{
		{name: "inline video", method: http.MethodGet, reply: `{"name":"op","done":true,"response":{"videos":[{"bytesBase64Encoded":"` + inline + `"}]}}`, status: http.StatusOK, body: "inline-video"},
		{name: "headers only", method: http.MethodHead, reply: `{"name":"op","done":true,"response":{"videos":[{"bytesBase64Encoded":"` + inline + `"}]}}`, status: http.StatusOK},
		{name: "URL-safe base64", method: http.MethodGet, reply: `{"done":true,"response":{"videos":[{"bytesBase64Encoded":"-_8="}]}}`, status: http.StatusOK, body: "\xfb\xff"},
		{name: "written to storage", method: http.MethodGet, reply: `{"done":true,"response":{"videos":[{"gcsUri":"gs://bucket/video.mp4"}]}}`, status: http.StatusGone, code: "artifact_gone"},
		{name: "operation failed", method: http.MethodGet, reply: `{"done":true,"error":{"code":3,"message":"bad request"}}`, status: http.StatusBadGateway, code: "artifact_upstream_error"},
		{name: "operation running", method: http.MethodGet, reply: `{"name":"op"}`, status: http.StatusConflict, code: "artifact_not_ready"},
		{name: "interaction completed", method: http.MethodGet, reply: `{"id":"int","status":"completed","response":{"videos":[{"bytesBase64Encoded":"` + inline + `"}]}}`, status: http.StatusOK, body: "inline-video"},
		{name: "interaction running", method: http.MethodGet, reply: `{"id":"int","status":"in_progress"}`, status: http.StatusConflict, code: "artifact_not_ready"},
		{name: "operation expired", method: http.MethodGet, replyStatus: http.StatusNotFound, reply: `{"error":{"code":404}}`, status: http.StatusGone, code: "artifact_gone"},
		{name: "reply too large", method: http.MethodGet, reply: `{"done":true,"response":{"videos":[{"bytesBase64Encoded":"` + strings.Repeat("A", 3<<20) + `"}]}}`, status: http.StatusBadGateway, code: "artifact_request_rejected"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			replyStatus, reply = http.StatusOK, testCase.reply
			if testCase.replyStatus != 0 {
				replyStatus = testCase.replyStatus
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(testCase.method, "/v1/tasks/task_generic/artifacts/video/content", nil)
			c.Request.Header.Set("Range", "bytes=0-3")

			err := proxyTaskMedia(c, task, &relaychannel.TaskContentRequest{
				URL: upstream.URL, Method: http.MethodPost, Body: []byte(`{"operationName":"op"}`),
				Headers:     map[string]string{"Authorization": "Bearer access-token"},
				ContentPath: []any{"response", "videos", int64(0), "bytesBase64Encoded"}, MimeType: "video/webm",
			})

			if testCase.code != "" {
				var proxyErr *taskMediaProxyError
				require.ErrorAs(t, err, &proxyErr)
				assert.Equal(t, testCase.status, proxyErr.status)
				assert.Equal(t, testCase.code, proxyErr.code)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.status, recorder.Code)
			assert.Equal(t, "video/webm", recorder.Header().Get("Content-Type"))
			assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
			if testCase.method == http.MethodHead {
				assert.Equal(t, "12", recorder.Header().Get("Content-Length"))
				assert.Empty(t, recorder.Body.String())
				return
			}
			assert.Equal(t, testCase.body, recorder.Body.String())
		})
	}
}

func TestWriteVideoDataURLStreamsAndSupportsHead(t *testing.T) {
	const dataURL = "data:video/mp4;base64,Y29tcGxldGUtYm9keQ=="

	getRecorder := httptest.NewRecorder()
	getContext, _ := gin.CreateTestContext(getRecorder)
	getContext.Request = httptest.NewRequest(http.MethodGet, "/content", nil)
	require.NoError(t, writeVideoDataURL(getContext, dataURL))
	assert.Equal(t, http.StatusOK, getRecorder.Code)
	assert.Equal(t, "complete-body", getRecorder.Body.String())
	assert.Equal(t, "13", getRecorder.Header().Get("Content-Length"))

	headRecorder := httptest.NewRecorder()
	headContext, _ := gin.CreateTestContext(headRecorder)
	headContext.Request = httptest.NewRequest(http.MethodHead, "/content", nil)
	require.NoError(t, writeVideoDataURL(headContext, dataURL))
	assert.Equal(t, http.StatusOK, headRecorder.Code)
	assert.Empty(t, headRecorder.Body.String())
	assert.Equal(t, "13", headRecorder.Header().Get("Content-Length"))
}

func TestWriteVideoDataURLRejectsOversizedPayloadBeforeDecode(t *testing.T) {
	previousLimit := taskMediaDataURLMaxEncodedBytes
	taskMediaDataURLMaxEncodedBytes = 32
	t.Cleanup(func() { taskMediaDataURLMaxEncodedBytes = previousLimit })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/content", nil)

	err := writeVideoDataURL(c, "data:video/mp4;base64,"+strings.Repeat("A", 64))

	assert.ErrorIs(t, err, errTaskMediaRequestRejected)
	assert.Empty(t, recorder.Header().Get("Content-Type"))
}

func TestProxyTaskMediaAllowsOnlyCredentiallessCrossOriginRedirect(t *testing.T) {
	task := setupGenericTaskTest(t)
	var destinationAuthorization, destinationRange string
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destinationAuthorization = r.Header.Get("Authorization")
		destinationRange = r.Header.Get("Range")
		_, _ = w.Write([]byte("redirected"))
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusFound)
	}))
	defer source.Close()
	allowPrivateTaskMediaTest(t)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/content", nil)
	c.Request.Header.Set("Range", "bytes=0-3")

	err := proxyTaskMedia(c, task, &relaychannel.TaskContentRequest{
		URL: source.URL, Method: http.MethodGet, Credentialless: true,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "redirected", recorder.Body.String())
	assert.Empty(t, destinationAuthorization)
	assert.Equal(t, "bytes=0-3", destinationRange)

	destinationRange = ""
	rejectedRecorder := httptest.NewRecorder()
	rejectedContext, _ := gin.CreateTestContext(rejectedRecorder)
	rejectedContext.Request = httptest.NewRequest(http.MethodGet, "/content", nil)
	err = proxyTaskMedia(rejectedContext, task, &relaychannel.TaskContentRequest{
		URL: source.URL, Method: http.MethodGet,
		Headers: map[string]string{"Authorization": "Bearer provider-secret"},
	})
	var proxyErr *taskMediaProxyError
	require.ErrorAs(t, err, &proxyErr)
	assert.Equal(t, "artifact_request_rejected", proxyErr.code)
	assert.Empty(t, destinationRange)
}

func TestTaskMediaRequestHeaderPolicy(t *testing.T) {
	header := http.Header{}
	require.NoError(t, applyTaskMediaRequestHeaders(header, map[string]string{
		"Authorization": "Bearer provider-secret",
		"X-Signature":   "signed",
	}))
	assert.Equal(t, "Bearer provider-secret", header.Get("Authorization"))
	assert.Equal(t, "signed", header.Get("X-Signature"))

	for _, name := range []string{"Host", "Content-Length", "Accept-Encoding", "Connection", "Proxy-Authorization", "Transfer-Encoding"} {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, applyTaskMediaRequestHeaders(http.Header{}, map[string]string{name: "bad"}), errTaskMediaRequestRejected)
		})
	}
	assert.ErrorIs(t, applyTaskMediaRequestHeaders(http.Header{}, map[string]string{"X-Test": "bad\r\ninjected"}), errTaskMediaRequestRejected)
}

func TestCredentiallessTaskMediaDescriptorRejectsCredentialsAndBody(t *testing.T) {
	task := setupGenericTaskTest(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/content", nil)

	for _, descriptor := range []*relaychannel.TaskContentRequest{
		{URL: "https://example.com/video", Method: http.MethodPost, Credentialless: true},
		{URL: "https://example.com/video", Method: http.MethodGet, Body: []byte("secret"), Credentialless: true},
		{URL: "https://example.com/video", Method: http.MethodGet, Headers: map[string]string{"X-Key": "secret"}, Credentialless: true},
	} {
		err := proxyTaskMedia(c, task, descriptor)
		var proxyErr *taskMediaProxyError
		require.ErrorAs(t, err, &proxyErr)
		assert.Equal(t, "artifact_request_rejected", proxyErr.code)
	}
}

func TestSelfTaskMediaURLGuard(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "https://gateway.example/v1/videos/task-1/content", nil)
	c.Request.Host = "gateway.example"

	selfURL, err := url.Parse("https://gateway.example/v1/videos/task-1/content")
	require.NoError(t, err)
	assert.True(t, isSelfTaskMediaURL(c, selfURL))

	remoteURL, err := url.Parse("https://cdn.example/v1/videos/task-1/content")
	require.NoError(t, err)
	assert.False(t, isSelfTaskMediaURL(c, remoteURL))
	assert.True(t, isTaskMediaFallbackLoop(remoteURL.String(), "task-1"))
	assert.False(t, isTaskMediaFallbackLoop(remoteURL.String(), "task-2"))
}
