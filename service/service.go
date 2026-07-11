package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"article-summarizer/connectors"
	"article-summarizer/constants"
	"article-summarizer/db"
	dto "article-summarizer/models"

	"github.com/agenticgokit/agenticgokit/v1beta"
	"github.com/go-telegram/bot/models"
)

var (
	reCredits      = regexp.MustCompile(`^/credits\s?$`)
	reCurrentModel = regexp.MustCompile(`^/model\s?$`)
	reUpdateModel  = regexp.MustCompile(`^/model\s.+`)
	reModels       = regexp.MustCompile(`^/models.*$`)
	reWebsite      = regexp.MustCompile(`^https?://`)
)

func HandleCommands(ctx context.Context, msg string, update *models.Update) string {
	switch {
	case reCredits.MatchString(msg):
		return creditsApiCall()
	case reCurrentModel.MatchString(msg):
		return getCurrentModel(update.Message.Chat.ID)
	case reUpdateModel.MatchString(msg):
		err := updateModel(msg, update.Message.Chat.ID)
		if err != nil {
			return err.Error()
		}
		return "Model updated successfully!"
	case reModels.MatchString(msg):
		return modelsApiCall(msg)
	}
	return callAgent(ctx, msg, update.Message.Chat.ID)
}

func creditsApiCall() string {
	body, err := connectors.GetOpenRouterCredits()
	if err != nil {
		log.Printf("Error occurred while fetching OpenRouter credits: %v", err)
		return ""
	}
	return string(body)
}

func getCurrentModel(chatId int64) string {
	model, err := db.GetDB().GetModelForChatId(fmt.Sprintf("%d", chatId))
	if err != nil {
		log.Printf("Error occurred while fetching model for chat ID %d: %v", chatId, err)
		return ""
	}
	return model
}

func updateModel(msg string, chatId int64) error {
	model := msg[len("/model "):]
	err := db.GetDB().UpdateChatInfo(fmt.Sprintf("%d", chatId), model, "")
	if err != nil {
		log.Printf("Error occurred while updating model for chat ID %d: %v", chatId, err)
	}
	return err
}

func modelsApiCall(msg string) string {
	filter := ""
	if len(msg) > len("/models ") {
		filter = strings.TrimSpace(msg[len("/models "):])
	}

	body, err := connectors.GetOpenRouterModels(filter)
	if err != nil {
		log.Printf("Error occurred while fetching OpenRouter models: %v", err)
		return ""
	}

	var response dto.ModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Error occurred while unmarshaling response: %v", err)
		return ""
	}

	var short []dto.SimplifiedModel
	var ids []string
	for _, model := range response.Data {

		simplified := dto.SimplifiedModel{
			ID:            model.ID,
			ContextLength: model.ContextLength,
			Modality:      model.Architecture.Modality,
			Pricing:       fmt.Sprintf("%s/%s", model.Pricing.Prompt, model.Pricing.Completion),
		}
		short = append(short, simplified)
		ids = append(ids, model.ID)

		if len(short) >= 10 {
			break
		}
	}

	// result, err := json.Marshal(short)
	// if err != nil {
	// 	log.Printf("Error occurred while marshaling filtered models: %v", err)
	// 	return ""
	// }

	return strings.Join(ids, "\n")
}

func callAgent(ctx context.Context, msg string, chatId int64) string {
	os.Setenv("AGK_TRACE", "true") // Enable observability

	model := getCurrentModel(chatId)
	requestCtx := context.WithValue(ctx, "chatId", chatId)

	agent, err := v1beta.NewBuilder("ChatAgent").
		WithConfig(&v1beta.Config{
			Name:         "ChatAgent",
			SystemPrompt: constants.SystemPrompt,
			Timeout:      120,
			LLM: v1beta.LLMConfig{
				Provider:  "openrouter",
				Model:     model,
				APIKey:    connectors.OpenRouterApiKey,
				MaxTokens: 50000,
			},
		}).
		WithTools().
		WithHandler(customHandler).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Execution
	result, err := agent.Run(requestCtx, msg)
	// stream, err := agent.RunStream(requestCtx, msg)
	if err != nil {
		log.Fatal(err)
	}
	// result, _ := stream.Wait()
	log.Print("Response: ", result.Content)
	log.Print(result.TokensUsed)
	return result.Content
}

func customHandler(ctx context.Context, input string, caps *v1beta.Capabilities) (string, error) {
	log.Print("customHandler: ", input)
	lines := strings.Split(input, "\n")
	var toolName string
	var toolInput string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "Action: "):
			toolName = strings.Split(line, ": ")[1]
		case strings.HasPrefix(line, "Action Input: "):
			toolInput = line[len("Action Input: "):]
		}
	}
	log.Print("customHandler: ", toolName, " called with ", toolInput)
	args := make(map[string]any)
	if err := json.Unmarshal([]byte(toolInput), &args); err != nil {
		log.Printf("Error parsing toolInput: %v", err)
		return caps.LLM(fmt.Sprintf("Error parsing toolInput: %v", err), input)
	}

	res, err := v1beta.ExecuteToolByName(ctx, toolName, args)
	if err != nil {
		if caps != nil && caps.LLM != nil {
			return caps.LLM("", fmt.Sprintf("Tool execution failed: %v", err))
		}
		return fmt.Sprintf("Tool execution failed: %v", err), nil
	}

	if caps != nil && caps.LLM != nil {
		return caps.LLM("", fmt.Sprint(res.Content))
	}

	return fmt.Sprint(res.Content), nil
}
