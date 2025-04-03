// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package anthropic

import (
	"reflect"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jumonmd/gengo/chat"
)

func TestConvertChatRequest(t *testing.T) {
	r := &chat.Request{
		Model: "claude-3-haiku-20240307",
		Config: chat.ModelConfig{
			MaxTokens:   100,
			Temperature: 0.7,
			TopP:        0.9,
			StopWords:   []string{"stop", "word"},
		},
		Tools: []chat.Tool{
			{
				Name:        "tool1",
				Description: "tool1 description",
			},
		},
		MustCallTool: true,
	}

	params := convertChatRequest(r, nil)

	if params.MaxTokens != 100 {
		t.Errorf("MaxTokens mismatch: expected %d, got %d", 100, params.MaxTokens)
	}

	if !cmp.Equal(params.Temperature.Value, 0.7, cmpopts.EquateApprox(0.001, 0)) {
		t.Errorf("Temperature mismatch: expected %f, got %f", 0.7, params.Temperature.Value)
	}

	if !cmp.Equal(params.TopP.Value, 0.9, cmpopts.EquateApprox(0.001, 0)) {
		t.Errorf("TopP mismatch: expected %f, got %f", 0.9, params.TopP.Value)
	}

	if !reflect.DeepEqual(params.StopSequences, []string{"stop", "word"}) {
		t.Errorf("StopSequences mismatch: expected %v, got %v", []string{"stop", "word"}, params.StopSequences)
	}

	if params.ToolChoice.OfToolChoiceAny == nil {
		t.Errorf("ToolChoice mismatch: expected %v, got %v", anthropic.ToolChoiceUnionParam{OfToolChoiceAny: &anthropic.ToolChoiceAnyParam{}}, params.ToolChoice)
	}

	r = &chat.Request{}
	params = convertChatRequest(r, nil)
	if params.MaxTokens != 2048 {
		t.Errorf("MaxTokens mismatch: expected %d, got %d", 2048, params.MaxTokens)
	}
}
