package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	commandStart  = "start"
	commandAdd    = "add"
	commandBudget = "budget"
	commandLang   = "lang"
	commandHelp   = "help"
)

// Cloud Expenses Bot for personal budget usage
type Bot struct {
	api          *tgbotapi.BotAPI
	sheetsClient *SheetsClient
	config       Config
	messages     Messages
	userLang     map[int64]string
}

// Localication
type Messages map[string]map[string]string

// JSON configs
type Config struct {
	SpreadsheetID string `json:"spreadsheet_id"`
	BotToken      string `json:"bot_token"`
	CellRanges    struct {
		DailyExpenses  string `json:"daily_expenses"`
		CategoryRange  string `json:"category_range"`
		CategoryColumn string `json:"category_column"`
		BudgetColumn   string `json:"budget_column"`
	} `json:"cell_ranges"`
}

// Google Sheets API
type SheetsClient struct {
	service *sheets.Service
}

func NewBot(ctx context.Context) (*Bot, error) {
	config, err := loadConfig("config.json")
	if err != nil {
		return nil, err
	}

	api, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	sheetsClient, err := NewSheetsClient(ctx)
	if err != nil {
		return nil, err
	}

	messages, err := loadMessages("lang.json")
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:          api,
		sheetsClient: sheetsClient,
		config:       config,
		messages:     messages,
		userLang:     make(map[int64]string),
	}, nil
}

func NewSheetsClient(ctx context.Context) (*SheetsClient, error) {
	creds, err := os.ReadFile("credentials.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials.json: %w", err)
	}

	config, err := google.ConfigFromJSON(creds, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Google config: %w", err)
	}

	client, err := getClient(ctx, config)
	if err != nil {
		return nil, err
	}

	service, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Sheets service: %w", err)
	}

	return &SheetsClient{service: service}, nil
}

func loadConfig(path string) (Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return config, nil
}

func loadMessages(path string) (Messages, error) {
	var messages Messages
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return messages, nil
}

func getClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	tokenFile := "token.json"
	token, err := tokenFromFile(tokenFile)
	if err != nil {
		token, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenFile, token); err != nil {
			return nil, err
		}
	}

	if !token.Valid() {
		tokenSource := config.TokenSource(ctx, token)
		token, err = tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		if err := saveToken(tokenFile, token); err != nil {
			return nil, err
		}
	}

	return config.Client(ctx, token), nil
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for authorization:\n%v\n", authURL)

	fmt.Print("Enter authorization code: ")
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	return token, nil
}

func saveToken(path string, token *oauth2.Token) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create token file %s: %w", path, err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(token); err != nil {
		return fmt.Errorf("failed to encode token: %w", err)
	}
	return nil
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var token oauth2.Token
	if err := json.NewDecoder(file).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (b *Bot) getSheetName(lang string) string {
	now := time.Now()
	year := now.Year()
	month := now.Month()

	monthKey := fmt.Sprintf("month_%d", month)
	monthName := b.messages[lang][monthKey]
	return fmt.Sprintf("%s %d", monthName, year)
}

func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			go b.handleUpdate(ctx, update.Message)
		} else if update.CallbackQuery != nil {
			go func() {
				b.handleCallbackQuery(update.CallbackQuery)
			}()
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, message *tgbotapi.Message) {
	userID := message.From.ID
	lang := b.getUserLanguage(userID)

	if message.IsCommand() {
		switch message.Command() {
		case commandStart:
			b.sendMessage(message.Chat.ID, b.messages[lang]["start"])
		case commandAdd:
			b.handleAddCommand(ctx, message, lang)
		case commandBudget:
			b.handleBudgetCommand(ctx, message, lang)
		case commandLang:
			b.handleLangCommand(message)
		case commandHelp:
			b.handleHelpCommand(message)
		default:
			b.sendMessage(message.Chat.ID, b.messages[lang]["unknown_command"])
		}
	} else {
		b.sendMessage(message.Chat.ID, b.messages[lang]["start"])
	}
}

func (b *Bot) handleCallbackQuery(cq *tgbotapi.CallbackQuery) {
	data := cq.Data
	var lang string
	switch data {
	case "help_ru":
		lang = "ru"
	case "help_en":
		lang = "en"
	default:
		return
	}

	helpMessage := b.messages[lang]["help_message"]

	msg := tgbotapi.NewMessage(cq.Message.Chat.ID, helpMessage)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send help message: %v", err)
	}

	callback := tgbotapi.NewCallback(cq.ID, "")
	if _, err := b.api.Request(callback); err != nil {
		log.Printf("Failed to answer callback query: %v", err)
	}
}

func (b *Bot) getUserLanguage(userID int64) string {
	if lang, exists := b.userLang[userID]; exists {
		return lang
	}
	return "ru" // Default language
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// /add
func (b *Bot) handleAddCommand(ctx context.Context, message *tgbotapi.Message, lang string) {
	args := message.CommandArguments()
	fields := strings.Fields(args)

	if len(fields) < 3 {
		b.sendMessage(message.Chat.ID, b.messages[lang]["add_usage"])
		return
	}

	amountStr, suffix, category := fields[0], fields[1], strings.Join(fields[2:], " ")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		b.sendMessage(message.Chat.ID, b.messages[lang]["invalid_amount"])
		return
	}

	isCard, err := b.parsePaymentMethod(suffix, lang)
	if err != nil {
		b.sendMessage(message.Chat.ID, err.Error())
		return
	}

	sheetName := b.getSheetName(lang)
	if err := b.sheetsClient.recordExpense(ctx, b.config, sheetName, category, amount, isCard, lang, b.messages); err != nil {
		log.Printf("Error recording expense: %v", err)
		b.sendMessage(message.Chat.ID, fmt.Sprintf(b.messages[lang]["error_occurred"], err))
		return
	}

	paymentMethod := b.messages[lang]["payment_cash"]
	if isCard {
		paymentMethod = b.messages[lang]["payment_card"]
	}

	reply := fmt.Sprintf(b.messages[lang]["expense_added"], amount, category, paymentMethod)
	b.sendMessage(message.Chat.ID, reply)
}

// parse the payment method from the suffix - card or cash
func (b *Bot) parsePaymentMethod(suffix, lang string) (bool, error) {
	switch suffix {
	case b.messages[lang]["suffix_card"]:
		return true, nil
	case b.messages[lang]["suffix_cash"]:
		return false, nil
	default:
		return false, errors.New(b.messages[lang]["invalid_suffix"])
	}
}

// /budget
func (b *Bot) handleBudgetCommand(ctx context.Context, message *tgbotapi.Message, lang string) {
	sheetName := b.getSheetName(lang)
	budget, err := b.sheetsClient.getDailyBudget(ctx, b.config, sheetName)
	if err != nil {
		log.Printf("Error getting daily budget: %v", err)
		b.sendMessage(message.Chat.ID, fmt.Sprintf(b.messages[lang]["error_occurred"], err))
		return
	}

	budgetValue, err := strconv.ParseFloat(budget, 64)
	if err != nil {
		log.Printf("Error parsing budget value: %v", err)
		b.sendMessage(message.Chat.ID, fmt.Sprintf(b.messages[lang]["error_occurred"], err))
		return
	}
	formattedBudget := fmt.Sprintf("%.2f", budgetValue)

	reply := fmt.Sprintf(b.messages[lang]["daily_budget"], formattedBudget)
	b.sendMessage(message.Chat.ID, reply)
}

// /lang
func (b *Bot) handleLangCommand(message *tgbotapi.Message) {
	args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
	if args == "en" || args == "ru" {
		b.userLang[message.From.ID] = args
		reply := fmt.Sprintf(b.messages[args]["language_set"], args)
		b.sendMessage(message.Chat.ID, reply)
	} else {
		lang := b.getUserLanguage(message.From.ID)
		b.sendMessage(message.Chat.ID, b.messages[lang]["select_language"])
	}
}

// /help
func (b *Bot) handleHelpCommand(message *tgbotapi.Message) {
	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Русский", "help_ru"),
		tgbotapi.NewInlineKeyboardButtonData("🇬🇧 English", "help_en"),
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)

	msg := tgbotapi.NewMessage(message.Chat.ID, "Please select your language / Пожалуйста, выберите язык:")
	msg.ReplyMarkup = keyboard
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send help message: %v", err)
	}
}

// write data to sheet
func (sc *SheetsClient) recordExpense(ctx context.Context, config Config, sheetName, category string, amount float64, isCard bool, lang string, messages Messages) error {
	if err := sc.writeExpenseToDailyCell(ctx, config, sheetName, amount, isCard); err != nil {
		return fmt.Errorf("failed to write to daily cell: %w", err)
	}

	if err := sc.writeExpenseToCategoryCell(ctx, config, sheetName, category, amount, isCard, lang, messages); err != nil {
		return fmt.Errorf("failed to write to category cell: %w", err)
	}

	return nil
}

func (sc *SheetsClient) writeExpenseToDailyCell(ctx context.Context, config Config, sheetName string, amount float64, isCard bool) error {
	now := time.Now()
	row := now.Day() + 1
	cell := fmt.Sprintf("%s!I%d", sheetName, row)

	currentValue, err := sc.getCellValue(ctx, config.SpreadsheetID, cell, "FORMULA")
	if err != nil {
		return err
	}

	formattedAmount := sc.formatAmount(amount, isCard)
	newValue := sc.buildNewFormula(currentValue, formattedAmount)

	return sc.updateCellValue(ctx, config.SpreadsheetID, cell, newValue)
}

func (sc *SheetsClient) writeExpenseToCategoryCell(ctx context.Context, config Config, sheetName, category string, amount float64, isCard bool, lang string, messages Messages) error {
	categoryRange := fmt.Sprintf("'%s'!%s", sheetName, config.CellRanges.CategoryRange)
	resp, err := sc.service.Spreadsheets.Values.Get(config.SpreadsheetID, categoryRange).Do()
	if err != nil {
		return fmt.Errorf("failed to get category range: %w", err)
	}

	rowIndex, found := sc.findCategoryRow(resp.Values, category)
	if !found {
		return fmt.Errorf(messages[lang]["category_not_found"], category)
	}

	row := 22 + rowIndex
	cell := fmt.Sprintf("%s!%s%d", sheetName, config.CellRanges.CategoryColumn, row)
	currentValue, err := sc.getCellValue(ctx, config.SpreadsheetID, cell, "FORMULA")
	if err != nil {
		return err
	}

	formattedAmount := sc.formatAmount(amount, isCard)
	newValue := sc.buildNewFormula(currentValue, formattedAmount)

	return sc.updateCellValue(ctx, config.SpreadsheetID, cell, newValue)
}

func (sc *SheetsClient) getDailyBudget(ctx context.Context, config Config, sheetName string) (string, error) {
	now := time.Now()
	row := now.Day() + 1
	cell := fmt.Sprintf("%s!%s%d", sheetName, config.CellRanges.BudgetColumn, row)

	value, err := sc.getCellValue(ctx, config.SpreadsheetID, cell, "UNFORMATTED_VALUE")
	if err != nil {
		return "", err
	}
	return value, nil
}

func (sc *SheetsClient) getCellValue(ctx context.Context, spreadsheetID, cell, valueRenderOption string) (string, error) {
	resp, err := sc.service.Spreadsheets.Values.Get(spreadsheetID, cell).ValueRenderOption(valueRenderOption).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get cell %s: %w", cell, err)
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
		return "", nil
	}

	value := fmt.Sprintf("%v", resp.Values[0][0])
	if valueRenderOption == "FORMULA" {
		value = strings.TrimPrefix(value, "=")
	}
	return value, nil
}

func (sc *SheetsClient) updateCellValue(ctx context.Context, spreadsheetID, cell, value string) error {
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{value}},
	}

	_, err := sc.service.Spreadsheets.Values.Update(spreadsheetID, cell, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to update cell %s: %w", cell, err)
	}
	return nil
}

func (sc *SheetsClient) formatAmount(amount float64, isCard bool) string {
	formatted := fmt.Sprintf("%.2f", amount)
	if !isCard {
		return fmt.Sprintf("(%s)", formatted)
	}
	return formatted
}

func (sc *SheetsClient) buildNewFormula(currentValue, formattedAmount string) string {
	if currentValue == "" {
		return fmt.Sprintf("=%s", formattedAmount)
	}
	return fmt.Sprintf("=%s+%s", currentValue, formattedAmount)
}

func (sc *SheetsClient) findCategoryRow(values [][]interface{}, category string) (int, bool) {
	for i, row := range values {
		if len(row) > 0 && strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", row[0])), strings.TrimSpace(category)) {
			return i, true
		}
	}
	return 0, false
}

func main() {
	ctx := context.Background()
	bot, err := NewBot(ctx)
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
	}
	bot.Start(ctx)
}
