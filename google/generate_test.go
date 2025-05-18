// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package google

import (
	"reflect"
	"testing"

	"github.com/jumonmd/gengo/chat"
	"github.com/jumonmd/gengo/jsonschema"
	"google.golang.org/genai"
)

func TestConfigFromRequest(t *testing.T) {
	r := &chat.Request{
		Config: chat.ModelConfig{
			MaxTokens:        100,
			Temperature:      0.7,
			TopP:             0.9,
			PresencePenalty:  0.5,
			FrequencyPenalty: 0.4,
			StopWords:        []string{"stop", "word"},
		},
	}

	config := convertChatConfig(r)

	if config.MaxOutputTokens != 100 {
		t.Errorf("MaxOutputTokens mismatch: expected %d, got %d", 100, config.MaxOutputTokens)
	}
	if *config.Temperature != 0.7 {
		t.Errorf("Temperature mismatch: expected %f, got %f", 0.7, *config.Temperature)
	}
	if *config.TopP != 0.9 {
		t.Errorf("TopP mismatch: expected %f, got %f", 0.9, *config.TopP)
	}
	if *config.FrequencyPenalty != 0.4 {
		t.Errorf("FrequencyPenalty mismatch: expected %f, got %f", 0.4, *config.FrequencyPenalty)
	}
	if *config.PresencePenalty != 0.5 {
		t.Errorf("PresencePenalty mismatch: expected %f, got %f", 0.5, *config.PresencePenalty)
	}
	if !reflect.DeepEqual(config.StopSequences, []string{"stop", "word"}) {
		t.Errorf("StopSequences mismatch: expected %v, got %v", []string{"stop", "word"}, config.StopSequences)
	}
}

func TestConvertChatTools(t *testing.T) {
	r := &chat.Request{
		Tools: []chat.Tool{
			{
				Name:        "tool1",
				Description: "tool1 description",
				InputSchema: jsonschema.MustParseJSONString(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
			},
		},
		MustCallTool: true,
	}

	tools, toolConfig, err := convertChatTools(r)
	if err != nil {
		t.Errorf("convertChatTools error: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("tools mismatch: expected %d, got %d", 1, len(tools))
	}
	t.Logf("toolConfig: %v", toolConfig)

	if toolConfig.FunctionCallingConfig.Mode != genai.FunctionCallingConfigModeAny {
		t.Errorf("toolConfig mismatch: expected %v, got %v", genai.FunctionCallingConfigModeAny, toolConfig.FunctionCallingConfig.Mode)
	}
}
