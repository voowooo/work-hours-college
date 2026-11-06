package notify

import (
	"fmt"
	"strings"
	"time"

	"github.com/workhourscounter/backend/internal/schedule"
)

// leadMinutes — за сколько минут до конца урока присылать сообщение.
const leadMinutes = 15

// lessonInfo — пара, о которой рассказываем в сообщении.
type lessonInfo struct {
	Period    int
	Group     string
	Subject   string
	Room      string
	Manual    bool
	Conducted bool
}

// dueRemaining возвращает минут до конца урока, если уведомление пора слать.
// Раньше чем за leadMinutes и позже конца урока — не шлём.
func dueRemaining(end, now int) (int, bool) {
	remaining := end - now
	if remaining <= 0 || remaining > leadMinutes {
		return 0, false
	}
	return remaining, true
}

// nextLesson — ближайшая пара после текущей (по номеру пары).
func nextLesson(lessons []lessonInfo, period int) *lessonInfo {
	for i := range lessons {
		if lessons[i].Period > period {
			return &lessons[i]
		}
	}
	return nil
}

func minutesWord(n int) string {
	mod10, mod100 := n%10, n%100
	if mod10 == 1 && mod100 != 11 {
		return "минуту"
	}
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
		return "минуты"
	}
	return "минут"
}

func inMinutesPhrase(n int) string {
	if n == 1 {
		return "через минуту"
	}
	return fmt.Sprintf("через %d %s", n, minutesWord(n))
}

func lessonLine(lesson lessonInfo) string {
	parts := make([]string, 0, 2)
	if lesson.Subject != "" {
		parts = append(parts, lesson.Subject)
	}
	if lesson.Group != "" {
		parts = append(parts, lesson.Group)
	}
	line := strings.Join(parts, " · ")
	if line == "" {
		line = "Урок"
	}
	if lesson.Room != "" {
		line += fmt.Sprintf(", ауд. %s", lesson.Room)
	}
	return line
}

// buildMessage собирает текст: конец текущего урока + следующий урок.
func buildMessage(date string, current lessonInfo, next *lessonInfo, remaining int) string {
	var b strings.Builder
	if _, end, ok := schedule.BellTimes(date, current.Period); ok {
		fmt.Fprintf(&b, "⏰ Урок закончится %s — в %s\n", inMinutesPhrase(remaining), schedule.FormatMinutes(end))
	} else {
		fmt.Fprintf(&b, "⏰ Урок закончится %s\n", inMinutesPhrase(remaining))
	}
	fmt.Fprintf(&b, "Пара %d · %s\n", current.Period, lessonLine(current))

	switch {
	case next != nil:
		if start, _, ok := schedule.BellTimes(date, next.Period); ok {
			fmt.Fprintf(&b, "\nСледующий урок в %s (пара %d)\n%s\n",
				schedule.FormatMinutes(start), next.Period, lessonLine(*next))
		} else {
			fmt.Fprintf(&b, "\nСледующий урок — пара %d\n%s\n", next.Period, lessonLine(*next))
		}
	default:
		b.WriteString("\nСледующего урока сегодня нет.")
	}
	return strings.TrimRight(b.String(), "\n")
}

// buildTestMessage — короткое сообщение для проверки из настроек.
func buildTestMessage(date string, next *lessonInfo) string {
	const header = "🔔 Тестовое уведомление\n\nБот пишет за 15 минут до конца урока и показывает следующую пару."
	if next == nil {
		return header
	}
	if start, _, ok := schedule.BellTimes(date, next.Period); ok {
		return fmt.Sprintf("%s\n\nБлижайший урок в %s (пара %d)\n%s",
			header, schedule.FormatMinutes(start), next.Period, lessonLine(*next))
	}
	return fmt.Sprintf("%s\n\nСледующий урок — пара %d\n%s", header, next.Period, lessonLine(*next))
}

// muteCallback — данные кнопки «отключить до завтра»: дата, до которой глушим.
func muteCallback(date string) string {
	return "mute:" + tomorrow(date)
}

// tomorrow возвращает следующий день после date в формате YYYY-MM-DD.
func tomorrow(date string) string {
	parsed, err := time.ParseInLocation("2006-01-02", date, time.UTC)
	if err != nil {
		return date
	}
	return parsed.AddDate(0, 0, 1).Format("2006-01-02")
}

// parseMuteCallback превращает «mute:2026-11-07» в момент наступления этого дня
// по Минску (00:00). Просроченная дата (старый повторный запрос) возвращает
// время в прошлом — тогда глушить нечего.
func parseMuteCallback(data string) (time.Time, error) {
	const prefix = "mute:"
	if !strings.HasPrefix(data, prefix) {
		return time.Time{}, fmt.Errorf("unknown callback %q", data)
	}
	day, err := time.ParseInLocation("2006-01-02", strings.TrimPrefix(data, prefix), minskZone())
	if err != nil {
		return time.Time{}, err
	}
	return day, nil
}
