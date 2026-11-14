package model

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

const ModelSquareDisplayOption = "ModelSquareDisplay"

type ModelSquareDecoration struct {
	Badge      string   `json:"badge,omitempty"`
	BadgeStyle string   `json:"badge_style,omitempty"`
	Color      string   `json:"color,omitempty"`
	Note       string   `json:"note,omitempty"`
	Highlight  bool     `json:"highlight,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// ModelSquareDisplay is site-owned presentation metadata. Concrete model names
// keep wildcard metadata rules and upstream catalog synchronization independent.
type ModelSquareDisplay struct {
	Order  []string                         `json:"order"`
	Models map[string]ModelSquareDecoration `json:"models"`
}

func (display *ModelSquareDisplay) Validate() error {
	if len(display.Order) > 10000 || len(display.Models) > 10000 {
		return common.NewMessage("Model square display supports at most 10000 models")
	}
	seen := make(map[string]bool, len(display.Order))
	for _, name := range display.Order {
		if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 256 || seen[name] {
			return common.NewMessage("Model square order contains an invalid or duplicate model name")
		}
		seen[name] = true
	}
	for name, decoration := range display.Models {
		if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 256 {
			return common.NewMessage("Model square display contains an invalid model name")
		}
		if utf8.RuneCountInString(decoration.Badge) > 24 || utf8.RuneCountInString(decoration.Note) > 120 {
			return common.NewMessage("Model square badges allow 24 characters and notes allow 120 characters")
		}
		if !slices.Contains([]string{"", "ribbon", "burst"}, decoration.BadgeStyle) ||
			!slices.Contains([]string{"", "red", "amber", "blue", "emerald", "violet"}, decoration.Color) {
			return common.NewMessage("Model square display contains an invalid style or color")
		}
		if len(decoration.Tags) > 10 {
			return common.NewMessage("Model square display allows up to 10 additional tags per model")
		}
		for _, tag := range decoration.Tags {
			if strings.TrimSpace(tag) == "" || utf8.RuneCountInString(tag) > 32 {
				return common.NewMessage("Model square tags must contain between 1 and 32 characters")
			}
		}
	}
	return nil
}

func GetModelSquareDisplay() ModelSquareDisplay {
	common.OptionMapRWMutex.RLock()
	value := common.OptionMap[ModelSquareDisplayOption]
	common.OptionMapRWMutex.RUnlock()
	display := ModelSquareDisplay{Order: []string{}, Models: map[string]ModelSquareDecoration{}}
	if value != "" {
		if err := common.UnmarshalJsonStr(value, &display); err != nil {
			common.SysError(common.LogText("failed to decode model square display settings") + ": " + err.Error())
			return ModelSquareDisplay{Order: []string{}, Models: map[string]ModelSquareDecoration{}}
		}
	}
	if display.Order == nil {
		display.Order = []string{}
	}
	if display.Models == nil {
		display.Models = map[string]ModelSquareDecoration{}
	}
	return display
}

// ForModels prevents the public catalog from exposing names in hidden or
// inaccessible groups. Missing models retain their saved settings for later.
func (display ModelSquareDisplay) ForModels(names []string) ModelSquareDisplay {
	visible := make(map[string]bool, len(names))
	result := ModelSquareDisplay{Order: []string{}, Models: map[string]ModelSquareDecoration{}}
	for _, name := range names {
		visible[name] = true
		if decoration, exists := display.Models[name]; exists {
			result.Models[name] = decoration
		}
	}
	for _, name := range display.Order {
		if visible[name] {
			result.Order = append(result.Order, name)
		}
	}
	return result
}
