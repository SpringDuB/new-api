package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

var taskArtifactArchives singleflight.Group

func init() {
	service.StoredTaskArtifactURLs = func(task *model.Task) map[string]string {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		urls := map[string]string{}
		store := service.GetTaskArtifactStore()
		provider, ok := store.(service.TaskArtifactURLProvider)
		if !ok {
			return urls
		}
		artifacts, err := taskStorageArtifacts(task)
		if err != nil {
			return urls
		}
		for _, artifact := range artifacts {
			if ctx.Err() != nil {
				break
			}
			ref, err := store.Resolve(task, artifact.Key)
			if err == nil && ref == nil && (artifact.Type == "video" || artifact.Type == "image") {
				ref, err = archiveTaskArtifact(ctx, task, artifact)
				if err != nil {
					logger.LogTaskArtifactFailure(ctx, task.TaskID, "storage", 0)
				}
			}
			if err != nil || ref == nil {
				continue
			}
			url, err := provider.SignedURL(context.Background(), task, ref, http.MethodGet)
			if err == nil {
				urls[artifact.Key] = url
			}
		}
		return urls
	}
	service.ArchiveTaskArtifacts = func(ctx context.Context, task *model.Task) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		artifacts, err := taskStorageArtifacts(task)
		if err != nil {
			return
		}
		for _, artifact := range artifacts {
			if ctx.Err() != nil {
				return
			}
			if artifact.Type != "video" && artifact.Type != "image" {
				continue
			}
			if _, err := archiveTaskArtifact(ctx, task, artifact); err != nil {
				logger.LogTaskArtifactFailure(ctx, task.TaskID, "storage", 0)
			}
		}
	}
}

// A synchronous response still holds its result in memory even when the
// submission deliberately omitted it from the database. Only this storage
// projection can read that transient result; retrieval keeps its usual rules.
func taskStorageArtifacts(task *model.Task) ([]relaychannel.TaskArtifact, error) {
	if task != nil && task.PrivateData.ResultDiscarded && len(task.Data) > 0 {
		transient := *task
		transient.PrivateData.ResultDiscarded = false
		task = &transient
	}
	return projectTaskArtifacts(task)
}

func archiveTaskArtifact(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact) (*service.StoredArtifactRef, error) {
	identity := fmt.Sprintf("%d/%d/%s/%s", task.UserId, task.ID, task.TaskID, artifact.Key)
	value, err, _ := taskArtifactArchives.Do(identity, func() (any, error) {
		store := service.GetTaskArtifactStore()
		if !store.Enabled() {
			return nil, service.ErrTaskArtifactStoreDisabled
		}
		if ref, err := store.Resolve(task, artifact.Key); err != nil || ref != nil {
			return ref, err
		}
		bounded, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		var descriptor *relaychannel.TaskContentRequest
		for _, image := range task.PrivateData.ImageArtifacts {
			if image.Key == artifact.Key {
				descriptor = &relaychannel.TaskContentRequest{URL: image.SourceURL, Method: http.MethodGet, Credentialless: true}
				break
			}
		}
		if descriptor == nil {
			adaptor, err := initTaskArtifactAdaptor(task)
			if err != nil {
				return nil, err
			}
			provider, ok := adaptor.(relaychannel.TaskContentRequestProvider)
			if !ok {
				return nil, errTaskArtifactPluginUnavailable
			}
			descriptor, err = provider.BuildContentRequest(bounded, task, artifact.Key, relaychannel.TaskArtifactClientRequest{Method: http.MethodGet})
			if err != nil {
				return nil, err
			}
		}
		request, err := http.NewRequestWithContext(bounded, http.MethodGet, "http://task-archive.invalid/", nil)
		if err != nil {
			return nil, err
		}
		c := &gin.Context{Request: request}
		var ref *service.StoredArtifactRef
		err = consumeTaskMedia(c, task, descriptor, func(content io.Reader) error {
			var persistErr error
			ref, persistErr = store.Persist(bounded, task, artifact, content)
			return persistErr
		})
		return ref, err
	})
	if err != nil {
		return nil, err
	}
	ref, _ := value.(*service.StoredArtifactRef)
	return ref, nil
}
