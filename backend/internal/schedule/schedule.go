package schedule

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ledongthuc/pdf"
)

const TeacherName = "ДурневДН"

const collegeHost = "https://seimtc.by"

type Lesson struct {
	Period  int    `json:"period"`
	Group   string `json:"group"`
	Subject string `json:"subject"`
	Room    string `json:"room"`
}

type DaySchedule struct {
	Date    string   `json:"date"`
	Found   bool     `json:"found"`
	Lessons []Lesson `json:"lessons"`
}

type cacheEntry struct {
	at   time.Time
	body DaySchedule
}

var (
	cacheMu  sync.Mutex
	cache    = map[string]cacheEntry{}
	cacheTTL = 30 * time.Minute
)

func Handler(w http.ResponseWriter, r *http.Request) {
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

	day, err := Fetch(ctx, date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(day)
}

func Fetch(ctx context.Context, date string) (DaySchedule, error) {
	if day, ok := cached(date); ok {
		return day, nil
	}
	day, err := fetchFrom(ctx, collegeHost, date)
	if err != nil {
		return DaySchedule{}, err
	}
	storeCache(date, day)
	return day, nil
}

func fetchFrom(ctx context.Context, host, date string) (DaySchedule, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return DaySchedule{}, fmt.Errorf("invalid date")
	}

	url := fmt.Sprintf("%s/images/raspisanie/%d/%02d_%02dp.pdf", host, t.Year(), t.Day(), int(t.Month()))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return DaySchedule{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; work-hours-college)")
	req.Header.Set("Accept", "application/pdf,*/*")

	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return DaySchedule{}, fmt.Errorf("schedule request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return DaySchedule{}, fmt.Errorf("read schedule: %w", err)
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return DaySchedule{Date: date, Found: false, Lessons: []Lesson{}}, nil
	}
	if resp.StatusCode != http.StatusOK || !bytes.HasPrefix(body, []byte("%PDF")) {
		return DaySchedule{Date: date, Found: false, Lessons: []Lesson{}}, nil
	}

	lessons, err := LessonsFromPDF(body, TeacherName)
	if err != nil {
		return DaySchedule{}, fmt.Errorf("parse schedule: %w", err)
	}
	if lessons == nil {
		lessons = []Lesson{}
	}
	return DaySchedule{Date: date, Found: true, Lessons: lessons}, nil
}

func LessonsFromPDF(data []byte, teacher string) ([]Lesson, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var lines []string
	for i := 1; i <= reader.NumPage(); i++ {
		rows, err := reader.Page(i).GetTextByRow()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			var b strings.Builder
			for _, text := range row.Content {
				b.WriteString(text.S)
			}
			lines = append(lines, b.String())
		}
	}
	return lessonsFromLines(lines, teacher), nil
}

func lessonsFromLines(lines []string, teacher string) []Lesson {
	target := compact(teacher)
	for i, line := range lines {
		if isSeparator(line) {
			continue
		}
		cells := rowCells(line)
		if len(cells) == 0 || compact(cells[0]) != target {
			continue
		}

		var detail [][]string
		for j := i + 1; j < len(lines) && len(detail) < 2; j++ {
			if isSeparator(lines[j]) {
				continue
			}
			next := rowCells(lines[j])
			if len(next) == 0 {
				continue
			}
			if next[0] != "" && compact(next[0]) != target {
				break
			}
			detail = append(detail, pad(next, 14))
		}

		groups := pad(cells, 14)
		var subjects, rooms []string
		if len(detail) > 0 {
			subjects = detail[0]
		}
		if len(detail) > 1 {
			rooms = detail[1]
		}

		lessons := make([]Lesson, 0, 8)
		for period := 1; period <= 13; period++ {
			group := groups[period]
			subject := cellAt(subjects, period)
			room := cellAt(rooms, period)
			if group == "" && subject == "" {
				continue
			}
			lessons = append(lessons, Lesson{
				Period:  period,
				Group:   group,
				Subject: subject,
				Room:    room,
			})
		}
		return lessons
	}
	return []Lesson{}
}

func rowCells(line string) []string {
	parts := strings.FieldsFunc(line, func(r rune) bool {
		return r == '│' || r == '║'
	})
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func pad(cells []string, n int) []string {
	for len(cells) < n {
		cells = append(cells, "")
	}
	return cells
}

func cellAt(cells []string, i int) string {
	if i < 0 || i >= len(cells) {
		return ""
	}
	return cells[i]
}

func isSeparator(line string) bool {
	return strings.ContainsRune(line, '├') || strings.ContainsRune(line, '└') || strings.ContainsRune(line, '┌')
}

func compact(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func cached(date string) (DaySchedule, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	entry, ok := cache[date]
	if !ok || time.Since(entry.at) > cacheTTL {
		return DaySchedule{}, false
	}
	return entry.body, true
}

func storeCache(date string, day DaySchedule) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache[date] = cacheEntry{at: time.Now(), body: day}
}
