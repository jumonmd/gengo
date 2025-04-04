// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

//go:build integration_test

// set OPENAI_API_KEY, GOOGLE_API_KEY, ANTHROPIC_API_KEY environment variables to run this test.

package gengo_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jumonmd/gengo"
	"github.com/jumonmd/gengo/chat"
	"github.com/jumonmd/gengo/jsonschema"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model         string
		hasToolCallID bool
	}{
		{"gpt-4o-mini", true},
		{"gemini-2.0-flash", false},
		{"claude-3-5-haiku-latest", true},
	}
	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			t.Parallel()
			runGenerate(t, &chat.Request{
				Model: test.model,
			})
			runGenerateStream(t, &chat.Request{
				Model: test.model,
			})
			runImageInput(t, &chat.Request{
				Model: test.model,
			})
			runToolcall(t, &chat.Request{
				Model: test.model,
			}, test.hasToolCallID)
			runResponseSchema(t, &chat.Request{
				Model: test.model,
			})
		})
	}
}

func runToolcall(t *testing.T, req *chat.Request, hasToolCallID bool) {
	t.Helper()

	req.Messages = append(req.Messages, chat.NewTextMessage(chat.MessageRoleHuman, "Hello, what is the weather in Tokyo?"))
	req.Tools = []chat.Tool{
		{
			Name:        "get_current_weather",
			Description: "Get the current weather in a given location",
			InputSchema: jsonschema.MustParseJSONString(`{"type": "object", "properties": {"location": {"type": "string"}}}`),
		},
	}

	resp, err := gengo.Generate(t.Context(), req)
	if err != nil {
		t.Fatalf("Error generating response: %v", err)
	}

	checkResponse(t, resp)

	if len(resp.ToolCalls()) == 0 {
		t.Fatalf("expected tool call, got %d", len(resp.ToolCalls()))
	}

	if resp.FinishReason != chat.FinishReasonToolUse {
		t.Fatalf("expected finish reason `tool_use`, got %s", resp.FinishReason)
	}

	for _, toolCall := range resp.ToolCalls() {
		if toolCall.ToolCall == nil {
			t.Fatalf("expected tool call, got nil")
		}
		if hasToolCallID && toolCall.ToolCall.ID == "" {
			t.Fatalf("expected tool call id, got empty string")
		}
		if !strings.Contains(toolCall.ToolCall.Name, "get_current_weather") {
			t.Fatalf("expected tool call name `get_current_weather`, got %s", toolCall.ToolCall.Name)
		}
		if !strings.Contains(toolCall.ToolCall.Arguments, "location") {
			t.Fatalf("expected tool call arguments `location`, got %s", toolCall.ToolCall.Arguments)
		}
		t.Logf("Tool call: %s, Arguments: %s", toolCall.ToolCall.Name, toolCall.ToolCall.Arguments)
	}

	req.Messages = append(req.Messages, resp.Messages...)

	// make tool response manually
	for _, msg := range resp.ToolCalls() {
		req.Messages = append(req.Messages, chat.NewToolResponseMessage(msg.ToolCall.Name, msg.ToolCall.ID, "Rainy"))
	}

	resp, err = gengo.Generate(t.Context(), req)
	if err != nil {
		t.Fatalf("Error generating response: %v", err)
	}

	checkResponse(t, resp)

	if resp.FinishReason != chat.FinishReasonStop {
		t.Fatalf("expected finish reason `stop`, got %s", resp.FinishReason)
	}
}

func runImageInput(t *testing.T, req *chat.Request) {
	t.Helper()

	msg, err := chat.NewTextImageMessage(chat.MessageRoleHuman, "OCR this image", "./testdata/image.png")
	if err != nil {
		t.Fatalf("Error creating image message: %v", err)
	}
	req.Messages = append(req.Messages, msg)

	resp, err := gengo.Generate(t.Context(), req)
	if err != nil {
		t.Fatalf("Error generating response: %v", err)
	}

	checkResponse(t, resp)

	if resp.FinishReason != chat.FinishReasonStop {
		t.Fatalf("expected finish reason `stop`, got %s", resp.FinishReason)
	}

	t.Logf("Content: %s", resp.Messages[0].ContentString())
	if !strings.Contains(resp.Messages[0].ContentString(), "GENGO") {
		t.Fatalf("expected content to contain `GENGO`, got %s", resp.Messages[0].ContentString())
	}
}

func runResponseSchema(t *testing.T, req *chat.Request) {
	t.Helper()

	req.Messages = append(req.Messages, chat.NewTextMessage(chat.MessageRoleHuman, "Extract the name from the following text. `I am Gengo`"))
	req.ResponseSchema = jsonschema.MustParseJSONString(`{"type": "object", "properties": {"name": {"type": "string"}}}`)
	resp, err := gengo.Generate(t.Context(), req)
	if err != nil {
		t.Fatalf("Error generating response: %v", err)
	}

	checkResponse(t, resp)

	if resp.FinishReason != chat.FinishReasonStop {
		t.Fatalf("expected finish reason `stop`, got %s", resp.FinishReason)
	}

	t.Logf("Content: %s", resp.Messages[0].ContentString())

	type name struct {
		Name string `json:"name"`
	}

	var result name
	err = json.Unmarshal([]byte(resp.Messages[0].ContentString()), &result)
	if err != nil {
		t.Fatalf("Error unmarshalling response: %v", err)
	}

	if result.Name != "Gengo" {
		t.Fatalf("expected name `Gengo`, got %s", result.Name)
	}
}

func runGenerate(t *testing.T, req *chat.Request) {
	t.Helper()

	req.Messages = append(req.Messages, chat.NewTextMessage(chat.MessageRoleHuman, "Hello!"))

	resp, err := gengo.Generate(t.Context(), req)
	if err != nil {
		t.Fatalf("Error generating response: %v", err)
	}

	checkResponse(t, resp)

	if resp.FinishReason != chat.FinishReasonStop {
		t.Fatalf("expected finish reason `stop`, got %s", resp.FinishReason)
	}

	t.Logf("Content: %s", resp.Messages[0].ContentString())
}

func runGenerateStream(t *testing.T, req *chat.Request) {
	t.Helper()

	req.Messages = append(req.Messages, chat.NewTextMessage(chat.MessageRoleHuman, "Hello!"))

	responses := []chat.StreamResponse{}
	streamer := func(resp *chat.StreamResponse) {
		responses = append(responses, *resp)
	}
	resp, err := gengo.Generate(t.Context(), req, chat.WithStream(streamer))
	if err != nil {
		t.Fatalf("Error generating response: %v", err)
	}

	checkResponse(t, resp)
	if resp.FinishReason != chat.FinishReasonStop {
		t.Fatalf("expected finish reason `stop`, got %s", resp.FinishReason)
	}

	content := ""
	for _, response := range responses {
		if response.Type != "text" {
			t.Fatalf("expected stream type, got %s", response.Type)
		}
		if response.Content == "" {
			t.Fatalf("expected stream content, got empty string")
		}
		content += response.Content
	}
	if content == "" {
		t.Fatalf("expected content, got empty string")
	}
	t.Logf("Content: %s, Streams: %d", content, len(responses))

	if content != resp.Messages[0].ContentString() {
		t.Fatalf("expected stream content and response content to be the same, got %s", content)
	}
}

func checkResponse(t *testing.T, resp *chat.Response) {
	t.Helper()

	if len(resp.Messages) == 0 {
		t.Fatalf("expected message, got %d", len(resp.Messages))
	}

	if resp.Messages[0].Role != chat.MessageRoleAI {
		t.Fatalf("expected assistant message, got %s", resp.Messages[0].Role)
	}

	if resp.Usage == nil || resp.Usage.TotalTokens == 0 || resp.Usage.InputTokens == 0 || resp.Usage.OutputTokens == 0 {
		t.Fatalf("expected token usage")
	}

	if resp.Usage.Cost == 0 {
		t.Fatalf("expected cost, got 0")
	}

	if resp.Messages[0].ToolCall == nil && resp.Messages[0].ContentString() == "" {
		t.Fatalf("expected content, got empty string")
	}

	if resp.FinishReason == "" {
		t.Fatalf("expected finish reason, got empty string")
	}
}
