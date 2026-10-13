package worker

import (
	"log"

	"go-help-cs/internal/analyzer"
	"go-help-cs/internal/config"
	"go-help-cs/internal/database"
	"go-help-cs/internal/freshdesk"
	"go-help-cs/internal/ollama"
	"go-help-cs/internal/queue"
	"go-help-cs/internal/telegram"
)

type Worker struct {
	config    *config.Config
	freshdesk *freshdesk.Client
	ollama    *ollama.Client
	database  *database.Database
	queue     *queue.Client
	analyzer  *analyzer.Analyzer
	telegram  *telegram.Client
}

func New(cfg *config.Config, freshdeskClient *freshdesk.Client, ollamaClient *ollama.Client, db *database.Database, queueClient *queue.Client) *Worker {
	telegramClient := telegram.New(cfg.Telegram.BotToken, cfg.Telegram.ChatID)

	return &Worker{
		config:    cfg,
		freshdesk: freshdeskClient,
		ollama:    ollamaClient,
		database:  db,
		queue:     queueClient,
		analyzer:  analyzer.New(ollamaClient),
		telegram:  telegramClient,
	}
}

func (w *Worker) Start() error {
	log.Println("Starting worker...")

	return w.queue.ConsumeTickets(w.processTicket)
}

func (w *Worker) processTicket(msg *queue.TicketMessage) error {
	log.Printf("Processing ticket %d (system: %v)", msg.TicketID, msg.IsSystem)

	// Сохраняем тикет в базу данных
	ticket := &database.Ticket{
		ID:          msg.TicketID,
		Subject:     msg.Subject,
		Description: msg.Description,
		IsSystem:    msg.IsSystem,
	}

	if err := w.database.SaveTicket(ticket); err != nil {
		log.Printf("Failed to save ticket %d to database: %v", msg.TicketID, err)
		return err
	}

	// Анализируем системные сообщения
	if msg.IsSystem {
		w.processSystemTicket(msg)
	}

	// Генерируем ответ с помощью Ollama
	response, err := w.ollama.GenerateTicketResponse(msg.Subject, msg.Description, msg.IsSystem, w.config.Ollama.ModelCS, w.config.Ollama.ModelSupport)
	if err != nil {
		log.Printf("Failed to generate response for ticket %d: %v", msg.TicketID, err)
		return err
	}

	log.Printf("Generated response for ticket %d: %s", msg.TicketID, response[:min(len(response), 100)]+"...")

	// Отправляем комментарий в Freshdesk
	isPrivate := msg.IsSystem // Системные тикеты получают приватные комментарии
	if err := w.freshdesk.CreateComment(msg.TicketID, response, isPrivate); err != nil {
		log.Printf("Failed to create comment for ticket %d: %v", msg.TicketID, err)
		return err
	}

	// Сохраняем комментарий в базу данных
	comment := &database.TicketComment{
		TicketID:  msg.TicketID,
		Body:      response,
		IsPrivate: isPrivate,
	}

	if err := w.database.SaveComment(comment); err != nil {
		log.Printf("Failed to save comment for ticket %d to database: %v", msg.TicketID, err)
		// Не возвращаем ошибку, так как комментарий уже отправлен в Freshdesk
	}

	// Отмечаем тикет как обработанный
	if err := w.database.MarkTicketProcessed(msg.TicketID); err != nil {
		log.Printf("Failed to mark ticket %d as processed: %v", msg.TicketID, err)
		// Не возвращаем ошибку, так как основная обработка завершена
	}

	log.Printf("Successfully processed ticket %d", msg.TicketID)
	return nil
}

func (w *Worker) processSystemTicket(msg *queue.TicketMessage) {
	// Анализируем системное сообщение
	analysis := w.analyzer.AnalyzeSystemMessage(msg.Subject, msg.Description)

	// Используем информацию о клиенте из анализатора или извлекаем вручную
	customerInfo := analysis.CustomerInfo
	if customerInfo == "" {
		customerInfo = w.analyzer.ExtractCustomerInfo(msg.Subject, msg.Description)
	}

	// Отправляем уведомление в Telegram
	if analysis.IsNoOrdersAlert {
		log.Printf("Sending no orders alert for ticket %d (days: %d)", msg.TicketID, analysis.DaysWithoutOrders)
		if err := w.telegram.SendNoOrdersAlert(msg.TicketID, customerInfo, analysis.DaysWithoutOrders); err != nil {
			log.Printf("Failed to send no orders alert: %v", err)
		}
	} else if analysis.Priority == "High" || analysis.RequiresAttention {
		log.Printf("Sending high priority alert for ticket %d (priority: %s)", msg.TicketID, analysis.Priority)
		if err := w.telegram.SendAlert(msg.TicketID, msg.Subject, msg.Description, customerInfo); err != nil {
			log.Printf("Failed to send alert: %v", err)
		}
	}

	// Логируем дополнительную информацию
	if analysis.TechnicalIssue {
		log.Printf("Technical issue detected in ticket %d", msg.TicketID)
	}
}
