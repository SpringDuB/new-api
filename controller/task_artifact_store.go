package controller

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func GetTaskArtifactStoreSettings(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	config := system_setting.LoadTaskArtifactStoreConfig()
	common.ApiSuccess(c, gin.H{"mode": config.Mode, "endpoint": config.OSSEndpoint, "bucket": config.OSSBucket, "region": config.OSSRegion, "prefix": config.OSSPrefix, "presign_ttl": config.OSSPresignTTLSeconds, "max_mb": config.OSSMaxBytes >> 20, "credentials_configured": config.OSSAccessKey != "" && config.OSSSecretKey != ""})
}

func UpdateTaskArtifactStoreSettings(c *gin.Context) {
	var request struct {
		Mode       string `json:"mode"`
		Endpoint   string `json:"endpoint"`
		Bucket     string `json:"bucket"`
		Region     string `json:"region"`
		Prefix     string `json:"prefix"`
		AccessKey  string `json:"access_key"`
		SecretKey  string `json:"secret_key"`
		PresignTTL int    `json:"presign_ttl"`
		MaxMB      int64  `json:"max_mb"`
	}
	if common.DecodeJson(c.Request.Body, &request) != nil || request.MaxMB < 1 || request.MaxMB > 4096 || request.Mode != "upstream" && request.Mode != "oss" {
		common.ApiErrorT(c, "Invalid task artifact storage configuration")
		return
	}
	config := system_setting.LoadTaskArtifactStoreConfig()
	config.Mode = request.Mode
	config.OSSEndpoint = request.Endpoint
	config.OSSBucket = request.Bucket
	config.OSSRegion = request.Region
	config.OSSPrefix = request.Prefix
	config.OSSPresignTTLSeconds = request.PresignTTL
	config.OSSMaxBytes = request.MaxMB << 20
	if request.AccessKey != "" {
		config.OSSAccessKey = request.AccessKey
	}
	if request.SecretKey != "" {
		config.OSSSecretKey = request.SecretKey
	}
	if system_setting.ValidateTaskArtifactStoreConfig(config) != nil {
		common.ApiErrorT(c, "Invalid task artifact storage configuration")
		return
	}
	encoded, err := common.Marshal(config)
	if err != nil {
		common.ApiErrorT(c, "Failed to save task artifact storage settings")
		return
	}
	if model.UpdateOptionsBulk(map[string]string{system_setting.TaskArtifactStoreOptionKey: string(encoded)}) != nil {
		common.ApiErrorT(c, "Failed to save task artifact storage settings")
		return
	}
	GetTaskArtifactStoreSettings(c)
}

func TestTaskArtifactStoreConnection(c *gin.Context) {
	store := service.GetTaskArtifactStore()
	probe, ok := store.(interface{ Probe(context.Context) error })
	if !ok || !store.Enabled() {
		common.ApiErrorT(c, "Task artifact storage is not enabled")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if probe.Probe(ctx) != nil {
		common.ApiErrorT(c, "OSS connection test failed; check endpoint, bucket and permissions")
		return
	}
	common.ApiSuccessT(c, "OSS connection test passed", nil)
}
