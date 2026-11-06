package notify

import (
	"strings"
	"testing"
	"time"

	"github.com/workhourscounter/backend/internal/schedule"
)

func TestDueRemaining(t *testing.T) {
	const (
		end = 8*60 + 45 // 08:45
	)
	cases := []struct {
		name   string
		now    int
		wantOK bool
		remain int
	}{
		{"за 15 минут", 8*60 + 30, true, 15},
		{"за минуту", 8*60 + 44, true, 1},
		{"ровно в конце", 8*60 + 45, false, 0},
		{"после конца", 9 * 60, false, 0},
		{"слишком рано", 8*60 + 14, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			remain, ok := dueRemaining(end, tc.now)
			if ok != tc.wantOK || remain != tc.remain {
				t.Fatalf("dueRemaining(%d, %d) = %d, %v; want %d, %v", end, tc.now, remain, ok, tc.remain, tc.wantOK)
			}
		})
	}
}

func TestNextLesson(t *testing.T) {
	lessons := []lessonInfo{
		{Period: 2, Group: "148ш", Subject: "Астр"},
		{Period: 3, Group: "60ф", Subject: "Астр"},
		{Period: 5, Group: "13п", Subject: "Стан"},
	}
	next := nextLesson(lessons, 2)
	if next == nil || next.Period != 3 {
		t.Fatalf("got %#v", next)
	}
	next = nextLesson(lessons, 3)
	if next == nil || next.Period != 5 {
		t.Fatalf("got %#v", next)
	}
	if nextLesson(lessons, 5) != nil {
		t.Fatal("после последней пары следующего урока нет")
	}
}

func TestBuildMessageWithNextLesson(t *testing.T) {
	// 5 ноября 2026 — четверг, будние звонки.
	current := lessonInfo{Period: 6, Group: "13п", Subject: "Стан", Room: "104"}
	next := lessonInfo{Period: 7, Group: "100б", Subject: "Физи", Room: "402"}

	text := buildMessage("2026-11-05", current, &next, 15)

	wantParts := []string{
		"через 15 минут",
		"в 13:30", // конец пары 6
		"Пара 6 · Стан · 13п, ауд. 104",
		"Следующий урок в 13:40 (пара 7)", // начало пары 7
		"Физи · 100б, ауд. 402",
	}
	for _, want := range wantParts {
		if !strings.Contains(text, want) {
			t.Errorf("в сообщении нет %q:\n%s", want, text)
		}
	}
}

func TestBuildMessageWithoutNextLesson(t *testing.T) {
	current := lessonInfo{Period: 8, Group: "188к", Subject: "Астр", Room: "402"}
	text := buildMessage("2026-11-05", current, nil, 3)
	if !strings.Contains(text, "через 3 минуты") {
		t.Errorf("нужно склонение:\n%s", text)
	}
	if !strings.Contains(text, "Следующего урока сегодня нет.") {
		t.Errorf("нет пометки о конце дня:\n%s", text)
	}
	if strings.Contains(text, "Следующий урок") {
		t.Errorf("лишняя строка о следующем уроке:\n%s", text)
	}
}

func TestBuildMessageSingleMinute(t *testing.T) {
	text := buildMessage("2026-11-05", lessonInfo{Period: 1, Group: "100б"}, nil, 1)
	if !strings.Contains(text, "через минуту") {
		t.Errorf("ожидалось «через минуту»:\n%s", text)
	}
}

func TestMuteCallbackRoundTrip(t *testing.T) {
	callback := muteCallback("2026-11-05")
	if callback != "mute:2026-11-06" {
		t.Fatalf("got %q", callback)
	}

	until, err := parseMuteCallback(callback)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 6, 0, 0, 0, 0, minskZone())
	if !until.Equal(want) {
		t.Fatalf("got %s, want %s", until, want)
	}

	// Повторная доставка старой кнопки не должна глушить уведомления.
	past, err := parseMuteCallback("mute:2020-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if past.After(time.Now()) {
		t.Fatal("старая кнопка вернула время в будущем")
	}

	if _, err := parseMuteCallback("something-else"); err == nil {
		t.Fatal("ожидалась ошибка для неизвестной кнопки")
	}
}

func TestBuildTestMessage(t *testing.T) {
	next := lessonInfo{Period: 1, Group: "100б", Subject: "Физи", Room: "402"}
	text := buildTestMessage("2026-11-05", &next)
	if !strings.Contains(text, "Ближайший урок в 08:00 (пара 1)") {
		t.Errorf("got:\n%s", text)
	}
	empty := buildTestMessage("2026-11-05", nil)
	if !strings.Contains(empty, "Тестовое уведомление") {
		t.Errorf("got:\n%s", empty)
	}
}

func TestFormatMinutes(t *testing.T) {
	if got := schedule.FormatMinutes(8*60 + 45); got != "08:45" {
		t.Fatalf("got %q", got)
	}
	if got := schedule.FormatMinutes(21 * 60); got != "21:00" {
		t.Fatalf("got %q", got)
	}
}
