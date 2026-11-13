package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

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
		if task.Status == model.TaskStatusSuccess && StoredTaskArtifactURLs != nil && GetTaskArtifactStore().Enabled() {
			urls := StoredTaskArtifactURLs(task)
			imageIndex, inlineIndex := 0, 0
			view.Data = replaceStoredTaskResultURLs(view.Data, urls, &imageIndex, &inlineIndex)
		}
	}
	return view, nil
}

// Replace output URL members only in the public view. Persisted provider data
// remains available for refresh, accounting and diagnostics.
func replaceStoredTaskResultURLs(value any, urls map[string]string, imageIndex, inlineIndex *int) any {
	switch typed := value.(type) {
	case string:
		if strings.HasPrefix(typed, "https://") || strings.HasPrefix(typed, "http://") {
			if urls["video"] != "" {
				return urls["video"]
			}
			*imageIndex++
			if url := urls[fmt.Sprintf("image-%d", *imageIndex)]; url != "" {
				return url
			}
		}
	case []any:
		for i := range typed {
			typed[i] = replaceStoredTaskResultURLs(typed[i], urls, imageIndex, inlineIndex)
		}
	case map[string]any:
		for _, field := range []string{"url", "video_url", "result_url", "download_url"} {
			if _, ok := typed[field].(string); ok && urls["video"] != "" {
				typed[field] = urls["video"]
			}
		}
		_, hasURL := typed["url"].(string)
		_, hasBase64 := typed["b64_json"].(string)
		if (hasURL || hasBase64) && urls["video"] == "" {
			storedURL := ""
			if hasBase64 && !hasURL {
				*inlineIndex++
				storedURL = urls[fmt.Sprintf("image-b64-%d", *inlineIndex)]
			}
			if storedURL == "" {
				*imageIndex++
				storedURL = urls[fmt.Sprintf("image-%d", *imageIndex)]
			}
			if storedURL != "" {
				typed["url"] = storedURL
			}
		}
		for _, field := range []string{"data", "images", "result", "output", "video", "metadata"} {
			if child, ok := typed[field]; ok {
				typed[field] = replaceStoredTaskResultURLs(child, urls, imageIndex, inlineIndex)
			}
		}
	}
	return value
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
