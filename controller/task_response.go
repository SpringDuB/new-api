package controller

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relay"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// buildDashboardTaskQueryResponse presents current state through the same
// protocol/native renderers used for client retrieval. It never submits work
// or substitutes the persisted upstream response for a missing presenter.
func buildDashboardTaskQueryResponse(c *gin.Context, task *model.Task) (*model.TaskResponseSnapshot, string, error) {
	if !task.ResultRetrievable() {
		return nil, "", nil
	}
	requestPath := ""
	if task.PrivateData.Execution != nil {
		requestPath = task.PrivateData.Execution.RequestPath
	}
	if requestPath == "" && task.PrivateData.ClientRequest != nil {
		requestPath = task.PrivateData.ClientRequest.Path
	}
	var body []byte
	var err error
	queryPath := ""
	status := http.StatusOK
	// Host /v1/videos retrieval hands native submissions back to their own
	// matching query route. Preserve that choice in the console as well.
	nativeVideo := false
	if requestPath == "/v1/videos" && task.PrivateData.Execution != nil && task.PrivateData.Execution.TaskPlugin != nil {
		if plugin, _, ok := resolveTaskPluginForProtocolRetrieve(task.Platform); ok {
			nativeVideo = slices.ContainsFunc(pluginruntime.PluginRouteBindings(plugin), func(binding pluginruntime.RouteBinding) bool {
				return binding.Route.Type != pluginruntime.RouteTypeQuery && binding.Route.Method == http.MethodPost && binding.Route.Path == requestPath
			})
		}
	}
	switch requestPath {
	case "/v1/videos":
		if nativeVideo {
			break
		}
		converter, ok := relay.GetTaskAdaptor(task.Platform).(relaychannel.OpenAIVideoConverter)
		if !ok {
			return nil, "", nil
		}
		queryPath = "/v1/videos/" + task.TaskID
		body, err = converter.ConvertToOpenAIVideo(task)
	case "/v1/video/generations":
		queryPath = "/v1/video/generations/" + task.TaskID
		body, err = common.Marshal(dto.TaskResponse[any]{Code: "success", Data: relay.TaskModel2Dto(task)})
		if err == nil {
			var value any
			value, err = service.ApplyTaskVideoDelivery(c.Request.Context(), task, common.RawMessage(body))
			if err == nil {
				body = value.(common.RawMessage)
			}
		}
	case "/v1/responses":
		plugin, generation, ok := resolveTaskPluginForProtocolRetrieve(task.Platform)
		if !ok || !slices.ContainsFunc(plugin.Meta.Protocols, func(claim pluginruntime.ProtocolClaim) bool { return claim.Name == "openai_responses" }) {
			return nil, "", nil
		}
		deps := defaultPluginProtocolBridgeDeps()
		responseID := "resp_" + strings.TrimPrefix(task.TaskID, "task_")
		queryPath = "/v1/responses/" + responseID
		createdAt := task.CreatedAt
		if createdAt == 0 {
			createdAt = task.SubmitTime
		}
		machine := relay.NewPluginResponsesMachine(task.TaskID, task.Properties.OriginModelName, createdAt, deps.protocolLimits)
		machine.SetBackground(task.PrivateData.ResponsesBackground)
		var value any
		switch task.Status {
		case model.TaskStatusFailure:
			value, err = machine.FinalResponse(nil, string(task.Status))
		case model.TaskStatusSuccess:
			pinned := pluginruntime.PinnedEndpoint{Generation: generation, Plugin: plugin, Protocol: "openai_responses", Model: task.Properties.OriginModelName}
			request := pluginruntime.ProtocolRequestContext{RouteRequestContext: pluginruntime.RouteRequestContext{Path: queryPath, Method: http.MethodGet, Params: map[string]string{"response_id": responseID}, Query: url.Values{}, Body: map[string]any{"kind": "none"}}, Protocol: "openai_responses", Operation: "retrieve", Model: task.Properties.OriginModelName}
			if plugin.Meta.ProtocolSupports("openai_responses", "sync") || plugin.Meta.ProtocolSupports("openai_responses", "background") {
				value, _, err = renderTaskPluginProtocolFinalResponse(c.Request.Context(), pinned, request, task, machine, deps)
			} else {
				value, _, err = renderTaskPluginProtocolEventsResponse(c.Request.Context(), pinned, request, task, machine, deps)
			}
			if err != nil {
				value, err = machine.FailureResponse(string(task.Status))
			}
		default:
			value = machine.PendingResponse(string(task.Status))
		}
		if err == nil {
			body, err = common.Marshal(value)
		}
	}
	if queryPath == "" {
		plugin, _, ok := resolveTaskPluginForProtocolRetrieve(task.Platform)
		if !ok || requestPath == "" {
			return nil, "", nil
		}
		request, renderer, ok := dashboardTaskNativeQuery(plugin, requestPath, task.TaskID)
		if !ok {
			return nil, "", nil
		}
		queryPath = request.Path
		if query := url.Values(request.Query).Encode(); query != "" {
			queryPath += "?" + query
		}
		var value any
		value, err = middleware.BuildTaskPluginQueryResult(c.Request.Context(), plugin, request, []*model.Task{task}, renderer, false)
		if err == nil {
			var reply middleware.NativeReply
			reply, err = middleware.BuildNativeReply(http.StatusOK, value)
			if err == nil {
				status = reply.Status
				if reply.HasBody {
					body, err = common.Marshal(reply.Body)
				}
			}
		}
	}
	if err != nil {
		return nil, queryPath, err
	}
	secrets := []string{task.PrivateData.Key}
	if channel, channelErr := model.CacheGetChannel(task.ChannelId); channelErr == nil {
		for key := range strings.SplitSeq(channel.Key, "\n") {
			if key != "" {
				secrets = append(secrets, key)
			}
		}
	}
	snapshot := service.BuildTaskResponseSnapshot(c, body, status, "application/json", "json", secrets...)
	snapshot.CapturedAt = 0 // This is a projection, not a historical wire response.
	return snapshot, queryPath, nil
}

// A native task may have several query surfaces. Choose the uniquely closest
// GET route to its submit path; ambiguous or body-dependent routes stay
// unavailable rather than claiming an unrelated response as the client's.
func dashboardTaskNativeQuery(plugin *pluginruntime.LoadedPlugin, submitPath, taskID string) (pluginruntime.RouteRequestContext, string, bool) {
	best, tied := -1, false
	var selected pluginruntime.Route
	for _, route := range plugin.Meta.Routes {
		if route.Type != pluginruntime.RouteTypeQuery || route.Method != http.MethodGet {
			continue
		}
		supported := true
		for part := range strings.SplitSeq(route.Path, "/") {
			if strings.HasPrefix(part, "*") || strings.HasPrefix(part, ":") && part != ":"+route.TaskIDParam {
				supported = false
			}
		}
		if !supported {
			continue
		}
		left := strings.Split(strings.TrimPrefix(submitPath, "/"+plugin.Meta.Key), "/")
		right := strings.Split(route.Path, "/")
		score := 0
		for i := range min(len(left), len(right)) {
			if left[i] != right[i] {
				break
			}
			score++
		}
		if score > best {
			best, selected, tied = score, route, false
		} else if score == best {
			tied = true
		}
	}
	if best < 0 || tied {
		return pluginruntime.RouteRequestContext{}, "", false
	}
	path := strings.ReplaceAll(selected.Path, ":"+selected.TaskIDParam, url.PathEscape(taskID))
	if strings.HasPrefix(submitPath, "/"+plugin.Meta.Key+"/") && !strings.HasPrefix(path, "/"+plugin.Meta.Key+"/") {
		path = "/" + plugin.Meta.Key + path
	}
	query := url.Values{}
	if selected.TaskIDInQuery() {
		query.Set(selected.TaskIDParam, taskID)
	}
	return pluginruntime.RouteRequestContext{Path: path, Method: http.MethodGet, Params: map[string]string{selected.TaskIDParam: taskID}, Query: query, Body: map[string]any{"kind": "none"}}, selected.Render, true
}
