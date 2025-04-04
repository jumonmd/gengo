// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/jumonmd/gengo/chat"
	"github.com/jumonmd/gengo/jsonschema"
	"google.golang.org/genai"
)

func Generate(ctx context.Context, r *chat.Request, opts ...chat.Option) (*chat.Response, error) {
	opt := chat.NewOptions(opts...)
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, err
	}

	// tool call will not use stream for simplicity
	if opt.Streamer != nil && len(r.Tools) == 0 {
		resp, err := generateContentStream(ctx, client, r, opt.Streamer)
		if err != nil {
			return nil, fmt.Errorf("generate content stream: %w", err)
		}
		opt.ModelCatalog.CalculateCost(r.Model, resp.Usage)
		return resp, nil
	}

	resp, err := generateContent(ctx, client, r)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}
	opt.ModelCatalog.CalculateCost(r.Model, resp.Usage)
	return resp, nil
}

func generateContent(ctx context.Context, client *genai.Client, r *chat.Request) (*chat.Response, error) {
	config := convertChatConfig(r)
	req, err := convertChatRequest(r, config)
	if err != nil {
		return nil, fmt.Errorf("convert chat request: %w", err)
	}

	result, err := client.Models.GenerateContent(ctx, r.Model, req.Contents, req.Config)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}

	response := convertGenerateContentResponse(result, r.Model)
	return response, nil
}

func generateContentStream(ctx context.Context, client *genai.Client, r *chat.Request, streamfunc chat.Streamer) (*chat.Response, error) {
	config := convertChatConfig(r)
	req, err := convertChatRequest(r, config)
	if err != nil {
		return nil, fmt.Errorf("convert chat request: %w", err)
	}

	usage := chat.Usage{}
	content := ""
	finishReason := genai.FinishReasonUnspecified
	for resp, err := range client.Models.GenerateContentStream(ctx, r.Model, req.Contents, req.Config) {
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("generate content stream: %w", err)
		}

		updateUsage(&usage, resp.UsageMetadata)

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			continue
		}

		for _, part := range resp.Candidates[0].Content.Parts {
			if c := part.Text; c != "" {
				content += c
				streamfunc(&chat.StreamResponse{
					Type:    "text",
					Content: c,
				})
			}
		}

		finishReason = resp.Candidates[0].FinishReason
	}

	return &chat.Response{
		Model:        r.Model,
		Messages:     []chat.Message{chat.NewTextMessage(chat.MessageRoleAI, content)},
		FinishReason: convertFinishReason(finishReason),
		Usage:        &usage,
	}, nil
}

func convertChatConfig(r *chat.Request) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	if r.Config.Temperature != 0 {
		config.Temperature = genai.Ptr(r.Config.Temperature)
	}
	if r.Config.MaxTokens != 0 {
		config.MaxOutputTokens = genai.Ptr(r.Config.MaxTokens)
	}
	if r.Config.TopP != 0 {
		config.TopP = genai.Ptr(r.Config.TopP)
	}
	if r.Config.PresencePenalty != 0 {
		config.PresencePenalty = genai.Ptr(r.Config.PresencePenalty)
	}
	if r.Config.FrequencyPenalty != 0 {
		config.FrequencyPenalty = genai.Ptr(r.Config.FrequencyPenalty)
	}
	if len(r.Config.StopWords) > 0 {
		config.StopSequences = r.Config.StopWords
	}

	return config
}

type generateContentRequest struct {
	Contents []*genai.Content
	Config   *genai.GenerateContentConfig
}

func convertChatRequest(r *chat.Request, config *genai.GenerateContentConfig) (*generateContentRequest, error) {
	contents, err := convertChatMessages(r.Messages)
	if err != nil {
		return nil, fmt.Errorf("convert chat messages: %w", err)
	}

	req := &generateContentRequest{
		Contents: contents,
		Config:   config,
	}

	if len(r.Tools) > 0 {
		tools, toolConfig, err := convertChatTools(r)
		if err != nil {
			return nil, fmt.Errorf("convert chat tools: %w", err)
		}
		config.Tools = tools
		config.ToolConfig = toolConfig
	}

	if r.ResponseSchema != nil {
		schema, err := convertChatSchema(r.ResponseSchema)
		if err != nil {
			return nil, fmt.Errorf("convert chat schema: %w", err)
		}
		config.ResponseMIMEType = "application/json"
		config.ResponseSchema = schema
	}

	return req, nil
}

func convertChatMessages(messages []chat.Message) ([]*genai.Content, error) {
	contents := []*genai.Content{}

	for _, msg := range messages {
		parts := []*genai.Part{}
		switch {
		case msg.IsToolResponse():
			output := map[string]any{
				"name":    msg.ToolResponse.Name,
				"content": msg.ToolResponse.Result,
			}
			parts = append(parts, genai.NewPartFromFunctionResponse(msg.ToolResponse.Name, output))
		case msg.IsToolCall():
			args := map[string]any{}
			if err := json.Unmarshal([]byte(msg.ToolCall.Arguments), &args); err != nil {
				return nil, fmt.Errorf("unmarshal tool call arguments: %w", err)
			}
			parts = append(parts, genai.NewPartFromFunctionCall(msg.ToolCall.Name, args))
		default:
			for _, part := range msg.Content {
				switch part.Type {
				case "text":
					parts = append(parts, genai.NewPartFromText(part.Text))
				case "image":
					if !chat.IsDataURL(part.DataURL) {
						return nil, fmt.Errorf("invalid data URL: %s", part.DataURL)
					}
					data, mimeType, err := chat.DecodeDataURL(part.DataURL)
					if err == nil {
						parts = append(parts, genai.NewPartFromBytes(data, mimeType))
					}
				}
			}
		}

		role := convertChatRole(msg.Role)
		content := &genai.Content{
			Role:  role,
			Parts: parts,
		}

		contents = append(contents, content)
	}

	return contents, nil
}

func convertChatRole(role chat.MessageRole) string {
	switch role {
	case chat.MessageRoleSystem:
		return "system"
	case chat.MessageRoleHuman:
		return "user"
	case chat.MessageRoleAI:
		return "model"
	case chat.MessageRoleTool:
		return "tool"
	default:
		return "user"
	}
}

func convertChatTools(r *chat.Request) ([]*genai.Tool, *genai.ToolConfig, error) {
	tools := []*genai.Tool{}

	for _, tool := range r.Tools {
		schema, err := convertChatSchema(tool.InputSchema)
		if err != nil {
			return nil, nil, fmt.Errorf("convert chat schema: %w", err)
		}
		functionDecl := &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  schema,
		}

		tools = append(tools, &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{functionDecl},
		})
	}

	toolConfig := &genai.ToolConfig{}
	if r.MustCallTool {
		toolConfig.FunctionCallingConfig = &genai.FunctionCallingConfig{
			Mode: genai.FunctionCallingConfigModeAny,
		}
	}

	return tools, toolConfig, nil
}

func convertChatSchema(schema jsonschema.Schema) (*genai.Schema, error) {
	gschema := &genai.Schema{}
	if err := json.Unmarshal(schema.JSON(), gschema); err != nil {
		return nil, fmt.Errorf("gemini unmarshal schema: %w", err)
	}

	return gschema, nil
}

func convertGenerateContentResponse(result *genai.GenerateContentResponse, model string) *chat.Response {
	msgs := []chat.Message{}

	finishreason := chat.FinishReasonUnknown

	if len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
		text := result.Text()
		if text != "" {
			msgs = append(msgs, chat.NewTextMessage(chat.MessageRoleAI, text))
		}
		functionCalls := result.FunctionCalls()
		for _, call := range functionCalls {
			argsJSON, err := json.Marshal(call.Args)
			if err != nil {
				continue
			}
			msgs = append(msgs, chat.NewToolCallMessage(call.Name, call.ID, string(argsJSON)))
		}
		if len(functionCalls) > 0 {
			finishreason = chat.FinishReasonToolUse
		} else {
			finishreason = convertFinishReason(result.Candidates[0].FinishReason)
		}
	}

	usage := &chat.Usage{}
	updateUsage(usage, result.UsageMetadata)

	response := &chat.Response{
		Model:        model,
		Messages:     msgs,
		FinishReason: finishreason,
		Usage:        usage,
	}
	return response
}

func convertFinishReason(reason genai.FinishReason) chat.FinishReason {
	switch reason {
	case genai.FinishReasonStop:
		return chat.FinishReasonStop
	case genai.FinishReasonMaxTokens:
		return chat.FinishReasonMaxTokens
	case genai.FinishReasonSafety,
		genai.FinishReasonImageSafety,
		genai.FinishReasonProhibitedContent,
		genai.FinishReasonRecitation,
		genai.FinishReasonBlocklist,
		genai.FinishReasonSPII:
		return chat.FinishReasonSafety
	case genai.FinishReasonMalformedFunctionCall:
		return chat.FinishReasonError
	case genai.FinishReasonOther,
		genai.FinishReasonUnspecified:
		return chat.FinishReasonUnknown
	default:
		return chat.FinishReasonUnknown
	}
}

func updateUsage(usage *chat.Usage, metadata *genai.GenerateContentResponseUsageMetadata) {
	if metadata != nil {
		if metadata.PromptTokenCount != nil {
			usage.InputTokens = int(*metadata.PromptTokenCount)
		}
		if metadata.CandidatesTokenCount != nil {
			usage.OutputTokens = int(*metadata.CandidatesTokenCount)
		}
		usage.TotalTokens = int(metadata.TotalTokenCount)
	}
}
