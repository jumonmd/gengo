// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package gengo

import (
	"context"
	"fmt"

	"github.com/jumonmd/gengo/anthropic"
	"github.com/jumonmd/gengo/chat"
	"github.com/jumonmd/gengo/google"
	"github.com/jumonmd/gengo/openai"
)

// Generate fetches responses from various AI models.
// Routes requests to the appropriate provider (OpenAI, Gemini, or Anthropic)
// based on the requested model name.
func Generate(ctx context.Context, req *chat.Request, opts ...chat.Option) (*chat.Response, error) {
	o := chat.NewOptions(opts...)

	model := o.ModelCatalog.GetModel(req.Model)
	if model == nil {
		return nil, fmt.Errorf("model not found: %s", req.Model)
	}

	switch model.Provider {
	case "anthropic":
		return anthropic.Generate(ctx, req, opts...)
	case "gemini":
		return google.Generate(ctx, req, opts...)
	case "openai":
		return openai.Generate(ctx, req, opts...)
	}

	return nil, fmt.Errorf("provider not found: %s", model.Provider)
}
