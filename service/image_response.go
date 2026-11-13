package service

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/tidwall/gjson"
)

type imageResponseProjection struct {
	artifacts    []model.TaskImageArtifact
	seen         map[[32]byte]bool
	allowBareURL bool
}

// ExtractImageResponseArtifacts reads image payloads in the output formats
// served by Images, Gemini, Responses and Chat. It never inspects request/input
// members. Inline sources are transient: callers must remove them before SQL.
// These artifacts are for storage/preview only, never billing quantities.
func ExtractImageResponseArtifacts(body []byte, allowBareURL bool) []model.TaskImageArtifact {
	projection := imageResponseProjection{seen: make(map[[32]byte]bool), allowBareURL: allowBareURL}
	if gjson.ValidBytes(body) {
		projection.visit(gjson.ParseBytes(body), "", 0)
	}
	return projection.artifacts
}

func (p *imageResponseProjection) add(source, mimeType string) {
	source = strings.TrimSpace(source)
	if source == "" || len(p.artifacts) >= 64 {
		return
	}
	if mimeType != "image/jpeg" && mimeType != "image/webp" && mimeType != "image/gif" && mimeType != "image/avif" {
		mimeType = "image/png"
	}
	if !strings.HasPrefix(source, "data:image/") {
		u, err := url.Parse(source)
		if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || len(source) > 16384 {
			return
		}
	}
	identity := sha256.Sum256([]byte(source))
	if p.seen[identity] {
		return
	}
	p.seen[identity] = true
	p.artifacts = append(p.artifacts, model.TaskImageArtifact{Key: fmt.Sprintf("image-%d", len(p.artifacts)+1), MimeType: mimeType, SourceURL: source})
}

func (p *imageResponseProjection) visit(value gjson.Result, parent string, depth int) {
	if depth > 32 || len(p.artifacts) >= 64 {
		return
	}
	if value.IsArray() {
		value.ForEach(func(_, item gjson.Result) bool { p.visit(item, parent, depth+1); return len(p.artifacts) < 64 })
		return
	}
	if !value.IsObject() {
		return
	}
	typ := value.Get("type").String()
	if strings.Contains(typ, "partial_image") || strings.Contains(typ, "delta") {
		return
	}
	mimeType := value.Get("mimeType").String()
	if mimeType == "" {
		mimeType = value.Get("mime_type").String()
	}
	if mimeType == "" {
		switch value.Get("output_format").String() {
		case "jpeg", "jpg":
			mimeType = "image/jpeg"
		case "webp":
			mimeType = "image/webp"
		}
	}
	if encoded := value.Get("b64_json").String(); encoded != "" {
		if mimeType == "" {
			mimeType = "image/png"
		}
		p.add("data:"+mimeType+";base64,"+encoded, mimeType)
	} else if encoded := value.Get("bytesBase64Encoded").String(); encoded != "" {
		p.add("data:image/png;base64,"+encoded, "image/png")
	} else if typ == "image_generation_call" && value.Get("result").Type == gjson.String {
		p.add("data:image/png;base64,"+value.Get("result").String(), "image/png")
	} else if source := value.Get("url").String(); source != "" &&
		(typ == "image_generation.partial_succeeded" || typ == "image_generation.completed" || typ == "image_edit.completed" || p.allowBareURL &&
			(parent == "data" || parent == "images" || parent == "results" || parent == "predictions" || parent == "artifacts")) {
		p.add(source, mimeType)
	}
	if source := value.Get("image"); p.allowBareURL && source.Type == gjson.String {
		p.add(source.String(), mimeType)
	}
	for _, key := range []string{"inlineData", "inline_data"} {
		inline := value.Get(key)
		mime := inline.Get("mimeType").String()
		if mime == "" {
			mime = inline.Get("mime_type").String()
		}
		if strings.HasPrefix(mime, "image/") && inline.Get("data").String() != "" {
			p.add("data:"+mime+";base64,"+inline.Get("data").String(), mime)
		}
	}
	if imageURL := value.Get("image_url"); imageURL.Exists() {
		source := imageURL.String()
		if imageURL.IsObject() {
			source = imageURL.Get("url").String()
		}
		p.add(source, mimeType)
	}
	for _, key := range []string{"data", "images", "candidates", "content", "parts", "predictions", "choices", "message", "delta", "output", "response", "results", "result", "events", "item", "artifacts"} {
		if child := value.Get(key); child.IsObject() || child.IsArray() {
			p.visit(child, key, depth+1)
		}
	}
}
