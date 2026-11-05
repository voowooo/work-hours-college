package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type contextKey string

const TelegramIDKey contextKey = "telegramID"
const TelegramUsernameKey contextKey = "telegramUsername"

func GetTelegramID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(TelegramIDKey).(int64)
	return id, ok
}

func GetTelegramUsername(ctx context.Context) string {
	username, _ := ctx.Value(TelegramUsernameKey).(string)
	return username
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Accept X-Telegram-User-ID header
		if userID := r.Header.Get("X-Telegram-User-ID"); userID != "" {
			id, err := strconv.ParseInt(userID, 10, 64)
			if err == nil {
				ctx := context.WithValue(r.Context(), TelegramIDKey, id)
				username := r.Header.Get("X-Telegram-Username")
				if username != "" {
					ctx = context.WithValue(ctx, TelegramUsernameKey, username)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// No auth provided
		http.Error(w, `{"errors":[{"message":"unauthorized"}]}`, http.StatusUnauthorized)
	})
}

func validateInitData(initData string) (int64, error) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return 0, fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	parts := strings.Split(initData, "&")
	var dataCheck string
	params := make(map[string]string)

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "hash" {
			dataCheck = kv[1]
		} else {
			params[kv[0]] = kv[1]
		}
	}

	if dataCheck == "" {
		return 0, fmt.Errorf("hash not found")
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var dataCheckString strings.Builder
	for i, k := range keys {
		if i > 0 {
			dataCheckString.WriteString("\n")
		}
		dataCheckString.WriteString(k + "=" + params[k])
	}

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	mac := hmac.New(sha256.New, secretKey.Sum(nil))
	mac.Write([]byte(dataCheckString.String()))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if expectedHash != dataCheck {
		return 0, fmt.Errorf("invalid hash")
	}

	userStr, ok := params["user"]
	if !ok {
		return 0, fmt.Errorf("user not found")
	}

	// Simple extraction of ID from JSON-like string
	// In production, use proper JSON parsing
	idStr := extractField(userStr, "id")
	if idStr == "" {
		return 0, fmt.Errorf("user id not found")
	}

	return strconv.ParseInt(idStr, 10, 64)
}

func extractField(json, field string) string {
	search := fmt.Sprintf("\"%s\":", field)
	idx := strings.Index(json, search)
	if idx == -1 {
		return ""
	}
	idx += len(search)

	end := strings.IndexAny(json[idx:], ",}")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(json[idx : idx+end])
}
