# Gengo

Gengo (Generative 言語 in Go) is a pragmatic, opinionated Go client for generative AI APIs.

Status: In development, specifications are subject to change.

## Key Features

- Easy-to-use, developer-friendly API
- Supports multiple providers with built-in cost calculation
- Supports streaming responses, tool calls, and image inputs
- Controlled JSON responses using schemas

## Supported Providers

- OpenAI (GPT)
- Anthropic (Claude)
- Google (Gemini)
- [Supported models](MODELS.md)


## Installation

```bash
go get github.com/jumonmd/gengo
```

## Quick Start

```go
package main

import (
    "context"
    "github.com/jumonmd/gengo"
    "github.com/jumonmd/gengo/chat"
)

func main() {
    ctx := context.Background()
    
    // Simple text generation
    resp, err := gengo.Generate(ctx, &chat.Request{
        Model: "gpt-4o-mini", // or gemini-2.0-flash, claude-3-5-haiku-latest
        Messages: []chat.Message{
            chat.NewTextMessage(chat.MessageRoleHuman, "Hello, how are you?"),
        },
    })
    if err != nil {
        panic(err)
    }
    
    // Print the response
    for _, msg := range resp.Messages {
        fmt.Printf("AI: %s\n", msg.ContentString())
    }
}
```

## Examples

### Streaming Response
```go
resp, err := gengo.Generate(ctx, &chat.Request{
    Model: "gpt-4o-mini",
    Messages: []chat.Message{
        chat.NewTextMessage(chat.MessageRoleHuman, "Tell me a story"),
    },
}, chat.WithStream(func(chunk *chat.StreamResponse) {
    fmt.Print(chunk.Content)
}))
```

### Tool Calling
```go
resp, err := gengo.Generate(ctx, &chat.Request{
    Model: "gpt-4o-mini",
    Messages: []chat.Message{
        chat.NewTextMessage(chat.MessageRoleHuman, "What's the weather in Tokyo?"),
    },
    Tools: []chat.Tool{
        {
            Name:        "get_current_weather",
            Description: "Get the current weather in a given location",
            InputSchema: jsonschema.MustParseJSONString(`{
                "type": "object",
                "properties": {
                    "location": {"type": "string"}
                }
            }`),
        },
    },
})
```

### Image Input
```go
msg, err := chat.NewTextImageMessage(chat.MessageRoleHuman, "OCR this image", "./testdata/image.png")
if err != nil {
    panic(err)
}

resp, err := gengo.Generate(ctx, &chat.Request{
    Model: "gpt-4o-mini",
    Messages: []chat.Message{msg},
})
```

### JSON Schema Response
```go
result, err := gengo.Generate(ctx, &chat.Request{
    Model: "gpt-4o-mini",
    Messages: []chat.Message{
        chat.NewTextMessage(chat.MessageRoleHuman, "Convert to JSON: Tokyo is the capital of Japan"),
    },
    ResponseSchema: jsonschema.MustParseJSONString(`{
        "type": "object",
        "properties": {
            "city": {"type": "string"},
            "country": {"type": "string"}
        }
    }`),
})
```

## Configuration

### Environment Variables
- `OPENAI_API_KEY`: OpenAI API key
- `GOOGLE_API_KEY`: Google API key
- `ANTHROPIC_API_KEY`: Anthropic API key

## Tasks

### test

```
go test -v ./...
```

### lint

```
golangci-lint run
```

### updatecatalog

```
go run scripts/updatecatalog/main.go
```

### integrationtest

API keys required

```
go test -tags=integration_test -v
```


## License

MIT License
