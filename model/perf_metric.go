package model

import (
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Route outcomes use the same retention and counter storage as model metrics,
// but never enter public model queries. The group column holds a channel ID.
const LecAPIRouteMetricPrefix = "lecapi-route#"

type LecAPIRouteStat struct {
	Route        string  `json:"route" gorm:"column:model_name"`
	ChannelID    int     `json:"channel_id"`
	RequestCount int64   `json:"request_count"`
	SuccessCount int64   `json:"success_count"`
	SuccessRate  float64 `json:"success_rate" gorm:"-"`
}

func RecordLecAPIRouteOutcome(channelID int, route string, success bool) error {
	if DB == nil || channelID <= 0 || !strings.HasPrefix(route, "lec-") || len(route)+len(LecAPIRouteMetricPrefix) > 128 {
		return nil
	}
	metric := PerfMetric{ModelName: LecAPIRouteMetricPrefix + route, Group: strconv.Itoa(channelID), BucketTs: time.Now().Unix() / 3600 * 3600, RequestCount: 1}
	if success {
		metric.SuccessCount = 1
	}
	return UpsertPerfMetric(&metric)
}

// LecAPIRouteStats aggregates up to seven days, per channel and upstream model.
// The short public ID is deliberately not part of the statistics key.
func LecAPIRouteStats(channelID int, routes []string) ([]LecAPIRouteStat, error) {
	if DB == nil {
		return nil, nil
	}
	var stats []LecAPIRouteStat
	now := time.Now().Unix()
	query := DB.Model(&PerfMetric{}).
		Select("model_name, "+commonGroupCol+" AS channel_id, SUM(request_count) AS request_count, SUM(success_count) AS success_count").
		Where("model_name LIKE ? AND bucket_ts BETWEEN ? AND ?", LecAPIRouteMetricPrefix+"%", now/3600*3600-167*3600, now)
	if channelID > 0 {
		query = query.Where(commonGroupCol+" = ?", strconv.Itoa(channelID))
	}
	if len(routes) > 0 {
		names := make([]string, len(routes))
		for i, route := range routes {
			names[i] = LecAPIRouteMetricPrefix + route
		}
		query = query.Where("model_name IN ?", names)
	}
	if err := query.Group("model_name, " + commonGroupCol).Having("SUM(request_count) > 0").Order("model_name, " + commonGroupCol).Find(&stats).Error; err != nil {
		return nil, err
	}
	for i := range stats {
		item := &stats[i]
		item.Route = strings.TrimPrefix(item.Route, LecAPIRouteMetricPrefix)
		item.SuccessRate = float64(item.SuccessCount) / float64(item.RequestCount) * 100
	}
	return stats, nil
}

// PerfMetric stores aggregated relay performance metrics for the model square.
type PerfMetric struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	ModelName      string `json:"model_name" gorm:"size:128;uniqueIndex:idx_perf_model_group_bucket,priority:1"`
	Group          string `json:"group" gorm:"column:group;size:64;uniqueIndex:idx_perf_model_group_bucket,priority:2"`
	BucketTs       int64  `json:"bucket_ts" gorm:"uniqueIndex:idx_perf_model_group_bucket,priority:3;index:idx_perf_bucket_ts"`
	RequestCount   int64  `json:"-" gorm:"default:0"`
	SuccessCount   int64  `json:"-" gorm:"default:0"`
	TotalLatencyMs int64  `json:"-" gorm:"default:0"`
	TtftSumMs      int64  `json:"-" gorm:"default:0"`
	TtftCount      int64  `json:"-" gorm:"default:0"`
	OutputTokens   int64  `json:"-" gorm:"default:0"`
	GenerationMs   int64  `json:"-" gorm:"default:0"`
}

func (PerfMetric) TableName() string {
	return "perf_metrics"
}

func UpsertPerfMetric(metric *PerfMetric) error {
	if metric == nil || metric.RequestCount == 0 {
		return nil
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "model_name"},
			{Name: "group"},
			{Name: "bucket_ts"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count":    gorm.Expr("perf_metrics.request_count + ?", metric.RequestCount),
			"success_count":    gorm.Expr("perf_metrics.success_count + ?", metric.SuccessCount),
			"total_latency_ms": gorm.Expr("perf_metrics.total_latency_ms + ?", metric.TotalLatencyMs),
			"ttft_sum_ms":      gorm.Expr("perf_metrics.ttft_sum_ms + ?", metric.TtftSumMs),
			"ttft_count":       gorm.Expr("perf_metrics.ttft_count + ?", metric.TtftCount),
			"output_tokens":    gorm.Expr("perf_metrics.output_tokens + ?", metric.OutputTokens),
			"generation_ms":    gorm.Expr("perf_metrics.generation_ms + ?", metric.GenerationMs),
		}),
	}).Create(metric).Error
}

func GetPerfMetrics(modelName string, group string, startTs int64, endTs int64) ([]PerfMetric, error) {
	var metrics []PerfMetric
	query := DB.Model(&PerfMetric{}).
		Where("model_name = ? AND model_name NOT LIKE ? AND bucket_ts >= ? AND bucket_ts <= ?", modelName, LecAPIRouteMetricPrefix+"%", startTs, endTs)
	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	err := query.Order("bucket_ts ASC").Find(&metrics).Error
	return metrics, err
}

type PerfMetricSummary struct {
	ModelName      string `json:"model_name"`
	RequestCount   int64  `json:"request_count"`
	SuccessCount   int64  `json:"success_count"`
	TotalLatencyMs int64  `json:"total_latency_ms"`
	OutputTokens   int64  `json:"output_tokens"`
	GenerationMs   int64  `json:"generation_ms"`
}

type PerfMetricSummaryBucket struct {
	ModelName      string `json:"model_name"`
	BucketTs       int64  `json:"bucket_ts"`
	RequestCount   int64  `json:"request_count"`
	SuccessCount   int64  `json:"success_count"`
	TotalLatencyMs int64  `json:"total_latency_ms"`
	OutputTokens   int64  `json:"output_tokens"`
	GenerationMs   int64  `json:"generation_ms"`
}

func GetPerfMetricsSummaryAll(startTs int64, endTs int64, groups []string) ([]PerfMetricSummary, error) {
	var summaries []PerfMetricSummary
	query := DB.Model(&PerfMetric{}).
		Select("model_name, SUM(request_count) as request_count, SUM(success_count) as success_count, SUM(total_latency_ms) as total_latency_ms, SUM(output_tokens) as output_tokens, SUM(generation_ms) as generation_ms").
		Where("model_name NOT LIKE ? AND bucket_ts >= ? AND bucket_ts <= ?", LecAPIRouteMetricPrefix+"%", startTs, endTs)
	if groups != nil {
		if len(groups) == 0 {
			return summaries, nil
		}
		query = query.Where(commonGroupCol+" IN ?", groups)
	}
	err := query.
		Group("model_name").
		Having("SUM(request_count) > 0").
		Find(&summaries).Error
	return summaries, err
}

func GetPerfMetricsSummaryBucketsAll(startTs int64, endTs int64, groups []string) ([]PerfMetricSummaryBucket, error) {
	var summaries []PerfMetricSummaryBucket
	query := DB.Model(&PerfMetric{}).
		Select("model_name, bucket_ts, SUM(request_count) as request_count, SUM(success_count) as success_count, SUM(total_latency_ms) as total_latency_ms, SUM(output_tokens) as output_tokens, SUM(generation_ms) as generation_ms").
		Where("model_name NOT LIKE ? AND bucket_ts >= ? AND bucket_ts <= ?", LecAPIRouteMetricPrefix+"%", startTs, endTs)
	if groups != nil {
		if len(groups) == 0 {
			return summaries, nil
		}
		query = query.Where(commonGroupCol+" IN ?", groups)
	}
	err := query.
		Group("model_name, bucket_ts").
		Having("SUM(request_count) > 0").
		Order("bucket_ts ASC").
		Find(&summaries).Error
	return summaries, err
}

// GetRecentPerfMetricSuccessBuckets reads the latest nonempty hours within
// the supplied history window, independently of the summary's reporting window.
func GetRecentPerfMetricSuccessBuckets(modelName string, startTs int64, endTs int64, groups []string, limit int) ([]PerfMetricSummaryBucket, error) {
	if modelName == "" || strings.HasPrefix(modelName, LecAPIRouteMetricPrefix) || limit <= 0 || (groups != nil && len(groups) == 0) {
		return nil, nil
	}
	const hourBucket = "bucket_ts - (bucket_ts % 3600)"
	query := DB.Model(&PerfMetric{}).
		Select("model_name, "+hourBucket+" AS bucket_ts, SUM(request_count) AS request_count, SUM(success_count) AS success_count").
		Where("model_name = ? AND bucket_ts >= ? AND bucket_ts <= ? AND request_count > 0", modelName, startTs, endTs)
	if groups != nil {
		query = query.Where(commonGroupCol+" IN ?", groups)
	}
	var buckets []PerfMetricSummaryBucket
	err := query.Group("model_name, " + hourBucket).
		Order("bucket_ts DESC").Limit(limit).Find(&buckets).Error
	return buckets, err
}

func DeletePerfMetricsBefore(cutoffTs int64) error {
	if cutoffTs <= 0 {
		return nil
	}
	return DB.Where("bucket_ts < ?", cutoffTs).Delete(&PerfMetric{}).Error
}

func PerfMetricStartTime(hours int) int64 {
	if hours <= 0 {
		hours = 24
	}
	return time.Now().Add(-time.Duration(hours) * time.Hour).Unix()
}
