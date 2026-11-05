package schedule

import "testing"
import "time"

func TestLessonEndedUsesMinskBells(t *testing.T) {
	// 5 ноября 2026, четверг, 16:40 по Минску.
	now := time.Date(2026, 11, 5, 16, 40, 0, 0, minsk)
	if !lessonEnded("2026-11-05", 8, now) {
		t.Fatal("lesson 8 should have ended at 15:20")
	}
	if lessonEnded("2026-11-05", 10, now) {
		t.Fatal("lesson 10 ends at 17:20")
	}
	if !lessonEnded("2026-11-04", 1, now) {
		t.Fatal("past day is conducted")
	}
	if lessonEnded("2026-11-06", 1, now) {
		t.Fatal("future lesson is not conducted")
	}

	saturday := time.Date(2026, 11, 7, 13, 25, 0, 0, minsk)
	if !lessonEnded("2026-11-07", 6, saturday) {
		t.Fatal("saturday lesson 6 ends at 13:20")
	}
	if lessonEnded("2026-11-07", 7, saturday) {
		t.Fatal("saturday lesson 7 ends at 14:15")
	}
}
