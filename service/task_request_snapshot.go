package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime"
	"net/url"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const TaskClientRequestContextKey = "task_client_request_snapshot"

// A snapshot is at most 64 KiB including metadata. Oversized bodies are
// omitted in full, never copied as an unsanitized or invalid JSON prefix.
// Retention follows the task row; there is no separate request log/archive.
const MaxTaskClientRequestSnapshotBytes = 64 << 10

var taskRequestURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)
var taskRequestCredentialPattern = regexp.MustCompile(`(?i)\b(?:Bearer|Basic)\s+[A-Za-z0-9._~+/=-]+|\bsk-[A-Za-z0-9_-]{6,}|\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)

type taskRequestRedactor struct {
	secrets  []string
	redacted bool
}

func taskRequestCredentialField(name string) bool {
	name = strings.ToLower(strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			return -1
		}
		return r
	}, name))
	return strings.Contains(name, "authorization") || strings.Contains(name, "cookie") ||
		strings.Contains(name, "apikey") || strings.Contains(name, "secret") ||
		strings.Contains(name, "password") || strings.Contains(name, "credential") ||
		strings.Contains(name, "signature") || strings.HasSuffix(name, "token") ||
		name == "auth" || name == "key" || name == "access" || name == "jwt" || name == "session" || name == "sessionid"
}

func (r *taskRequestRedactor) text(value string) string {
	original := value
	for _, secret := range r.secrets {
		if value == secret {
			value = "[REDACTED]"
		} else if len(secret) >= 6 {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	value = taskRequestCredentialPattern.ReplaceAllString(value, "[REDACTED]")
	value = taskRequestURLPattern.ReplaceAllStringFunc(value, func(raw string) string {
		u, err := url.Parse(raw)
		if err != nil {
			return "[REDACTED URL]"
		}
		u.User = nil
		if u.RawQuery != "" {
			u.RawQuery = "[REDACTED]"
		}
		if u.Fragment != "" {
			u.Fragment = "[REDACTED]"
		}
		return u.String()
	})
	r.redacted = r.redacted || value != original
	return value
}

func (r *taskRequestRedactor) json(raw json.RawMessage, depth int) (json.RawMessage, error) {
	if depth > 32 {
		return nil, errors.New("request snapshot nesting limit")
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		redacted := make(map[string]json.RawMessage, len(object))
		for key, value := range object {
			if taskRequestCredentialField(key) {
				r.redacted = true
				redacted[r.text(key)] = json.RawMessage(`"[REDACTED]"`)
				continue
			}
			encoded, err := r.json(value, depth+1)
			if err != nil {
				return nil, err
			}
			redacted[r.text(key)] = encoded
		}
		return common.Marshal(redacted)
	case "array":
		var values []json.RawMessage
		if err := common.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		for i := range values {
			value, err := r.json(values[i], depth+1)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return common.Marshal(values)
	case "string":
		var value string
		if err := common.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		// Multipart text fields may themselves contain JSON credentials.
		trimmed := bytes.TrimSpace([]byte(value))
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			var nested json.RawMessage
			if common.Unmarshal(trimmed, &nested) == nil {
				encoded, err := r.json(nested, depth+1)
				if err != nil {
					return nil, err
				}
				value = string(encoded)
			}
		}
		return common.Marshal(r.text(value))
	default:
		return raw, nil
	}
}

// CaptureTaskClientRequest runs before plugin decoding or request rewriting.
// Repeated candidate decoding and retries never replace the first snapshot.
// parsedBody uses the host body union; rawJSON, when present, is the client's
// original bytes, preserving numbers beyond JavaScript's integer precision.
func CaptureTaskClientRequest(c *gin.Context, parsedBody any, rawJSON []byte) {
	if c == nil || c.Request == nil || c.Request.Method != "POST" {
		return
	}
	if _, exists := c.Get(TaskClientRequestContextKey); exists {
		return
	}
	redactor := taskRequestRedactor{}
	for name, values := range c.Request.Header {
		if !taskRequestCredentialField(name) {
			continue
		}
		for _, value := range values {
			if value != "" {
				redactor.secrets = append(redactor.secrets, value)
			}
			if _, credential, found := strings.Cut(value, " "); found && credential != "" {
				redactor.secrets = append(redactor.secrets, credential)
			}
		}
	}
	for _, cookie := range c.Request.Cookies() {
		if cookie.Value != "" {
			redactor.secrets = append(redactor.secrets, cookie.Value)
		}
	}
	snapshot := &model.TaskClientRequestSnapshot{
		Method: c.Request.Method, Path: redactor.text(c.Request.URL.Path),
		ContentType: redactor.text(c.GetHeader("Content-Type")), BodyKind: "none",
	}
	if len(snapshot.Path) > 2048 {
		snapshot.Path = "[OMITTED]"
	}
	if len(snapshot.ContentType) > 1024 {
		snapshot.ContentType = "[OMITTED]"
	}
	body, _ := parsedBody.(map[string]any)
	if kind, ok := body["kind"].(string); ok {
		snapshot.BodyKind = kind
	}
	if parsedBody == nil {
		mediaType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
			snapshot.BodyKind = "json"
			storage, err := common.GetBodyStorage(c)
			if err == nil {
				rawJSON, err = storage.Bytes()
			}
			if err != nil {
				snapshot.BodyOmitted = "unavailable"
			}
		}
	}
	var raw []byte
	switch snapshot.BodyKind {
	case "json":
		raw = rawJSON
	case "form", "multipart":
		files := make([]map[string]any, 0)
		if entries, ok := body["files"].([]map[string]any); ok {
			for _, file := range entries {
				files = append(files, map[string]any{"field": file["field"], "filename": file["filename"], "mime_type": file["mimeType"], "size": file["size"]})
			}
		}
		snapshot.FilesOmitted = len(files) > 0
		raw, _ = common.Marshal(map[string]any{"fields": body["fields"], "files": files})
	}
	if len(raw) > MaxTaskClientRequestSnapshotBytes {
		snapshot.BodyOmitted = "size_limit"
	} else if len(raw) > 0 {
		encoded, err := redactor.json(raw, 0)
		if err != nil {
			snapshot.BodyOmitted = "invalid_or_too_deep"
		} else {
			snapshot.Body = encoded
		}
	}
	snapshot.Redacted = redactor.redacted
	encoded, err := common.Marshal(snapshot)
	if err != nil || len(encoded) > MaxTaskClientRequestSnapshotBytes {
		snapshot.Body = nil
		snapshot.BodyOmitted = "size_limit"
	}
	c.Set(TaskClientRequestContextKey, snapshot)
}
