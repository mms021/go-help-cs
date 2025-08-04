package analyzer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go-help-cs/internal/ollama"
)

type Analyzer struct {
	noOrdersPattern *regexp.Regexp
	ollamaClient    *ollama.Client
}

type AnalysisResult struct {
	IsNoOrdersAlert   bool
	DaysWithoutOrders int
	IsSystemAlert     bool
	Priority          string
	CustomerInfo      string
	TechnicalIssue    bool
	RequiresAttention bool
}

type TicketSummary struct {
	Type                       string `json:"type"`
	Priority                   string `json:"priority"`
	IsNoOrders                 bool   `json:"is_no_orders"`
	DaysWithoutOrders          int    `json:"days_without_orders"`
	CustomerInfo               string `json:"customer_info"`
	TechnicalIssue             bool   `json:"technical_issue"`
	RequiresImmediateAttention bool   `json:"requires_immediate_attention"`
}

func New(ollamaClient *ollama.Client) *Analyzer {
	// Паттерн для поиска сообщений о отсутствии заказов
	noOrdersPattern := regexp.MustCompile(`(?i)(no orders?|заказов? нет|нет заказов?)\s*(?:for|в течение|за)?\s*(\d+)\s*(?:days?|дней?)`)

	return &Analyzer{
		noOrdersPattern: noOrdersPattern,
		ollamaClient:    ollamaClient,
	}
}

func (a *Analyzer) AnalyzeSystemMessage(subject, description string) *AnalysisResult {
	// Объединяем тему и описание для анализа
	text := strings.ToLower(subject + " " + description)

	// Получаем краткую информацию о тикете от LLM
	ticketSummary, err := a.getTicketSummary(subject, description)
	if err != nil {
		// Если не удалось получить от LLM, используем базовый анализ
		return a.basicAnalysis(text)
	}

	// Анализируем с помощью LLM
	return a.analyzeWithLLM(text, ticketSummary)
}

func (a *Analyzer) getTicketSummary(subject, description string) (string, error) {
	prompt := fmt.Sprintf(`Проанализируй этот тикет и предоставь краткую информацию в формате JSON:

Тема: %s
Описание: %s

Ответь в формате JSON:
{
  "type": "тип_проблемы",
  "priority": "высокий/средний/низкий",
  "is_no_orders": true/false,
  "days_without_orders": число_дней,
  "customer_info": "информация_о_клиенте",
  "technical_issue": true/false,
  "requires_immediate_attention": true/false
}`, subject, description)

	return a.ollamaClient.GenerateResponse(prompt)
}

func (a *Analyzer) analyzeWithLLM(text, llmResponse string) *AnalysisResult {
	result := &AnalysisResult{
		IsSystemAlert: true,
		Priority:      "Medium",
	}

	// Пытаемся распарсить JSON ответ от LLM
	var summary TicketSummary
	if err := json.Unmarshal([]byte(llmResponse), &summary); err == nil {
		// Успешно распарсили JSON
		result.IsNoOrdersAlert = summary.IsNoOrders
		result.DaysWithoutOrders = summary.DaysWithoutOrders
		result.CustomerInfo = summary.CustomerInfo
		result.TechnicalIssue = summary.TechnicalIssue
		result.RequiresAttention = summary.RequiresImmediateAttention

		// Устанавливаем приоритет
		switch strings.ToLower(summary.Priority) {
		case "высокий", "high":
			result.Priority = "High"
		case "низкий", "low":
			result.Priority = "Low"
		default:
			result.Priority = "Medium"
		}
	} else {
		// Если не удалось распарсить JSON, используем базовый анализ
		return a.basicAnalysis(text)
	}

	return result
}

func (a *Analyzer) basicAnalysis(text string) *AnalysisResult {
	result := &AnalysisResult{
		IsSystemAlert: true,
		Priority:      "Medium",
	}

	// Проверяем на сообщения о отсутствии заказов
	if a.noOrdersPattern.MatchString(text) {
		result.IsNoOrdersAlert = true
		result.Priority = "High"

		// Извлекаем количество дней
		matches := a.noOrdersPattern.FindStringSubmatch(text)
		if len(matches) > 1 {
			// Пытаемся извлечь число дней из текста
			daysPattern := regexp.MustCompile(`(\d+)`)
			daysMatches := daysPattern.FindStringSubmatch(text)
			if len(daysMatches) > 1 {
				result.DaysWithoutOrders = 5
			}
		}
	}

	if a.isHighPriorityAlert(text) {
		result.Priority = "High"
	}

	return result
}

func (a *Analyzer) isHighPriorityAlert(text string) bool {
	highPriorityPatterns := []string{
		"error", "ошибка", "critical", "критическая",
		"down", "не работает", "сбой", "failure",
		"emergency", "срочно", "urgent", "критично",
		"security", "безопасность", "breach", "взлом",
	}

	for _, pattern := range highPriorityPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	return false
}

func (a *Analyzer) ExtractCustomerInfo(subject, description string) string {
	text := subject + " " + description

	// Ищем email
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	if email := emailPattern.FindString(text); email != "" {
		return "Email: " + email
	}

	// Ищем номер телефона
	phonePattern := regexp.MustCompile(`(?:\+7|8)[- (]?\d{3}[- )]?\d{3}[- ]?\d{2}[- ]?\d{2}`)
	if phone := phonePattern.FindString(text); phone != "" {
		return "Phone: " + phone
	}

	// Ищем ID клиента
	idPattern := regexp.MustCompile(`(?:client|клиент|id)[\s:]*(\d+)`)
	if matches := idPattern.FindStringSubmatch(text); len(matches) > 1 {
		return "Client ID: " + matches[1]
	}

	return "Информация о клиенте не найдена"
}
