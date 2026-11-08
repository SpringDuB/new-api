package jsplugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"

	"github.com/QuantumNous/new-api/common"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/abema/go-mp4"
	"github.com/gin-gonic/gin"
)

// Media references use the protected download client, never channel credentials.
// Cache measured facts only for this submission; URL contents may change later.
func mediaProbeContext(ctx context.Context, c *gin.Context) context.Context {
	const cacheKey = "task_plugin_media_prober"
	if existing, ok := c.Get(cacheKey); ok {
		return pluginruntime.WithMediaProber(ctx, existing.(pluginruntime.Fetcher))
	}
	measured := make(map[string]pluginruntime.FetchResponse)
	var totalBytes int64
	prober := pluginruntime.Fetcher(func(ctx context.Context, request pluginruntime.FetchRequest) (pluginruntime.FetchResponse, error) {
		key := request.URL + "\x00" + request.FileRef
		if cached, ok := measured[key]; ok {
			return cached, nil
		}
		limit := maxInlineFileBytes()
		var reader io.ReadSeeker
		if request.FileRef != "" {
			field, index, valid := pluginruntime.ParseFileReference(request.FileRef)
			form, err := common.ParseMultipartFormReusable(c)
			if err != nil {
				return pluginruntime.FetchResponse{}, fmt.Errorf("could not read uploaded reference video")
			}
			defer form.RemoveAll()
			if !valid || index >= len(form.File[field]) {
				return pluginruntime.FetchResponse{}, fmt.Errorf("unknown reference video file")
			}
			header := form.File[field][index]
			if header.Size <= 0 || header.Size > limit-totalBytes {
				return pluginruntime.FetchResponse{}, fmt.Errorf("reference videos exceed the download byte limit")
			}
			file, err := header.Open()
			if err != nil {
				return pluginruntime.FetchResponse{}, fmt.Errorf("could not open reference video")
			}
			defer file.Close()
			reader = file
			totalBytes += header.Size
		} else {
			if err := service.ValidateSSRFProtectedFetchURL(request.URL); err != nil {
				return pluginruntime.FetchResponse{}, fmt.Errorf("reference video URL is blocked")
			}
			client := service.GetSSRFProtectedHTTPClient()
			if client == nil {
				return pluginruntime.FetchResponse{}, fmt.Errorf("media download client is unavailable")
			}
			protected := *client
			protected.Jar = nil
			protected.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				if len(via) >= 5 || (via[len(via)-1].URL.Scheme == "https" && next.URL.Scheme != "https") {
					return fmt.Errorf("reference video redirect is blocked")
				}
				next.Header = make(http.Header)
				return service.ValidateSSRFProtectedFetchURL(next.URL.String())
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, request.URL, nil)
			if err != nil {
				return pluginruntime.FetchResponse{}, fmt.Errorf("invalid reference video URL")
			}
			resp, err := protected.Do(req)
			if err != nil {
				return pluginruntime.FetchResponse{}, fmt.Errorf("could not download reference video")
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return pluginruntime.FetchResponse{}, fmt.Errorf("reference video download returned HTTP %d", resp.StatusCode)
			}
			if resp.ContentLength > limit-totalBytes {
				return pluginruntime.FetchResponse{}, fmt.Errorf("reference videos exceed the download byte limit")
			}
			file, err := os.CreateTemp("", "new-api-media-*")
			if err != nil {
				return pluginruntime.FetchResponse{}, fmt.Errorf("could not open media temporary file")
			}
			defer os.Remove(file.Name())
			defer file.Close()
			n, err := io.Copy(file, io.LimitReader(resp.Body, limit-totalBytes+1))
			if err != nil || n > limit-totalBytes {
				return pluginruntime.FetchResponse{}, fmt.Errorf("reference video download is incomplete or exceeds the byte limit")
			}
			totalBytes += n
			reader = file
		}
		seconds, err := referenceVideoDuration(mediaReader{Context: ctx, ReadSeeker: reader})
		if err != nil {
			return pluginruntime.FetchResponse{}, err
		}
		body, err := common.Marshal(seconds)
		if err != nil {
			return pluginruntime.FetchResponse{}, err
		}
		result := pluginruntime.FetchResponse{Status: http.StatusOK, Body: body}
		measured[key] = result
		return result, nil
	})
	c.Set(cacheKey, prober)
	return pluginruntime.WithMediaProber(ctx, prober)
}

type mediaReader struct {
	context.Context
	io.ReadSeeker
}

func (r mediaReader) Read(p []byte) (int, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	return r.ReadSeeker.Read(p)
}

func (r mediaReader) Seek(offset int64, whence int) (int64, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	return r.ReadSeeker.Seek(offset, whence)
}

// Read video-track duration, so a longer audio track is not charged as video.
// MP4 and QuickTime/MOV share these boxes. Unsupported or missing durations
// fail closed before upstream submission, rather than trusting client metadata.
func referenceVideoDuration(reader io.ReadSeeker) (float64, error) {
	tracks, err := mp4.ExtractBoxes(reader, nil, []mp4.BoxPath{{mp4.BoxTypeMoov(), mp4.BoxTypeTrak()}})
	if err != nil {
		return 0, fmt.Errorf("could not parse reference video; use an MP4 or MOV with a finite duration")
	}
	var seconds float64
	for _, track := range tracks {
		boxes, err := mp4.ExtractBoxesWithPayload(reader, track, []mp4.BoxPath{{mp4.BoxTypeMdia(), mp4.BoxTypeHdlr()}, {mp4.BoxTypeMdia(), mp4.BoxTypeMdhd()}})
		if err != nil {
			return 0, fmt.Errorf("could not parse reference video track")
		}
		var duration *mp4.Mdhd
		video := false
		for _, box := range boxes {
			switch payload := box.Payload.(type) {
			case *mp4.Hdlr:
				video = bytes.Equal(payload.HandlerType[:], []byte("vide"))
			case *mp4.Mdhd:
				duration = payload
			}
		}
		if !video {
			continue
		}
		if duration == nil || duration.Timescale == 0 {
			return 0, fmt.Errorf("reference video has no measurable duration")
		}
		seconds = max(seconds, float64(duration.GetDuration())/float64(duration.Timescale))
	}
	if !math.IsInf(seconds, 0) && !math.IsNaN(seconds) && seconds > 0 && seconds <= relaycommon.MaxTaskDurationSeconds {
		return seconds, nil
	}
	return 0, fmt.Errorf("reference video duration must be positive and at most %d seconds; use MP4 or MOV", relaycommon.MaxTaskDurationSeconds)
}
