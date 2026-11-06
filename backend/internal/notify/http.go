package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/workhourscounter/backend/internal/schedule"
)

type Status struct {
	Enabled    bool   `json:"enabled"`
	MutedUntil string `json:"mutedUntil"`
}

// Notifications — GET: текущие настройки, PUT: включить/выключить.
// Включение заодно снимает режим «до завтра».
func (s *Service) Notifications(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleStatus(w, r)
	case http.MethodPut:
		s.handleSetEnabled(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStatus возвращает текущие настройки уведомлений.
func (s *Service) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	telegramID, _, ok := authUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	status, err := s.loadStatus(r.Context(), telegramID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, status)
}

// handleSetEnabled включает/выключает уведомления.
func (s *Service) handleSetEnabled(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	telegramID, username, ok := authUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || body.Enabled == nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	enabled := *body.Enabled

	ctx := r.Context()
	userID, err := s.Schedule.EnsureUser(ctx, telegramID, username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO notification_settings (user_id, enabled, muted_until)
		VALUES ($1, $2, NULL)
		ON CONFLICT (user_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			muted_until = CASE WHEN EXCLUDED.enabled THEN NULL ELSE notification_settings.muted_until END,
			updated_at = NOW()
	`, userID, enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status, err := s.loadStatusByUser(ctx, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, status)
}

// Test отправляет пробное уведомление с той же кнопкой «отключить до завтра».
func (s *Service) Test(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.ready(w) {
		return
	}
	telegramID, username, ok := authUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.Token == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "TELEGRAM_BOT_TOKEN не задан на сервере"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	userID, err := s.Schedule.EnsureUser(ctx, telegramID, username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	date, minutes := schedule.MinskClock(time.Now())
	next := s.upcomingLesson(ctx, userID, date, minutes)
	text := buildTestMessage(date, next)

	if err := s.sendMessage(ctx, telegramID, text, muteCallback(date)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": friendlySendError(err)})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// upcomingLesson — ближайший ещё не закончившийся урок дня (или первый).
func (s *Service) upcomingLesson(ctx context.Context, userID, date string, minutes int) *lessonInfo {
	lessons, err := s.userLessons(ctx, userID, date)
	if err != nil || len(lessons) == 0 {
		return nil
	}
	var first *lessonInfo
	for i := range lessons {
		if first == nil {
			first = &lessons[i]
		}
		_, end, ok := schedule.BellTimes(date, lessons[i].Period)
		if ok && end > minutes {
			return &lessons[i]
		}
	}
	return first
}

func (s *Service) userLessons(ctx context.Context, userID, date string) ([]lessonInfo, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT period, group_name, subject, room, manual, conducted
		FROM schedule_lessons
		WHERE user_id = $1 AND date = $2
		ORDER BY period
	`, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []lessonInfo
	for rows.Next() {
		var lesson lessonInfo
		if err := rows.Scan(&lesson.Period, &lesson.Group, &lesson.Subject, &lesson.Room, &lesson.Manual, &lesson.Conducted); err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	return lessons, rows.Err()
}

func (s *Service) loadStatus(ctx context.Context, telegramID int64) (Status, error) {
	return s.queryStatus(ctx, `
		SELECT ns.enabled, to_char(ns.muted_until AT TIME ZONE 'Europe/Minsk', 'YYYY-MM-DD HH24:MI')
		FROM notification_settings ns
		JOIN users u ON u.id = ns.user_id
		WHERE u.telegram_id = $1
	`, telegramID)
}

func (s *Service) loadStatusByUser(ctx context.Context, userID string) (Status, error) {
	return s.queryStatus(ctx, `
		SELECT enabled, to_char(muted_until AT TIME ZONE 'Europe/Minsk', 'YYYY-MM-DD HH24:MI')
		FROM notification_settings
		WHERE user_id = $1
	`, userID)
}

func (s *Service) queryStatus(ctx context.Context, query string, arg any) (Status, error) {
	var status Status
	var muted sql.NullString
	if err := s.DB.QueryRowContext(ctx, query, arg).Scan(&status.Enabled, &muted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Настроек ещё нет: по умолчанию уведомления включены.
			return Status{Enabled: true}, nil
		}
		return Status{}, err
	}
	if muted.Valid {
		status.MutedUntil = muted.String
	}
	return status, nil
}

func (s *Service) ready(w http.ResponseWriter) bool {
	if s.DB == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return false
	}
	return true
}

func friendlySendError(err error) string {
	if isBlocked(err) {
		return "Бот не смог вам написать. Откройте бота, нажмите /start и повторите."
	}
	return fmt.Sprintf("Не удалось отправить сообщение: %v", err)
}

func authUser(r *http.Request) (int64, string, bool) {
	id, err := strconv.ParseInt(r.Header.Get("X-Telegram-User-ID"), 10, 64)
	if err != nil || id == 0 {
		return 0, "", false
	}
	return id, r.Header.Get("X-Telegram-Username"), true
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
