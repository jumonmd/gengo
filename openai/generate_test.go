// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package openai

import (
	"reflect"
	"testing"

	"github.com/jumonmd/gengo/chat"
)

func TestConvertChatRequest(t *testing.T) {
	r := &chat.Request{
		Config: chat.ModelConfig{
			MaxTokens:        100,
			Temperature:      0.7,
			TopP:             0.9,
			PresencePenalty:  0.5,
			FrequencyPenalty: 0.4,
			StopWords:        []string{"stop", "word"},
		},
		Tools: []chat.Tool{
			{
				Name:        "tool1",
				Description: "tool1 description",
			},
		},
		MustCallTool: true,
	}

	req := convertChatRequest(r)
	if req.MaxTokens != 100 {
		t.Errorf("MaxTokens mismatch: expected %d, got %d", 100, req.MaxTokens)
	}
	if req.Temperature != 0.7 {
		t.Errorf("Temperature mismatch: expected %f, got %f", 0.7, req.Temperature)
	}
	if req.TopP != 0.9 {
		t.Errorf("TopP mismatch: expected %f, got %f", 0.9, req.TopP)
	}
	if req.FrequencyPenalty != 0.4 {
		t.Errorf("FrequencyPenalty mismatch: expected %f, got %f", 0.4, req.FrequencyPenalty)
	}
	if req.PresencePenalty != 0.5 {
		t.Errorf("PresencePenalty mismatch: expected %f, got %f", 0.5, req.PresencePenalty)
	}
	if !reflect.DeepEqual(req.Stop, []string{"stop", "word"}) {
		t.Errorf("StopWords mismatch: expected %v, got %v", []string{"stop", "word"}, req.Stop)
	}
	if len(req.Tools) != len(r.Tools) {
		t.Errorf("Tools mismatch: expected %v, got %v", len(r.Tools), len(req.Tools))
	}
	if req.ToolChoice != "required" {
		t.Errorf("ToolChoice mismatch: expected %s, got %s", "required", req.ToolChoice)
	}
}
