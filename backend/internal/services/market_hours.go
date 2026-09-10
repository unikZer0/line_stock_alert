package services

import "time"

// usRegularMarketStatus returns the NYSE/Nasdaq regular-session state. The
// schedule is 09:30-16:00 America/New_York, Monday through Friday.
func usRegularMarketStatus(now time.Time) (string, time.Time) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		newYork = time.FixedZone("ET", -5*60*60)
	}
	local := now.In(newYork)
	open := time.Date(local.Year(), local.Month(), local.Day(), 9, 30, 0, 0, newYork)
	closeAt := time.Date(local.Year(), local.Month(), local.Day(), 16, 0, 0, 0, newYork)
	if isUSTradingDay(local) && !local.Before(open) && local.Before(closeAt) {
		return "OPEN", closeAt.UTC()
	}
	if isUSTradingDay(local) && local.Before(open) {
		return "CLOSED", open.UTC()
	}
	next := local.AddDate(0, 0, 1)
	for !isUSTradingDay(next) {
		next = next.AddDate(0, 0, 1)
	}
	nextOpen := time.Date(next.Year(), next.Month(), next.Day(), 9, 30, 0, 0, newYork)
	return "CLOSED", nextOpen.UTC()
}

func isWeekday(value time.Time) bool {
	return value.Weekday() != time.Saturday && value.Weekday() != time.Sunday
}

func isUSTradingDay(value time.Time) bool {
	return isWeekday(value) && !isUSMarketHoliday(value)
}

func isUSMarketHoliday(value time.Time) bool {
	year, location := value.Year(), value.Location()
	holiday := func(month time.Month, day int) time.Time {
		return observedHoliday(time.Date(year, month, day, 0, 0, 0, 0, location))
	}
	dates := []time.Time{
		holiday(time.January, 1), holiday(time.July, 4), holiday(time.December, 25),
		nthWeekday(year, time.January, time.Monday, 3, location),
		nthWeekday(year, time.February, time.Monday, 3, location),
		lastWeekday(year, time.May, time.Monday, location),
		nthWeekday(year, time.September, time.Monday, 1, location),
		nthWeekday(year, time.November, time.Thursday, 4, location),
		easterSunday(year, location).AddDate(0, 0, -2),
	}
	if year >= 2022 {
		dates = append(dates, holiday(time.June, 19))
	}
	// New Year's Day can be observed on December 31 of the previous year.
	dates = append(dates, observedHoliday(time.Date(year+1, time.January, 1, 0, 0, 0, 0, location)))
	for _, date := range dates {
		if value.Year() == date.Year() && value.YearDay() == date.YearDay() {
			return true
		}
	}
	return false
}

func observedHoliday(date time.Time) time.Time {
	switch date.Weekday() {
	case time.Saturday:
		return date.AddDate(0, 0, -1)
	case time.Sunday:
		return date.AddDate(0, 0, 1)
	}
	return date
}

func nthWeekday(year int, month time.Month, weekday time.Weekday, occurrence int, location *time.Location) time.Time {
	date := time.Date(year, month, 1, 0, 0, 0, 0, location)
	for date.Weekday() != weekday {
		date = date.AddDate(0, 0, 1)
	}
	return date.AddDate(0, 0, (occurrence-1)*7)
}

func lastWeekday(year int, month time.Month, weekday time.Weekday, location *time.Location) time.Time {
	date := time.Date(year, month+1, 0, 0, 0, 0, 0, location)
	for date.Weekday() != weekday {
		date = date.AddDate(0, 0, -1)
	}
	return date
}

func easterSunday(year int, location *time.Location) time.Time {
	a, b, c := year%19, year/100, year%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := time.Month((h + l - 7*m + 114) / 31)
	day := (h+l-7*m+114)%31 + 1
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}
