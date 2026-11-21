package controller

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/service"
	"golang.org/x/sync/singleflight"
)

var taskVideoDeliveries singleflight.Group

func init() {
	service.ResolveTaskVideoDelivery = resolveTaskVideoDelivery
}

func videoArtifactsForTask(task *model.Task) ([]relaychannel.TaskArtifact, error) {
	artifacts, err := taskStorageArtifacts(task)
	if err != nil {
		return nil, err
	}
	if !taskHasPluginExecution(task) && (legacyVideoAvailable(task) || task.GetResultURL() != "" && len(artifacts) == 0 && !model.IsImageTaskAction(task.Action)) {
		return []relaychannel.TaskArtifact{{Key: "video", Type: "video", MimeType: "video/mp4"}}, nil
	}
	return artifacts, nil
}

func primaryVideoArtifact(artifacts []relaychannel.TaskArtifact) (relaychannel.TaskArtifact, bool) {
	var primary relaychannel.TaskArtifact
	found := false
	for _, artifact := range artifacts {
		if artifact.Type != "video" {
			continue
		}
		if artifact.Key == "video" {
			return artifact, true
		}
		if !found || artifact.Key < primary.Key {
			primary, found = artifact, true
		}
	}
	return primary, found
}

func taskVideoContentRequest(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact, clientRequest relaychannel.TaskArtifactClientRequest) (*relaychannel.TaskContentRequest, error) {
	descriptor := &relaychannel.TaskContentRequest{URL: task.GetResultURL(), Method: clientRequest.Method, Credentialless: true}
	if !taskHasPluginExecution(task) {
		return descriptor, nil
	}
	adaptor, err := initTaskArtifactAdaptor(task)
	if err != nil {
		return nil, err
	}
	provider, ok := adaptor.(relaychannel.TaskContentRequestProvider)
	if !ok {
		return nil, errTaskArtifactPluginUnavailable
	}
	descriptor, err = provider.BuildContentRequest(ctx, task, artifact.Key, clientRequest)
	if err != nil {
		return nil, err
	}
	if descriptor == nil {
		return nil, errors.New("video content request is unavailable")
	}
	return descriptor, nil
}

// Resolve delivery independently of the vendor's JSON envelope. The original
// snapshot remains the source for driver refreshes and background polling.
func resolveTaskVideoDelivery(ctx context.Context, task *model.Task) (map[string]service.TaskVideoDelivery, error) {
	deliveries := make(map[string]service.TaskVideoDelivery)
	store := service.GetTaskArtifactStore()
	if task == nil || task.Status != model.TaskStatusSuccess {
		return deliveries, nil
	}
	artifacts, err := videoArtifactsForTask(task)
	if err != nil {
		return nil, service.ErrTaskVideoDelivery
	}
	primary, hasPrimary := primaryVideoArtifact(artifacts)
	primaryKey := ""
	if hasPrimary {
		primaryKey = primary.Key
	}
	for _, artifact := range artifacts {
		if artifact.Type != "video" {
			continue
		}
		delivery, err := taskVideoDelivery(ctx, task, artifact, store)
		if err != nil {
			return nil, service.ErrTaskVideoDelivery
		}
		if artifact.Key == primaryKey && task.GetResultURL() != "" {
			delivery.SourceURLs = append(delivery.SourceURLs, task.GetResultURL())
		}
		deliveries[artifact.Key] = delivery
	}
	return deliveries, nil
}

func taskVideoDelivery(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact, store service.TaskArtifactStore) (service.TaskVideoDelivery, error) {
	ctx, cancel := context.WithTimeout(ctx, service.TaskArtifactTransferTimeout())
	defer cancel()
	// The shared download owns its snapshot; a caller may leave while another
	// request still waits for the same artifact.
	snapshot := *task
	snapshot.Data = bytes.Clone(task.Data)
	if task.PrivateData.ArtifactArchive != nil {
		archive := *task.PrivateData.ArtifactArchive
		archive.Artifacts = slices.Clone(archive.Artifacts)
		snapshot.PrivateData.ArtifactArchive = &archive
	}
	identity := fmt.Sprintf("%d/%d/%s/%s", task.UserId, task.ID, task.TaskID, artifact.Key)
	result := taskVideoDeliveries.DoChan(identity, func() (value any, err error) {
		defer func() {
			if recover() != nil {
				logger.LogTaskArtifactFailure(ctx, snapshot.TaskID, "delivery_worker", 0)
				value, err = nil, service.ErrTaskVideoDelivery
			}
		}()
		return fetchTaskVideoDelivery(ctx, &snapshot, artifact, store)
	})
	select {
	case <-ctx.Done():
		return service.TaskVideoDelivery{}, ctx.Err()
	case resolved := <-result:
		if resolved.Err != nil {
			return service.TaskVideoDelivery{}, resolved.Err
		}
		delivery := resolved.Val.(service.TaskVideoDelivery)
		delivery.SourceURLs = slices.Clone(delivery.SourceURLs)
		if err := recordTaskVideoDelivery(ctx, task, artifact, delivery); err != nil {
			return service.TaskVideoDelivery{}, err
		}
		return delivery, nil
	}
}

func recordTaskVideoDelivery(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact, delivery service.TaskVideoDelivery) error {
	if task.ID <= 0 {
		return nil
	}
	stored := taskArchivedArtifact(artifact, delivery.Ref)
	if previous := task.PrivateData.ArtifactArchive; previous != nil && previous.Status == model.TaskArchiveCompleted && slices.Contains(previous.Artifacts, stored) {
		return nil
	}
	artifacts, err := taskStorageArtifacts(task)
	if err != nil {
		return err
	}
	// Enabling OSS after completion must still archive the task's images;
	// recording this video's partial manifest would otherwise claim the job.
	if task.PrivateData.ArtifactArchive == nil && slices.ContainsFunc(artifacts, func(item relaychannel.TaskArtifact) bool { return item.Type == "image" }) {
		service.ScheduleTaskArtifactArchive(task)
	}
	archive := model.TaskArtifactArchive{Status: model.TaskArchiveFailed, UpdatedAt: time.Now().Unix()}
	if task.PrivateData.ArtifactArchive != nil {
		archive.Status = task.PrivateData.ArtifactArchive.Status
	}
	for _, expected := range artifacts {
		if expected.Key != artifact.Key {
			archive.Artifacts = append(archive.Artifacts, taskArchivedArtifact(expected, nil))
		}
	}
	archive.Artifacts = append(archive.Artifacts, stored)
	_, err = model.UpdateTaskArtifactArchive(ctx, task, archive)
	return err
}

func fetchTaskVideoDelivery(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact, store service.TaskArtifactStore) (service.TaskVideoDelivery, error) {
	delivery := service.TaskVideoDelivery{}
	reader := store
	if task.PrivateData.ArtifactArchive != nil {
		for _, archived := range task.PrivateData.ArtifactArchive.Artifacts {
			if archived.Key == artifact.Key && archived.Backend == "oss" {
				var err error
				reader, err = service.GetTaskArtifactReadStore(task, artifact.Key)
				if err != nil {
					return delivery, err
				}
				break
			}
		}
	}
	ref, lookupErr := reader.Resolve(task, artifact.Key)
	if lookupErr != nil {
		return delivery, lookupErr
	}
	if ref == nil {
		if !store.Enabled() {
			return delivery, service.ErrTaskArtifactStoreDisabled
		}
		release, err := service.AcquireTaskArtifactTransfer(ctx)
		if err != nil {
			return delivery, err
		}
		defer release()
	}
	descriptor, err := taskVideoContentRequest(ctx, task, artifact, relaychannel.TaskArtifactClientRequest{Method: http.MethodGet})
	if descriptor != nil && descriptor.URL != "" {
		delivery.SourceURLs = []string{descriptor.URL}
	}
	if err == nil && descriptor != nil && ref == nil {
		reader = store
		ref, err = persistTaskMedia(ctx, task, artifact, reader, descriptor)
	}
	if ref == nil {
		if err == nil {
			err = errors.New("video content is unavailable")
		}
		return delivery, err
	}
	provider, ok := reader.(service.TaskArtifactURLProvider)
	if !ok {
		return delivery, errors.New("video store cannot sign delivery URLs")
	}
	delivery.URL, err = provider.SignedURL(ctx, task, ref, http.MethodGet)
	delivery.Store, delivery.Ref = reader, ref
	return delivery, err
}

func validatedTaskVideoContent(content io.Reader, response *http.Response) (io.Reader, error) {
	if response != nil {
		mediaType := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Type")))
		if strings.HasPrefix(mediaType, "text/") || strings.Contains(mediaType, "json") || strings.Contains(mediaType, "xml") {
			return nil, errors.New("upstream did not return video content")
		}
	}
	buffered := bufio.NewReader(content)
	if _, err := buffered.Peek(1); err != nil {
		return nil, errors.New("upstream returned empty video content")
	}
	return buffered, nil
}
