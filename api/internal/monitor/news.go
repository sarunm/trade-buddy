package monitor

import "time"

// NewsEvent represents a recurring monthly high-impact event.
type NewsEvent struct {
	Name        string
	WeekOfMonth int
	Weekday     time.Weekday
	HourUTC     int
	MinuteUTC   int
	// Months: nil means every month; else specific months (1-12).
	Months []int
}

// DefaultNewsEvents is the static schedule for XAUUSD high-impact events.
var DefaultNewsEvents = []NewsEvent{
	{Name: "NFP", WeekOfMonth: 1, Weekday: time.Friday, HourUTC: 13, MinuteUTC: 30},
	{Name: "CPI", WeekOfMonth: 2, Weekday: time.Wednesday, HourUTC: 13, MinuteUTC: 30},
	{Name: "PPI", WeekOfMonth: 2, Weekday: time.Thursday, HourUTC: 13, MinuteUTC: 30},
	{Name: "FOMC", WeekOfMonth: 3, Weekday: time.Wednesday, HourUTC: 19, MinuteUTC: 0, Months: []int{1, 3, 5, 7, 9, 11}},
}

// IsNewsBlocked returns true if t is within +/-15 minutes of any event this month.
func IsNewsBlocked(t time.Time) bool {
	return BlockReason(t) != ""
}

// BlockReason returns the event name if blocked, else "".
func BlockReason(t time.Time) string {
	utc := t.UTC()
	for _, event := range DefaultNewsEvents {
		if !eventAppliesToMonth(event, utc.Month()) {
			continue
		}

		eventDay := nthWeekdayOfMonth(utc.Year(), utc.Month(), event.Weekday, event.WeekOfMonth)
		if eventDay == -1 {
			continue
		}

		eventTime := time.Date(utc.Year(), utc.Month(), eventDay, event.HourUTC, event.MinuteUTC, 0, 0, time.UTC)
		diff := utc.Sub(eventTime)
		if diff < 0 {
			diff = -diff
		}
		if diff <= 15*time.Minute {
			return event.Name
		}
	}
	return ""
}

func nthWeekdayOfMonth(year int, month time.Month, weekday time.Weekday, n int) int {
	if n <= 0 {
		return -1
	}

	count := 0
	for day := 1; ; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if date.Month() != month {
			break
		}
		if date.Weekday() != weekday {
			continue
		}
		count++
		if count == n {
			return day
		}
	}
	return -1
}

func eventAppliesToMonth(event NewsEvent, month time.Month) bool {
	if event.Months == nil {
		return true
	}
	for _, eventMonth := range event.Months {
		if time.Month(eventMonth) == month {
			return true
		}
	}
	return false
}
