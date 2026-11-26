package controller

import (
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

type canvasCatalogPrice struct {
	Currency                   string                              `json:"currency"`
	Rates                      []canvasPriceRate                   `json:"rates,omitempty"`
	Varies                     bool                                `json:"varies,omitempty"`
	Estimate                   []canvasPriceAmount                 `json:"estimate,omitempty"`
	EstimatesByResolution      map[string][]canvasPriceAmount      `json:"estimates_by_resolution,omitempty"`
	VideoEstimatesByResolution map[string]canvasVideoPriceEstimate `json:"video_estimates_by_resolution,omitempty"`
	VideoTokenEstimate         *jsplugin.VideoTokenEstimate        `json:"video_token_estimate,omitempty"`
}

type canvasVideoPriceEstimate struct {
	WithoutReference []canvasPriceAmount `json:"without_reference,omitempty"`
	WithReference    []canvasPriceAmount `json:"with_reference,omitempty"`
}

type canvasPriceRate struct {
	Unit string  `json:"unit"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

type canvasPriceAmount struct {
	Unit   string  `json:"unit"`
	Amount float64 `json:"amount"`
}

type canvasPriceExpression struct {
	node   ast.Node
	schema map[string]jsplugin.UsageFieldSchema
}

func canvasPriceSummary(item model.Pricing, groups modelListGroups, plugins map[string]bool, resolutions []string, video bool, tokenEstimate *jsplugin.VideoTokenEstimate) *canvasCatalogPrice {
	price := &canvasCatalogPrice{Currency: operation_setting.GetCurrencySymbol()}
	if price.Currency == "" {
		price.Currency = "$"
	}
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	var ratios []float64
	for _, group := range groups.ownerGroups {
		if !slices.Contains(item.EnableGroup, group) && !slices.Contains(item.EnableGroup, "all") {
			continue
		}
		ratio, ok := ratio_setting.GetGroupGroupRatio(groups.userGroup, group)
		if !ok {
			ratio = ratio_setting.GetGroupRatio(group)
		}
		ratios = append(ratios, ratio)
	}
	if len(ratios) == 0 {
		return nil
	}
	var tiers []map[string]float64
	var expressions []canvasPriceExpression
	if item.BillingExpr != "" || len(item.BillingPluginVariants) != 0 {
		if item.BillingExpr != "" && len(item.BillingPluginVariants) == 0 {
			expressions = append(expressions, parseCanvasPriceExpression(item.BillingExpr, item.BillingUsageSchema))
		}
		for _, variant := range item.BillingPluginVariants {
			if plugins[variant.PluginKey] {
				expressions = append(expressions, parseCanvasPriceExpression(variant.BillingExpr, variant.BillingUsageSchema))
			}
		}
		tiers = canvasPriceTiers(expressions, nil)
		if len(tiers) > 0 {
			price.Estimate = canvasPriceEstimate(tiers, ratios, false)
		}
		if len(price.Estimate) == 0 && len(expressions) > 0 {
			for _, resolution := range resolutions {
				if estimate := canvasPriceEstimate(canvasPriceTiers(expressions, map[string]string{"resolution": resolution}), ratios, false); len(estimate) > 0 {
					if price.EstimatesByResolution == nil {
						price.EstimatesByResolution = make(map[string][]canvasPriceAmount)
					}
					price.EstimatesByResolution[resolution] = estimate
				}
			}
		}
		if video && len(price.Estimate) == 0 && len(expressions) > 0 {
			for _, resolution := range resolutions {
				if len(price.EstimatesByResolution[resolution]) > 0 {
					continue
				}
				allowTokens := tokenEstimate != nil && tokenEstimate.TokensPerSecond[resolution] > 0
				quote := canvasVideoPriceEstimate{
					WithoutReference: canvasPriceEstimate(canvasPriceTiers(expressions, map[string]string{"resolution": resolution, "video_input": "none"}), ratios, allowTokens),
					WithReference:    canvasPriceEstimate(canvasPriceTiers(expressions, map[string]string{"resolution": resolution, "video_input": "video"}), ratios, allowTokens),
				}
				if len(quote.WithoutReference) == 0 && len(quote.WithReference) == 0 {
					continue
				}
				if price.VideoEstimatesByResolution == nil {
					price.VideoEstimatesByResolution = make(map[string]canvasVideoPriceEstimate)
				}
				price.VideoEstimatesByResolution[resolution] = quote
				if allowTokens {
					price.VideoTokenEstimate = tokenEstimate
				}
			}
		}
		if len(tiers) == 0 {
			price.Varies = true
			return price
		}
	} else if item.QuotaType == 1 {
		tiers = []map[string]float64{{"request": item.ModelPrice}}
		price.Estimate = canvasPriceEstimate(tiers, ratios, false)
	} else if _, ok, _ := ratio_setting.GetModelRatio(item.ModelName); ok {
		base := item.ModelRatio * 2
		tiers = []map[string]float64{{"input_tokens": base, "output_tokens": base * item.CompletionRatio}}
	} else {
		return nil
	}
	amounts := make(map[string]canvasPriceRate)
	for _, tier := range tiers {
		for unit, base := range tier {
			for _, ratio := range ratios {
				amount := base * ratio * rate
				if amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
					return &canvasCatalogPrice{Currency: price.Currency, Varies: true}
				}
				current, ok := amounts[unit]
				if !ok {
					current = canvasPriceRate{Unit: unit, Min: amount, Max: amount}
				}
				current.Min, current.Max = min(current.Min, amount), max(current.Max, amount)
				amounts[unit] = current
			}
		}
	}
	for _, unit := range []string{"second", "request", "image", "video_tokens", "input_tokens", "output_tokens", "image_tokens", "cache_tokens", "reference_second", "reference_image", "credit"} {
		if amount, ok := amounts[unit]; ok {
			price.Rates = append(price.Rates, amount)
		}
	}
	price.Varies = len(tiers) > 1 || len(price.Rates) > 1 || len(ratios) > 1
	return price
}

// Canvas credits and the Stolas balance share the same face value. The price
// list can display another currency, but the estimate must not use its FX rate.
func canvasPriceEstimate(tiers []map[string]float64, ratios []float64, allowVideoTokens bool) []canvasPriceAmount {
	if len(tiers) == 0 || len(ratios) == 0 {
		return nil
	}
	for _, tier := range tiers[1:] {
		if !maps.Equal(tiers[0], tier) {
			return nil
		}
	}
	var estimate []canvasPriceAmount
	for _, unit := range []string{"request", "second", "image", "reference_second", "reference_image", "video_tokens"} {
		if unit == "video_tokens" && !allowVideoTokens {
			continue
		}
		if base, ok := tiers[0][unit]; ok {
			amount := base * ratios[0]
			if amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) || slices.ContainsFunc(ratios[1:], func(ratio float64) bool {
				return base*ratio != amount
			}) {
				return nil
			}
			estimate = append(estimate, canvasPriceAmount{Unit: unit, Amount: amount})
		}
	}
	if len(estimate) != len(tiers[0]) {
		return nil // Token/other unknown usage cannot be quoted from Canvas settings.
	}
	return estimate
}

// Recognize linear price tiers without evaluating a billing expression in the
// catalog path or disclosing request rules, route names, or usage internals.
func parseCanvasPriceExpression(expression string, schema map[string]jsplugin.UsageFieldSchema) canvasPriceExpression {
	_, body := billingexpr.ParseExprVersion(expression)
	if strings.TrimSpace(body) == "" || len(body) > 64*1024 {
		return canvasPriceExpression{}
	}
	tree, err := parser.Parse(body)
	if err != nil {
		return canvasPriceExpression{}
	}
	return canvasPriceExpression{node: tree.Node, schema: schema}
}

func canvasPriceTiers(expressions []canvasPriceExpression, knownUsage map[string]string) []map[string]float64 {
	var tiers []map[string]float64
	var collect func(ast.Node, map[string]jsplugin.UsageFieldSchema) bool
	collect = func(node ast.Node, schema map[string]jsplugin.UsageFieldSchema) bool {
		if branch, ok := node.(*ast.ConditionalNode); ok {
			if matched, known := canvasUsageCondition(branch.Cond, knownUsage); known {
				if matched {
					return collect(branch.Exp1, schema)
				}
				return collect(branch.Exp2, schema)
			}
			return collect(branch.Exp1, schema) && collect(branch.Exp2, schema)
		}
		amounts := make(map[string]float64)
		if !canvasPriceTerms(node, schema, amounts, 1) || len(amounts) == 0 {
			return false
		}
		tiers = append(tiers, amounts)
		return true
	}
	for _, expression := range expressions {
		if !collect(expression.node, expression.schema) {
			// Never publish a partial price that omits a possible route or tier.
			return nil
		}
	}
	return tiers
}

// Unknown usage facts keep both branches.
func canvasUsageCondition(node ast.Node, knownUsage map[string]string) (matched, known bool) {
	part, ok := node.(*ast.BinaryNode)
	if !ok || len(knownUsage) == 0 {
		return false, false
	}
	switch part.Operator {
	case "&&", "and":
		left, leftKnown := canvasUsageCondition(part.Left, knownUsage)
		right, rightKnown := canvasUsageCondition(part.Right, knownUsage)
		return left && right, (leftKnown && rightKnown) || (leftKnown && !left) || (rightKnown && !right)
	case "||", "or":
		left, leftKnown := canvasUsageCondition(part.Left, knownUsage)
		right, rightKnown := canvasUsageCondition(part.Right, knownUsage)
		return left || right, (leftKnown && rightKnown) || (leftKnown && left) || (rightKnown && right)
	case "==", "!=":
		call, ok := part.Left.(*ast.CallNode)
		literal, literalOK := part.Right.(*ast.StringNode)
		if !ok || !literalOK || len(call.Arguments) != 1 {
			return false, false
		}
		callee, calleeOK := call.Callee.(*ast.IdentifierNode)
		key, keyOK := call.Arguments[0].(*ast.StringNode)
		if calleeOK && callee.Value == "u" && keyOK {
			if value, known := knownUsage[key.Value]; known {
				return (value == literal.Value) == (part.Operator == "=="), true
			}
		}
	}
	return false, false
}

func canvasPriceTerms(node ast.Node, schema map[string]jsplugin.UsageFieldSchema, terms map[string]float64, factor float64) bool {
	if value, ok := canvasPriceNumber(node); ok {
		if schema == nil {
			return false // A bare constant in a token expression isn't a dollar price.
		}
		terms["request"] += value * factor
		return true
	}
	switch part := node.(type) {
	case *ast.BinaryNode:
		switch part.Operator {
		case "+":
			return canvasPriceTerms(part.Left, schema, terms, factor) && canvasPriceTerms(part.Right, schema, terms, factor)
		case "*":
			if n, ok := canvasPriceNumber(part.Right); ok {
				return canvasPriceTerms(part.Left, schema, terms, factor*n)
			}
			if n, ok := canvasPriceNumber(part.Left); ok {
				return canvasPriceTerms(part.Right, schema, terms, factor*n)
			}
			if id, ok := part.Right.(*ast.IdentifierNode); ok && id.Value == "image_count" && schema == nil {
				amounts := make(map[string]float64)
				if canvasPriceTerms(part.Left, schema, amounts, factor) && len(amounts) == 1 {
					if amount, ok := amounts["request"]; ok {
						terms["image"] += amount
						return true
					}
				}
			}
		case "/":
			if n, ok := canvasPriceNumber(part.Right); ok && n > 0 {
				return canvasPriceTerms(part.Left, schema, terms, factor/n)
			}
		}
	case *ast.IdentifierNode:
		if schema == nil {
			unit := map[string]string{"p": "input_tokens", "c": "output_tokens", "img": "image_tokens", "img_o": "output_tokens", "cr": "cache_tokens", "img_cr": "cache_tokens"}[part.Value]
			if unit != "" {
				// Distinct tokens must not be accidentally added into one unit.
				if _, exists := terms[unit]; exists {
					return false
				}
				terms[unit] = factor
				return true
			}
		}
	case *ast.CallNode:
		callee, ok := part.Callee.(*ast.IdentifierNode)
		if !ok {
			return false
		}
		switch callee.Value {
		case "tier":
			if len(part.Arguments) == 2 {
				return canvasPriceTerms(part.Arguments[1], schema, terms, factor)
			}
		case "fixed":
			if schema == nil && len(part.Arguments) == 1 {
				if n, ok := canvasPriceNumber(part.Arguments[0]); ok {
					terms["request"] += n * factor
					return true
				}
			}
		case "u":
			if len(part.Arguments) != 1 {
				return false
			}
			key, ok := part.Arguments[0].(*ast.StringNode)
			if !ok {
				return false
			}
			var unit string
			switch schema[key.Value].Unit {
			case "second":
				unit = map[string]string{"seconds": "second", "reference_seconds": "reference_second"}[key.Value]
			case "count":
				unit = map[string]string{"requests": "request", "image_count": "image", "extra_images": "reference_image"}[key.Value]
			case "token":
				if key.Value == "tokens" {
					unit, factor = "video_tokens", factor*1_000_000
				}
			case "credit":
				unit = "credit"
			}
			if unit != "" {
				terms[unit] += factor
				return true
			}
		}
	}
	return false
}

func canvasPriceNumber(node ast.Node) (float64, bool) {
	switch v := node.(type) {
	case *ast.IntegerNode:
		return float64(v.Value), v.Value >= 0
	case *ast.FloatNode:
		return v.Value, v.Value >= 0 && !math.IsInf(v.Value, 0)
	default:
		return 0, false
	}
}
