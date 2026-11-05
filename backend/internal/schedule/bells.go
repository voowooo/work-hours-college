package schedule

import "time"

// Минск круглый год UTC+3, без перехода на летнее время.
var minsk = time.FixedZone("Europe/Minsk", 3*60*60)

type bell struct {
	start int
	end   int
}

var weekdayBells = []bell{
	{8*60 + 0, 8*60 + 45},
	{8*60 + 55, 9*60 + 40},
	{9*60 + 50, 10*60 + 35},
	{10*60 + 45, 11*60 + 30},
	{11*60 + 40, 12*60 + 25},
	{12*60 + 45, 13*60 + 30},
	{13*60 + 40, 14*60 + 25},
	{14*60 + 35, 15*60 + 20},
	{15*60 + 30, 16*60 + 15},
	{16*60 + 35, 17*60 + 20},
	{17*60 + 30, 18*60 + 15},
	{18*60 + 25, 19*60 + 10},
	{19*60 + 20, 20*60 + 5},
	{20*60 + 15, 21*60 + 0},
}

var saturdayBells = []bell{
	{8*60 + 0, 8*60 + 45},
	{8*60 + 55, 9*60 + 40},
	{9*60 + 50, 10*60 + 35},
	{10*60 + 45, 11*60 + 30},
	{11*60 + 40, 12*60 + 25},
	{12*60 + 35, 13*60 + 20},
	{13*60 + 30, 14*60 + 15},
	{14*60 + 25, 15*60 + 10},
}

func minskClock(now time.Time) (string, int) {
	local := now.In(minsk)
	return local.Format("2006-01-02"), local.Hour()*60 + local.Minute()
}

func lessonEnded(date string, period int, now time.Time) bool {
	today, minutes := minskClock(now)
	if date < today {
		return true
	}
	if date > today {
		return false
	}
	end, ok := lessonEndMinutes(date, period)
	if !ok {
		return false
	}
	return minutes >= end
}

func lessonEndMinutes(date string, period int) (int, bool) {
	parsed, err := time.ParseInLocation("2006-01-02", date, minsk)
	if err != nil || period < 1 {
		return 0, false
	}
	table := weekdayBells
	switch parsed.Weekday() {
	case time.Saturday:
		table = saturdayBells
	case time.Sunday:
		return 0, false
	}
	if period > len(table) {
		return 0, false
	}
	return table[period-1].end, true
}
