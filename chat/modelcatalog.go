// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package chat

import (
	"encoding/json"
	"io"
	"strings"
)

// ModelCatalog is a map of model name to model info.
type ModelCatalog []*ModelInfo

// ModelInfo is the model info like max tokens, cost per token, etc.
type ModelInfo struct {
	Model                  string  `json:"model"`
	Provider               string  `json:"provider"`
	MaxTokens              int     `json:"max_tokens"`
	MaxInputTokens         int     `json:"max_input_tokens"`
	MaxOutputTokens        int     `json:"max_output_tokens"`
	InputTokenCost         float64 `json:"input_cost_per_token"`
	OutputTokenCost        float64 `json:"output_cost_per_token"`
	CacheCreationTokenCost float64 `json:"cache_creation_input_token_cost"`
	CacheReadTokenCost     float64 `json:"cache_read_input_token_cost"`
	SupportsWebSearch      bool    `json:"supports_web_search"`
	SupportsVision         bool    `json:"supports_vision"`
	SupportsPDFInput       bool    `json:"supports_pdf_input"`
}

// NewModelCatalog creates a new model catalog from a JSON reader input.
func NewModelCatalog(r io.Reader) (ModelCatalog, error) {
	var catalog ModelCatalog
	if err := json.NewDecoder(r).Decode(&catalog); err != nil {
		return nil, err
	}
	return catalog, nil
}

// GetModel returns a model info from the catalog.
// If the model is not exact match, it will find the "/" separated model name.
func (c ModelCatalog) GetModel(model string) *ModelInfo {
	for _, info := range c {
		if info.Model == model {
			return info
		}
		// eg. "gemini/gemini-2.0-flash" -> "gemini-2.0-flash"
		if cmp := strings.Split(info.Model, "/"); len(cmp) > 1 {
			if cmp[1] == model {
				return info
			}
		}
	}
	return nil
}

// CalculateCost put cost into the usage in USD.
// Returns true if the model is found and add cost to the usage.
func (c ModelCatalog) CalculateCost(model string, usage *Usage) bool {
	if m := c.GetModel(model); m != nil {
		cost := calculateCost(m, usage)
		usage.Cost = cost
		return true
	}
	return false
}

func calculateCost(model *ModelInfo, usage *Usage) float64 {
	cost := 0.0
	cost += model.InputTokenCost * float64(usage.InputTokens)
	cost += model.OutputTokenCost * float64(usage.OutputTokens)

	return cost
}
