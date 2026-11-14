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
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

var taskVideoDeliveries singleflight.Group

func init() {
	service.ResolveTaskVideoDelivery = resolveTaskVideoDelivery
}

// Resolve delivery independently of the vendor's JSON envelope. The original
// snapshot remains the source for driver refreshes and background polling.
func resolveTaskVideoDelivery(ctx context.Context, task *model.Task) (map[string]service.TaskVideoDelivery, error) {
	deliveries := make(map[string]service.TaskVideoDelivery)
	store := service.GetTaskArtifactStore()
	if task == nil || task.Status != model.TaskStatusSuccess || !store.Enabled() {
		return deliveries, nil
	}
	artifacts, err := projectTaskArtifacts(task)
	if err != nil {
		return nil, service.ErrTaskVideoDelivery
	}
	if legacyVideoAvailable(task) {
		artifacts = []relaychannel.TaskArtifact{{Key: "video", Type: "video", MimeType: "video/mp4"}}
	}
	primaryKey := ""
	for _, artifact := range artifacts {
		if artifact.Type != "video" {
			continue
		}
		if artifact.Key == "video" {
			primaryKey = artifact.Key
			break
		}
		if primaryKey == "" || artifact.Key < primaryKey {
			primaryKey = artifact.Key
		}
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
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
	stored.Direct = delivery.Ref == nil
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
	if ref == nil {
		release, err := service.AcquireTaskArtifactTransfer(ctx)
		if err != nil {
			return delivery, err
		}
		defer release()
	}
	descriptor := &relaychannel.TaskContentRequest{URL: task.GetResultURL(), Method: http.MethodGet, Credentialless: true}
	var err error
	if taskHasPluginExecution(task) {
		var adaptor relaychannel.TaskAdaptor
		adaptor, err = initTaskArtifactAdaptor(task)
		if err == nil {
			if provider, ok := adaptor.(relaychannel.TaskContentRequestProvider); ok {
				descriptor, err = provider.BuildContentRequest(ctx, task, artifact.Key, relaychannel.TaskArtifactClientRequest{Method: http.MethodGet})
			} else {
				err = errTaskArtifactPluginUnavailable
			}
		}
	}
	if descriptor != nil && descriptor.URL != "" {
		delivery.SourceURLs = []string{descriptor.URL}
	}
	if err == nil && descriptor != nil && ref == nil {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://task-delivery.invalid/", nil)
		if requestErr != nil {
			return delivery, requestErr
		}
		c := &gin.Context{Request: request}
		// Probe with GET rather than HEAD: signed download endpoints frequently
		// implement only GET. Closing a direct reply avoids downloading it twice.
		if len(descriptor.ContentPath) == 0 && (strings.HasPrefix(descriptor.URL, "http://") || strings.HasPrefix(descriptor.URL, "https://")) {
			anonymous := &relaychannel.TaskContentRequest{URL: descriptor.URL, Method: http.MethodGet, Credentialless: true}
			accepted := false
			err = consumeTaskMedia(c, task, anonymous, func(content io.Reader, response *http.Response) error {
				content, err = validatedTaskVideoContent(content, response)
				if err != nil {
					return err
				}
				accepted = true
				if response.Request.Response == nil {
					delivery.URL = descriptor.URL
					return nil
				}
				if lookupErr != nil {
					return lookupErr
				}
				reader = store
				ref, err = store.Persist(ctx, task, artifact, content)
				return err
			})
			if err == nil && delivery.URL != "" {
				return delivery, nil
			}
			if err != nil && (accepted || descriptor.Credentialless) && ref == nil {
				return delivery, err
			}
		}
		if ref == nil {
			if lookupErr != nil {
				return delivery, lookupErr
			}
			err = consumeTaskMedia(c, task, descriptor, func(content io.Reader, response *http.Response) error {
				content, err = validatedTaskVideoContent(content, response)
				if err != nil {
					return err
				}
				reader = store
				ref, err = store.Persist(ctx, task, artifact, content)
				return err
			})
		}
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
