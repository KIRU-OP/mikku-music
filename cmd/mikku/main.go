package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-telegram/bot"
	"github.com/KIRU-OP/mikku-music/internal/app"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" { log.Fatal("BOT_TOKEN is required") }
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	service := app.New()
	tg, err := bot.New(token, bot.WithDefaultHandler(service.Handle))
	if err != nil { log.Fatalf("initialize Telegram bot: %v", err) }
	log.Println("Mikku Music Go bot starting")
	tg.Start(ctx)
	log.Println("Mikku Music Go bot stopped")
}
