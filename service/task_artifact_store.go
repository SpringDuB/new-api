package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// StoredArtifactRef identifies a private object without persisting signed URLs.
type StoredArtifactRef struct {
	Backend     string
	Endpoint    string
	Region      string
	Bucket      string
	Prefix      string
	ObjectKey   string
	MimeType    string
	Size        int64
	ArtifactKey string
}

// TaskArtifactStore is the persistence boundary for generated artifact bytes.
// types.TaskArtifact is re-exported by relay/channel as channel.TaskArtifact.
type TaskArtifactStore interface {
	Enabled() bool
	Resolve(task *model.Task, artifactKey string) (*StoredArtifactRef, error)
	Persist(ctx context.Context, task *model.Task, artifact types.TaskArtifact, content io.Reader) (*StoredArtifactRef, error)
	Serve(c *gin.Context, task *model.Task, ref *StoredArtifactRef) error
}

var ErrTaskArtifactStoreDisabled = errors.New("task artifact store is disabled")

type disabledArtifactStore struct{}

func (disabledArtifactStore) Enabled() bool {
	return false
}

func (disabledArtifactStore) Resolve(*model.Task, string) (*StoredArtifactRef, error) {
	return nil, nil
}

func (disabledArtifactStore) Persist(context.Context, *model.Task, types.TaskArtifact, io.Reader) (*StoredArtifactRef, error) {
	return nil, ErrTaskArtifactStoreDisabled
}

func (disabledArtifactStore) Serve(*gin.Context, *model.Task, *StoredArtifactRef) error {
	return ErrTaskArtifactStoreDisabled
}

var taskArtifactStore TaskArtifactStore = &disabledArtifactStore{}

var taskArtifactStoreMutex sync.Mutex
var taskArtifactStoreConfig system_setting.TaskArtifactStoreConfig

// The host registers its media downloader without a service -> controller
// dependency. Archival runs after durable completion and never affects quota.
var ArchiveTaskArtifacts func(context.Context, *model.Task, TaskArtifactStore) error

var taskArtifactArchiveSlots = make(chan struct{}, 2)
var taskArtifactArchiveQueue = make(chan struct{}, 8)
var taskArtifactArchiveJobs sync.Map

func ScheduleTaskArtifactArchive(task *model.Task) {
	if task == nil || task.ID <= 0 || task.Status != model.TaskStatusSuccess || task.PrivateData.ArtifactArchive != nil || ArchiveTaskArtifacts == nil {
		return
	}
	store := GetTaskArtifactStore()
	if !store.Enabled() {
		return
	}
	if _, loaded := taskArtifactArchiveJobs.LoadOrStore(task.ID, struct{}{}); loaded {
		task.PrivateData.ArtifactArchive = &model.TaskArtifactArchive{Status: model.TaskArchivePending, UpdatedAt: time.Now().Unix()}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	archive := model.TaskArtifactArchive{Status: model.TaskArchivePending, UpdatedAt: time.Now().Unix()}
	claimed, err := model.UpdateTaskArtifactArchive(ctx, task, archive)
	cancel()
	if err != nil || !claimed {
		taskArtifactArchiveJobs.Delete(task.ID)
		if err != nil {
			logger.LogTaskArtifactFailure(context.Background(), task.TaskID, "archive_state", 0)
		}
		return
	}
	task.PrivateData.ArtifactArchive = &archive
	select {
	case taskArtifactArchiveQueue <- struct{}{}:
	default:
		archive.Status = model.TaskArchiveFailed
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, _ = model.UpdateTaskArtifactArchive(ctx, task, archive)
		cancel()
		taskArtifactArchiveJobs.Delete(task.ID)
		logger.LogTaskArtifactFailure(context.Background(), task.TaskID, "archive_queue", 0)
		return
	}
	// Own transient media deliberately omitted from SQL. The response
	// observer can continue modifying its task after enqueueing this copy.
	snapshot := *task
	archiveSnapshot := archive
	snapshot.PrivateData.ArtifactArchive = &archiveSnapshot
	snapshot.Data = bytes.Clone(task.Data)
	snapshot.PrivateData.ImageArtifacts = append([]model.TaskImageArtifact(nil), task.PrivateData.ImageArtifacts...)
	archiveTask := ArchiveTaskArtifacts
	gopool.Go(func() {
		defer func() {
			<-taskArtifactArchiveQueue
			taskArtifactArchiveJobs.Delete(snapshot.ID)
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		status := model.TaskArchiveFailed
		defer func() {
			if recover() != nil {
				status = model.TaskArchiveFailed
				logger.LogTaskArtifactFailure(context.Background(), snapshot.TaskID, "archive_worker", 0)
			}
			writeCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
			defer stop()
			finalArchive := *snapshot.PrivateData.ArtifactArchive
			finalArchive.Status, finalArchive.UpdatedAt = status, time.Now().Unix()
			if _, err := model.UpdateTaskArtifactArchive(writeCtx, &snapshot, finalArchive); err != nil {
				logger.LogTaskArtifactFailure(writeCtx, snapshot.TaskID, "archive_state", 0)
			}
		}()
		select {
		case taskArtifactArchiveSlots <- struct{}{}:
			defer func() { <-taskArtifactArchiveSlots }()
		case <-ctx.Done():
			return
		}
		if _, err := model.UpdateTaskArtifactArchive(ctx, &snapshot, model.TaskArtifactArchive{Status: model.TaskArchiveRunning, UpdatedAt: time.Now().Unix()}); err != nil {
			return
		}
		if err := archiveTask(ctx, &snapshot, store); err == nil {
			status = model.TaskArchiveCompleted
		}
	})
}

// GetTaskArtifactReadStore uses current credentials with the immutable storage
// identity recorded during archival. Disabling new writes leaves reads intact.
func GetTaskArtifactReadStore(task *model.Task, artifactKey string) (TaskArtifactStore, error) {
	config := system_setting.LoadTaskArtifactStoreConfig()
	var stored *model.TaskArchivedArtifact
	if task != nil && task.PrivateData.ArtifactArchive != nil && len(task.PrivateData.ArtifactArchive.Artifacts) > 0 {
		for i := range task.PrivateData.ArtifactArchive.Artifacts {
			artifact := &task.PrivateData.ArtifactArchive.Artifacts[i]
			if artifact.Key == artifactKey {
				stored = artifact
				break
			}
		}
		if stored == nil || stored.Backend != "oss" {
			return nil, errors.New("stored artifact is unavailable")
		}
		config.OSSEndpoint, config.OSSRegion = stored.Endpoint, stored.Region
		config.OSSBucket, config.OSSPrefix = stored.Bucket, stored.Prefix
	}
	config.Mode = system_setting.TaskArtifactStoreModeOSS
	if err := system_setting.ValidateTaskArtifactStoreConfig(config); err != nil {
		return nil, errors.New("artifact read configuration is unavailable")
	}
	store, err := newOSSArtifactStore(config, nil)
	if err != nil {
		return nil, err
	}
	reader := store.(*ossArtifactStore)
	key, err := reader.objectKey(task, artifactKey)
	if err != nil || stored != nil && stored.ObjectKey != key {
		return nil, errors.New("invalid stored artifact ownership")
	}
	reader.readOnly = true
	return reader, nil
}

// Interrupted workers cannot leave the dashboard displaying progress forever.
func TaskArtifactArchiveStatus(task *model.Task) string {
	archive := task.PrivateData.ArtifactArchive
	if archive == nil {
		return ""
	}
	if (archive.Status == model.TaskArchivePending || archive.Status == model.TaskArchiveRunning) && time.Now().Unix()-archive.UpdatedAt > 6*60 {
		return model.TaskArchiveFailed
	}
	return archive.Status
}

type TaskArtifactURLProvider interface {
	SignedURL(context.Context, *model.Task, *StoredArtifactRef, string) (string, error)
}

func init() {
	_ = system_setting.LoadTaskArtifactStoreConfig()
}

// GetTaskArtifactStore applies the current administrator storage configuration.
func GetTaskArtifactStore() TaskArtifactStore {
	config := system_setting.LoadTaskArtifactStoreConfig()
	taskArtifactStoreMutex.Lock()
	defer taskArtifactStoreMutex.Unlock()
	if config == taskArtifactStoreConfig {
		return taskArtifactStore
	}
	taskArtifactStoreConfig = config
	taskArtifactStore = &disabledArtifactStore{}
	if config.Mode == system_setting.TaskArtifactStoreModeOSS {
		if store, err := newOSSArtifactStore(config, nil); err == nil {
			taskArtifactStore = store
		}
	}
	return taskArtifactStore
}
