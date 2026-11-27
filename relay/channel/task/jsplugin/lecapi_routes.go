package jsplugin

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// Keep persistence bounded independently of the plugin's small state limit.
// A single reference clip can be 20 MiB before encoding.
const maxLecAPIRetryPayloadBytes = 32 << 20

type lecapiRetryPayload struct {
	Shared map[string]any            `json:"shared"`
	Routes map[string]map[string]any `json:"routes"`
}

type lecapiRouteCandidate struct {
	Route string         `json:"route"`
	Body  map[string]any `json:"body"`
}

type lecapiRouteCost struct {
	OutputSecond    float64 `json:"outputSecond"`
	ReferenceSecond float64 `json:"referenceSecond"`
	Durations       []int   `json:"durations"`
}

// FilterPricedTaskRoutes freezes the affordable set before either submission or
// delayed retry can see it. Compare known costs to the actual group-adjusted
// charge at each duration the route can report, including the reservation.
func (a *TaskAdaptor) FilterPricedTaskRoutes(info *relaycommon.RelayInfo) error {
	if a.plugin.Meta.Key != "lecapi" || a.submit == nil || a.submit.LecAPIRouteCosts == nil {
		return nil
	}
	snap := info.TieredBillingSnapshot
	if snap == nil || snap.QuotaPerUnit <= 0 || math.IsNaN(snap.QuotaPerUnit) || math.IsInf(snap.QuotaPerUnit, 0) {
		return errors.New("the video route pool requires a valid usage price")
	}
	primary, ok := a.submit.Body.(map[string]any)
	seconds, validSeconds := snap.UsageFacts["seconds"].(float64)
	referenceSeconds, _ := snap.UsageFacts["reference_seconds"].(float64)
	const maxSeedanceDuration = 30
	if !ok || !validSeconds || math.Trunc(seconds) != seconds || seconds < 1 || seconds > maxSeedanceDuration ||
		math.IsNaN(referenceSeconds) || math.IsInf(referenceSeconds, 0) || referenceSeconds < 0 {
		return errors.New("invalid video usage for the route cost check")
	}
	quotaPerUnit := decimal.NewFromFloat(snap.QuotaPerUnit)
	inputSeconds := decimal.NewFromFloat(referenceSeconds)
	charges := map[int]int{int(seconds): snap.EstimatedQuotaAfterGroup}
	candidates := append([]map[string]any{primary}, a.submit.LecAPIRouteAlternatives...)
	eligible := candidates[:0]
	for _, body := range candidates {
		name, _ := body["model"].(string)
		rate, ok := a.submit.LecAPIRouteCosts[name]
		if !ok || rate.OutputSecond <= 0 || rate.ReferenceSecond < 0 ||
			(rate.ReferenceSecond > 0 && referenceSeconds == 0) || !slices.Contains(rate.Durations, int(seconds)) {
			continue
		}
		outputQuotaPerSecond := decimal.NewFromFloat(rate.OutputSecond).Mul(quotaPerUnit)
		inputQuota := decimal.NewFromFloat(rate.ReferenceSecond).Mul(inputSeconds).Mul(quotaPerUnit)
		covered := true
		for _, duration := range rate.Durations {
			if duration < 1 || duration > maxSeedanceDuration {
				covered = false
				break
			}
			charge, resolved := charges[duration]
			if !resolved {
				durationTier := "short"
				if duration > 15 {
					durationTier = "long"
				}
				result, _, err := service.EvaluateTaskCompletionUsage(snap, map[string]any{"seconds": float64(duration), "duration_tier": durationTier})
				if err != nil {
					covered = false
					break
				}
				charge = result.ActualQuotaAfterGroup
				charges[duration] = charge
			}
			cost := outputQuotaPerSecond.Mul(decimal.NewFromInt(int64(duration))).Add(inputQuota)
			if cost.GreaterThan(decimal.NewFromInt(int64(charge))) {
				covered = false
				break
			}
		}
		if covered {
			eligible = append(eligible, body)
		}
	}
	if len(eligible) == 0 {
		return errors.New("no video route is available at the current price for this request")
	}
	a.submit.Body = eligible[0]
	a.submit.LecAPIRouteAlternatives = eligible[1:]
	return nil
}

// Synchronous images have no pollable task to recover. A timeout or an
// ambiguous reply ends submission; only a definitive refusal permits replay.
func (a *TaskAdaptor) submitLecAPIImagePool(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	if a.submit.ResponseType != "json" || a.submit.BodyType == "multipart" || c.Request.Method != http.MethodPost {
		return nil, errors.New("invalid LecAPI image pool submission")
	}
	resp, err := channel.DoTaskApiRequest(a, c, info, body)
	for _, alternative := range a.submit.LecAPIImageAlternatives[:min(2, len(a.submit.LecAPIImageAlternatives))] {
		if err != nil || resp == nil || c.Request.Context().Err() != nil || !lecapiDefinitiveRouteRejection(resp) {
			return resp, err
		}
		resp.Body.Close()
		value, err := inlineJSONFilePlaceholders(c, alternative)
		if err != nil {
			return nil, err
		}
		encoded, err := common.Marshal(value)
		if err != nil {
			return nil, err
		}
		resp, err = channel.DoTaskApiRequest(a, c, info, bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
	}
	return resp, err
}

// The plugin validates each body before quota reservation. All alternatives use
// the same descriptor and reservation, and only an explicit refusal can advance.
func (a *TaskAdaptor) submitLecAPIVideoPool(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	const maxAttempts = 3
	primary, ok := a.submit.Body.(map[string]any)
	if !ok || a.submit.ResponseType != "json" || a.submit.BodyType == "multipart" || c.Request.Method != http.MethodPost {
		return nil, fmt.Errorf("invalid LecAPI video pool submission")
	}
	candidates := make([]lecapiRouteCandidate, 0, 1+len(a.submit.LecAPIRouteAlternatives))
	names := make([]string, 0, cap(candidates))
	for _, value := range append([]map[string]any{primary}, a.submit.LecAPIRouteAlternatives...) {
		name, _ := value["model"].(string)
		if !strings.HasPrefix(name, "lec-") || len(name) > 100 || slices.Contains(names, name) {
			return nil, fmt.Errorf("invalid LecAPI video pool route")
		}
		names = append(names, name)
		candidates = append(candidates, lecapiRouteCandidate{Route: name, Body: value})
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
		slices.SortStableFunc(candidates, func(l, r lecapiRouteCandidate) int { return cmp.Compare(scores[r.Route], scores[l.Route]) })
	}
	candidates = candidates[:min(len(candidates), maxAttempts)]
	for i, route := range candidates {
		a.lecapiSubmitOutcomeRecorded = false
		requestBody := body
		if route.Route != names[0] {
			value, err := inlineJSONFilePlaceholders(c, route.Body)
			if err != nil {
				return nil, err
			}
			encoded, err := common.Marshal(value)
			if err != nil {
				return nil, err
			}
			requestBody = bytes.NewReader(encoded)
		}
		a.lecapiSubmitRoute = route.Route
		resp, err := channel.DoTaskApiRequest(a, c, info, requestBody)
		if err != nil || resp == nil {
			return resp, err
		}
		if a.plugin.IsFactory() && a.plugin.Meta.Key == "lecapi" && lecapiGatewayError(resp.StatusCode) {
			reply, poll := lecapiGatewaySubmission(resp, info.PublicTaskID)
			a.lecapiSubmitPoll = poll
			a.storeLecAPIRetryCandidates(c, a.submit.URL, candidates[i+1:])
			return reply, nil
		}
		mayRetry := lecapiDefinitiveRouteRejection(resp)
		if mayRetry {
			if err = model.RecordLecAPIRouteOutcome(info.ChannelId, route.Route, false); err != nil {
				logger.LogWarn(c, common.LogText("could not record LecAPI route outcome: %s", err.Error()))
			} else {
				a.lecapiSubmitOutcomeRecorded = true
			}
		}
		if !mayRetry {
			a.storeLecAPIRetryCandidates(c, a.submit.URL, candidates[i+1:])
			return resp, nil
		}
		if i == len(candidates)-1 || c.Request.Context().Err() != nil {
			return resp, nil
		}
		resp.Body.Close()
	}
	return nil, fmt.Errorf("no LecAPI video pool route")
}

func (a *TaskAdaptor) storeLecAPIRetryCandidates(c *gin.Context, url string, candidates []lecapiRouteCandidate) {
	a.lecapiRetryURL = ""
	a.lecapiRetryCandidates = nil
	a.lecapiRetryPayload = nil
	if len(candidates) == 0 {
		return
	}

	stored := make([]lecapiRouteCandidate, 0, len(candidates))
	materialized := make(map[string]map[string]any, len(candidates))
	for _, candidate := range candidates {
		body, err := inlineJSONFilePlaceholders(c, candidate.Body)
		if err != nil {
			logger.LogWarn(c, common.LogText("could not retain LecAPI route for delayed retry: %s", err.Error()))
			return
		}
		bodyObject, ok := body.(map[string]any)
		if !ok {
			return
		}
		materialized[candidate.Route] = bodyObject
		stored = append(stored, lecapiRouteCandidate{Route: candidate.Route, Body: map[string]any{"model": candidate.Route}})
	}
	shared := make(map[string]any)
	for name, value := range materialized[candidates[0].Route] {
		if name == "model" {
			continue
		}
		if !slices.ContainsFunc(candidates[1:], func(candidate lecapiRouteCandidate) bool {
			other, ok := materialized[candidate.Route][name]
			return !ok || !reflect.DeepEqual(other, value)
		}) {
			shared[name] = value
		}
	}
	for _, body := range materialized {
		for name := range shared {
			delete(body, name)
		}
	}
	encoded, err := common.Marshal(lecapiRetryPayload{Shared: shared, Routes: materialized})
	if err != nil || len(encoded) > maxLecAPIRetryPayloadBytes {
		logger.LogWarn(c, common.LogText("LecAPI route candidates exceed the delayed retry payload limit"))
		return
	}
	// Leave room for the task snapshot and the rest of the SQL statement. An
	// undersized MySQL packet must not make the whole accepted task unpersistable.
	const rowOverhead = 2 << 20
	if model.DB != nil && common.UsingMainDatabase(common.DatabaseTypeMySQL) {
		var packetLimit int
		if err := model.DB.WithContext(c.Request.Context()).Raw("SELECT @@session.max_allowed_packet").Scan(&packetLimit).Error; err != nil || len(encoded) > packetLimit-rowOverhead {
			logger.LogWarn(c, common.LogText("MySQL packet limit does not allow storing LecAPI route retry payload"))
			return
		}
	}
	a.lecapiRetryURL = url
	a.lecapiRetryCandidates = stored
	a.lecapiRetryPayload = encoded
}

func markLecAPIRetryAmbiguous(task *model.Task, result *relaycommon.TaskInfo, state map[string]any) (bool, error) {
	state["lecapiRetryAmbiguous"] = true
	encoded, err := encodeLecAPIRetryState(state)
	if err != nil {
		return false, err
	}
	task.PrivateData.PluginState = encoded
	task.PrivateData.LecAPIPoll.RetryUntil = time.Now().Add(model.LecAPIGatewayRecheckDelay).UnixMilli()
	task.Status = model.TaskStatusSubmitted
	task.Progress = taskcommon.ProgressSubmitted
	task.FailReason = ""
	result.Status = string(model.TaskStatusSubmitted)
	return true, nil
}

func withoutLecAPIRetryState(pluginState []byte) []byte {
	var state map[string]any
	if common.Unmarshal(pluginState, &state) != nil {
		return nil
	}
	delete(state, "lecapiRetryURL")
	delete(state, "lecapiRetryCandidates")
	delete(state, "lecapiRetryInFlight")
	encoded, err := common.Marshal(state)
	if err != nil {
		return nil
	}
	return encoded
}

func decodeLecAPIRetryState(task *model.Task) (map[string]any, string, string, []lecapiRouteCandidate, *lecapiRouteCandidate, error) {
	if task == nil || len(task.PrivateData.PluginState) == 0 {
		return nil, "", "", nil, nil, fmt.Errorf("LecAPI route retry state is missing")
	}
	var state map[string]any
	if err := common.Unmarshal(task.PrivateData.PluginState, &state); err != nil {
		return nil, "", "", nil, nil, fmt.Errorf("LecAPI route retry state is invalid")
	}
	currentRoute, _ := state["lecapiSubmitRoute"].(string)
	url, _ := state["lecapiRetryURL"].(string)
	candidates, err := decodeLecAPIRouteCandidates(state["lecapiRetryCandidates"])
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	var inFlight *lecapiRouteCandidate
	if value, exists := state["lecapiRetryInFlight"]; exists && value != nil {
		decoded, err := decodeLecAPIRouteCandidate(value)
		if err != nil {
			return nil, "", "", nil, nil, err
		}
		inFlight = &decoded
	}
	return state, currentRoute, url, candidates, inFlight, nil
}

func decodeLecAPIRouteCandidates(value any) ([]lecapiRouteCandidate, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := common.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("LecAPI route retry candidates are invalid")
	}
	var candidates []lecapiRouteCandidate
	if err = common.Unmarshal(encoded, &candidates); err != nil {
		return nil, fmt.Errorf("LecAPI route retry candidates are invalid")
	}
	if len(candidates) > 3 {
		return nil, fmt.Errorf("too many LecAPI route retry candidates")
	}
	for _, candidate := range candidates {
		if !validLecAPIRouteCandidate(candidate) {
			return nil, fmt.Errorf("LecAPI route retry candidate is invalid")
		}
	}
	return candidates, nil
}

func decodeLecAPIRouteCandidate(value any) (lecapiRouteCandidate, error) {
	encoded, err := common.Marshal(value)
	if err != nil {
		return lecapiRouteCandidate{}, fmt.Errorf("LecAPI route retry candidate is invalid")
	}
	var candidate lecapiRouteCandidate
	if err = common.Unmarshal(encoded, &candidate); err != nil || !validLecAPIRouteCandidate(candidate) {
		return lecapiRouteCandidate{}, fmt.Errorf("LecAPI route retry candidate is invalid")
	}
	return candidate, nil
}

func validLecAPIRouteCandidate(candidate lecapiRouteCandidate) bool {
	modelName, _ := candidate.Body["model"].(string)
	return strings.HasPrefix(candidate.Route, "lec-") && len(candidate.Route) <= 100 &&
		candidate.Body != nil && modelName == candidate.Route
}

func encodeLecAPIRetryState(state map[string]any) ([]byte, error) {
	encoded, err := common.Marshal(state)
	if err != nil || len(encoded) > maxTaskPluginPersistedJSONBytes {
		return nil, fmt.Errorf("LecAPI route retry state exceeds the task state limit")
	}
	return encoded, nil
}

// PrepareTaskFailureRetry claims the next persisted route before any request is
// sent. The polling service persists the status transition with CAS, preventing
// concurrent pollers from submitting the same fallback route.
func (a *TaskAdaptor) PrepareTaskFailureRetry(task *model.Task, recoverExpired bool) (bool, bool) {
	if a.plugin.Meta.Key != "lecapi" || !a.plugin.IsFactory() || a.info == nil || !a.info.HasChannelMeta() ||
		a.info.ChannelType == constant.ChannelTypeNewAPI || task == nil || task.PrivateData.Execution == nil ||
		task.PrivateData.Execution.TaskPlugin == nil || task.PrivateData.Execution.TaskPlugin.Key != "lecapi" {
		return false, false
	}
	state, currentRoute, url, candidates, inFlight, err := decodeLecAPIRetryState(task)
	if err != nil || currentRoute == "" || url == "" {
		return false, false
	}
	if inFlight != nil {
		if !recoverExpired {
			return false, true
		}
		poll := task.PrivateData.LecAPIPoll
		if poll.RetryUntil != 0 && !poll.RetryDispatched {
			return true, false // The expired owner never marked the request for dispatch.
		}
		// A dispatched request or a legacy claim may have been accepted.
		// Leave that candidate consumed even though no upstream ID survived.
		delete(state, "lecapiRetryInFlight")
		delete(state, "lecapiRetryAmbiguous")
	}
	if len(candidates) == 0 {
		return false, false
	}

	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.Route)
	}
	stats, err := model.LecAPIRouteStats(a.info.ChannelId, names)
	if err != nil {
		logger.LogWarn(context.Background(), common.LogText("could not load LecAPI route statistics for delayed retry: %s", err.Error()))
	}
	scores := make(map[string]float64, len(candidates))
	for _, stat := range stats {
		scores[stat.Route] = float64(stat.SuccessCount+2) / float64(stat.RequestCount+4)
	}
	for _, route := range names {
		if _, known := scores[route]; !known {
			scores[route] = 0.5
		}
	}
	slices.SortStableFunc(candidates, func(left, right lecapiRouteCandidate) int {
		return cmp.Compare(scores[right.Route], scores[left.Route])
	})
	state["lecapiRetryCandidates"] = candidates[1:]
	state["lecapiRetryInFlight"] = candidates[0]
	encoded, err := encodeLecAPIRetryState(state)
	if err != nil {
		return false, false
	}
	task.PrivateData.PluginState = encoded
	return true, false
}

// RetryTaskAfterFailure submits the claimed fallback route(s). Explicit route
// rejections can advance immediately. Gateway errors return to the poller for
// the scheduled recheck. Ambiguous attempts are skipped after their retry claim expires.
func (a *TaskAdaptor) RetryTaskAfterFailure(ctx context.Context, task *model.Task, result *relaycommon.TaskInfo) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, currentRoute, retryURL, candidates, inFlight, err := decodeLecAPIRetryState(task)
	if err != nil || inFlight == nil || currentRoute == "" {
		return false, err
	}
	if a.info == nil || !a.info.HasChannelMeta() || a.info.ChannelType == constant.ChannelTypeNewAPI || result == nil {
		return false, nil
	}
	key, baseURL, proxy := a.queryCredentials()
	if task.PrivateData.Key != "" {
		key = task.PrivateData.Key
	}
	if err = pluginruntime.ValidateRequestURL(retryURL, baseURL, a.plugin.Meta.AllowedHosts); err != nil {
		delete(state, "lecapiRetryInFlight")
		task.PrivateData.PluginState, _ = encodeLecAPIRetryState(state)
		return false, err
	}

	var payload lecapiRetryPayload
	if len(task.PrivateData.LecAPIRetryPayload) > 0 {
		if err := common.Unmarshal(task.PrivateData.LecAPIRetryPayload, &payload); err != nil || len(payload.Routes) == 0 {
			return false, fmt.Errorf("invalid retained LecAPI route payload")
		}
	}
	for {
		candidate := *inFlight
		if payload.Routes != nil {
			overrides, exists := payload.Routes[candidate.Route]
			if !exists {
				return false, fmt.Errorf("retained LecAPI route payload is missing")
			}
			candidate.Body = make(map[string]any, len(payload.Shared)+len(overrides))
			maps.Copy(candidate.Body, payload.Shared)
			maps.Copy(candidate.Body, overrides)
		}
		encodedBody, marshalErr := common.Marshal(candidate.Body)
		if marshalErr != nil || len(encodedBody) > maxLecAPIRetryPayloadBytes || !validLecAPIRouteCandidate(candidate) {
			delete(state, "lecapiRetryInFlight")
			task.PrivateData.PluginState, _ = encodeLecAPIRetryState(state)
			return false, fmt.Errorf("LecAPI route retry body is invalid or too large")
		}

		poll := &task.PrivateData.LecAPIPoll
		if ctx.Err() != nil || poll.RetryUntil != 0 && poll.RetryUntil <= time.Now().UnixMilli() {
			return false, model.ErrLecAPIRetryClaimLost
		}
		submitCtx, cancel := context.WithTimeout(ctx, model.LecAPIRetrySubmitTimeout)
		defer cancel()
		deadline, _ := submitCtx.Deadline()
		poll.RetryInFlight = true
		poll.RetryDispatched = true
		poll.RetryUntil = deadline.Add(model.LecAPIGatewayRecheckDelay).UnixMilli()
		won, updateErr := task.UpdateWithStatus(model.TaskStatusSubmitted)
		if !won || updateErr != nil || submitCtx.Err() != nil {
			return false, errors.Join(model.ErrLecAPIRetryClaimLost, updateErr)
		}

		requestInfo := *a.info
		channelMeta := *a.info.ChannelMeta
		channelMeta.ApiKey = key
		requestInfo.ChannelMeta = &channelMeta
		requestInfo.IsChannelTest = true
		a.submit = &requestDescriptor{URL: retryURL, Method: http.MethodPost, Headers: map[string]string{
			"Authorization": "Bearer " + key,
			"Content-Type":  "application/json",
		}}
		req, requestErr := http.NewRequestWithContext(submitCtx, http.MethodPost, "http://task-retry.local/", bytes.NewReader(encodedBody))
		if requestErr != nil {
			return false, requestErr
		}
		req.Header.Set("Content-Type", "application/json")
		c := &gin.Context{Request: req, Keys: make(map[string]any)}
		resp, requestErr := channel.DoTaskApiRequest(a, c, &requestInfo, bytes.NewReader(encodedBody))
		if requestErr != nil || resp == nil {
			if requestErr == nil {
				requestErr = fmt.Errorf("LecAPI route retry returned an empty response")
			}
			deferred, deferErr := markLecAPIRetryAmbiguous(task, result, state)
			return deferred, errors.Join(requestErr, deferErr)
		}

		if lecapiDefinitiveRouteRejection(resp) {
			_ = resp.Body.Close()
			if err = model.RecordLecAPIRouteOutcome(task.ChannelId, candidate.Route, false); err != nil {
				logger.LogWarn(ctx, common.LogText("could not record LecAPI route outcome: %s", err.Error()))
			}
			if len(candidates) == 0 {
				delete(state, "lecapiRetryInFlight")
				task.PrivateData.PluginState, _ = encodeLecAPIRetryState(state)
				return false, nil
			}
			inFlight = &candidates[0]
			candidates = candidates[1:]
			state["lecapiRetryCandidates"] = candidates
			state["lecapiRetryInFlight"] = inFlight
			task.PrivateData.PluginState, err = encodeLecAPIRetryState(state)
			if err != nil {
				return false, err
			}
			poll.RetryDispatched = false
			poll.RetryUntil = time.Now().Add(model.LecAPIGatewayRecheckDelay).UnixMilli()
			won, updateErr = task.UpdateWithStatus(model.TaskStatusSubmitted)
			if !won || updateErr != nil {
				return false, errors.Join(model.ErrLecAPIRetryClaimLost, updateErr)
			}
			continue
		}

		nextPoll := model.LecAPIPollState{Revision: task.PrivateData.LecAPIPoll.Revision}
		if lecapiGatewayError(resp.StatusCode) {
			var gatewayPoll model.LecAPIPollState
			resp, gatewayPoll = lecapiGatewaySubmission(resp, task.TaskID)
			nextPoll.RecheckAt, nextPoll.CreateGatewayError = gatewayPoll.RecheckAt, gatewayPoll.CreateGatewayError
		} else if resp.StatusCode/100 != 2 {
			code := resp.StatusCode
			_ = resp.Body.Close()
			deferred, deferErr := markLecAPIRetryAmbiguous(task, result, state)
			return deferred, errors.Join(fmt.Errorf("LecAPI route retry returned HTTP %d", code), deferErr)
		}

		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxTaskPluginPersistedJSONBytes+1))
		_ = resp.Body.Close()
		if readErr != nil || len(responseBody) > maxTaskPluginPersistedJSONBytes {
			if readErr == nil {
				readErr = fmt.Errorf("LecAPI route retry response exceeds the task limit")
			}
			deferred, deferErr := markLecAPIRetryAmbiguous(task, result, state)
			return deferred, errors.Join(readErr, deferErr)
		}
		var responseValue any
		if common.Unmarshal(responseBody, &responseValue) != nil {
			responseValue = string(responseBody)
		}
		hookContext, contextErr := a.queryContext(task, key, baseURL, proxy)
		if contextErr != nil {
			deferred, deferErr := markLecAPIRetryAmbiguous(task, result, state)
			return deferred, errors.Join(contextErr, deferErr)
		}
		hookContext["lecapiSubmitRoute"] = candidate.Route
		hookContext["lecapiRetryURL"] = retryURL
		hookContext["lecapiRetryCandidates"] = jsonValue(candidates)
		headers := make(map[string][]string, len(resp.Header))
		for name, values := range resp.Header {
			headers[name] = values
		}
		value, hookErr := a.plugin.Engine.Call(pluginruntime.WithFetcher(submitCtx, a.hookFetcher(baseURL, proxy)), "parseSubmitResponse", hookContext,
			map[string]any{"statusCode": resp.StatusCode, "headers": headers, "body": responseValue})
		if hookErr != nil {
			deferred, deferErr := markLecAPIRetryAmbiguous(task, result, state)
			return deferred, errors.Join(hookErr, deferErr)
		}
		var parsed submitResponse
		if err = convert(value, &parsed); err != nil {
			deferred, deferErr := markLecAPIRetryAmbiguous(task, result, state)
			return deferred, errors.Join(err, deferErr)
		}
		pluginState, _ := encodeReturnedPluginState(value)
		if len(pluginState) > maxTaskPluginPersistedJSONBytes {
			pluginState = withoutLecAPIRetryState(pluginState)
		}
		if len(pluginState) > maxTaskPluginPersistedJSONBytes {
			return markLecAPIRetryAmbiguous(task, result, state)
		}
		var taskData []byte
		if parsed.TaskData != nil {
			taskData, err = common.Marshal(parsed.TaskData)
			if err != nil || len(taskData) > maxTaskPluginPersistedJSONBytes {
				return markLecAPIRetryAmbiguous(task, result, state)
			}
		}
		if parsed.Immediate != nil && parsed.Immediate.Status == model.TaskStatusFailure {
			if err = model.RecordLecAPIRouteOutcome(task.ChannelId, candidate.Route, false); err != nil {
				logger.LogWarn(ctx, common.LogText("could not record LecAPI route outcome: %s", err.Error()))
			}
			if len(candidates) == 0 {
				delete(state, "lecapiRetryInFlight")
				task.PrivateData.PluginState, _ = encodeLecAPIRetryState(state)
				return false, nil
			}
			inFlight = &candidates[0]
			candidates = candidates[1:]
			state["lecapiRetryCandidates"] = candidates
			state["lecapiRetryInFlight"] = inFlight
			task.PrivateData.PluginState, err = encodeLecAPIRetryState(state)
			if err != nil {
				return false, err
			}
			poll.RetryDispatched = false
			poll.RetryUntil = time.Now().Add(model.LecAPIGatewayRecheckDelay).UnixMilli()
			won, updateErr = task.UpdateWithStatus(model.TaskStatusSubmitted)
			if !won || updateErr != nil {
				return false, errors.Join(model.ErrLecAPIRetryClaimLost, updateErr)
			}
			continue
		}
		if parsed.TaskID == "" {
			return markLecAPIRetryAmbiguous(task, result, state)
		}
		if err = model.RecordLecAPIRouteOutcome(task.ChannelId, currentRoute, false); err != nil {
			logger.LogWarn(ctx, common.LogText("could not record LecAPI route outcome: %s", err.Error()))
		}
		task.PrivateData.UpstreamTaskID = parsed.TaskID
		task.PrivateData.LecAPIPoll = nextPoll
		if taskData != nil {
			task.Data = taskData
		}
		if len(pluginState) > 0 {
			task.PrivateData.PluginState = pluginState
		}
		if len(candidates) == 0 {
			task.PrivateData.LecAPIRetryPayload = nil
		}
		task.FailReason = ""
		task.PrivateData.PollFailures = 0
		if parsed.Immediate == nil {
			task.Status = model.TaskStatusSubmitted
			task.Progress = taskcommon.ProgressSubmitted
			result.Status = string(model.TaskStatusSubmitted)
			return true, nil
		}
		immediate := &relaycommon.TaskInfo{
			Code: parsed.Immediate.Code, TaskID: parsed.Immediate.TaskID, Status: parsed.Immediate.Status,
			Progress: parsed.Immediate.Progress, Reason: parsed.Immediate.Reason, Url: parsed.Immediate.URL,
			RemoteUrl: parsed.Immediate.RemoteURL, CompletionTokens: positiveInt(parsed.Immediate.CompletionTokens),
			TotalTokens: positiveInt(parsed.Immediate.TotalTokens),
		}
		if immediate.Status != model.TaskStatusSuccess {
			return markLecAPIRetryAmbiguous(task, result, state)
		}
		if a.hasHook(ctx, "extractUsageOnComplete") {
			facts, usageErr := a.plugin.Engine.Call(pluginruntime.WithFetcher(submitCtx, a.hookFetcher(baseURL, proxy)), "extractUsageOnComplete", hookContext, jsonValue(immediate), parsed.TaskData)
			if usageErr == nil {
				_ = a.applyCompletionUsageFacts(immediate, facts, task.Properties.UpstreamModelName, task.Properties.OriginModelName)
			}
		}
		task.Status = model.TaskStatusSuccess
		task.Progress = taskcommon.ProgressComplete
		task.FinishTime = time.Now().Unix()
		if immediate.Url != "" {
			task.PrivateData.ResultURL = immediate.Url
		} else {
			task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		}
		*result = *immediate
		return true, nil
	}
}

func lecapiGatewayError(code int) bool {
	return code == http.StatusBadGateway || code == http.StatusGatewayTimeout
}

// A gateway error may include an upstream task ID. Keep it only when the
// reply identifies it as a task, since a proxy's own request ID cannot be polled.
func lecapiGatewaySubmission(resp *http.Response, publicID string) (*http.Response, model.LecAPIPollState) {
	var body map[string]any
	if resp.Body != nil {
		_ = common.DecodeJson(io.LimitReader(resp.Body, 64<<10), &body)
		_ = resp.Body.Close()
	}
	id, _ := body["task_id"].(string)
	if data, ok := body["data"].(map[string]any); ok && id == "" {
		id, _ = data["task_id"].(string)
	}
	if metadata, ok := body["metadata"].(map[string]any); ok && id == "" {
		id, _ = metadata["task_id"].(string)
	}
	if body["object"] == "lecapi.task" && id == "" {
		id, _ = body["id"].(string)
	}
	id = strings.TrimSpace(id)
	poll := model.LecAPIPollState{
		RecheckAt:          time.Now().Add(model.LecAPIGatewayRecheckDelay).UnixMilli(),
		CreateGatewayError: id == "",
	}
	if id == "" {
		id = publicID
	}
	encoded, _ := common.Marshal(map[string]string{"task_id": id, "status": "queued"})
	return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(encoded))}, poll
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
