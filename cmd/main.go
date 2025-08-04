package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-help-cs/internal/config"
	"go-help-cs/internal/database"
	"go-help-cs/internal/freshdesk"
	"go-help-cs/internal/ollama"
	"go-help-cs/internal/queue"
	"go-help-cs/internal/scheduler"
	"go-help-cs/internal/worker"
)

func main() {
	log.Println("Starting Ticket Processing System...")

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализируем базу данных
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Инициализируем клиенты
	freshdeskClient := freshdesk.New(&cfg.Freshdesk)
	ollamaClient := ollama.New(&cfg.Ollama)
	queueClient, err := queue.New(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}
	defer queueClient.Close()

	// Создаем воркер
	w := worker.New(cfg, freshdeskClient, ollamaClient, db, queueClient)

	// Создаем планировщик
	sched := scheduler.New(cfg, freshdeskClient, queueClient)

	// Создаем контекст для graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем воркер
	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	// Запускаем планировщик в отдельной горутине
	go sched.Start()

	log.Println("System started successfully. Press Ctrl+C to stop.")

	// Ожидаем сигнала для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Received shutdown signal. Stopping...")

	// Останавливаем планировщик
	sched.Stop()

	log.Println("System stopped gracefully.")
}
