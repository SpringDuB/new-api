package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisabledTaskArtifactStoreHasNoStorageBehavior(t *testing.T) {
	store := GetTaskArtifactStore()
	require.NotNil(t, store)
	assert.False(t, store.Enabled())

	task := &model.Task{TaskID: "task-disabled-store"}
	ref, err := store.Resolve(task, "video")
	require.NoError(t, err)
	assert.Nil(t, ref)

	ref, err = store.Persist(t.Context(), task, types.TaskArtifact{Key: "video", Type: "video"}, strings.NewReader("content"))
	assert.Nil(t, ref)
	assert.ErrorIs(t, err, ErrTaskArtifactStoreDisabled)
	assert.ErrorIs(t, store.Serve(&gin.Context{}, task, &StoredArtifactRef{Backend: "s3"}), ErrTaskArtifactStoreDisabled)
	assert.Same(t, store, GetTaskArtifactStore())
}

var _ TaskArtifactStore = disabledArtifactStore{}

type taskArtifactUploadTransport func(*http.Request) (*http.Response, error)

func (f taskArtifactUploadTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestOSSArtifactTransferAllowsSlowUploadsAndHonorsDeadlines(t *testing.T) {
	for _, test := range []struct {
		name           string
		configured     string
		uploadDuration time.Duration
		callerTimeout  time.Duration
		wantFailure    bool
	}{
		{name: "default permits upload beyond 120 seconds", uploadDuration: 121 * time.Second},
		{name: "configured transfer permits upload beyond five minutes", configured: "600", uploadDuration: 6 * time.Minute},
		{name: "transfer deadline cancels upload", configured: "180", uploadDuration: 181 * time.Second, wantFailure: true},
		{name: "caller deadline still applies", configured: "600", uploadDuration: 121 * time.Second, callerTimeout: 30 * time.Second, wantFailure: true},
		{name: "zero uses bounded default", configured: "0", uploadDuration: 121 * time.Second},
		{name: "oversized value uses bounded default", configured: "86401", uploadDuration: 121 * time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("TASK_ARTIFACT_TRANSFER_TIMEOUT_SECONDS", test.configured)
			synctest.Test(t, func(t *testing.T) {
				var uploaded []byte
				previousTransport := http.DefaultTransport
				http.DefaultTransport = taskArtifactUploadTransport(func(request *http.Request) (*http.Response, error) {
					status := http.StatusNotFound
					if request.Method == http.MethodPut {
						timer := time.NewTimer(test.uploadDuration)
						defer timer.Stop()
						select {
						case <-request.Context().Done():
							return nil, request.Context().Err()
						case <-timer.C:
						}
						var err error
						uploaded, err = io.ReadAll(request.Body)
						if err != nil {
							return nil, err
						}
						status = http.StatusOK
					}
					return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
				})
				t.Cleanup(func() { http.DefaultTransport = previousTransport })
				config := system_setting.TaskArtifactStoreConfig{Mode: "oss", OSSEndpoint: "https://oss-cn-shanghai.aliyuncs.com", OSSRegion: "cn-shanghai", OSSBucket: "task-results", OSSAccessKey: "test-access", OSSSecretKey: "test-secret", OSSPrefix: "tasks/v1", OSSMaxBytes: 1024}
				store, err := newOSSArtifactStore(config, nil)
				require.NoError(t, err)
				ctx := t.Context()
				if test.callerTimeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, test.callerTimeout)
					defer cancel()
				}
				ref, err := store.Persist(ctx, &model.Task{ID: 7, TaskID: "task-slow-upload", UserId: 12}, types.TaskArtifact{Key: "video", Type: "video", MimeType: "video/mp4"}, strings.NewReader("video-bytes"))
				if test.wantFailure {
					require.Error(t, err)
					assert.Nil(t, ref, "a canceled transfer must not publish a stored resource")
					assert.Empty(t, uploaded)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, []byte("video-bytes"), uploaded)
				assert.EqualValues(t, len(uploaded), ref.Size)
			})
		})
	}
}

func TestTaskArtifactArchiveStatusUsesTransferBudget(t *testing.T) {
	t.Setenv("TASK_ARTIFACT_TRANSFER_TIMEOUT_SECONDS", "1800")
	for _, status := range []string{model.TaskArchivePending, model.TaskArchiveRunning} {
		t.Run(status, func(t *testing.T) {
			task := &model.Task{PrivateData: model.TaskPrivateData{ArtifactArchive: &model.TaskArtifactArchive{Status: status, UpdatedAt: time.Now().Add(-10 * time.Minute).Unix()}}}
			assert.Equal(t, status, TaskArtifactArchiveStatus(task), "a slow transfer within budget must not appear failed")
			task.PrivateData.ArtifactArchive.UpdatedAt = time.Now().Add(-32 * time.Minute).Unix()
			assert.Equal(t, model.TaskArchiveFailed, TaskArtifactArchiveStatus(task), "abandoned transfers must still expire")
		})
	}
}

func TestOSSArtifactStoreProbeReportsFailedOperationWithoutSecrets(t *testing.T) {
	var logs bytes.Buffer
	previous := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logs
	t.Cleanup(func() { gin.DefaultErrorWriter = previous })
	for _, test := range []struct {
		name      string
		operation string
		method    string
		code      string
		wantCode  string
	}{
		{"invalid access key", "PutObject", http.MethodPut, "InvalidAccessKeyId", "InvalidAccessKeyId"},
		{"read denied", "HeadObject", http.MethodHead, "AccessDenied", "BadErrorResponse"},
		{"delete denied", "DeleteObject", http.MethodDelete, "AccessDenied", "AccessDenied"},
		{"unsafe error metadata", "PutObject", http.MethodPut, "https://secret.example/?signature=signature-canary", "Forbidden"},
	} {
		t.Run(test.name, func(t *testing.T) {
			logs.Reset()
			methods := make([]string, 0, 3)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				w.Header().Set("X-Oss-Request-Id", "0123456789ABCDEF01234567")
				if r.Method == test.method {
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(http.StatusForbidden)
					_, _ = fmt.Fprintf(w, `<Error><Code>%s</Code><Message>credential-canary https://secret.example/?signature=signature-canary</Message><RequestId>0123456789ABCDEF01234567</RequestId></Error>`, test.code)
					return
				}
				w.Header().Set("ETag", `"test"`)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			store, err := newOSSArtifactStore(system_setting.TaskArtifactStoreConfig{Mode: "oss", OSSEndpoint: server.URL, OSSRegion: "cn-shanghai", OSSBucket: "results", OSSAccessKey: "access-canary", OSSSecretKey: "secret-canary", OSSPrefix: "tasks/v1"}, server.Client())
			require.NoError(t, err)
			err = store.(interface{ Probe(context.Context) error }).Probe(t.Context())
			require.Error(t, err)
			var message *common.Message
			require.True(t, errors.As(err, &message), "the web console receives a translatable diagnostic")
			assert.Equal(t, test.operation, message.Params["operation"])
			assert.Equal(t, http.StatusForbidden, message.Params["status"])
			assert.Equal(t, "0123456789ABCDEF01234567", message.Params["request_id"])
			assert.Equal(t, test.wantCode, message.Params["code"])
			for _, secret := range []string{"access-canary", "secret-canary", "credential-canary", "signature-canary", "secret.example", server.URL} {
				assert.NotContains(t, err.Error(), secret)
				assert.NotContains(t, logs.String(), secret)
			}
			if test.operation == "HeadObject" {
				assert.Equal(t, []string{http.MethodPut, http.MethodHead, http.MethodDelete}, methods, "read failures still clean up the test object")
			}
		})
	}
}

func TestOSSArtifactStoreProbeClassifiesTransportFailuresWithoutSecrets(t *testing.T) {
	var logs bytes.Buffer
	previous := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logs
	t.Cleanup(func() { gin.DefaultErrorWriter = previous })
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{"timeout", context.DeadlineExceeded, "RequestTimeout"},
		{"DNS", &net.DNSError{Err: "credential-canary", Name: "secret.example", IsNotFound: true}, "DNSLookupFailed"},
		{"certificate", &tls.CertificateVerificationError{Err: errors.New("credential-canary")}, "TLSCertificateInvalid"},
		{"canceled", context.Canceled, "RequestCanceled"},
		{"network", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("credential-canary")}, "NetworkError"},
		{"request", &url.Error{Op: "Put", URL: "https://secret.example/?signature=signature-canary", Err: errors.New("credential-canary")}, "NetworkError"},
		{"unknown", errors.New("credential-canary https://secret.example/?signature=signature-canary"), "RequestFailed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			logs.Reset()
			err := taskArtifactOSSProbeError(t.Context(), "PutObject", fmt.Errorf("request: %w", test.err))
			var message *common.Message
			require.ErrorAs(t, err, &message)
			assert.Equal(t, test.code, message.Params["code"])
			assert.Equal(t, 0, message.Params["status"])
			assert.Equal(t, "-", message.Params["request_id"])
			for _, secret := range []string{"secret.example", "signature-canary", "credential-canary"} {
				assert.NotContains(t, err.Error(), secret)
				assert.NotContains(t, logs.String(), secret)
			}
		})
	}
}

func TestOSSArtifactStorePersistsPrivateObjectsAndSignsClientMethods(t *testing.T) {
	var object []byte
	objects := map[string][]byte{}
	puts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotContains(t, r.URL.Path, "../")
		switch r.Method {
		case http.MethodHead:
			stored := objects[r.URL.Path]
			if len(stored) == 0 {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", fmt.Sprint(len(stored)))
			w.Header().Set("Content-Type", "video/mp4")
		case http.MethodPut:
			assert.Contains(t, r.Header.Get("Authorization"), "OSS4-HMAC-SHA256")
			assert.Equal(t, "private", r.Header.Get("X-Oss-Object-Acl"))
			assert.Equal(t, "true", r.Header.Get("X-Oss-Forbid-Overwrite"))
			var err error
			object, err = io.ReadAll(r.Body)
			objects[r.URL.Path] = object
			require.NoError(t, err)
			puts++
			w.Header().Set("ETag", `"test-etag"`)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	config := system_setting.TaskArtifactStoreConfig{Mode: "oss", OSSEndpoint: server.URL, OSSRegion: "cn-hangzhou", OSSBucket: "task-results", OSSAccessKey: "test-access", OSSSecretKey: "test-secret", OSSPrefix: "tasks/v1", OSSPresignTTLSeconds: 900, OSSMaxBytes: 1024}
	store, err := newOSSArtifactStore(config, server.Client())
	require.NoError(t, err)
	require.True(t, store.Enabled())
	task := &model.Task{ID: 7, TaskID: "task-public", UserId: 12}
	ref, err := store.Resolve(task, "video")
	require.NoError(t, err)
	require.Nil(t, ref)
	ref, err = store.Persist(t.Context(), task, types.TaskArtifact{Key: "video", Type: "video", MimeType: "video/mp4"}, bytes.NewReader([]byte("video-bytes")))
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, []byte("video-bytes"), object)
	assert.Equal(t, int64(11), ref.Size)
	_, err = store.Persist(t.Context(), task, types.TaskArtifact{Key: "video", Type: "video"}, strings.NewReader("replacement"))
	require.NoError(t, err)
	assert.Equal(t, 1, puts)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(method, "/content", nil)
		require.NoError(t, store.Serve(c, task, ref))
		signed, err := url.Parse(c.Writer.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "900", signed.Query().Get("x-oss-expires"))
		assert.NotEmpty(t, signed.Query().Get("x-oss-signature"))
		assert.NotContains(t, signed.String(), "test-secret")
		assert.Equal(t, "private, no-store", c.Writer.Header().Get("Cache-Control"))
	}
	other := &model.Task{ID: 8, TaskID: "task-other", UserId: 13}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/content", nil)
	assert.Error(t, store.Serve(c, other, ref))
	config.OSSMaxBytes = 3
	limited, err := newOSSArtifactStore(config, server.Client())
	require.NoError(t, err)
	_, err = limited.Persist(context.Background(), other, types.TaskArtifact{Key: "video", Type: "video"}, strings.NewReader("too large"))
	require.Error(t, err)
	assert.Equal(t, 1, puts, "an oversized/incomplete download is never uploaded")
}
