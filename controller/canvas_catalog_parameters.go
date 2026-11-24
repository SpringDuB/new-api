package controller

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	dto "github.com/QuantumNous/new-api/relaykit/dto"
)

type canvasMediaParameter struct {
	Values   []string `json:"values,omitempty"`
	Default  string   `json:"default,omitempty"`
	Required bool     `json:"required,omitempty"`
	Min      int      `json:"min,omitempty"`
	Max      int      `json:"max,omitempty"`
	Inline   bool     `json:"inline,omitempty"`
}

var (
	canvasParamHeading = regexp.MustCompile(`(?:^|[\n;；])\s*([a-z][a-z0-9_]*)\s*[:：]\s*`)
	canvasParamOptions = regexp.MustCompile(`(?i)(?:可选\s*|options:\s*)([^，,;；]+)`)
	canvasParamDefault = regexp.MustCompile(`(?i)(?:默认\s*|default\s+)([a-z0-9:.\-]+)`)
	canvasParamRange   = regexp.MustCompile(`(-?\d+)\s*[–—~至-]\s*(\d+)`)
	canvasParamMax     = regexp.MustCompile(`(?i)(?:最多\s*|需传入\s*|up to\s+|exactly\s+)(\d+)`)
	canvasParamValue   = regexp.MustCompile(`(?i)^[a-z0-9:.\-]+$`)
)

// The model detail page already publishes these named parameter notes. Only
// known request fields and their machine-readable values cross the bridge.
func canvasDocumentedParameters(description, kind string) map[string]canvasMediaParameter {
	result := make(map[string]canvasMediaParameter)
	headings := canvasParamHeading.FindAllStringSubmatchIndex(description, -1)
	for i, heading := range headings {
		name := description[heading[2]:heading[3]]
		end := len(description)
		if i+1 < len(headings) {
			end = headings[i+1][0]
		}
		note := strings.TrimSpace(description[heading[1]:end])
		if strings.Contains(note, "当前模型不支持") || strings.Contains(note, "not supported by this model") {
			continue
		}
		param := canvasMediaParameter{Required: strings.Contains(note, "，必填") || strings.Contains(note, ", required")}
		if match := canvasParamDefault.FindStringSubmatch(note); len(match) > 0 {
			param.Default = match[1]
		}
		if match := canvasParamOptions.FindStringSubmatch(note); len(match) > 0 {
			for option := range strings.SplitSeq(match[1], "/") {
				value, _, _ := strings.Cut(strings.TrimSpace(option), "（")
				value = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(value), " 秒"), " seconds")
				if canvasParamValue.MatchString(value) && !slices.Contains(param.Values, value) {
					param.Values = append(param.Values, value)
				}
			}
		}
		switch name {
		case "duration", "seconds", "n":
			if (name == "n") != (kind == "image") {
				continue
			}
			upper := relaycommon.MaxTaskDurationSeconds
			if name == "n" {
				upper = dto.MaxImageN
			} else {
				name = "duration"
			}
			if match := canvasParamRange.FindStringSubmatch(note); len(param.Values) == 0 && len(match) > 0 {
				param.Min, _ = strconv.Atoi(match[1])
				param.Max, _ = strconv.Atoi(match[2])
				if param.Min < 1 || param.Max < param.Min || param.Max > upper {
					continue
				}
			}
			param.Values = slices.DeleteFunc(param.Values, func(value string) bool {
				n, err := strconv.Atoi(value)
				return err != nil || n < 1 || n > upper
			})
			if param.Max == 0 && len(param.Values) == 0 {
				continue
			}
		case "first_frame", "last_frame":
			if kind != "video" {
				continue
			}
			param.Max = 1
			if param.Required {
				param.Min = 1
			}
		case "images", "videos", "audios":
			if kind == "image" && name != "images" {
				continue
			}
			// Without a declared count, permit one, not an inferred family-wide limit.
			param.Max = 1
			if match := canvasParamMax.FindStringSubmatch(note); len(match) > 0 {
				param.Max, _ = strconv.Atoi(match[1])
			} else if match := canvasParamRange.FindStringSubmatch(note); len(match) > 0 {
				param.Min, _ = strconv.Atoi(match[1])
				param.Max, _ = strconv.Atoi(match[2])
			}
			if param.Max < 1 || param.Max > 64 || param.Min < 0 || param.Min > param.Max {
				continue
			}
			if param.Required && param.Min == 0 {
				param.Min = 1
			}
			param.Inline = strings.Contains(note, "Base64 data URL")
		case "aspect_ratio", "resolution", "quality", "output_format":
			if len(param.Values) == 0 || ((name == "quality" || name == "output_format") && kind != "image") {
				continue
			}
			if !slices.Contains(param.Values, param.Default) {
				param.Default = param.Values[0]
			}
		case "audio", "generate_audio", "watermark":
			if kind != "video" || (!strings.Contains(note, "布尔类型") && !strings.Contains(note, ", boolean")) {
				continue
			}
			param.Values = []string{"false", "true"}
		default:
			continue
		}
		result[name] = param
	}
	return result
}

func canvasModelIcon(family, icon string) string {
	switch f := strings.ToLower(family); {
	case strings.HasPrefix(f, "seedance-"), f == "seedream":
		return "ByteDance.Color"
	case f == "wan3":
		return "Qwen.Color"
	case f == "minimax h3":
		return "Minimax.Color"
	case f == "gemini omni", f == "banana":
		return "Gemini.Color"
	case f == "gpt-image-2":
		return "OpenAI"
	case f == "muse", f == "meta", strings.HasPrefix(f, "llama"):
		return "Meta.Color"
	default:
		return icon
	}
}
