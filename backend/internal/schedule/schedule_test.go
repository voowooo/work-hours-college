package schedule

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLessonsFromSamplePDF(t *testing.T) {
	data, err := os.ReadFile("testdata/05_11p.pdf")
	if err != nil {
		t.Fatal(err)
	}
	lessons, err := LessonsFromPDF(data, TeacherName)
	if err != nil {
		t.Fatal(err)
	}

	want := []Lesson{
		{Period: 2, Group: "148ш", Subject: "Астр", Room: "402"},
		{Period: 3, Group: "60ф", Subject: "Астр", Room: "402"},
		{Period: 4, Group: "148ш", Subject: "Астр", Room: "402"},
		{Period: 5, Group: "13п", Subject: "Стан", Room: "104"},
		{Period: 7, Group: "100б", Subject: "Физи", Room: "402"},
		{Period: 8, Group: "188к", Subject: "Астр", Room: "402"},
	}
	if len(lessons) != len(want) {
		t.Fatalf("got %d lessons %#v", len(lessons), lessons)
	}
	for i := range want {
		if lessons[i] != want[i] {
			t.Fatalf("lesson %d: got %#v want %#v", i, lessons[i], want[i])
		}
	}
}

func TestFetchMissingSchedule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ru/images/raspisanie/2026/01_01p.pdf" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Error"))
			return
		}
		http.Redirect(w, r, "/ru"+r.URL.Path, http.StatusMovedPermanently)
	}))
	defer srv.Close()

	day, err := fetchFrom(context.Background(), srv.URL, "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if day.Found {
		t.Fatal("expected missing schedule")
	}
	if len(day.Lessons) != 0 {
		t.Fatalf("expected no lessons, got %#v", day.Lessons)
	}
}

func TestFetchPublishedSchedule(t *testing.T) {
	data, err := os.ReadFile("testdata/05_11p.pdf")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/raspisanie/2026/05_11p.pdf" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	day, err := fetchFrom(context.Background(), srv.URL, "2026-11-05")
	if err != nil {
		t.Fatal(err)
	}
	if !day.Found || len(day.Lessons) != 6 {
		t.Fatalf("got %#v", day)
	}
}
