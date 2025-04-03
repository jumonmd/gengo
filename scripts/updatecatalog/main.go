// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jumonmd/gengo/chat"
)

const (
	modelCostURL       = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"
	modelCostCopyright = "Data from [BerriAI/litellm](https://github.com/BerriAI/litellm/blob/main/model_prices_and_context_window.json) Copyright Berri AI, MIT License."
	jsonFileName       = "./chat/modelcatalog.json"
	markdownFileName   = "./MODELS.md"
)

var (
	providers = []string{"openai", "anthropic", "gemini"}
	excludes  = []string{
		"ft:",
		"-audio-",
		"-realtime-",
		"-search-",
		"chatgpt-",
	}
)

type LiteLLMModelInfo struct {
	Mode                   string  `json:"mode"`
	Model                  string  `json:"model"`
	Provider               string  `json:"litellm_provider"`
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
type ModelCatalog map[string]LiteLLMModelInfo

func main() {
	modelData, err := fetchModelData()
	if err != nil {
		log.Fatalf("Failed to fetch model data: %v", err)
	}

	catalog, err := filterModels(modelData)
	if err != nil {
		log.Fatalf("Failed to filter models: %v", err)
	}

	if err := writeJSON(catalog); err != nil {
		log.Fatalf("Failed to write JSON output: %v", err)
	}

	if err := writeMarkdown(catalog); err != nil {
		log.Fatalf("Failed to write Markdown output: %v", err)
	}
}

func fetchModelData() ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, modelCostURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

func filterModels(rawdata []byte) (ModelCatalog, error) {
	var rawModels map[string]map[string]any
	if err := json.Unmarshal(rawdata, &rawModels); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}
	filteredRawModels := make(map[string]map[string]any)
	for key, model := range rawModels {
		if model["mode"] == "chat" {
			filteredRawModels[key] = model
		}
	}

	data, err := json.Marshal(filteredRawModels)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	var allModels ModelCatalog
	if err := json.Unmarshal(data, &allModels); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	filteredModels := make(ModelCatalog)
	for _, provider := range providers {
		for modelName, modelInfo := range allModels {
			if !isModelEligible(modelName, modelInfo, provider) {
				continue
			}
			filteredModels[modelName] = modelInfo
		}
	}
	return filteredModels, nil
}

func isModelEligible(modelName string, info LiteLLMModelInfo, provider string) bool {
	if info.Mode != "chat" {
		return false
	}

	for _, excludePattern := range excludes {
		if strings.Contains(modelName, excludePattern) {
			return false
		}
	}

	return info.Provider == provider
}

func writeJSON(catalog ModelCatalog) error {
	models := []*chat.ModelInfo{}
	for key, model := range catalog {
		models = append(models, &chat.ModelInfo{
			Model:                  key,
			Provider:               model.Provider,
			MaxTokens:              model.MaxTokens,
			MaxInputTokens:         model.MaxInputTokens,
			MaxOutputTokens:        model.MaxOutputTokens,
			InputTokenCost:         model.InputTokenCost,
			OutputTokenCost:        model.OutputTokenCost,
			CacheCreationTokenCost: model.CacheCreationTokenCost,
			CacheReadTokenCost:     model.CacheReadTokenCost,
			SupportsWebSearch:      model.SupportsWebSearch,
			SupportsVision:         model.SupportsVision,
			SupportsPDFInput:       model.SupportsPDFInput,
		})
	}

	jsonData, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	file, err := os.Create(jsonFileName)
	if err != nil {
		return fmt.Errorf("error creating JSON file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(jsonData); err != nil {
		return fmt.Errorf("error writing to JSON file: %w", err)
	}

	return nil
}

func writeMarkdown(catalog ModelCatalog) error {
	file, err := os.Create(markdownFileName)
	if err != nil {
		return fmt.Errorf("error creating markdown file: %w", err)
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "# Model Catalog\n\n"); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}

	for _, provider := range providers {
		if err := writeProviderSection(catalog, provider, file); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(file, "%s\n", modelCostCopyright); err != nil {
		return fmt.Errorf("error writing copyright: %w", err)
	}

	return nil
}

func writeProviderSection(catalog ModelCatalog, provider string, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "## %s\n\n", provider); err != nil {
		return fmt.Errorf("error writing provider header: %w", err)
	}

	providerModels := []string{}
	for modelName, modelInfo := range catalog {
		if modelInfo.Provider == provider {
			providerModels = append(providerModels, modelName)
		}
	}

	sort.Strings(providerModels)

	for _, modelName := range providerModels {
		cleanModelName := strings.ReplaceAll(modelName, "gemini/", "")
		if _, err := fmt.Fprintf(w, "- `%s`\n", cleanModelName); err != nil {
			return fmt.Errorf("error writing model entry: %w", err)
		}
	}

	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return fmt.Errorf("error writing section footer: %w", err)
	}

	return nil
}
