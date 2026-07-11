package tools

import (
	"article-summarizer/connectors"
	"article-summarizer/constants"
	"article-summarizer/db"
	"context"
	"fmt"
	"log"

	"github.com/agenticgokit/agenticgokit/v1beta"
)

type JinaTool struct{}

func (t *JinaTool) Name() string        { return "jina_parser" }
func (t *JinaTool) Description() string { return constants.JinaToolDescription }
func (t *JinaTool) Execute(ctx context.Context, args map[string]any) (*v1beta.ToolResult, error) {
	log.Print("jina_parser called, args are", args)
	chatId := ctx.Value("chatId")
	if chatId == nil {
		return &v1beta.ToolResult{Success: false, Content: "Error missing chat context"}, fmt.Errorf("chat id is missing from request context")
	}
	url, ok := args["url"].(string)
	if !ok {
		return &v1beta.ToolResult{Success: false, Content: "Error parsing url"}, fmt.Errorf("url is missing, invalid, or not a string")
	}
	response, err := connectors.CallJina(url)
	if err != nil {
		log.Printf("Error calling Jina API: %v", err)
		return &v1beta.ToolResult{Success: false}, err
	}
	db.GetDB().UpdateChatInfo(fmt.Sprintf("%d", chatId), "", response)
	return &v1beta.ToolResult{Success: true, Content: response}, nil
}

type LastArticleTool struct{}

func (t *LastArticleTool) Name() string        { return "get_last_article" }
func (t *LastArticleTool) Description() string { return constants.LastArticleToolDescription }
func (t *LastArticleTool) Execute(ctx context.Context, args map[string]any) (*v1beta.ToolResult, error) {
	log.Print("get_last_article called, args are", args)
	chatId := ctx.Value("chatId")
	if chatId == nil {
		return &v1beta.ToolResult{Success: false, Content: "Error retrieving chat id"}, fmt.Errorf("chat id is missing, invalid, or not a string")
	}
	response, err := db.GetDB().GetLastSummary(fmt.Sprintf("%d", chatId))
	if err != nil {
		log.Printf("Error fetching last article: %v", err)
	}
	return &v1beta.ToolResult{Success: true, Content: response}, nil
}
