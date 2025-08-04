package scheduler

import (
	"log"
	"time"

	"go-help-cs/internal/config"
	"go-help-cs/internal/freshdesk"
	"go-help-cs/internal/queue"
)

type Scheduler struct {
	config    *config.Config
	freshdesk *freshdesk.Client
	queue     *queue.Client
	stopChan  chan struct{}
}

func New(cfg *config.Config, freshdeskClient *freshdesk.Client, queueClient *queue.Client) *Scheduler {
	return &Scheduler{
		config:    cfg,
		freshdesk: freshdeskClient,
		queue:     queueClient,
		stopChan:  make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	log.Printf("Starting scheduler with interval: %v", s.config.Scheduler.Interval)

	ticker := time.NewTicker(s.config.Scheduler.Interval)
	defer ticker.Stop()

	// Запускаем первую проверку сразу
	s.fetchNewTickets()

	for {
		select {
		case <-ticker.C:
			s.fetchNewTickets()
		case <-s.stopChan:
			log.Println("Scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
}

func (s *Scheduler) fetchNewTickets() {
	log.Println("Fetching new tickets from Freshdesk...")

	tickets, err := s.freshdesk.GetNewTickets()
	if err != nil {
		log.Printf("Failed to fetch tickets: %v", err)
		return
	}

	log.Printf("Found %d tickets", len(tickets))

	for _, ticket := range tickets {
		// Проверяем, является ли тикет системным
		isSystem := s.freshdesk.IsSystemTicket(ticket, s.config.Email.SystemEmails)

		// Создаем сообщение для очереди
		msg := queue.NewTicketMessage(
			ticket.ID,
			ticket.Subject,
			ticket.Description,
			isSystem,
		)

		// Отправляем в очередь
		if err := s.queue.PublishTicket(msg); err != nil {
			log.Printf("Failed to publish ticket %d to queue: %v", ticket.ID, err)
			continue
		}

		log.Printf("Published ticket %d to queue (system: %v)", ticket.ID, isSystem)
	}
}
