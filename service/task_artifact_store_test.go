package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
