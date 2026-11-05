package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/workhourscounter/backend/internal/db"
)

type Service struct {
	DB *db.DB
}

type Plan struct {
	Group        string `json:"group"`
	Subject      string `json:"subject"`
	PlannedHours int    `json:"plannedHours"`
	Conducted    int    `json:"conducted"`
}

type importLesson struct {
	Date      string `json:"date"`
	Period    int    `json:"period"`
	Group     string `json:"group"`
	Subject   string `json:"subject"`
	Room      string `json:"room"`
	Conducted bool   `json:"conducted"`
}

type importBody struct {
	Lessons []importLesson `json:"lessons"`
	Plans   []Plan         `json:"plans"`
}

// markBody — ручная отметка урока. Ручной приоритетнее авто-расчета по звонкам,
// пока пользователь не снимет отметку.
type markBody struct {
	Date      string `json:"date"`
	Period    int    `json:"period"`
	Group     string `json:"group"`
	Subject   string `json:"subject"`
	Room      string `json:"room"`
	Conducted bool   `json:"conducted"`
}

func (s *Service) Schedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	date := r.URL.Query().Get("date")
	if _, err := time.Parse("2006-01-02", date); err != nil {
		http.Error(w, "date must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	day, err := s.day(ctx, r, date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, day)
}

func (s *Service) Plans(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	telegramID, username, ok := telegramUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		if err := s.sweep(ctx, telegramID, username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		plans, err := s.listPlans(ctx, telegramID, username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"plans": plans})
	case http.MethodPut:
		var body Plan
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if body.PlannedHours < 0 {
			body.PlannedHours = 0
		}
		if err := s.setPlan(ctx, telegramID, username, body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, body)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Mark — ручная отметка урока как проведённого/непроведённого.
func (s *Service) Mark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.DB == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	telegramID, username, ok := telegramUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body markBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", body.Date); err != nil || body.Period < 1 || body.Period > 14 {
		http.Error(w, "date must be YYYY-MM-DD and period 1-14", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := s.markLesson(ctx, telegramID, username, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Service) Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.DB == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	telegramID, username, ok := telegramUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body importBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := s.importLocal(ctx, telegramID, username, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Service) day(ctx context.Context, r *http.Request, date string) (DaySchedule, error) {
	telegramID, username, hasUser := telegramUser(r)
	now := time.Now()

	var stored []Lesson
	if hasUser && s.DB != nil {
		if err := s.sweep(ctx, telegramID, username); err != nil {
			return DaySchedule{}, err
		}
		var err error
		stored, err = s.loadLessons(ctx, telegramID, date)
		if err != nil {
			return DaySchedule{}, err
		}
		today, _ := minskClock(now)
		if date < today && len(stored) > 0 && allConducted(stored) {
			return DaySchedule{Date: date, Found: true, Lessons: stored}, nil
		}
	}

	remote, err := Fetch(ctx, date)
	if err != nil {
		if len(stored) > 0 {
			return DaySchedule{Date: date, Found: true, Lessons: stored}, nil
		}
		return DaySchedule{}, err
	}
	if !remote.Found {
		if len(stored) > 0 {
			return DaySchedule{Date: date, Found: true, Lessons: stored}, nil
		}
		remote.Lessons = []Lesson{}
		return remote, nil
	}

	if !hasUser || s.DB == nil {
		for i := range remote.Lessons {
			remote.Lessons[i].Conducted = lessonEnded(date, remote.Lessons[i].Period, now)
		}
		return remote, nil
	}

	merged, err := s.mergeLessons(ctx, telegramID, username, date, remote.Lessons, now)
	if err != nil {
		return DaySchedule{}, err
	}
	return DaySchedule{Date: date, Found: true, Lessons: merged}, nil
}

func (s *Service) ensureUser(ctx context.Context, telegramID int64, username string) (string, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO users (telegram_id, username)
		VALUES ($1, NULLIF($2, ''))
		ON CONFLICT (telegram_id) DO UPDATE
		SET username = COALESCE(NULLIF($2, ''), users.username)
		RETURNING id::text
	`, telegramID, username).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("ensure user: %w", err)
	}
	return id, nil
}

func (s *Service) loadLessons(ctx context.Context, telegramID int64, date string) ([]Lesson, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT period, group_name, subject, room, conducted
		FROM schedule_lessons
		WHERE user_id = (SELECT id FROM users WHERE telegram_id = $1)
		  AND date = $2
		ORDER BY period
	`, telegramID, date)
	if err != nil {
		return nil, fmt.Errorf("load lessons: %w", err)
	}
	defer rows.Close()

	lessons := []Lesson{}
	for rows.Next() {
		var lesson Lesson
		if err := rows.Scan(&lesson.Period, &lesson.Group, &lesson.Subject, &lesson.Room, &lesson.Conducted); err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	return lessons, rows.Err()
}

func (s *Service) mergeLessons(ctx context.Context, telegramID int64, username, date string, incoming []Lesson, now time.Time) ([]Lesson, error) {
	userID, err := s.ensureUser(ctx, telegramID, username)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	periods := make([]int, 0, len(incoming))
	for _, lesson := range incoming {
		conducted := lessonEnded(date, lesson.Period, now)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schedule_lessons (user_id, date, period, group_name, subject, room, conducted, conducted_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, CASE WHEN $7 THEN NOW() ELSE NULL END)
			ON CONFLICT (user_id, date, period) DO UPDATE SET
				group_name = EXCLUDED.group_name,
				subject = EXCLUDED.subject,
				room = EXCLUDED.room,
				conducted = CASE
					WHEN schedule_lessons.manual THEN schedule_lessons.conducted
					ELSE (schedule_lessons.conducted OR EXCLUDED.conducted)
				END,
				conducted_at = CASE
					WHEN schedule_lessons.manual THEN schedule_lessons.conducted_at
					WHEN schedule_lessons.conducted THEN schedule_lessons.conducted_at
					WHEN EXCLUDED.conducted THEN NOW()
					ELSE NULL
				END,
				updated_at = NOW()
		`, userID, date, lesson.Period, lesson.Group, lesson.Subject, lesson.Room, conducted); err != nil {
			return nil, fmt.Errorf("save lesson: %w", err)
		}
		if err := ensurePlan(ctx, tx, userID, lesson.Group, lesson.Subject); err != nil {
			return nil, err
		}
		periods = append(periods, lesson.Period)
	}

	if len(periods) == 0 {
		periods = []int{-1}
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM schedule_lessons
		WHERE user_id = $1 AND date = $2 AND conducted = FALSE AND manual = FALSE
		  AND NOT (period = ANY($3))
	`, userID, date, pq.Array(periods)); err != nil {
		return nil, fmt.Errorf("prune lessons: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.loadLessons(ctx, telegramID, date)
}

func (s *Service) sweep(ctx context.Context, telegramID int64, username string) error {
	userID, err := s.ensureUser(ctx, telegramID, username)
	if err != nil {
		return err
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT date::text, period
		FROM schedule_lessons
		WHERE user_id = $1 AND conducted = FALSE AND manual = FALSE
	`, userID)
	if err != nil {
		return fmt.Errorf("sweep lessons: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	type dueLesson struct {
		date   string
		period int
	}
	var due []dueLesson
	for rows.Next() {
		var item dueLesson
		if err := rows.Scan(&item.date, &item.period); err != nil {
			return err
		}
		if lessonEnded(item.date, item.period, now) {
			due = append(due, item)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range due {
		if _, err := s.DB.ExecContext(ctx, `
			UPDATE schedule_lessons
			SET conducted = TRUE, conducted_at = NOW(), updated_at = NOW()
			WHERE user_id = $1 AND date = $2 AND period = $3 AND conducted = FALSE AND manual = FALSE
		`, userID, item.date, item.period); err != nil {
			return fmt.Errorf("mark conducted: %w", err)
		}
	}
	return nil
}

func (s *Service) listPlans(ctx context.Context, telegramID int64, username string) ([]Plan, error) {
	userID, err := s.ensureUser(ctx, telegramID, username)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT p.group_name, p.subject, p.planned_hours,
		       COUNT(l.id) FILTER (WHERE l.conducted)
		FROM subject_plans p
		LEFT JOIN schedule_lessons l
		  ON l.user_id = p.user_id
		 AND l.group_name = p.group_name
		 AND l.subject = p.subject
		WHERE p.user_id = $1
		GROUP BY p.id
		ORDER BY p.group_name, p.subject
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	plans := []Plan{}
	for rows.Next() {
		var plan Plan
		if err := rows.Scan(&plan.Group, &plan.Subject, &plan.PlannedHours, &plan.Conducted); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (s *Service) setPlan(ctx context.Context, telegramID int64, username string, plan Plan) error {
	userID, err := s.ensureUser(ctx, telegramID, username)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO subject_plans (user_id, group_name, subject, planned_hours)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, group_name, subject) DO UPDATE
		SET planned_hours = EXCLUDED.planned_hours, updated_at = NOW()
	`, userID, plan.Group, plan.Subject, plan.PlannedHours)
	if err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	return nil
}

func (s *Service) markLesson(ctx context.Context, telegramID int64, username string, body markBody) error {
	userID, err := s.ensureUser(ctx, telegramID, username)
	if err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO schedule_lessons (user_id, date, period, group_name, subject, room, conducted, manual, conducted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE,
		        CASE WHEN $7 THEN NOW() ELSE NULL END)
		ON CONFLICT (user_id, date, period) DO UPDATE SET
			group_name = EXCLUDED.group_name,
			subject = EXCLUDED.subject,
			room = EXCLUDED.room,
			conducted = EXCLUDED.conducted,
			manual = TRUE,
			conducted_at = CASE
				WHEN EXCLUDED.conducted THEN COALESCE(schedule_lessons.conducted_at, NOW())
				ELSE NULL
			END,
			updated_at = NOW()
	`, userID, body.Date, body.Period, body.Group, body.Subject, body.Room, body.Conducted); err != nil {
		return fmt.Errorf("mark lesson: %w", err)
	}
	if body.Group != "" && body.Subject != "" {
		if err := ensurePlan(ctx, s.DB, userID, body.Group, body.Subject); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) importLocal(ctx context.Context, telegramID int64, username string, body importBody) error {
	userID, err := s.ensureUser(ctx, telegramID, username)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, lesson := range body.Lessons {
		if _, err := time.Parse("2006-01-02", lesson.Date); err != nil || lesson.Period < 1 {
			continue
		}
		conducted := lesson.Conducted || lessonEnded(lesson.Date, lesson.Period, now)
		// Старые отметки из localStorage ставились руками, поэтому фиксируем их
		// как ручные, чтобы авто-расчёт по звонкам их не перекрыл.
		manual := lesson.Conducted
		if _, err := s.DB.ExecContext(ctx, `
			INSERT INTO schedule_lessons (user_id, date, period, group_name, subject, room, conducted, manual, conducted_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CASE WHEN $7 THEN NOW() ELSE NULL END)
			ON CONFLICT (user_id, date, period) DO UPDATE SET
				group_name = EXCLUDED.group_name,
				subject = EXCLUDED.subject,
				room = EXCLUDED.room,
				conducted = schedule_lessons.conducted OR EXCLUDED.conducted,
				manual = schedule_lessons.manual OR EXCLUDED.manual,
				conducted_at = CASE
					WHEN schedule_lessons.conducted THEN schedule_lessons.conducted_at
					WHEN EXCLUDED.conducted THEN NOW()
					ELSE schedule_lessons.conducted_at
				END,
				updated_at = NOW()
		`, userID, lesson.Date, lesson.Period, lesson.Group, lesson.Subject, lesson.Room, conducted, manual); err != nil {
			return fmt.Errorf("import lesson: %w", err)
		}
		if err := ensurePlan(ctx, s.DB, userID, lesson.Group, lesson.Subject); err != nil {
			return err
		}
	}
	for _, plan := range body.Plans {
		if err := s.setPlan(ctx, telegramID, username, plan); err != nil {
			return err
		}
	}
	return s.sweep(ctx, telegramID, username)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func ensurePlan(ctx context.Context, exec execer, userID, group, subject string) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO subject_plans (user_id, group_name, subject, planned_hours)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (user_id, group_name, subject) DO NOTHING
	`, userID, group, subject)
	if err != nil {
		return fmt.Errorf("ensure plan: %w", err)
	}
	return nil
}

func allConducted(lessons []Lesson) bool {
	for _, lesson := range lessons {
		if !lesson.Conducted {
			return false
		}
	}
	return true
}

func telegramUser(r *http.Request) (int64, string, bool) {
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
