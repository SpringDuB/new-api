package system_setting

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
)

const (
	TaskArtifactStoreModeUpstream = "upstream"
	TaskArtifactStoreModeS3       = "s3"
	TaskArtifactStoreModeOSS      = "oss"
	TaskArtifactStoreOptionKey    = "TaskArtifactStoreSecret"

	DefaultTaskArtifactStorePresignTTLSeconds = 900
	MaxTaskArtifactStorePresignTTLSeconds     = 7 * 24 * 60 * 60
)

const (
	TaskArtifactStoreModeEnv         = "TASK_ARTIFACT_STORE_MODE"
	TaskArtifactStoreS3EndpointEnv   = "TASK_ARTIFACT_STORE_S3_ENDPOINT"
	TaskArtifactStoreS3BucketEnv     = "TASK_ARTIFACT_STORE_S3_BUCKET"
	TaskArtifactStoreS3RegionEnv     = "TASK_ARTIFACT_STORE_S3_REGION"
	TaskArtifactStoreS3AccessKeyEnv  = "TASK_ARTIFACT_STORE_S3_ACCESS_KEY"
	TaskArtifactStoreS3SecretKeyEnv  = "TASK_ARTIFACT_STORE_S3_SECRET_KEY"
	TaskArtifactStoreS3PrefixEnv     = "TASK_ARTIFACT_STORE_S3_PREFIX"
	TaskArtifactStoreS3PresignTTLEnv = "TASK_ARTIFACT_STORE_S3_PRESIGN_TTL"
)

var (
	taskArtifactStoreBucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	taskArtifactStoreRegionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

// TaskArtifactStoreConfig supports native Aliyun OSS. S3 fields retain their
// existing reserved contract; unsupported S3 mode falls back to upstream.
type TaskArtifactStoreConfig struct {
	Mode                 string `json:"mode"`
	S3Endpoint           string
	S3Bucket             string
	S3Region             string
	S3AccessKey          string
	S3SecretKey          string
	S3Prefix             string
	S3PresignTTLSeconds  int
	OSSEndpoint          string `json:"oss_endpoint"`
	OSSBucket            string `json:"oss_bucket"`
	OSSRegion            string `json:"oss_region"`
	OSSAccessKey         string `json:"oss_access_key"`
	OSSSecretKey         string `json:"oss_secret_key"`
	OSSPrefix            string `json:"oss_prefix"`
	OSSPresignTTLSeconds int    `json:"oss_presign_ttl_seconds"`
	OSSMaxBytes          int64  `json:"oss_max_bytes"`
}

// LoadTaskArtifactStoreConfig reads environment defaults and the atomic root
// settings override. Secrets never appear in generic options responses.
func LoadTaskArtifactStoreConfig() TaskArtifactStoreConfig {
	config := TaskArtifactStoreConfig{
		Mode:                 common.GetEnvOrDefaultString(TaskArtifactStoreModeEnv, TaskArtifactStoreModeUpstream),
		S3Endpoint:           common.GetEnvOrDefaultString(TaskArtifactStoreS3EndpointEnv, ""),
		S3Bucket:             common.GetEnvOrDefaultString(TaskArtifactStoreS3BucketEnv, ""),
		S3Region:             common.GetEnvOrDefaultString(TaskArtifactStoreS3RegionEnv, ""),
		S3AccessKey:          common.GetEnvOrDefaultString(TaskArtifactStoreS3AccessKeyEnv, ""),
		S3SecretKey:          common.GetEnvOrDefaultString(TaskArtifactStoreS3SecretKeyEnv, ""),
		S3Prefix:             common.GetEnvOrDefaultString(TaskArtifactStoreS3PrefixEnv, ""),
		S3PresignTTLSeconds:  common.GetEnvOrDefault(TaskArtifactStoreS3PresignTTLEnv, DefaultTaskArtifactStorePresignTTLSeconds),
		OSSEndpoint:          common.GetEnvOrDefaultString("TASK_ARTIFACT_STORE_OSS_ENDPOINT", ""),
		OSSBucket:            common.GetEnvOrDefaultString("TASK_ARTIFACT_STORE_OSS_BUCKET", ""),
		OSSRegion:            common.GetEnvOrDefaultString("TASK_ARTIFACT_STORE_OSS_REGION", ""),
		OSSAccessKey:         common.GetEnvOrDefaultString("TASK_ARTIFACT_STORE_OSS_ACCESS_KEY", ""),
		OSSSecretKey:         common.GetEnvOrDefaultString("TASK_ARTIFACT_STORE_OSS_SECRET_KEY", ""),
		OSSPrefix:            common.GetEnvOrDefaultString("TASK_ARTIFACT_STORE_OSS_PREFIX", "tasks/v1"),
		OSSPresignTTLSeconds: common.GetEnvOrDefault("TASK_ARTIFACT_STORE_OSS_PRESIGN_TTL", 900),
		OSSMaxBytes:          int64(common.GetEnvOrDefault("TASK_ARTIFACT_STORE_OSS_MAX_MB", 512)) << 20,
	}
	common.OptionMapRWMutex.RLock()
	stored := common.OptionMap[TaskArtifactStoreOptionKey]
	common.OptionMapRWMutex.RUnlock()
	if stored != "" {
		if common.UnmarshalJsonStr(stored, &config) != nil {
			config.Mode = TaskArtifactStoreModeUpstream
		}
	}
	if err := ValidateTaskArtifactStoreConfig(config); err != nil {
		common.SysError(common.LogText("invalid task artifact store configuration: %s; using upstream mode", err.Error()))
		config.Mode = TaskArtifactStoreModeUpstream
		return config
	}
	if config.Mode == TaskArtifactStoreModeS3 {
		common.SysError(common.LogText("task artifact S3 storage is not implemented; using upstream mode"))
		config.Mode = TaskArtifactStoreModeUpstream
	}
	return config
}

// ValidateTaskArtifactStoreConfig performs syntax checks only. It never
// resolves hosts, contacts an endpoint, or verifies credentials.
func ValidateTaskArtifactStoreConfig(config TaskArtifactStoreConfig) error {
	if config.Mode == TaskArtifactStoreModeOSS {
		if config.OSSPresignTTLSeconds <= 0 || config.OSSPresignTTLSeconds > MaxTaskArtifactStorePresignTTLSeconds {
			return errors.New("invalid OSS signed URL lifetime")
		}
		if config.OSSMaxBytes <= 0 || config.OSSMaxBytes > 4<<30 {
			return errors.New("OSS object limit must be between 1 byte and 4 GiB")
		}
		endpoint, err := url.Parse(config.OSSEndpoint)
		if err != nil || endpoint.Scheme != "https" || endpoint.Path != "" && endpoint.Path != "/" {
			return errors.New("OSS endpoint must use HTTPS without a path")
		}
		return ValidateTaskArtifactStoreConfig(TaskArtifactStoreConfig{
			Mode: TaskArtifactStoreModeS3, S3Endpoint: config.OSSEndpoint, S3Bucket: config.OSSBucket,
			S3Region: config.OSSRegion, S3AccessKey: config.OSSAccessKey, S3SecretKey: config.OSSSecretKey,
			S3Prefix: config.OSSPrefix, S3PresignTTLSeconds: config.OSSPresignTTLSeconds,
		})
	}
	if config.Mode != TaskArtifactStoreModeUpstream && config.Mode != TaskArtifactStoreModeS3 {
		return fmt.Errorf("unsupported mode %q", config.Mode)
	}
	if config.S3PresignTTLSeconds <= 0 || config.S3PresignTTLSeconds > MaxTaskArtifactStorePresignTTLSeconds {
		return fmt.Errorf("S3 presign TTL must be between 1 and %d seconds", MaxTaskArtifactStorePresignTTLSeconds)
	}

	requireS3Fields := config.Mode == TaskArtifactStoreModeS3
	if requireS3Fields && config.S3Endpoint == "" {
		return errors.New("S3 endpoint is required")
	}
	if config.S3Endpoint != "" {
		if config.S3Endpoint != strings.TrimSpace(config.S3Endpoint) {
			return errors.New("S3 endpoint must not contain surrounding whitespace")
		}
		endpoint, err := url.Parse(config.S3Endpoint)
		if err != nil || endpoint == nil || endpoint.Host == "" || endpoint.User != nil || endpoint.Opaque != "" {
			return errors.New("S3 endpoint must be an absolute URL without userinfo")
		}
		if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
			return errors.New("S3 endpoint must use http or https")
		}
		if endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
			return errors.New("S3 endpoint must not contain a query or fragment")
		}
	}

	if requireS3Fields && config.S3Bucket == "" {
		return errors.New("S3 bucket is required")
	}
	if config.S3Bucket != "" {
		if !taskArtifactStoreBucketPattern.MatchString(config.S3Bucket) ||
			strings.Contains(config.S3Bucket, "..") || net.ParseIP(config.S3Bucket) != nil {
			return errors.New("S3 bucket syntax is invalid")
		}
	}

	if requireS3Fields && config.S3Region == "" {
		return errors.New("S3 region is required")
	}
	if config.S3Region != "" && !taskArtifactStoreRegionPattern.MatchString(config.S3Region) {
		return errors.New("S3 region syntax is invalid")
	}
	if err := validateTaskArtifactStoreCredential("access key", config.S3AccessKey, 256, requireS3Fields); err != nil {
		return err
	}
	if err := validateTaskArtifactStoreCredential("secret key", config.S3SecretKey, 1024, requireS3Fields); err != nil {
		return err
	}

	if config.S3Prefix != "" {
		if config.S3Prefix != strings.TrimSpace(config.S3Prefix) || len(config.S3Prefix) > 512 ||
			strings.HasPrefix(config.S3Prefix, "/") || strings.Contains(config.S3Prefix, "\\") {
			return errors.New("S3 prefix syntax is invalid")
		}
		for part := range strings.SplitSeq(config.S3Prefix, "/") {
			if part == "." || part == ".." {
				return errors.New("S3 prefix must not contain dot segments")
			}
		}
		for _, character := range config.S3Prefix {
			if unicode.IsControl(character) {
				return errors.New("S3 prefix must not contain control characters")
			}
		}
	}
	return nil
}

func validateTaskArtifactStoreCredential(name, value string, maxLength int, required bool) error {
	if required && value == "" {
		return fmt.Errorf("S3 %s is required", name)
	}
	if value == "" {
		return nil
	}
	if value != strings.TrimSpace(value) || len(value) > maxLength {
		return fmt.Errorf("S3 %s syntax is invalid", name)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("S3 %s syntax is invalid", name)
		}
	}
	return nil
}
