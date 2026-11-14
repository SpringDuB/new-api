package controller

import (
	"context"
	"errors"
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
	service.ArchiveTaskArtifacts = func(ctx context.Context, task *model.Task, store service.TaskArtifactStore) error {
		artifacts, err := taskStorageArtifacts(task)
		if err != nil {
			return err
		}
		var failures []error
		archive := *task.PrivateData.ArtifactArchive
		archive.Artifacts = make([]model.TaskArchivedArtifact, len(artifacts))
		for i, artifact := range artifacts {
			archive.Artifacts[i] = taskArchivedArtifact(artifact, nil)
		}
		defer func() { task.PrivateData.ArtifactArchive = &archive }()
		for i, artifact := range artifacts {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if artifact.Type != "video" && artifact.Type != "image" {
				continue
			}
			var ref *service.StoredArtifactRef
			if artifact.Type == "video" {
				delivery, deliveryErr := taskVideoDelivery(ctx, task, artifact, store)
				ref, err = delivery.Ref, deliveryErr
			} else {
				ref, err = archiveTaskArtifact(ctx, task, artifact, store)
			}
			if err != nil || ref == nil && artifact.Type != "video" {
				logger.LogTaskArtifactFailure(ctx, task.TaskID, "storage", 0)
				failures = append(failures, errors.New("artifact archival failed"))
				continue
			}
			if ref == nil {
				archive.Artifacts[i].Direct = true
				continue
			}
			archive.Artifacts[i] = taskArchivedArtifact(artifact, ref)
			for i := range task.PrivateData.ImageArtifacts {
				if task.PrivateData.ImageArtifacts[i].Key == artifact.Key {
					task.PrivateData.ImageArtifacts[i].Stored = true
				}
			}
		}
		return errors.Join(failures...)
	}
}

func taskArchivedArtifact(artifact relaychannel.TaskArtifact, ref *service.StoredArtifactRef) model.TaskArchivedArtifact {
	stored := model.TaskArchivedArtifact{Key: artifact.Key, Type: artifact.Type, MimeType: artifact.MimeType}
	if ref != nil {
		stored.Backend, stored.Endpoint, stored.Region = ref.Backend, ref.Endpoint, ref.Region
		stored.Bucket, stored.Prefix, stored.ObjectKey = ref.Bucket, ref.Prefix, ref.ObjectKey
		stored.Size = ref.Size
	}
	return stored
}

// Legacy archives have objects but no manifest. While their original plugin
// is available, verify those objects and backfill without fetching media again.
func archivedTaskArtifacts(ctx context.Context, task *model.Task) ([]relaychannel.TaskArtifact, error) {
	archive := *task.PrivateData.ArtifactArchive
	if len(archive.Artifacts) == 0 {
		artifacts, err := taskStorageArtifacts(task)
		if err != nil {
			return nil, err
		}
		archive.Artifacts = make([]model.TaskArchivedArtifact, 0, len(artifacts))
		for _, artifact := range artifacts {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			var ref *service.StoredArtifactRef
			if artifact.Type == "video" || artifact.Type == "image" {
				store, err := service.GetTaskArtifactReadStore(task, artifact.Key)
				if err != nil {
					return nil, err
				}
				ref, err = store.Resolve(task, artifact.Key)
				if err != nil || ref == nil {
					return nil, errors.New("archived artifact is unavailable")
				}
			}
			archive.Artifacts = append(archive.Artifacts, taskArchivedArtifact(artifact, ref))
		}
		if len(archive.Artifacts) > 0 {
			if _, err := model.UpdateTaskArtifactArchive(ctx, task, archive); err != nil {
				return nil, err
			}
			task.PrivateData.ArtifactArchive = &archive
		}
	}
	artifacts := make([]relaychannel.TaskArtifact, 0, len(archive.Artifacts))
	for _, artifact := range archive.Artifacts {
		artifacts = append(artifacts, relaychannel.TaskArtifact{Key: artifact.Key, Type: artifact.Type, MimeType: artifact.MimeType})
	}
	return validateProjectedTaskArtifacts(artifacts)
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
	if task != nil && len(task.PrivateData.ImageArtifacts) > 0 {
		artifacts := make([]relaychannel.TaskArtifact, 0, len(task.PrivateData.ImageArtifacts))
		for _, image := range task.PrivateData.ImageArtifacts {
			artifacts = append(artifacts, imageLogArtifact(image))
		}
		return validateProjectedTaskArtifacts(artifacts)
	}
	return projectTaskArtifacts(task)
}

func archiveTaskArtifact(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact, store service.TaskArtifactStore) (*service.StoredArtifactRef, error) {
	identity := fmt.Sprintf("%d/%d/%s/%s", task.UserId, task.ID, task.TaskID, artifact.Key)
	value, err, _ := taskArtifactArchives.Do(identity, func() (any, error) {
		if !store.Enabled() {
			return nil, service.ErrTaskArtifactStoreDisabled
		}
		if ref, err := store.Resolve(task, artifact.Key); err != nil || ref != nil {
			return ref, err
		}
		bounded, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		release, err := service.AcquireTaskArtifactTransfer(bounded)
		if err != nil {
			return nil, err
		}
		defer release()
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
		err = consumeTaskMedia(c, task, descriptor, func(content io.Reader, _ *http.Response) error {
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
