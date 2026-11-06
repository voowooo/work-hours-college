package notify

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/workhourscounter/backend/internal/db"
	"github.com/workhourscounter/backend/internal/schedule"
)

// maxAttempts — сколько раз повторить отправку в пределах 15-минутного окна.
const maxAttempts = 5

// Service рассылает уведомления от бота: за 15 минут до конца урока и по
// кнопке «отключить до завтра».
type Service struct {
	DB       *db.DB
	Token    string
	Schedule *schedule.Service

	client     *http.Client
	pollClient *http.Client

	mu       sync.Mutex
	offset   int64
	checked  map[string]bool
	attempts map[string]int
	conflict bool
}

// dueRow — урок пользователя на сегодня.
type dueRow struct {
	id         string
	userID     string
	chatID     int64
	period     int
	group      string
	subject    string
	room       string
	manual     bool
	conducted  bool
	notifiedAt bool
}

func New(database *db.DB, token string, svc *schedule.Service) *Service {
	return &Service{
		DB:         database,
		Token:      token,
		Schedule:   svc,
		client:     &http.Client{Timeout: 15 * time.Second},
		pollClient: &http.Client{Timeout: 45 * time.Second},
		checked:    map[string]bool{},
		attempts:   map[string]int{},
	}
}

func minskZone() *time.Location {
	return time.FixedZone("Europe/Minsk", 3*3600)
}

// Run блокируется, пока не отменят ctx: опрашивает Telegram и шлёт уведомления.
func (s *Service) Run(ctx context.Context) {
	if s.DB == nil {
		log.Println("notify: база недоступна, уведомления отключены")
		return
	}
	if s.Token == "" {
		log.Println("notify: TELEGRAM_BOT_TOKEN не задан, уведомления отключены")
		return
	}
	bot, err := s.getMe(ctx)
	if err != nil {
		log.Printf("notify: бот недоступен, уведомления отключены: %v", err)
		return
	}
	log.Printf("notify: бот @%s запущен", bot)

	go s.pollLoop(ctx)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.tick(ctx); err != nil {
				log.Printf("notify: %v", err)
			}
		}
	}
}

func (s *Service) tick(ctx context.Context) error {
	date, minutes := schedule.MinskClock(time.Now())
	s.ensureToday(ctx, date)

	rows, err := s.todayLessons(ctx, date)
	if err != nil {
		return err
	}
	for chatID, list := range groupByChat(rows) {
		s.notifyUser(ctx, date, minutes, chatID, list)
	}
	return nil
}

// ensureToday сохраняет в базу расписание дня для тех, у кого его ещё нет:
// бот должен уведомлять, даже если приложение не открывали.
func (s *Service) ensureToday(ctx context.Context, date string) {
	// Ночью сайт не трогаем — на уроки всё равно рано.
	_, minutes := schedule.MinskClock(time.Now())
	if minutes < 7*60 || minutes > 23*60 {
		return
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT u.id::text, u.telegram_id
		FROM users u
		JOIN notification_settings ns ON ns.user_id = u.id
		WHERE ns.enabled
		  AND (ns.muted_until IS NULL OR ns.muted_until <= NOW())
		  AND NOT EXISTS (
			SELECT 1 FROM schedule_lessons sl WHERE sl.user_id = u.id AND sl.date = $1
		  )
	`, date)
	if err != nil {
		log.Printf("notify: не удалось получить список пользователей: %v", err)
		return
	}
	type pending struct {
		id         string
		telegramID int64
	}
	var users []pending
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.id, &item.telegramID); err != nil {
			rows.Close()
			log.Printf("notify: %v", err)
			return
		}
		users = append(users, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		log.Printf("notify: %v", err)
		return
	}

	need := make([]pending, 0, len(users))
	s.mu.Lock()
	for _, user := range users {
		if !s.checked[checkKey(user.id, date)] {
			need = append(need, user)
		}
	}
	s.mu.Unlock()
	if len(need) == 0 {
		return
	}

	// PDF один на весь день — schedule.Fetch сам кэширует ответ.
	day, err := schedule.Fetch(ctx, date)
	if err != nil {
		log.Printf("notify: не удалось получить расписание на %s: %v", date, err)
		return
	}

	for _, user := range need {
		s.mu.Lock()
		s.checked[checkKey(user.id, date)] = true
		s.mu.Unlock()
		if !day.Found {
			continue
		}
		if err := s.Schedule.StoreLessons(ctx, user.telegramID, date, day.Lessons); err != nil {
			log.Printf("notify: не удалось сохранить расписание: %v", err)
		}
	}
}

func checkKey(userID, date string) string {
	return userID + "|" + date
}

func (s *Service) todayLessons(ctx context.Context, date string) ([]dueRow, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT sl.id::text, sl.user_id::text, u.telegram_id, sl.period,
		       sl.group_name, sl.subject, sl.room, sl.manual, sl.conducted,
		       sl.notify_sent_at IS NOT NULL
		FROM schedule_lessons sl
		JOIN users u ON u.id = sl.user_id
		JOIN notification_settings ns ON ns.user_id = sl.user_id
		WHERE sl.date = $1
		  AND ns.enabled
		  AND (ns.muted_until IS NULL OR ns.muted_until <= NOW())
		ORDER BY u.telegram_id, sl.period
	`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []dueRow
	for rows.Next() {
		var row dueRow
		if err := rows.Scan(&row.id, &row.userID, &row.chatID, &row.period,
			&row.group, &row.subject, &row.room, &row.manual, &row.conducted, &row.notifiedAt); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func groupByChat(rows []dueRow) map[int64][]dueRow {
	grouped := make(map[int64][]dueRow, 1)
	for _, row := range rows {
		grouped[row.chatID] = append(grouped[row.chatID], row)
	}
	return grouped
}

func toLessonInfo(rows []dueRow) []lessonInfo {
	lessons := make([]lessonInfo, 0, len(rows))
	for _, row := range rows {
		lessons = append(lessons, lessonInfo{
			Period:    row.period,
			Group:     row.group,
			Subject:   row.subject,
			Room:      row.room,
			Manual:    row.manual,
			Conducted: row.conducted,
		})
	}
	return lessons
}

func (s *Service) notifyUser(ctx context.Context, date string, minutes int, chatID int64, rows []dueRow) {
	lessons := toLessonInfo(rows)
	for _, row := range rows {
		if row.notifiedAt {
			continue
		}
		// Урок отмечен вручную как непроведенный — писать о нём нечего.
		if row.manual && !row.conducted {
			s.markSent(ctx, row)
			continue
		}
		_, end, ok := schedule.BellTimes(date, row.period)
		if !ok {
			s.markSent(ctx, row)
			continue
		}
		remaining, due := dueRemaining(end, minutes)
		if !due {
			continue
		}

		text := buildMessage(date, lessonInfo{
			Period:  row.period,
			Group:   row.group,
			Subject: row.subject,
			Room:    row.room,
		}, nextLesson(lessons, row.period), remaining)

		if err := s.sendMessage(ctx, chatID, text, muteCallback(date)); err != nil {
			log.Printf("notify: не удалось отправить пользователю %d (пара %d): %v", chatID, row.period, err)
			if isBlocked(err) {
				// Пользователь не запускал бота или заблокировал его.
				s.disableNotifications(ctx, row.userID)
				return
			}
			if s.shouldRetry(row, err) {
				continue
			}
		}
		s.clearAttempts(row)
		s.markSent(ctx, row)
	}
}

func (s *Service) markSent(ctx context.Context, row dueRow) {
	if _, err := s.DB.ExecContext(ctx, `
		UPDATE schedule_lessons SET notify_sent_at = NOW()
		WHERE id = $1 AND notify_sent_at IS NULL
	`, row.id); err != nil {
		log.Printf("notify: не удалось отметить отправку: %v", err)
	}
}

func (s *Service) disableNotifications(ctx context.Context, userID string) {
	if _, err := s.DB.ExecContext(ctx, `
		UPDATE notification_settings SET enabled = FALSE, updated_at = NOW()
		WHERE user_id = $1
	`, userID); err != nil {
		log.Printf("notify: не удалось отключить уведомления: %v", err)
	}
}

func (s *Service) shouldRetry(row dueRow, err error) bool {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests {
		return true
	}
	key := attemptKey(row)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[key]++
	return s.attempts[key] < maxAttempts
}

func (s *Service) clearAttempts(row dueRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.attempts, attemptKey(row))
}

func attemptKey(row dueRow) string {
	return row.id
}

// --- Обработка входящих сообщений и нажатий на кнопки ---------------------

func (s *Service) pollLoop(ctx context.Context) {
	for ctx.Err() == nil {
		updates, err := s.getUpdates(ctx, s.getOffset())
		if err != nil {
			log.Printf("notify: getUpdates: %v", err)
			delay := 30 * time.Second
			if isConflict(err) {
				s.mu.Lock()
				if !s.conflict {
					s.conflict = true
					log.Println("notify: Telegram сообщает о втором опроснике/вебхуке — переподключаемся реже")
				}
				s.mu.Unlock()
				delay = 5 * time.Minute
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			continue
		}
		for _, update := range updates {
			s.setOffset(update.UpdateID + 1)
			s.handleUpdate(ctx, update)
		}
	}
}

func (s *Service) getOffset() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offset
}

func (s *Service) setOffset(value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value > s.offset {
		s.offset = value
	}
}

func (s *Service) handleUpdate(ctx context.Context, update update) {
	switch {
	case update.CallbackQuery != nil:
		s.handleCallback(ctx, *update.CallbackQuery)
	case update.Message != nil:
		s.handleMessage(ctx, *update.Message)
	}
}

func (s *Service) handleCallback(ctx context.Context, query tgCallbackQuery) {
	until, err := parseMuteCallback(query.Data)
	if err != nil {
		_ = s.answerCallback(ctx, query.ID, "Неизвестная команда")
		return
	}
	if !until.After(time.Now()) {
		// Повторная доставка старой кнопки — глушить уже нечего.
		_ = s.answerCallback(ctx, query.ID, "Уведомления уже включены")
		return
	}

	userID, err := s.Schedule.EnsureUser(ctx, query.From.ID, query.From.Username)
	if err != nil {
		log.Printf("notify: %v", err)
		return
	}
	if err := s.setMutedUntil(ctx, userID, until); err != nil {
		log.Printf("notify: %v", err)
		return
	}
	label := until.Format("02.01, 15:04")
	_ = s.answerCallback(ctx, query.ID, "Уведомления отключены до "+label)
}

func (s *Service) setMutedUntil(ctx context.Context, userID string, until time.Time) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO notification_settings (user_id, enabled, muted_until)
		VALUES ($1, TRUE, $2)
		ON CONFLICT (user_id) DO UPDATE SET
			enabled = TRUE,
			muted_until = EXCLUDED.muted_until,
			updated_at = NOW()
	`, userID, until)
	return err
}

func (s *Service) handleMessage(ctx context.Context, message tgMessage) {
	text := strings.TrimSpace(message.Text)
	if text == "" || !strings.HasPrefix(text, "/") {
		return
	}
	if message.Chat.Type != "" && message.Chat.Type != "private" {
		return
	}
	// Не отвечаем на старые сообщения после перезапуска.
	if message.Date > 0 && time.Since(time.Unix(message.Date, 0)) > 2*time.Minute {
		return
	}
	if message.Chat.ID == 0 {
		return
	}

	switch {
	case strings.HasPrefix(text, "/start"):
		userID, err := s.Schedule.EnsureUser(ctx, message.Chat.ID, usernameOf(message.From))
		if err != nil {
			log.Printf("notify: %v", err)
			return
		}
		if _, err := s.DB.ExecContext(ctx, `
			INSERT INTO notification_settings (user_id, enabled, muted_until)
			VALUES ($1, TRUE, NULL)
			ON CONFLICT (user_id) DO UPDATE SET
				enabled = TRUE, muted_until = NULL, updated_at = NOW()
		`, userID); err != nil {
			log.Printf("notify: %v", err)
			return
		}
		date, _ := schedule.MinskClock(time.Now())
		reply := "🔔 Уведомления включены\n\n" +
			"Бот напишет за 15 минут до конца урока: покажет время конца и следующую пару. " +
			"Кнопка под сообщением отключает уведомления до завтра."
		if err := s.sendMessage(ctx, message.Chat.ID, reply, muteCallback(date)); err != nil {
			log.Printf("notify: %v", err)
		}
	case strings.HasPrefix(text, "/stop"):
		userID, err := s.Schedule.EnsureUser(ctx, message.Chat.ID, usernameOf(message.From))
		if err != nil {
			log.Printf("notify: %v", err)
			return
		}
		s.disableNotifications(ctx, userID)
		if err := s.sendMessage(ctx, message.Chat.ID, "Уведомления выключены. Вернуть их можно командой /start.", ""); err != nil {
			log.Printf("notify: %v", err)
		}
	}
}

func usernameOf(user *tgUser) string {
	if user == nil {
		return ""
	}
	return user.Username
}
