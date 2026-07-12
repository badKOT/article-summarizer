package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"article-summarizer/connectors"
	"article-summarizer/db"
	"article-summarizer/service"
	"article-summarizer/tools"

	"github.com/agenticgokit/agenticgokit/v1beta"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// init db
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "summarizer")
	dbPassword := getEnv("DB_PASSWORD", "summarizer123")
	dbName := getEnv("DB_NAME", "article_summarizer")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	if err := db.InitDB(connStr); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Print("Database initialized successfully!")
	defer db.Close()

	// init tools for agents
	jina := tools.JinaTool{}
	lastArticle := tools.LastArticleTool{}
	v1beta.RegisterInternalTool(jina.Name(), func() v1beta.Tool { return &jina })
	v1beta.RegisterInternalTool(lastArticle.Name(), func() v1beta.Tool { return &lastArticle })
	connectors.OpenRouterApiKey = os.Getenv("OPENROUTER_API_KEY")
	connectors.JinaApiKey = os.Getenv("JINA_READER_API_KEY")

	// init bot and webhook
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(os.Getenv("TELEGRAM_API_TOKEN"), opts...)
	if err != nil {
		panic(err)
	}
	log.Print("Telegram Bot was initialized!")

	b.SetWebhook(ctx, &bot.SetWebhookParams{
		URL: os.Getenv("WEBHOOK_URL"),
	})

	go func() {
		log.Print("Starting HTTP webhook server on :8080")
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.Handle("/", b.WebhookHandler())
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	b.StartWebhook(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message.Text
	log.Printf("Received message from chat %d: '%s'", update.Message.Chat.ID, msg)
	response := service.HandleCommands(ctx, msg, update)
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   response,
	}); err != nil {
		log.Printf("Failed to send message to chat %d: %v", update.Message.Chat.ID, err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
