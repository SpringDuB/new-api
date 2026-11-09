package jsplugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Calcium-Ion/moejs"
)

// Fetcher sends one utils.fetch request for the hook call whose context
// carries it. The host binds it to the executing channel; the engine passes a
// context already bounded by the call's fetch budget, so the fetcher only sends.
type Fetcher func(ctx context.Context, request FetchRequest) (FetchResponse, error)

type FetchRequest struct {
	Hook      string // the calling hook, filled by the engine for logs and errors
	URL       string
	Method    string
	Headers   map[string]string
	FileRef   string // utils.uploadFile only; bytes remain in the host request.
	JSONField string // utils.uploadFile only; optional Base64 JSON field instead of multipart.
}

type FetchResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

// The utils.fetch limits are part of the published plugin contract
// (docs/plugin-api/v1.md). A fetcher reads at most MaxFetchBodyBytes+1 bytes
// so that the engine can reject a longer body.
const (
	maxFetchesPerCall   = 4
	fetchTimeout        = 15 * time.Second
	maxFetchTimePerCall = 30 * time.Second
	MaxFetchBodyBytes   = 1 << 20
	maxFetchURLBytes    = 64 << 10
)

type fetcherContextKey struct{}
type fileUploaderContextKey struct{}
type mediaProberContextKey struct{}

// WithMediaProber binds a credentialless, request-scoped media duration reader.
func WithMediaProber(ctx context.Context, prober Fetcher) context.Context {
	return context.WithValue(ctx, mediaProberContextKey{}, prober)
}

// WithFetcher makes utils.fetch available to the hook called with ctx.
func WithFetcher(ctx context.Context, fetcher Fetcher) context.Context {
	return context.WithValue(ctx, fetcherContextKey{}, fetcher)
}

// WithFileUploader binds request-scoped multipart uploads during submit building.
// It must not be bound to decoders, polling, renderers, or usage hooks.
func WithFileUploader(ctx context.Context, uploader Fetcher) context.Context {
	return context.WithValue(ctx, fileUploaderContextKey{}, uploader)
}

// runtimeCall is the hook call a runtime is running, as its host functions
// see it. The engine sets it for one call and clears it when the call ends,
// so an idle pooled runtime keeps no request data.
type runtimeCall struct {
	engine   *Engine
	runtime  *moejs.Runtime
	ctx      context.Context
	hook     string
	fetcher  Fetcher
	uploader Fetcher
	prober   Fetcher
	watchdog *runtimeWatchdog
	// holdsSlot and parked are what the call holds of the engine's
	// semaphore and parked tokens; release returns exactly that.
	holdsSlot bool
	parked    bool
	fetches   int
	uploads   int
	probes    int
	fetchTime time.Duration
}

func (c *runtimeCall) release() {
	if c.holdsSlot {
		<-c.engine.semaphore
		c.holdsSlot = false
	}
	if c.parked {
		<-c.engine.parked
		c.parked = false
	}
}

// fetch is utils.fetch. While the fetcher waits on the network the call's
// JavaScript clock is stopped and its execution slot is lent to other calls.
// Errors name the host only: hook errors reach API callers, and paths and
// queries can carry credentials.
func (c *runtimeCall) fetch(r *moejs.Realm, argument moejs.Value) (moejs.Value, error) {
	if c.fetcher == nil {
		return moejs.Undefined(), fmt.Errorf("utils.fetch is not available in %s", c.hook)
	}
	request, host, wantBytes, err := fetchRequest(r, argument)
	if err != nil {
		return moejs.Undefined(), err
	}
	request.Hook = c.hook
	if c.fetches >= maxFetchesPerCall {
		return moejs.Undefined(), fmt.Errorf("utils.fetch allows at most %d requests per hook call", maxFetchesPerCall)
	}
	c.fetches++
	return c.performFetch(r, request, host, wantBytes, c.fetcher, "utils.fetch")
}

func (c *runtimeCall) uploadFile(r *moejs.Realm, argument moejs.Value) (moejs.Value, error) {
	if c.uploader == nil || c.hook != "buildSubmitRequest" {
		return moejs.Undefined(), fmt.Errorf("utils.uploadFile is not available in %s", c.hook)
	}
	exported, err := r.ToGoStrict(argument)
	if err != nil {
		return moejs.Undefined(), err
	}
	members, ok := exported.(map[string]any)
	if !ok {
		return moejs.Undefined(), r.TypeError("utils.uploadFile request must be an object")
	}
	for name := range members {
		if name != "url" && name != "headers" && name != "fileRef" && name != "jsonField" {
			return moejs.Undefined(), r.TypeError("utils.uploadFile request has unsupported member %q", name)
		}
	}
	ref, _ := members["fileRef"].(string)
	if _, _, valid := ParseFileReference(ref); !valid {
		return moejs.Undefined(), r.TypeError("utils.uploadFile requires an uploaded file reference")
	}
	jsonField := ""
	if value, exists := members["jsonField"]; exists {
		var valid bool
		jsonField, valid = value.(string)
		if !valid || len(jsonField) == 0 || len(jsonField) > 64 {
			return moejs.Undefined(), r.TypeError("utils.uploadFile jsonField must contain 1 to 64 ASCII letters, digits or underscores")
		}
		for _, char := range jsonField {
			if char != '_' && (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
				return moejs.Undefined(), r.TypeError("utils.uploadFile jsonField must contain 1 to 64 ASCII letters, digits or underscores")
			}
		}
	}
	// Reuse the ordinary URL/header validation without enabling writes in fetch.
	validated, err := r.FromGo(map[string]any{"url": members["url"], "headers": members["headers"]})
	if err != nil {
		return moejs.Undefined(), err
	}
	request, host, _, err := fetchRequest(r, validated)
	if err != nil {
		return moejs.Undefined(), err
	}
	if c.uploads >= 16 {
		return moejs.Undefined(), fmt.Errorf("utils.uploadFile allows at most 16 uploads per hook call")
	}
	c.uploads++
	request.Method, request.FileRef, request.JSONField = http.MethodPost, ref, jsonField
	return c.performFetch(r, request, host, false, c.uploader, "utils.uploadFile")
}

func (c *runtimeCall) mediaDuration(r *moejs.Realm, argument moejs.Value) (moejs.Value, error) {
	if c.prober == nil || (c.hook != "extractUsage" && c.hook != "buildSubmitRequest") {
		return moejs.Undefined(), fmt.Errorf("utils.mediaDuration is not available in %s", c.hook)
	}
	value, err := r.ToGoStrict(argument)
	if err != nil {
		return moejs.Undefined(), err
	}
	members, ok := value.(map[string]any)
	if !ok || len(members) != 1 {
		return moejs.Undefined(), r.TypeError("mediaDuration requires one url or fileRef")
	}
	request := FetchRequest{Method: http.MethodGet}
	host := "uploaded video"
	if raw, ok := members["url"].(string); ok {
		parsed, err := url.Parse(raw)
		if err != nil || len(raw) > maxFetchURLBytes || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return moejs.Undefined(), r.TypeError("mediaDuration requires an absolute HTTP(S) URL without credentials")
		}
		request.URL, host = raw, parsed.Host
	} else if ref, ok := members["fileRef"].(string); ok {
		if _, _, valid := ParseFileReference(ref); !valid {
			return moejs.Undefined(), r.TypeError("mediaDuration requires a request file reference")
		}
		request.FileRef = ref
	} else {
		return moejs.Undefined(), r.TypeError("mediaDuration requires one url or fileRef")
	}
	if c.probes >= 16 {
		return moejs.Undefined(), fmt.Errorf("mediaDuration allows at most 16 references")
	}
	c.probes++
	result, err := c.performFetch(r, request, host, false, c.prober, "utils.mediaDuration")
	if err != nil {
		return moejs.Undefined(), err
	}
	decoded, err := r.ToGoStrict(result)
	if err != nil {
		return moejs.Undefined(), err
	}
	return r.FromGo(decoded.(map[string]any)["body"])
}

// performFetch shares the bounded network wait and JSON response handling.
func (c *runtimeCall) performFetch(r *moejs.Realm, request FetchRequest, host string, wantBytes bool, fetcher Fetcher, tool string) (moejs.Value, error) {
	request.Hook = c.hook
	remaining := maxFetchTimePerCall - c.fetchTime
	if remaining <= 0 {
		return moejs.Undefined(), fmt.Errorf("%s requests used up their %s per hook call", tool, maxFetchTimePerCall)
	}
	timeout, timeoutCause := fetchTimeout, fmt.Errorf("%s to %s timed out after %s", tool, host, fetchTimeout)
	if remaining < fetchTimeout {
		timeout, timeoutCause = remaining, fmt.Errorf("%s to %s ran past the %s of requests per hook call", tool, host, maxFetchTimePerCall)
	}

	if !c.watchdog.pause() {
		// The call timed out or its caller left; the interrupt ends it.
		return moejs.Undefined(), r.CheckInterrupt()
	}
	c.yieldSlot()
	fetchContext, cancel := context.WithTimeoutCause(c.ctx, timeout, timeoutCause)
	started := time.Now()
	response, err := fetcher(fetchContext, request)
	c.fetchTime += time.Since(started)
	if err != nil && fetchContext.Err() != nil {
		err = context.Cause(fetchContext)
	}
	cancel()
	if !c.reclaimSlot() {
		// JavaScript must not run without a slot, and a thrown error could be
		// caught; interrupting discards the runtime as a timeout does.
		cause := context.Cause(c.ctx)
		if cause == nil {
			cause = fmt.Errorf("plugin call could not resume after utils.fetch: %w", ErrCallAdmissionTimeout)
		}
		c.runtime.Interrupt(cause)
		return moejs.Undefined(), r.CheckInterrupt()
	}
	c.watchdog.resume()
	if interrupted := r.CheckInterrupt(); interrupted != nil {
		return moejs.Undefined(), interrupted
	}

	if err != nil {
		var transportErr *url.Error
		if errors.As(err, &transportErr) {
			err = transportErr.Err
		}
		return moejs.Undefined(), fmt.Errorf("%s to %s failed: %v", tool, host, err)
	}
	if len(response.Body) > MaxFetchBodyBytes {
		return moejs.Undefined(), fmt.Errorf("%s response from %s exceeds %d bytes", tool, host, MaxFetchBodyBytes)
	}
	// The result is built from JavaScript objects, not a host map: a hook may
	// return it, or part of it, as it is.
	headers := r.NewObject()
	for name, value := range response.Headers {
		if err = headers.CreateDataPropertyOrThrow(r, r.KeyFromGoString(name), moejs.String(value)); err != nil {
			return moejs.Undefined(), err
		}
	}
	headersValue, err := r.FromGo(headers)
	if err != nil {
		return moejs.Undefined(), err
	}
	body := moejs.String("")
	if wantBytes {
		// An ArrayBuffer over the response bytes, which belong to this call.
		if response.Body == nil {
			response.Body = []byte{}
		}
		if body, err = r.FromGo(response.Body); err != nil {
			return moejs.Undefined(), err
		}
	} else if request.Method != http.MethodHead {
		text := string(response.Body)
		if body, err = c.runtime.ParseJSONString(text); err != nil {
			body = moejs.String(text)
		}
	}
	result := r.NewObject()
	for _, member := range []struct {
		name  string
		value moejs.Value
	}{{"status", moejs.Int(int64(response.Status))}, {"headers", headersValue}, {"body", body}} {
		if err = result.CreateDataPropertyOrThrow(r, r.KeyFromGoString(member.name), member.value); err != nil {
			return moejs.Undefined(), err
		}
	}
	return r.FromGo(result)
}

// fetchRequest reads the utils.fetch argument: {url, method?, headers?,
// responseType?}. wantBytes is a responseType of "bytes": the body is
// returned as an ArrayBuffer instead of parsed JSON or text.
func fetchRequest(r *moejs.Realm, argument moejs.Value) (request FetchRequest, host string, wantBytes bool, err error) {
	if !argument.IsObject() {
		return FetchRequest{}, "", false, r.TypeError("utils.fetch request must be an object")
	}
	exported, err := r.ToGoStrict(argument)
	if err != nil {
		return FetchRequest{}, "", false, err
	}
	members, ok := exported.(map[string]any)
	if !ok {
		return FetchRequest{}, "", false, r.TypeError("utils.fetch request must be an object")
	}
	request = FetchRequest{Method: http.MethodGet}
	for name, value := range members {
		switch name {
		case "url":
			if request.URL, ok = value.(string); !ok {
				return FetchRequest{}, "", false, r.TypeError("utils.fetch url must be a string")
			}
		case "method":
			if value == nil {
				continue
			}
			method, isString := value.(string)
			if !isString {
				return FetchRequest{}, "", false, r.TypeError("utils.fetch method must be a string")
			}
			if method = strings.ToUpper(strings.TrimSpace(method)); method != "" {
				request.Method = method
			}
		case "headers":
			if value == nil {
				continue
			}
			headers, isObject := value.(map[string]any)
			if !isObject {
				return FetchRequest{}, "", false, r.TypeError("utils.fetch headers must be an object")
			}
			request.Headers = make(map[string]string, len(headers))
			for headerName, headerValue := range headers {
				text, isString := headerValue.(string)
				if !isString {
					return FetchRequest{}, "", false, r.TypeError("utils.fetch header %q must be a string", headerName)
				}
				request.Headers[strings.TrimSpace(headerName)] = text
			}
		case "responseType":
			if value == nil {
				continue
			}
			responseType, isString := value.(string)
			if !isString || (responseType != "json" && responseType != "bytes") {
				return FetchRequest{}, "", false, r.TypeError(`utils.fetch responseType must be "json" or "bytes"`)
			}
			wantBytes = responseType == "bytes"
		default:
			return FetchRequest{}, "", false, r.TypeError("utils.fetch request has unsupported member %q", name)
		}
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return FetchRequest{}, "", false, r.TypeError("utils.fetch supports only GET and HEAD")
	}
	if len(request.URL) > maxFetchURLBytes {
		return FetchRequest{}, "", false, r.TypeError("utils.fetch url exceeds %d bytes", maxFetchURLBytes)
	}
	parsed, err := url.Parse(request.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return FetchRequest{}, "", false, r.TypeError("utils.fetch url must be an absolute http(s) URL")
	}
	if err = ValidateRequestHeaders(request.Headers); err != nil {
		return FetchRequest{}, "", false, r.TypeError("utils.fetch: %s", err)
	}
	return request, parsed.Host, wantBytes, nil
}

// yieldSlot lends the call's execution slot to other calls while it waits on
// the network. Parked runtimes are bounded by the engine's parked tokens;
// without one the call keeps its slot and waits holding it.
func (c *runtimeCall) yieldSlot() {
	select {
	case c.engine.parked <- struct{}{}:
	default:
		return
	}
	c.parked = true
	<-c.engine.semaphore
	c.holdsSlot = false
}

// reclaimSlot takes an execution slot back for the call's JavaScript, waiting
// no longer than the caller and fetchTimeout allow.
func (c *runtimeCall) reclaimSlot() bool {
	if c.holdsSlot {
		return true
	}
	timer := time.NewTimer(fetchTimeout)
	defer timer.Stop()
	select {
	case c.engine.semaphore <- struct{}{}:
		c.holdsSlot = true
	case <-c.ctx.Done():
	case <-timer.C:
	}
	<-c.engine.parked
	c.parked = false
	return c.holdsSlot
}
