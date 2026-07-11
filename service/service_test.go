package service

import (
	"article-summarizer/constants"
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/agenticgokit/agenticgokit/v1beta"
)

func TestCustomHandlerHandlesMissingCapabilities(t *testing.T) {
	// setup
	v1beta.RegisterInternalTool("jina_parser", func() v1beta.Tool { return &JinaStubTool{} })
	ctx := context.Background()
	ctx = context.WithValue(ctx, "chatId", "12345")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("customHandler panicked: %v", r)
		}
	}()
	caps := &v1beta.Capabilities{LLM: llmStub}

	response, err := customHandler(context.Background(), "hello", caps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if response == "" {
		t.Fatal("expected a fallback response")
	}
}

func TestCustomHandler(t *testing.T) {
	// setup
	v1beta.RegisterInternalTool("jina_parser", func() v1beta.Tool { return &JinaStubTool{} })
	ctx := context.Background()
	ctx = context.WithValue(ctx, "chatId", "12345")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("customHandler panickedL: %v", r)
		}
	}()
	caps := &v1beta.Capabilities{LLM: llmStub}

	response, err := customHandler(context.Background(), "I'll fetch that URL for you.\nAction: jina_parser\nAction Input: {\"url\": \"https://example.com\"}", caps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	log.Print(response)
	if response == "" {
		t.Fatal("expected a response")
	}
}

func TestCustomHandlerNumber(t *testing.T) {
	// setup
	v1beta.RegisterInternalTool("jina_parser", func() v1beta.Tool { return &JinaStubTool{} })
	ctx := context.Background()
	ctx = context.WithValue(ctx, "chatId", "12345")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("customHandler panickedL: %v", r)
		}
	}()
	caps := &v1beta.Capabilities{LLM: llmStub}

	response, err := customHandler(context.Background(), "I'll fetch that URL for you.\nAction: jina_parser\nAction Input: {\"url\": 123}", caps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	log.Print(response)
	if response == "" {
		t.Fatal("expected a response")
	}
}

func llmStub(system, user string) (string, error) {
	log.Printf("LLM Stub Received: System: %s, User: %s", system, user)
	return fmt.Sprintf("%s %s", system, user), nil
}

type JinaStubTool struct{}

func (t *JinaStubTool) Name() string        { return "jina_parser" }
func (t *JinaStubTool) Description() string { return constants.JinaToolDescription }
func (t *JinaStubTool) Execute(ctx context.Context, args map[string]any) (*v1beta.ToolResult, error) {
	return &v1beta.ToolResult{Success: true, Content: "Jina Stub received you request!"}, nil
}
