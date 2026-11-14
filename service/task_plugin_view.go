package service

import (
	"bytes"
	"context"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

// BuildTaskPluginDeliveryView leaves the upstream snapshot intact for polling
// and content refreshes while presenting usable video addresses to clients.
func BuildTaskPluginDeliveryView(ctx context.Context, task *model.Task) (dto.TaskView, error) {
	view, err := BuildTaskPluginView(task)
	if err != nil {
		return view, err
	}
	view.Data, err = ApplyTaskVideoDelivery(ctx, task, view.Data)
	return view, err
}

// ApplyTaskVideoDelivery replaces artifact sources in a decoded public response.
// Renewed download signatures retain their resource identity. Callers own value;
// the persisted upstream data is untouched.
func ApplyTaskVideoDelivery(ctx context.Context, task *model.Task, value any) (any, error) {
	if ResolveTaskVideoDelivery == nil || task.Status != model.TaskStatusSuccess || !GetTaskArtifactStore().Enabled() {
		return value, nil
	}
	deliveries, err := ResolveTaskVideoDelivery(ctx, task)
	if err != nil {
		return nil, err
	}
	replacements := make(map[string]string)
	identities := make(map[string]string)
	for _, delivery := range deliveries {
		for _, source := range delivery.SourceURLs {
			if source != "" {
				replacements[source] = delivery.URL
				if identity := taskVideoSourceIdentity(source); identity != "" {
					previous, exists := identities[identity]
					if exists && previous != delivery.URL {
						identities[identity] = "" // Ambiguous resources still require an exact match.
					} else if !exists {
						identities[identity] = delivery.URL
					}
				}
			}
		}
	}
	if len(replacements) == 0 {
		return value, nil
	}
	return replaceTaskVideoURLs(value, replacements, identities), nil
}

// taskVideoSourceIdentity ignores only established signing parameters, retaining
// the origin, path and every resource/transform parameter. A token or arbitrary
// query value may identify a different resource and is deliberately retained.
func taskVideoSourceIdentity(source string) string {
	parsed, err := url.Parse(source)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return ""
	}
	signed := false
	for key := range query {
		switch strings.ToLower(key) {
		case "signature", "sig", "x-amz-signature", "x-oss-signature", "x-goog-signature":
			signed = true
		}
	}
	if !signed {
		return ""
	}
	for key := range query {
		switch strings.ToLower(key) {
		case "signature", "expires", "ossaccesskeyid", "googleaccessid", "security-token",
			"x-amz-signature", "x-amz-algorithm", "x-amz-credential", "x-amz-date", "x-amz-expires", "x-amz-security-token", "x-amz-signedheaders",
			"x-oss-signature", "x-oss-signature-version", "x-oss-credential", "x-oss-date", "x-oss-expires", "x-oss-security-token", "x-oss-additional-headers",
			"x-goog-signature", "x-goog-algorithm", "x-goog-credential", "x-goog-date", "x-goog-expires", "x-goog-signedheaders",
			"sig", "se", "st":
			query.Del(key)
		}
	}
	parsed.Scheme, parsed.Host = strings.ToLower(parsed.Scheme), strings.ToLower(parsed.Host)
	parsed.RawQuery, parsed.Fragment = query.Encode(), ""
	return parsed.String()
}

func replaceTaskVideoURLs(value any, replacements, identities map[string]string) any {
	switch typed := value.(type) {
	case common.RawMessage:
		trimmed := bytes.TrimSpace(typed)
		if len(trimmed) == 0 {
			return value
		}
		switch trimmed[0] {
		case '"':
			var source string
			if common.Unmarshal(trimmed, &source) == nil {
				if replacement := replaceTaskVideoURLs(source, replacements, identities).(string); replacement != source {
					encoded, _ := common.Marshal(replacement)
					return common.RawMessage(encoded)
				}
			}
		case '{':
			var object map[string]common.RawMessage
			if common.Unmarshal(trimmed, &object) == nil {
				for key, item := range object {
					object[key] = replaceTaskVideoURLs(item, replacements, identities).(common.RawMessage)
				}
				encoded, _ := common.Marshal(object)
				return common.RawMessage(encoded)
			}
		case '[':
			var items []common.RawMessage
			if common.Unmarshal(trimmed, &items) == nil {
				for i, item := range items {
					items[i] = replaceTaskVideoURLs(item, replacements, identities).(common.RawMessage)
				}
				encoded, _ := common.Marshal(items)
				return common.RawMessage(encoded)
			}
		}
	case string:
		if replacement, ok := replacements[typed]; ok {
			return replacement
		}
		if replacement := identities[taskVideoSourceIdentity(typed)]; replacement != "" {
			return replacement
		}
	case []any:
		for i, item := range typed {
			typed[i] = replaceTaskVideoURLs(item, replacements, identities)
		}
	case map[string]any:
		for key, item := range typed {
			typed[key] = replaceTaskVideoURLs(item, replacements, identities)
		}
	}
	return value
}

// BuildTaskPluginView converts a persisted task into the deliberately narrow
// public shape permitted at JavaScript plugin boundaries.
func BuildTaskPluginView(task *model.Task) (dto.TaskView, error) {
	createdAt := task.CreatedAt
	if createdAt == 0 {
		createdAt = task.SubmitTime
	}
	view := dto.TaskView{
		TaskID:     task.TaskID,
		Platform:   string(task.Platform),
		Status:     string(task.Status),
		Progress:   task.Progress,
		FailReason: task.FailReason,
		CreatedAt:  createdAt,
		UpdatedAt:  task.UpdatedAt,
		FinishedAt: task.FinishTime,
	}
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &view.Data); err != nil {
			return dto.TaskView{}, err
		}
		view.Data = replacePrivateTaskID(view.Data, task.PrivateData.UpstreamTaskID, task.TaskID)
	}
	return view, nil
}

// replacePrivateTaskID rewrites exact private IDs only in known task-ID fields.
// Map keys and opaque strings, including URLs containing the ID, are preserved.
func replacePrivateTaskID(value any, privateTaskID, publicTaskID string) any {
	if privateTaskID == "" || privateTaskID == publicTaskID {
		return value
	}
	switch typed := value.(type) {
	case []any:
		replaced := make([]any, len(typed))
		for index, item := range typed {
			replaced[index] = replacePrivateTaskID(item, privateTaskID, publicTaskID)
		}
		return replaced
	case map[string]any:
		replaced := make(map[string]any, len(typed))
		for key, item := range typed {
			if (key == "id" || key == "task_id" || key == "taskId") && item == privateTaskID {
				replaced[key] = publicTaskID
				continue
			}
			replaced[key] = replacePrivateTaskID(item, privateTaskID, publicTaskID)
		}
		return replaced
	default:
		return value
	}
}
