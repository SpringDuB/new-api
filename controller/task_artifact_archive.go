package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/service"
	"golang.org/x/sync/singleflight"
)

var taskArtifactArchives singleflight.Group

func init() {
	service.ResolveTaskImageDelivery = resolveTaskImageDelivery
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
			if err != nil || ref == nil {
				logger.LogTaskArtifactFailure(ctx, task.TaskID, "storage", 0)
				failures = append(failures, errors.New("artifact archival failed"))
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

func resolveTaskImageDelivery(ctx context.Context, task *model.Task) (map[string]string, error) {
	artifacts, err := taskStorageArtifacts(task)
	if err != nil {
		return nil, errImageDelivery
	}
	urls := make(map[string]string)
	for _, artifact := range artifacts {
		if artifact.Type != "image" {
			continue
		}
		store, ref, err := storedTaskImage(ctx, task, artifact)
		if err != nil {
			return nil, errImageDelivery
		}
		signer, ok := store.(service.TaskArtifactURLProvider)
		if !ok {
			return nil, errImageDelivery
		}
		url, err := signer.SignedURL(ctx, task, ref, http.MethodGet)
		if err != nil || url == "" {
			return nil, errImageDelivery
		}
		urls[artifact.Key] = url
	}
	return urls, nil
}

func storedTaskImage(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact) (service.TaskArtifactStore, *service.StoredArtifactRef, error) {
	store, err := service.GetTaskArtifactReadStore(task, artifact.Key)
	if err == nil {
		ref, lookupErr := store.Resolve(task, artifact.Key)
		if lookupErr == nil && ref != nil {
			return store, ref, nil
		}
	}
	store = service.GetTaskArtifactStore()
	ref, err := archiveTaskArtifact(ctx, task, artifact, store)
	if err != nil || ref == nil {
		return nil, nil, errImageDelivery
	}
	return store, ref, nil
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
		bounded, cancel := context.WithTimeout(ctx, service.TaskArtifactTransferTimeout())
		defer cancel()
		release, err := service.AcquireTaskArtifactTransfer(bounded)
		if err != nil {
			return nil, err
		}
		defer release()
		var descriptor *relaychannel.TaskContentRequest
		for _, image := range task.PrivateData.ImageArtifacts {
			if image.Key != artifact.Key {
				continue
			}
			if strings.HasPrefix(image.SourceURL, "data:") {
				header, encoded, ok := strings.Cut(image.SourceURL, ",")
				mediaType, isDataURL := strings.CutPrefix(header, "data:")
				mediaType, isBase64 := strings.CutSuffix(mediaType, ";base64")
				if !ok || !isDataURL || !isBase64 || mediaType != image.MimeType {
					return nil, errors.New("invalid inline image artifact")
				}
				content := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
				return store.Persist(bounded, task, artifact, content)
			}
			descriptor = &relaychannel.TaskContentRequest{URL: image.SourceURL, Method: http.MethodGet, Credentialless: true}
			break
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
		return persistTaskMedia(bounded, task, artifact, store, descriptor)
	})
	if err != nil {
		return nil, err
	}
	ref, _ := value.(*service.StoredArtifactRef)
	return ref, nil
}
