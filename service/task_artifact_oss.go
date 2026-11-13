package service

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/gin-gonic/gin"
)

type ossArtifactStore struct {
	client   *oss.Client
	config   system_setting.TaskArtifactStoreConfig
	readOnly bool
}

var ossProbeCodePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,63}$`)
var ossProbeRequestIDPattern = regexp.MustCompile(`^[A-Fa-f0-9]{16,64}$`)

func newOSSArtifactStore(config system_setting.TaskArtifactStoreConfig, client *http.Client) (TaskArtifactStore, error) {
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	noRedirects := *client
	noRedirects.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	endpoint, err := url.Parse(config.OSSEndpoint)
	if err != nil {
		return nil, errors.New("invalid OSS endpoint")
	}
	settings := oss.LoadDefaultConfig().WithRegion(config.OSSRegion).WithEndpoint(config.OSSEndpoint).
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.OSSAccessKey, config.OSSSecretKey)).
		WithHttpClient(&noRedirects).WithRetryMaxAttempts(1).WithSignatureVersion(oss.SignatureVersionV4)
	if net.ParseIP(endpoint.Hostname()) != nil {
		settings.WithUsePathStyle(true)
	}
	return &ossArtifactStore{client: oss.NewClient(settings), config: config}, nil
}

func (s *ossArtifactStore) Enabled() bool { return !s.readOnly }

// Probe creates and removes an isolated object only when root explicitly
// requests a connection test. It verifies read, write and delete permissions.
func (s *ossArtifactStore) Probe(ctx context.Context) error {
	if s.readOnly {
		return ErrTaskArtifactStoreDisabled
	}
	key := path.Join(s.config.OSSPrefix, ".connection-test", fmt.Sprint(time.Now().UnixNano()))
	_, err := s.client.PutObject(ctx, &oss.PutObjectRequest{Bucket: oss.Ptr(s.config.OSSBucket), Key: oss.Ptr(key), Body: strings.NewReader("new-api OSS connection test"), Acl: oss.ObjectACLPrivate, ForbidOverwrite: oss.Ptr("true")})
	if err != nil {
		return taskArtifactOSSProbeError(ctx, "PutObject", err)
	}
	_, headErr := s.client.HeadObject(ctx, &oss.HeadObjectRequest{Bucket: oss.Ptr(s.config.OSSBucket), Key: oss.Ptr(key)})
	_, deleteErr := s.client.DeleteObject(ctx, &oss.DeleteObjectRequest{Bucket: oss.Ptr(s.config.OSSBucket), Key: oss.Ptr(key)})
	if deleteErr != nil {
		cleanupErr := taskArtifactOSSProbeError(ctx, "DeleteObject", deleteErr)
		if headErr == nil {
			return cleanupErr
		}
	}
	if headErr != nil {
		return taskArtifactOSSProbeError(ctx, "HeadObject", headErr)
	}
	return nil
}

// Preserve diagnostic identifiers only. SDK error text, XML snapshots and
// request targets may contain credentials and are never logged or returned.
func taskArtifactOSSProbeError(ctx context.Context, operation string, err error) error {
	status := 0
	code, requestID := "RequestFailed", "-"
	var upstream *oss.ServiceError
	var dns *net.DNSError
	var certificate *tls.CertificateVerificationError
	var network net.Error
	switch {
	case errors.As(err, &upstream):
		status = upstream.StatusCode
		code = "UnknownServiceError"
		if ossProbeCodePattern.MatchString(upstream.Code) {
			code = upstream.Code
		} else if text := http.StatusText(status); text != "" {
			code = strings.ReplaceAll(text, " ", "")
		}
		id := upstream.RequestID
		if id == "" {
			id = upstream.Headers.Get("X-Oss-Request-Id")
		}
		if ossProbeRequestIDPattern.MatchString(id) {
			requestID = id
		}
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &network) && network.Timeout():
		code = "RequestTimeout"
	case errors.As(err, &dns):
		code = "DNSLookupFailed"
	case errors.As(err, &certificate):
		code = "TLSCertificateInvalid"
	case errors.Is(err, context.Canceled):
		code = "RequestCanceled"
	case errors.As(err, &network):
		code = "NetworkError"
	}
	logger.LogWarn(ctx, common.LogText("task artifact OSS probe failed operation=%s http=%d code=%s request_id=%s", operation, status, code, requestID))
	return common.NewMessage("OSS connection test failed at {{operation}} (HTTP {{status}}, code {{code}}, request ID {{request_id}})", map[string]any{
		"operation": operation, "status": status, "code": code, "request_id": requestID,
	})
}

func (s *ossArtifactStore) objectKey(task *model.Task, artifactKey string) (string, error) {
	if task == nil || task.TaskID == "" || task.UserId <= 0 || artifactKey == "" || len(artifactKey) > 128 || strings.ContainsAny(artifactKey, "/\\\x00\r\n") || artifactKey == "." || artifactKey == ".." {
		return "", errors.New("invalid stored artifact identity")
	}
	identity := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s", task.UserId, task.ID, task.TaskID)))
	return path.Join(s.config.OSSPrefix, fmt.Sprintf("%x", identity), artifactKey), nil
}

func (s *ossArtifactStore) Resolve(task *model.Task, artifactKey string) (*StoredArtifactRef, error) {
	key, err := s.objectKey(task, artifactKey)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := s.client.HeadObject(ctx, &oss.HeadObjectRequest{Bucket: oss.Ptr(s.config.OSSBucket), Key: oss.Ptr(key)})
	if err != nil {
		var upstream *oss.ServiceError
		if errors.As(err, &upstream) && upstream.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, errors.New("OSS object lookup failed")
	}
	return &StoredArtifactRef{Backend: "oss", Endpoint: s.config.OSSEndpoint, Region: s.config.OSSRegion, Bucket: s.config.OSSBucket, Prefix: s.config.OSSPrefix, ObjectKey: key, ArtifactKey: artifactKey, MimeType: oss.ToString(result.ContentType), Size: result.ContentLength}, nil
}

func (s *ossArtifactStore) Persist(ctx context.Context, task *model.Task, artifact types.TaskArtifact, content io.Reader) (*StoredArtifactRef, error) {
	if s.readOnly {
		return nil, ErrTaskArtifactStoreDisabled
	}
	key, err := s.objectKey(task, artifact.Key)
	if err != nil {
		return nil, err
	}
	if ref, err := s.Resolve(task, artifact.Key); err != nil || ref != nil {
		return ref, err
	}
	file, err := os.CreateTemp("", "new-api-task-artifact-*")
	if err != nil {
		return nil, errors.New("artifact spool unavailable")
	}
	defer os.Remove(file.Name())
	defer file.Close()
	size, err := io.Copy(file, io.LimitReader(content, s.config.OSSMaxBytes+1))
	if err != nil || size == 0 || size > s.config.OSSMaxBytes {
		return nil, errors.New("artifact download incomplete or exceeds storage limit")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.New("artifact spool unavailable")
	}
	mimeType := artifact.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	_, err = s.client.PutObject(ctx, &oss.PutObjectRequest{Bucket: oss.Ptr(s.config.OSSBucket), Key: oss.Ptr(key), Body: file, ContentLength: oss.Ptr(size), ContentType: oss.Ptr(mimeType), Acl: oss.ObjectACLPrivate, ForbidOverwrite: oss.Ptr("true")})
	if err != nil {
		var upstream *oss.ServiceError
		if errors.As(err, &upstream) && upstream.StatusCode == http.StatusConflict {
			return s.Resolve(task, artifact.Key)
		}
		return nil, errors.New("OSS object upload failed")
	}
	return &StoredArtifactRef{Backend: "oss", Endpoint: s.config.OSSEndpoint, Region: s.config.OSSRegion, Bucket: s.config.OSSBucket, Prefix: s.config.OSSPrefix, ObjectKey: key, ArtifactKey: artifact.Key, MimeType: mimeType, Size: size}, nil
}

func (s *ossArtifactStore) SignedURL(ctx context.Context, task *model.Task, ref *StoredArtifactRef, method string) (string, error) {
	if ref == nil || ref.Backend != "oss" || ref.Bucket != s.config.OSSBucket {
		return "", errors.New("invalid stored artifact")
	}
	key, err := s.objectKey(task, ref.ArtifactKey)
	if err != nil || ref.ObjectKey != key {
		return "", errors.New("invalid stored artifact ownership")
	}
	var request any = &oss.GetObjectRequest{Bucket: oss.Ptr(ref.Bucket), Key: oss.Ptr(ref.ObjectKey)}
	if method == http.MethodHead {
		request = &oss.HeadObjectRequest{Bucket: oss.Ptr(ref.Bucket), Key: oss.Ptr(ref.ObjectKey)}
	} else if method != http.MethodGet {
		return "", errors.New("unsupported artifact method")
	}
	result, err := s.client.Presign(ctx, request, oss.PresignExpires(time.Duration(s.config.OSSPresignTTLSeconds)*time.Second))
	if err != nil {
		return "", errors.New("OSS URL signing failed")
	}
	return result.URL, nil
}

func (s *ossArtifactStore) Serve(c *gin.Context, task *model.Task, ref *StoredArtifactRef) error {
	url, err := s.SignedURL(c.Request.Context(), task, ref, c.Request.Method)
	if err != nil {
		return err
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Location", url)
	c.Status(http.StatusTemporaryRedirect)
	c.Writer.WriteHeaderNow()
	return nil
}
