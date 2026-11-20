package controller

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

var errImageDelivery = service.ErrTaskImageDelivery

const imageDeliveryManifestContextKey = "image_delivery_manifest"
const privateImageRelayContextKey = "private_image_relay"
const maxInlineImageBase64Bytes = 64 << 20

type imageDeliveryManifest struct {
	taskID  string
	archive model.TaskArtifactArchive
}

// Image adaptors still parse and account for their original responses. Their
// body and headers stay private until the gateway has uploaded every image.
type imageDeliveryWriter struct {
	gin.ResponseWriter
	header  http.Header
	status  int
	size    int
	capture imageLogResponseWriter
}

func (w *imageDeliveryWriter) Header() http.Header { return w.header }
func (w *imageDeliveryWriter) Status() int         { return w.status }
func (w *imageDeliveryWriter) Size() int           { return w.size }
func (w *imageDeliveryWriter) Written() bool       { return w.size >= 0 }
func (w *imageDeliveryWriter) WriteHeader(status int) {
	if !w.Written() && status > 0 {
		w.status = status
	}
}
func (w *imageDeliveryWriter) WriteHeaderNow() { w.size = max(0, w.size) }
func (w *imageDeliveryWriter) Flush()          { w.WriteHeaderNow() }
func (w *imageDeliveryWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *imageDeliveryWriter) Write(p []byte) (int, error) {
	w.WriteHeaderNow()
	w.capture.capture(p)
	w.size += len(p)
	return len(p), nil
}

func relayStoredImages(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	if !service.TaskArtifactOSSRequired() {
		return relay.ImageHelper(c, info, nil)
	}
	store := service.GetTaskArtifactStore()
	c.Set(privateImageRelayContextKey, true)
	if !store.Enabled() {
		return types.NewOpenAIError(errImageDelivery, types.ErrorCodeBadResponse, http.StatusServiceUnavailable, types.ErrOptionWithSkipRetry())
	}
	writer := &imageDeliveryWriter{ResponseWriter: c.Writer, header: make(http.Header), status: http.StatusOK, size: -1}
	writer.capture.ResponseWriter = writer
	c.Writer = writer
	var response storedImageResponse
	var format imageDeliveryFormat
	err := func() *types.NewAPIError {
		defer func() { c.Writer = writer.ResponseWriter }()
		return relay.ImageHelper(c, info, func() *types.NewAPIError {
			failed := writer.capture.omitted || writer.capture.streamError || writer.status != http.StatusOK
			if info.StreamStatus != nil {
				outcome := info.StreamStatus.OutcomeSnapshot()
				failed = failed || outcome.HasErrors || outcome.EndReason != relaycommon.StreamEndReasonDone && outcome.EndReason != relaycommon.StreamEndReasonEOF
			}
			if !failed {
				task := &model.Task{TaskID: model.GenerateTaskID(), Platform: "image", UserId: c.GetInt("id"), ChannelId: info.ChannelId}
				request, _ := info.Request.(*dto.ImageRequest)
				format = imageDeliveryFormat{stream: writer.capture.stream, jsonAsStream: c.GetBool(relaycommon.ImageJSONAsStreamKey), inlineBase64: request != nil && strings.EqualFold(strings.TrimSpace(request.ResponseFormat), "b64_json")}
				var deliveryErr error
				response, deliveryErr = prepareStoredImages(c, task, store, writer.capture.body.Bytes(), writer.capture.events, format)
				if deliveryErr == nil {
					return nil
				}
			}
			return types.NewOpenAIError(errImageDelivery, types.ErrorCodeBadResponse, http.StatusBadGateway, types.ErrOptionWithSkipRetry())
		})
	}()
	if err != nil {
		return err
	}
	response.write(c, format.stream)
	return nil
}

type storedImageResponse struct {
	Created int64            `json:"created"`
	Data    []storedImage    `json:"data"`
	Usage   map[string]int64 `json:"usage,omitempty"`
}

type storedImage struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
}

type imageDeliveryFormat struct {
	stream       bool
	jsonAsStream bool
	inlineBase64 bool
}

func writeStoredImages(c *gin.Context, task *model.Task, store service.TaskArtifactStore, body []byte, events []common.RawMessage, format imageDeliveryFormat) error {
	response, err := prepareStoredImages(c, task, store, body, events, format)
	if err == nil {
		response.write(c, format.stream)
	}
	return err
}

func prepareStoredImages(c *gin.Context, task *model.Task, store service.TaskArtifactStore, body []byte, events []common.RawMessage, format imageDeliveryFormat) (storedImageResponse, error) {
	signer, ok := store.(service.TaskArtifactURLProvider)
	if !store.Enabled() || !ok && !format.inlineBase64 {
		return storedImageResponse{}, errImageDelivery
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), service.TaskArtifactTransferTimeout())
	defer cancel()
	release, err := service.AcquireTaskArtifactTransfer(ctx)
	if err != nil {
		return storedImageResponse{}, errImageDelivery
	}
	defer release()

	response := storedImageResponse{Created: time.Now().Unix()}
	archive := model.TaskArtifactArchive{Status: model.TaskArchiveCompleted, UpdatedAt: time.Now().Unix()}
	var images []gjson.Result
	if format.stream {
		for _, event := range events {
			value := gjson.ParseBytes(event)
			switch value.Get("type").String() {
			case "image_generation.completed", "image_edit.completed", "image_generation.partial_succeeded":
				images = append(images, value)
			}
		}
	} else {
		if !gjson.ValidBytes(body) || gjson.GetBytes(body, "error").Exists() && gjson.GetBytes(body, "error").Type != gjson.Null {
			return storedImageResponse{}, errImageDelivery
		}
		images = gjson.GetBytes(body, "data").Array()
		if created := gjson.GetBytes(body, "created").Int(); created > 0 {
			response.Created = created
		}
		for _, key := range []string{"input_tokens", "output_tokens", "total_tokens"} {
			if value := gjson.GetBytes(body, "usage."+key); value.Type == gjson.Number {
				if response.Usage == nil {
					response.Usage = make(map[string]int64)
				}
				response.Usage[key] = value.Int()
			}
		}
	}
	payloads := make([]storedImage, 0, len(images))
	var urlOnly, inlineOnly []int
	for _, image := range images {
		item := storedImage{URL: image.Get("url").String(), B64JSON: image.Get("b64_json").String()}
		if item.URL == "" && item.B64JSON == "" {
			continue
		}
		if !format.stream || format.jsonAsStream {
			if item.B64JSON == "" {
				urlOnly = append(urlOnly, len(payloads))
			} else if item.URL == "" {
				inlineOnly = append(inlineOnly, len(payloads))
			}
		}
		payloads = append(payloads, item)
	}
	// A compatible JSON provider may split the two representations of each
	// image into separate rows. Pair them in order, without collapsing real SSE events.
	pairs := min(len(urlOnly), len(inlineOnly))
	for i := range pairs {
		urlIndex, inlineIndex := urlOnly[i], inlineOnly[i]
		first, second := min(urlIndex, inlineIndex), max(urlIndex, inlineIndex)
		payloads[first] = storedImage{URL: payloads[urlIndex].URL, B64JSON: payloads[inlineIndex].B64JSON}
		payloads[second] = storedImage{}
	}
	if count := len(payloads) - pairs; count == 0 || count > 64 {
		return storedImageResponse{}, errImageDelivery
	}
	remainingBase64 := maxInlineImageBase64Bytes
	for _, image := range payloads {
		source := image.URL
		if encoded := image.B64JSON; encoded != "" {
			// The content type is detected from bytes, not the upstream claim.
			source = "data:application/octet-stream;base64," + encoded
		}
		if source == "" {
			continue
		}
		artifact := relaychannel.TaskArtifact{Key: fmt.Sprintf("delivery-image-%d", len(response.Data)+1), Type: "image"}
		descriptor := &relaychannel.TaskContentRequest{URL: source, Method: http.MethodGet, Credentialless: true}
		var result storedImage
		var ref *service.StoredArtifactRef
		var err error
		if format.inlineBase64 {
			if len(image.B64JSON) > remainingBase64 {
				return storedImageResponse{}, errImageDelivery
			}
			err = consumeGeneratedMedia(ctx, task, artifact, descriptor, func(artifact relaychannel.TaskArtifact, content io.Reader) error {
				limit := int64(remainingBase64 / 4 * 3)
				data, err := io.ReadAll(io.LimitReader(content, limit+1))
				if err != nil || int64(len(data)) > limit {
					return errImageDelivery
				}
				ref, err = store.Persist(ctx, task, artifact, bytes.NewReader(data))
				if err == nil && ref != nil {
					result.B64JSON = image.B64JSON
					if result.B64JSON == "" {
						result.B64JSON = base64.StdEncoding.EncodeToString(data)
					}
					remainingBase64 -= len(result.B64JSON)
				}
				return err
			})
		} else {
			ref, err = persistTaskMedia(ctx, task, artifact, store, descriptor)
		}
		if err != nil || ref == nil {
			logger.LogTaskArtifactFailure(ctx, task.TaskID, "image_delivery", 0)
			return storedImageResponse{}, errImageDelivery
		}
		if !format.inlineBase64 {
			result.URL, err = signer.SignedURL(ctx, task, ref, http.MethodGet)
			if err != nil || result.URL == "" {
				return storedImageResponse{}, errImageDelivery
			}
		}
		response.Data = append(response.Data, result)
		artifact.MimeType = ref.MimeType
		archive.Artifacts = append(archive.Artifacts, taskArchivedArtifact(artifact, ref))
	}
	if task.Platform == "image" || task.PrivateData.ResultDiscarded {
		c.Set(imageDeliveryManifestContextKey, imageDeliveryManifest{taskID: task.TaskID, archive: archive})
	}
	return response, nil
}

func (response storedImageResponse) write(c *gin.Context, stream bool) {
	// Commit only after all objects are available. Never copy upstream headers,
	// revised prompts, vendor metadata, partial previews or stream error bodies.
	c.Header("Cache-Control", "private, no-store")
	if !stream {
		c.JSON(http.StatusOK, response)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("X-Accel-Buffering", "no")
	eventType := "image_generation.completed"
	if strings.HasSuffix(c.Request.URL.Path, "/edits") {
		eventType = "image_edit.completed"
	}
	for _, image := range response.Data {
		encoded, _ := common.Marshal(struct {
			Type      string `json:"type"`
			CreatedAt int64  `json:"created_at"`
			storedImage
		}{Type: eventType, CreatedAt: response.Created, storedImage: image})
		if err := helper.ResponseChunkData(c, dto.ResponsesStreamResponse{Type: eventType}, string(encoded)); err != nil {
			return // The client disconnected after delivery was committed.
		}
	}
	helper.Done(c)
}

func persistTaskMedia(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact, store service.TaskArtifactStore, descriptor *relaychannel.TaskContentRequest) (*service.StoredArtifactRef, error) {
	var ref *service.StoredArtifactRef
	err := consumeGeneratedMedia(ctx, task, artifact, descriptor, func(artifact relaychannel.TaskArtifact, content io.Reader) error {
		var err error
		ref, err = store.Persist(ctx, task, artifact, content)
		return err
	})
	return ref, err
}

func consumeGeneratedMedia(ctx context.Context, task *model.Task, artifact relaychannel.TaskArtifact, descriptor *relaychannel.TaskContentRequest, consume func(relaychannel.TaskArtifact, io.Reader) error) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://task-delivery.invalid/", nil)
	if err != nil {
		return err
	}
	c := &gin.Context{Request: request}
	return consumeTaskMedia(c, task, descriptor, func(content io.Reader, response *http.Response) error {
		if artifact.Type == "image" {
			buffered := bufio.NewReader(content)
			sample, err := buffered.Peek(512)
			if err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			artifact.MimeType = http.DetectContentType(sample)
			if len(sample) >= 12 && string(sample[4:8]) == "ftyp" && (string(sample[8:12]) == "avif" || string(sample[8:12]) == "avis") {
				artifact.MimeType = "image/avif"
			}
			if !strings.HasPrefix(artifact.MimeType, "image/") {
				return errImageDelivery
			}
			content = buffered
		} else {
			var err error
			content, err = validatedTaskVideoContent(content, response)
			if err != nil {
				return err
			}
			if artifact.MimeType == "" {
				artifact.MimeType = "video/mp4"
				if response != nil {
					mediaType, _, _ := strings.Cut(response.Header.Get("Content-Type"), ";")
					switch mediaType {
					case "video/webm", "video/quicktime", "video/mpeg":
						artifact.MimeType = mediaType
					}
				}
			}
		}
		return consume(artifact, content)
	})
}
