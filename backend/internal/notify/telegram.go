package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const telegramAPI = "https://api.telegram.org"

// muteButtonLabel — текст кнопки под сообщением.
const muteButtonLabel = "🔕 Отключить до завтра"

type apiResponse struct {
	OK          bool            `json:"ok"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

type apiError struct {
	Code        int
	Description string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("telegram %d: %s", e.Code, e.Description)
}

// isBlocked — пользователь не запускал бота или заблокировал его.
func isBlocked(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusForbidden
}

// isConflict — бот уже опрашивает Telegram в другом процессе (409).
func isConflict(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) &&
		(apiErr.Code == http.StatusConflict || strings.Contains(strings.ToLower(apiErr.Description), "conflict"))
}

type inlineKeyboard struct {
	InlineKeyboard [][]keyboardButton `json:"inline_keyboard"`
}

type keyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

type tgChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type tgUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

type tgMessage struct {
	Text string  `json:"text"`
	Date int64   `json:"date"`
	Chat tgChat  `json:"chat"`
	From *tgUser `json:"from"`
}

type tgCallbackQuery struct {
	ID      string     `json:"id"`
	Data    string     `json:"data"`
	From    tgUser     `json:"from"`
	Message *tgMessage `json:"message"`
}

type update struct {
	UpdateID      int64            `json:"update_id"`
	Message       *tgMessage       `json:"message"`
	CallbackQuery *tgCallbackQuery `json:"callback_query"`
}

func (s *Service) call(ctx context.Context, client *http.Client, method string, payload any) (*apiResponse, error) {
	if client == nil {
		client = s.client
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/bot%s/%s", telegramAPI, s.Token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var out apiResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("telegram %s: некорректный ответ (HTTP %d)", method, resp.StatusCode)
	}
	if !out.OK {
		return nil, &apiError{Code: out.ErrorCode, Description: out.Description}
	}
	return &out, nil
}

func (s *Service) getMe(ctx context.Context) (string, error) {
	out, err := s.call(ctx, s.client, "getMe", struct{}{})
	if err != nil {
		return "", err
	}
	var bot struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(out.Result, &bot); err != nil {
		return "", err
	}
	return bot.Username, nil
}

// sendMessage шлём сообщение; callback != "" добавляет кнопку под ним.
func (s *Service) sendMessage(ctx context.Context, chatID int64, text, callback string) error {
	payload := struct {
		ChatID      int64           `json:"chat_id"`
		Text        string          `json:"text"`
		ReplyMarkup *inlineKeyboard `json:"reply_markup,omitempty"`
	}{ChatID: chatID, Text: text}

	if callback != "" {
		payload.ReplyMarkup = &inlineKeyboard{
			InlineKeyboard: [][]keyboardButton{
				{{Text: muteButtonLabel, CallbackData: callback}},
			},
		}
	}

	_, err := s.call(ctx, s.client, "sendMessage", payload)
	return err
}

func (s *Service) answerCallback(ctx context.Context, id, text string) error {
	_, err := s.call(ctx, s.client, "answerCallbackQuery", map[string]any{
		"callback_query_id": id,
		"text":              text,
	})
	return err
}

func (s *Service) getUpdates(ctx context.Context, offset int64) ([]update, error) {
	out, err := s.call(ctx, s.pollClient, "getUpdates", map[string]any{
		"offset":          offset,
		"timeout":         25,
		"allowed_updates": []string{"message", "callback_query"},
	})
	if err != nil {
		return nil, err
	}
	var updates []update
	if err := json.Unmarshal(out.Result, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}
