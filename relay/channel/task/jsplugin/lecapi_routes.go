package jsplugin

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

// The plugin validates each body before quota reservation. All alternatives use
// the same descriptor and reservation, and only an explicit refusal can advance.
func (a *TaskAdaptor) submitLecAPIVideoPool(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	const maxAttempts = 3
	primary, ok := a.submit.Body.(map[string]any)
	if !ok || a.submit.ResponseType != "json" || a.submit.BodyType == "multipart" || c.Request.Method != http.MethodPost {
		return nil, fmt.Errorf("invalid LecAPI video pool submission")
	}
	type candidate struct {
		name string
		body map[string]any
	}
	candidates := make([]candidate, 0, 1+len(a.submit.LecAPIRouteAlternatives))
	names := make([]string, 0, cap(candidates))
	for _, value := range append([]map[string]any{primary}, a.submit.LecAPIRouteAlternatives...) {
		name, _ := value["model"].(string)
		if !strings.HasPrefix(name, "lec-") || len(name) > 100 || slices.Contains(names, name) {
			return nil, fmt.Errorf("invalid LecAPI video pool route")
		}
		names = append(names, name)
		candidates = append(candidates, candidate{name: name, body: value})
	}
	if len(candidates) > 1 && info.ChannelId > 0 {
		stats, err := model.LecAPIRouteStats(info.ChannelId, names)
		if err != nil {
			logger.LogWarn(c, "could not load LecAPI route statistics: "+err.Error())
		}
		// A small neutral prior prevents a single success from dominating.
		scores := make(map[string]float64, len(stats))
		for _, stat := range stats {
			scores[stat.Route] = float64(stat.SuccessCount+2) / float64(stat.RequestCount+4)
		}
		for _, name := range names {
			if _, known := scores[name]; !known {
				scores[name] = 0.5
			}
		}
		slices.SortStableFunc(candidates, func(l, r candidate) int { return cmp.Compare(scores[r.name], scores[l.name]) })
	}
	candidates = candidates[:min(len(candidates), maxAttempts)]
	for i, route := range candidates {
		requestBody := body
		if route.name != names[0] {
			value, err := inlineJSONFilePlaceholders(c, route.body)
			if err != nil {
				return nil, err
			}
			encoded, err := common.Marshal(value)
			if err != nil {
				return nil, err
			}
			requestBody = bytes.NewReader(encoded)
		}
		a.lecapiSubmitRoute = route.name
		resp, err := channel.DoTaskApiRequest(a, c, info, requestBody)
		if err != nil || resp == nil {
			return resp, err
		}
		mayRetry := lecapiDefinitiveRouteRejection(resp)
		if mayRetry || resp.StatusCode/100 == 5 {
			if err = model.RecordLecAPIRouteOutcome(info.ChannelId, route.name, false); err != nil {
				logger.LogWarn(c, common.LogText("could not record LecAPI route outcome: %s", err.Error()))
			}
		}
		if !mayRetry || i == len(candidates)-1 || c.Request.Context().Err() != nil {
			return resp, nil
		}
		resp.Body.Close()
	}
	return nil, fmt.Errorf("no LecAPI video pool route")
}

func lecapiDefinitiveRouteRejection(resp *http.Response) bool {
	if resp.Body == nil {
		return false
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusBadRequest, http.StatusNotFound, http.StatusGone, http.StatusServiceUnavailable:
	default:
		return false
	}
	const limit = 64 << 10
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	// Preserve the original response for the caller if it is not retried.
	resp.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(body), resp.Body), resp.Body}
	if err != nil || len(body) > limit {
		return false
	}
	var reply map[string]any
	if common.Unmarshal(body, &reply) != nil || reply["id"] != nil || reply["task_id"] != nil || reply["taskId"] != nil || reply["data"] != nil || reply["metadata"] != nil || reply["success"] == true {
		return false
	}
	failure, _ := reply["error"].(map[string]any)
	code, _ := failure["code"].(string)
	if code == "" {
		code, _ = failure["type"].(string)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return code == "rate_limit_exceeded"
	}
	return code == "model_not_found" || code == "model_unavailable" || code == "no_available_channel"
}
