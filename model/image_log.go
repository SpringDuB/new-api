package model

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Image actions already emitted by the built-in task plugins, plus host image
// relays. Classifying on output action avoids treating vision inputs as draws.
var ImageTaskActions = []string{"image_generation", "image_edit", "text_to_image", "image_to_image", "imagine", "edits", "blend", "hd_upscale"}

func IsImageTaskAction(action string) bool { return slices.Contains(ImageTaskActions, action) }

// UpdateTaskImageLog merges audit fields into the persisted terminal row.
// A protocol observer can hold a stale task after losing the poller's CAS;
// its plugin state, original request and billing context must not be restored.
func UpdateTaskImageLog(ctx context.Context, task *Task) error {
	return DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current Task
		if err := lockForUpdate(tx).First(&current, task.ID).Error; err != nil {
			return err
		}
		if current.Status != TaskStatusSuccess && current.Status != TaskStatusFailure {
			return nil
		}
		if current.PrivateData.Response == nil {
			current.PrivateData.Response = task.PrivateData.Response
		}
		current.PrivateData.ImageRequestMode = task.PrivateData.ImageRequestMode
		if task.PrivateData.ImageArtifacts != nil && current.Status == task.Status {
			current.PrivateData.ImageArtifacts = task.PrivateData.ImageArtifacts
		}
		return tx.Model(&current).Update("private_data", current.PrivateData).Error
	})
}

type ImageLogQuery struct {
	UserID         int
	TaskID         string
	ChannelID      string
	StartTimestamp string
	EndTimestamp   string
}

// GetImageLogs paginates the union of task and legacy MJ identities in SQL,
// then loads only this page. Sorting before LIMIT/OFFSET preserves global
// ordering/counts even when records alternate between the two sources.
func GetImageLogs(offset, limit int, params ImageLogQuery) ([]*Task, int64, error) {
	tasks := DB.Model(&Task{}).Where("action IN ?", ImageTaskActions)
	mj := DB.Model(&Midjourney{})
	if params.UserID > 0 {
		tasks = tasks.Where("user_id = ?", params.UserID)
		mj = mj.Where("user_id = ?", params.UserID)
	}
	if params.ChannelID != "" {
		tasks = tasks.Where("channel_id = ?", params.ChannelID)
		mj = mj.Where("channel_id = ?", params.ChannelID)
	}
	if params.TaskID != "" {
		tasks = tasks.Where("task_id = ?", params.TaskID)
		mj = mj.Where("mj_id = ?", params.TaskID)
	}
	for _, bound := range []struct{ value, operator string }{{params.StartTimestamp, ">="}, {params.EndTimestamp, "<="}} {
		if bound.value == "" {
			continue
		}
		seconds, err := strconv.ParseInt(bound.value, 10, 64)
		if err != nil || seconds < 0 || seconds > 253402300799 {
			return nil, 0, fmt.Errorf("invalid image log timestamp")
		}
		tasks = tasks.Where("submit_time "+bound.operator+" ?", seconds)
		millis := seconds * 1000
		if bound.operator == "<=" {
			millis += 999
		}
		mj = mj.Where("submit_time "+bound.operator+" ?", millis)
	}
	union := DB.Table("(? UNION ALL ?) AS drawing_rows",
		tasks.Select("id, submit_time * 1000 AS sort_time, 0 AS source"),
		mj.Select("id, submit_time AS sort_time, 1 AS source"))
	var total int64
	if err := union.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID       int64
		Source   int
		SortTime int64
	}
	if err := union.Select("id, source, sort_time").Order("sort_time DESC, source ASC, id DESC").Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	ids, mjIDs := make([]int64, 0), make([]int64, 0)
	for _, row := range rows {
		if row.Source == 0 {
			ids = append(ids, row.ID)
		} else {
			mjIDs = append(mjIDs, row.ID)
		}
	}
	byID, mjByID := make(map[int64]*Task), make(map[int64]*Task)
	if len(ids) > 0 {
		var page []*Task
		query := DB.Where("id IN ?", ids).Omit("data")
		if params.UserID > 0 {
			query = query.Where("user_id = ?", params.UserID)
		}
		if err := query.Find(&page).Error; err != nil {
			return nil, 0, err
		}
		for _, task := range page {
			byID[task.ID] = task
		}
	}
	if len(mjIDs) > 0 {
		var page []Midjourney
		query := DB.Where("id IN ?", mjIDs)
		if params.UserID > 0 {
			query = query.Where("user_id = ?", params.UserID)
		}
		if err := query.Find(&page).Error; err != nil {
			return nil, 0, err
		}
		for _, item := range page {
			mjByID[int64(item.Id)] = MidjourneyImageLog(item)
		}
	}
	result := make([]*Task, 0, len(rows))
	for _, row := range rows {
		task := byID[row.ID]
		if row.Source == 1 {
			task = mjByID[row.ID]
		}
		if task != nil {
			task.Data = nil
			result = append(result, task)
		}
	}
	return result, total, nil
}

func MidjourneyImageLog(item Midjourney) *Task {
	task := &Task{ID: -int64(item.Id), TaskID: fmt.Sprintf("mjlog_%d", item.Id), Platform: "mj", UserId: item.UserId, ChannelId: item.ChannelId,
		Action: item.Action, Status: TaskStatus(item.Status), Progress: item.Progress, FailReason: item.FailReason,
		SubmitTime: item.SubmitTime / 1000, StartTime: item.StartTime / 1000, FinishTime: item.FinishTime / 1000, Quota: item.Quota,
		Properties: Properties{Input: item.Prompt}, PrivateData: TaskPrivateData{UpstreamTaskID: item.MjId, ImageRequestMode: "asynchronous"}}
	if item.ImageUrl != "" {
		task.PrivateData.ImageArtifacts = []TaskImageArtifact{{Key: "image", MimeType: "image/png", SourceURL: item.ImageUrl}}
	}
	task.SetData(map[string]any{"mj_id": item.MjId, "status": item.Status, "progress": item.Progress, "prompt": item.Prompt, "image_url": item.ImageUrl, "fail_reason": item.FailReason})
	return task
}

func GetMidjourneyImageLog(taskID string, userID int, admin bool) (*Task, bool, error) {
	identifier, ok := strings.CutPrefix(taskID, "mjlog_")
	if !ok {
		return nil, false, nil
	}
	id, err := strconv.ParseInt(identifier, 10, 64)
	if err != nil || id <= 0 {
		return nil, false, nil
	}
	query := DB.Where("id = ?", id)
	if !admin {
		query = query.Where("user_id = ?", userID)
	}
	var item Midjourney
	err = query.First(&item).Error
	exists, err := RecordExist(err)
	if err != nil || !exists {
		return nil, exists, err
	}
	return MidjourneyImageLog(item), true, nil
}
