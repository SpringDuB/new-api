package model

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	DB = db
	LOG_DB = db

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true
	initCol()

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&Task{},
		&User{},
		&UserSession{},
		&AuthFlow{},
		&ExternalIdentityClaim{},
		&Token{},
		&PasskeyCredential{},
		&TwoFA{},
		&TwoFABackupCode{},
		&Log{},
		&Channel{},
		&QuotaData{},
		&Ability{},
		&TopUp{},
		&SubscriptionPlan{},
		&SubscriptionOrder{},
		&UserSubscription{},
		&UserOAuthBinding{},
		&PerfMetric{},
		&SystemInstance{},
		&SystemTask{},
		&SystemTaskLock{},
		&UserAccessToken{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	os.Exit(m.Run())
}

func truncateTables(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		DB.Exec("DELETE FROM tasks")
		DB.Exec("DELETE FROM auth_flows")
		DB.Exec("DELETE FROM external_identity_claims")
		DB.Exec("DELETE FROM user_sessions")
		DB.Exec("DELETE FROM passkey_credentials")
		DB.Exec("DELETE FROM two_fa_backup_codes")
		DB.Exec("DELETE FROM two_fas")
		DB.Exec("DELETE FROM tokens")
		DB.Exec("DELETE FROM user_access_tokens")
		DB.Exec("DELETE FROM user_oauth_bindings")
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
		DB.Exec("DELETE FROM channels")
		DB.Exec("DELETE FROM quota_data")
		DB.Exec("DELETE FROM abilities")
		DB.Exec("DELETE FROM top_ups")
		DB.Exec("DELETE FROM subscription_orders")
		DB.Exec("DELETE FROM subscription_plans")
		DB.Exec("DELETE FROM user_subscriptions")
		DB.Exec("DELETE FROM perf_metrics")
		DB.Exec("DELETE FROM system_instances")
		DB.Exec("DELETE FROM system_task_locks")
		DB.Exec("DELETE FROM system_tasks")
	})
}

func insertTask(t *testing.T, task *Task) {
	t.Helper()
	task.CreatedAt = time.Now().Unix()
	task.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(task).Error)
}

func TestGetTaskForProtocolObservationScopesOwnerAndPlatform(t *testing.T) {
	truncateTables(t)
	task := &Task{
		TaskID:   "task_protocol_scope",
		UserId:   7,
		Platform: "plugin-a",
		Status:   TaskStatusInProgress,
	}
	insertTask(t, task)

	got, exists, err := GetTaskForProtocolObservation(context.Background(), 7, "plugin-a", task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, task.ID, got.ID)

	for _, query := range []struct {
		userID   int
		platform string
	}{
		{userID: 8, platform: "plugin-a"},
		{userID: 7, platform: "plugin-b"},
	} {
		got, exists, err = GetTaskForProtocolObservation(context.Background(), query.userID, constant.TaskPlatform(query.platform), task.TaskID)
		require.NoError(t, err)
		assert.False(t, exists)
		assert.Nil(t, got)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = GetTaskForProtocolObservation(cancelled, 7, "plugin-a", task.TaskID)
	require.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Snapshot / Equal — pure logic tests (no DB)
// ---------------------------------------------------------------------------

func TestSnapshotEqual_Same(t *testing.T) {
	s := taskSnapshot{
		Status:     TaskStatusInProgress,
		Progress:   "50%",
		StartTime:  1000,
		FinishTime: 0,
		FailReason: "",
		ResultURL:  "",
		Data:       json.RawMessage(`{"key":"value"}`),
	}
	assert.True(t, s.Equal(s))
}

func TestSnapshotEqual_DifferentStatus(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage(`{}`)}
	b := taskSnapshot{Status: TaskStatusSuccess, Data: json.RawMessage(`{}`)}
	assert.False(t, a.Equal(b))
}

func TestSnapshotEqual_DifferentProgress(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Progress: "30%", Data: json.RawMessage(`{}`)}
	b := taskSnapshot{Status: TaskStatusInProgress, Progress: "60%", Data: json.RawMessage(`{}`)}
	assert.False(t, a.Equal(b))
}

func TestSnapshotEqual_DifferentData(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage(`{"a":1}`)}
	b := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage(`{"a":2}`)}
	assert.False(t, a.Equal(b))
}

func TestSnapshotEqual_NilVsEmpty(t *testing.T) {
	a := taskSnapshot{Status: TaskStatusInProgress, Data: nil}
	b := taskSnapshot{Status: TaskStatusInProgress, Data: json.RawMessage{}}
	// bytes.Equal(nil, []byte{}) == true
	assert.True(t, a.Equal(b))
}

func TestSnapshotEqual_PluginStateAndPollFailures(t *testing.T) {
	base := taskSnapshot{
		Status:       TaskStatusInProgress,
		PluginState:  json.RawMessage(`{"req_key":"a"}`),
		PollFailures: 2,
	}
	assert.True(t, base.Equal(taskSnapshot{
		Status:       TaskStatusInProgress,
		PluginState:  json.RawMessage(`{"req_key":"a"}`),
		PollFailures: 2,
	}))
	assert.False(t, base.Equal(taskSnapshot{
		Status:       TaskStatusInProgress,
		PluginState:  json.RawMessage(`{"req_key":"b"}`),
		PollFailures: 2,
	}))
	assert.False(t, base.Equal(taskSnapshot{
		Status:       TaskStatusInProgress,
		PluginState:  json.RawMessage(`{"req_key":"a"}`),
		PollFailures: 3,
	}))
}

func TestSnapshot_Roundtrip(t *testing.T) {
	task := &Task{
		Status:     TaskStatusInProgress,
		Progress:   "42%",
		StartTime:  1234,
		FinishTime: 5678,
		FailReason: "timeout",
		PrivateData: TaskPrivateData{
			ResultURL:    "https://example.com/result.mp4",
			PluginState:  json.RawMessage(`{"req_key":"keep"}`),
			PollFailures: 3,
		},
		Data: json.RawMessage(`{"model":"test-model"}`),
	}
	snap := task.Snapshot()
	assert.Equal(t, task.Status, snap.Status)
	assert.Equal(t, task.Progress, snap.Progress)
	assert.Equal(t, task.StartTime, snap.StartTime)
	assert.Equal(t, task.FinishTime, snap.FinishTime)
	assert.Equal(t, task.FailReason, snap.FailReason)
	assert.Equal(t, task.PrivateData.ResultURL, snap.ResultURL)
	assert.JSONEq(t, string(task.Data), string(snap.Data))
	assert.Equal(t, task.PrivateData.PluginState, snap.PluginState)
	assert.Equal(t, task.PrivateData.PollFailures, snap.PollFailures)
}

// ---------------------------------------------------------------------------
// UpdateWithStatus CAS — DB integration tests
// ---------------------------------------------------------------------------

func TestUpdateWithStatus_Win(t *testing.T) {
	truncateTables(t)

	task := &Task{
		TaskID:   "task_cas_win",
		Status:   TaskStatusInProgress,
		Progress: "50%",
		Data:     json.RawMessage(`{}`),
	}
	insertTask(t, task)

	task.Status = TaskStatusSuccess
	task.Progress = "100%"
	won, err := task.UpdateWithStatus(TaskStatusInProgress)
	require.NoError(t, err)
	assert.True(t, won)

	var reloaded Task
	require.NoError(t, DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, TaskStatusSuccess, reloaded.Status)
	assert.Equal(t, "100%", reloaded.Progress)
}

func TestUpdateWithStatus_Lose(t *testing.T) {
	truncateTables(t)

	task := &Task{
		TaskID: "task_cas_lose",
		Status: TaskStatusFailure,
		Data:   json.RawMessage(`{}`),
	}
	insertTask(t, task)

	task.Status = TaskStatusSuccess
	won, err := task.UpdateWithStatus(TaskStatusInProgress) // wrong fromStatus
	require.NoError(t, err)
	assert.False(t, won)

	var reloaded Task
	require.NoError(t, DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, TaskStatusFailure, reloaded.Status) // unchanged
}

func TestUpdateWithStatus_ConcurrentWinner(t *testing.T) {
	truncateTables(t)

	task := &Task{
		TaskID: "task_cas_race",
		Status: TaskStatusInProgress,
		Quota:  1000,
		Data:   json.RawMessage(`{}`),
	}
	insertTask(t, task)

	const goroutines = 5
	wins := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			t := &Task{}
			*t = Task{
				ID:       task.ID,
				TaskID:   task.TaskID,
				Status:   TaskStatusSuccess,
				Progress: "100%",
				Quota:    task.Quota,
				Data:     json.RawMessage(`{}`),
			}
			t.CreatedAt = task.CreatedAt
			t.UpdatedAt = time.Now().Unix()
			won, err := t.UpdateWithStatus(TaskStatusInProgress)
			if err == nil {
				wins[idx] = won
			}
		}(i)
	}
	wg.Wait()

	winCount := 0
	for _, w := range wins {
		if w {
			winCount++
		}
	}
	assert.Equal(t, 1, winCount, "exactly one goroutine should win the CAS")
}

func TestUpdateWithStatus_PersistsPluginStateAndPollFailures(t *testing.T) {
	truncateTables(t)

	task := &Task{
		TaskID: "task_cas_plugin_state",
		Status: TaskStatusInProgress,
		Data:   json.RawMessage(`{}`),
		PrivateData: TaskPrivateData{
			PluginState:  json.RawMessage(`{"req_key":"old"}`),
			PollFailures: 1,
		},
	}
	insertTask(t, task)

	task.PrivateData.PluginState = json.RawMessage(`{"req_key":"new"}`)
	task.PrivateData.PollFailures = 4
	won, err := task.UpdateWithStatus(TaskStatusInProgress)
	require.NoError(t, err)
	require.True(t, won)

	var reloaded Task
	require.NoError(t, DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, TaskStatusInProgress, reloaded.Status)
	assert.JSONEq(t, `{"req_key":"new"}`, string(reloaded.PrivateData.PluginState))
	assert.Equal(t, 4, reloaded.PrivateData.PollFailures)
}

func TestLecAPIRouteReplacementDatabaseMatrix(t *testing.T) {
	for _, engine := range []common.DatabaseType{common.DatabaseTypeSQLite, common.DatabaseTypeMySQL, common.DatabaseTypePostgreSQL} {
		t.Run(string(engine), func(t *testing.T) {
			var driver gorm.Dialector
			versionSQL := "SELECT version()"
			switch engine {
			case common.DatabaseTypeSQLite:
				driver, versionSQL = sqlite.Open(":memory:"), "SELECT sqlite_version()"
			case common.DatabaseTypeMySQL:
				if os.Getenv("TEST_MYSQL_DSN") == "" {
					t.Skip("TEST_MYSQL_DSN is not configured")
				}
				driver = mysql.Open(os.Getenv("TEST_MYSQL_DSN"))
			case common.DatabaseTypePostgreSQL:
				if os.Getenv("TEST_POSTGRES_DSN") == "" {
					t.Skip("TEST_POSTGRES_DSN is not configured")
				}
				driver = postgres.Open(os.Getenv("TEST_POSTGRES_DSN"))
			}
			config := newGormConfig(false)
			config.NamingStrategy = schema.NamingStrategy{TablePrefix: "lec_retry_test_"}
			db, err := gorm.Open(driver, config)
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			previousDB, previousMain, previousLog := DB, common.MainDatabaseType(), common.LogDatabaseType()
			DB = db
			common.SetDatabaseTypes(engine, previousLog)
			t.Cleanup(func() {
				DB = previousDB
				common.SetDatabaseTypes(previousMain, previousLog)
				require.NoError(t, db.Migrator().DropTable(&Task{}))
				require.NoError(t, sqlDB.Close())
			})
			require.NoError(t, db.AutoMigrate(&Task{}))
			var version string
			require.NoError(t, db.Raw(versionSQL).Scan(&version).Error)
			t.Logf("database version: %s", version)

			reference := strings.Repeat("a", 2<<20)
			payload, err := common.Marshal(map[string]string{"reference": reference})
			require.NoError(t, err)
			expired := LecAPIPollState{RetryInFlight: true, RetryDispatched: true, RetryUntil: time.Now().Add(-time.Minute).UnixMilli()}
			original := &Task{TaskID: "task_same_public_id", Platform: "lecapi", Status: TaskStatusSubmitted,
				PrivateData: TaskPrivateData{UpstreamTaskID: "route-a", LecAPIRetryPayload: payload, LecAPIPoll: expired,
					Execution: &TaskExecutionSnapshot{TaskPlugin: &TaskPluginSnapshot{Key: "lecapi"}}}}
			require.NoError(t, DB.Create(original).Error)
			var stale Task
			require.NoError(t, DB.First(&stale, original.ID).Error)
			assert.JSONEq(t, string(payload), string(stale.PrivateData.LecAPIRetryPayload))
			assert.Equal(t, expired, stale.PrivateData.LecAPIPoll)
			original.PrivateData.LecAPIPoll = LecAPIPollState{RetryInFlight: true, RetryUntil: time.Now().Add(time.Minute).UnixMilli()}
			original.PrivateData.UpstreamTaskID = "route-b"
			claimed, err := original.UpdateWithStatus(TaskStatusSubmitted)
			require.NoError(t, err)
			require.True(t, claimed)
			stale.PrivateData.UpstreamTaskID = "late-response"
			stale.PrivateData.LecAPIPoll = LecAPIPollState{}
			claimed, err = stale.UpdateWithStatus(TaskStatusSubmitted)
			require.NoError(t, err)
			assert.False(t, claimed)
			original.Status = TaskStatusSuccess
			original.PrivateData.LecAPIPoll = LecAPIPollState{Revision: original.PrivateData.LecAPIPoll.Revision}
			claimed, err = original.UpdateWithStatus(TaskStatusSubmitted)
			require.NoError(t, err)
			require.True(t, claimed)
			var final Task
			require.NoError(t, DB.First(&final, original.ID).Error)
			assert.Equal(t, "task_same_public_id", final.TaskID)
			assert.Equal(t, "route-b", final.GetUpstreamTaskID())
			assert.Empty(t, final.PrivateData.LecAPIRetryPayload)
		})
	}
}

func TestAsyncImageTaskQueriesExcludeGenerationJobsFromProviderPolling(t *testing.T) {
	truncateTables(t)
	imageTask := &Task{
		TaskID: "task_async_image", Platform: constant.TaskPlatformAsyncImage,
		Action: "image_generation", Status: TaskStatusNotStart, Progress: "0%",
	}
	insertTask(t, imageTask)
	providerTask := &Task{
		TaskID: "task_provider", Platform: constant.TaskPlatformSuno,
		Action: constant.TaskActionTextToVideo, Status: TaskStatusInProgress, Progress: "30%",
	}
	insertTask(t, providerTask)

	pending, err := GetPendingAsyncImageTasks(10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, imageTask.TaskID, pending[0].TaskID)
	assert.True(t, HasPendingAsyncImageTasks())

	forPolling := GetAllUnFinishSyncTasks(10)
	require.Len(t, forPolling, 1)
	assert.Equal(t, providerTask.TaskID, forPolling[0].TaskID)
	assert.True(t, HasUnfinishedSyncTasks())

	imageTask.Status = TaskStatusSuccess
	imageTask.Progress = "100%"
	require.NoError(t, imageTask.Update())
	assert.False(t, HasPendingAsyncImageTasks())
}

func TestPersistAsyncImageTaskBillingReservation(t *testing.T) {
	truncateTables(t)
	task := &Task{
		TaskID:     "task_async_image_billing",
		Platform:   constant.TaskPlatformAsyncImage,
		Action:     "image_generation",
		Status:     TaskStatusInProgress,
		Progress:   "1%",
		Quota:      2400,
		ChannelId:  19,
		Group:      "premium",
		Properties: Properties{OriginModelName: "image-model", UpstreamModelName: "provider-image-model"},
		PrivateData: TaskPrivateData{
			TokenId:        23,
			BillingSource:  "subscription",
			SubscriptionId: 29,
			BillingContext: &TaskBillingContext{ModelPrice: 0.02, GroupRatio: 1.5, ModelRatio: 2, OtherRatios: map[string]float64{"n": 3}, OriginModelName: "image-model", PerCallBilling: true},
		},
	}
	insertTask(t, task)

	updated, err := PersistAsyncImageTaskBillingReservation(task)
	require.NoError(t, err)
	require.True(t, updated)

	var stored Task
	require.NoError(t, DB.First(&stored, task.ID).Error)
	assert.Equal(t, task.Quota, stored.Quota)
	assert.Equal(t, task.ChannelId, stored.ChannelId)
	assert.Equal(t, task.Group, stored.Group)
	assert.Equal(t, task.Properties, stored.Properties)
	assert.Equal(t, task.PrivateData.TokenId, stored.PrivateData.TokenId)
	assert.Equal(t, task.PrivateData.BillingSource, stored.PrivateData.BillingSource)
	assert.Equal(t, task.PrivateData.SubscriptionId, stored.PrivateData.SubscriptionId)
	require.NotNil(t, stored.PrivateData.BillingContext)
	assert.Equal(t, task.PrivateData.BillingContext, stored.PrivateData.BillingContext)

	task.Status = TaskStatusFailure
	require.NoError(t, task.Update())
	updated, err = PersistAsyncImageTaskBillingReservation(task)
	require.NoError(t, err)
	assert.False(t, updated)
}

func TestLinkAsyncImageSourceTask(t *testing.T) {
	truncateTables(t)
	wrapper := &Task{TaskID: "task_async_image_wrapper", UserId: 7, Platform: constant.TaskPlatformAsyncImage, Action: "image_generation", Status: TaskStatusInProgress}
	source := &Task{TaskID: "task_lecapi_source", UserId: 7, Platform: "lecapi", Action: "image_generation", Status: TaskStatusSubmitted}
	insertTask(t, wrapper)
	insertTask(t, source)

	linked, err := LinkAsyncImageSourceTask(wrapper.UserId, wrapper.TaskID, source.TaskID)
	require.NoError(t, err)
	require.True(t, linked)

	var stored Task
	require.NoError(t, DB.First(&stored, wrapper.ID).Error)
	assert.Equal(t, source.TaskID, stored.PrivateData.AsyncImageSourceTaskID)

	wrapper.Status = TaskStatusFailure
	require.NoError(t, wrapper.Update())
	linked, err = LinkAsyncImageSourceTask(wrapper.UserId, wrapper.TaskID, source.TaskID)
	require.NoError(t, err)
	assert.False(t, linked)
}
