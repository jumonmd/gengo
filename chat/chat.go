// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jumonmd/gengo/jsonschema"
)

type MessageRole string

const (
	MessageRoleSystem MessageRole = "system"
	MessageRoleAI     MessageRole = "ai"
	MessageRoleHuman  MessageRole = "human"
	MessageRoleTool   MessageRole = "tool"
)

type Request struct {
	Model          string            `json:"model"`
	Config         ModelConfig       `json:"config"`
	Metadata       Metadata          `json:"metadata"`
	Messages       []Message         `json:"messages"`
	Tools          []Tool            `json:"tools"`
	MustCallTool   bool              `json:"must_call_tool"`
	ResponseType   string            `json:"response_type"`
	ResponseSchema jsonschema.Schema `json:"response_schema"`
}

type ModelConfig struct {
	MaxTokens        int32    `json:"max_tokens,omitempty"`
	Temperature      float32  `json:"temperature,omitempty"`
	TopP             float32  `json:"top_p,omitempty"`
	PresencePenalty  float32  `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32  `json:"frequency_penalty,omitempty"`
	StopWords        []string `json:"stop_words,omitempty"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Parameters.
	InputSchema jsonschema.Schema `json:"input_schema"`
}

type Metadata map[string]string

type Message struct {
	// Type for extension. Default type is message.
	//   possible values: web_search_call, file_search_call...
	Type    string        `json:"type,omitempty"`
	Role    MessageRole   `json:"role"`
	Content []ContentPart `json:"content,omitempty"`
	// ToolCall by AI. Role should be AI.
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	// ToolResponse from tool. Role should be tool.
	ToolResponse *ToolResponse `json:"tool_response,omitempty"`
}

func (m *Message) IsToolCall() bool {
	return m.ToolCall != nil && m.Role == MessageRoleAI
}

func (m *Message) IsToolResponse() bool {
	return m.ToolResponse != nil && m.Role == MessageRoleTool
}

type ContentPart struct {
	// Type is the content part type. text, image or file.
	Type string `json:"type"`
	// Text for text type.
	Text string `json:"text,omitempty"`
	// DataURL for image or file type.
	DataURL string `json:"data_url,omitempty"`
}

type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Arguments is stringified json.
	Arguments string `json:"arguments"`
}

type ToolResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Result is stringified json.
	Result string `json:"result"`
}

type Response struct {
	Model        string       `json:"model"`
	FinishReason FinishReason `json:"finish_reason"`
	Messages     []Message    `json:"messages"`
	Metadata     Metadata     `json:"metadata"`
	Usage        *Usage       `json:"usage,omitempty"`
}

type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonMaxTokens FinishReason = "max_tokens"
	FinishReasonToolUse   FinishReason = "tool_use"
	FinishReasonSafety    FinishReason = "safety"
	FinishReasonError     FinishReason = "error"
	FinishReasonUnknown   FinishReason = "unknown"
)

type Usage struct {
	InputTokens         int     `json:"input_tokens"`
	OutputTokens        int     `json:"output_tokens"`
	ReasoningTokens     int     `json:"reasoning_tokens"`
	CacheCreationTokens int     `json:"cache_creation_tokens"`
	CachedTokens        int     `json:"cached_tokens"`
	TotalTokens         int     `json:"total_tokens"`
	Cost                float64 `json:"cost"`
}

type Streamer func(resp *StreamResponse)

type StreamResponse struct {
	// Type is the type of the stream response for extension.
	//   possible values: chat.completion.chunk, chat.thinking.chunk...
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (s *StreamResponse) JSON() []byte {
	json, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return json
}

func NewTextMessage(role MessageRole, text string) Message {
	return Message{
		Role: role,
		Content: []ContentPart{{
			Type: "text",
			Text: text,
		}},
	}
}

// NewTextImageMessage creates a message with text and image.
// If text is empty, image only content is returned.
func NewTextImageMessage(role MessageRole, text, path string) (Message, error) {
	dataurl, mimeType, err := EncodeDataURLFromPath(path)
	if err != nil {
		return Message{}, err
	}

	if !strings.HasPrefix(mimeType, "image/") {
		return Message{}, fmt.Errorf("not an image: %s", mimeType)
	}

	content := []ContentPart{}

	if text != "" {
		textpart := ContentPart{
			Type: "text",
			Text: text,
		}
		content = append(content, textpart)
	}

	content = append(content, ContentPart{
		Type:    "image",
		DataURL: dataurl,
	})

	return Message{
		Role:    role,
		Content: content,
	}, nil
}

// NewToolCallMessage creates a AI tool call message with name, callID and arguments(stringified json).
func NewToolCallMessage(name, callID, arguments string) Message {
	return Message{
		Role: MessageRoleAI,
		ToolCall: &ToolCall{
			Name:      name,
			ID:        callID,
			Arguments: arguments,
		},
	}
}

// NewToolResponseMessage creates a tool response message with name, callID and result.
func NewToolResponseMessage(name, callID, result string) Message {
	return Message{
		Role: MessageRoleTool,
		ToolResponse: &ToolResponse{
			Name:   name,
			ID:     callID,
			Result: result,
		},
	}
}

// ToolCalls returns tool call messages from AI.
func (r *Response) ToolCalls() []Message {
	toolcalls := []Message{}
	for _, m := range r.Messages {
		if m.IsToolCall() {
			toolcalls = append(toolcalls, m)
		}
	}
	return toolcalls
}

// ToolResponses returns tool response messages from tool.
func (r *Response) ToolResponses() []Message {
	toolresponses := []Message{}
	for _, m := range r.Messages {
		if m.IsToolResponse() {
			toolresponses = append(toolresponses, m)
		}
	}
	return toolresponses
}

func (p ContentPart) String() string {
	if p.Type == "text" {
		return p.Text
	}
	return ""
}

func (m *Message) ContentString() string {
	parts := []string{}
	for _, p := range m.Content {
		parts = append(parts, p.String())
	}
	return strings.Join(parts, "")
}

func (m *Message) String() string {
	parts := []string{}
	if m.ContentString() != "" {
		parts = append(parts, fmt.Sprintf("%s: %s", strings.ToUpper(string(m.Role)), m.ContentString()))
	}
	if m.ToolCall != nil {
		parts = append(parts, fmt.Sprintf("tool_calls: [CallID: %s, Name: %s, Arguments: %s]", m.ToolCall.ID, m.ToolCall.Name, m.ToolCall.Arguments))
	}
	return strings.Join(parts, "\n")
}
