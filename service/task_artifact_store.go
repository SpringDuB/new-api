package service

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// StoredArtifactRef identifies a private object without persisting signed URLs.
type StoredArtifactRef struct {
	Backend     string
	Bucket      string
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
var ArchiveTaskArtifacts func(context.Context, *model.Task)

// StoredTaskArtifactURLs supplies current OSS links to authorized presenters.
var StoredTaskArtifactURLs func(*model.Task) map[string]string
var taskArtifactArchiveSlots = make(chan struct{}, 2)

func ScheduleTaskArtifactArchive(task *model.Task) {
	if task == nil || task.Status != model.TaskStatusSuccess || ArchiveTaskArtifacts == nil || !GetTaskArtifactStore().Enabled() {
		return
	}
	taskID := task.ID
	gopool.Go(func() {
		taskArtifactArchiveSlots <- struct{}{}
		defer func() { <-taskArtifactArchiveSlots }()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		var stored model.Task
		if model.DB.WithContext(ctx).First(&stored, taskID).Error == nil {
			ArchiveTaskArtifacts(ctx, &stored)
		}
	})
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
