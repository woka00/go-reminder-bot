package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var monthRuToTime = map[string]time.Month{
	"января":   time.January,
	"февраля":  time.February,
	"марта":    time.March,
	"апреля":   time.April,
	"мая":      time.May,
	"июня":     time.June,
	"июля":     time.July,
	"августа":  time.August,
	"сентября": time.September,
	"октября":  time.October,
	"ноября":   time.November,
	"декабря":  time.December,
}

var (
	reAfterTomorrow = regexp.MustCompile(`(?i)послезавтра`)
	reTomorrow      = regexp.MustCompile(`(?i)завтра`)
	reToday         = regexp.MustCompile(`(?i)сегодня`)
	reDayMonth      = regexp.MustCompile(`(?i)(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)`)
)

type dateResult struct {
	Year  int
	Month time.Month
	Day   int
	Found bool
}

func extractDate(input string, now time.Time) (string, dateResult, error) {
	if loc := reAfterTomorrow.FindStringIndex(input); loc != nil {
		t := now.AddDate(0, 0, 2)
		return cut(input, loc), newDateResult(t), nil
	}
	if loc := reTomorrow.FindStringIndex(input); loc != nil {
		t := now.AddDate(0, 0, 1)
		return cut(input, loc), newDateResult(t), nil
	}
	if loc := reToday.FindStringIndex(input); loc != nil {
		return cut(input, loc), newDateResult(now), nil
	}
	if m := reDayMonth.FindStringSubmatchIndex(input); m != nil {
		day, err := strconv.Atoi(input[m[2]:m[3]])
		if err != nil {
			return input, dateResult{}, fmt.Errorf("parse day: %w", err)
		}
		monthWord := strings.ToLower(input[m[4]:m[5]])
		month, ok := monthRuToTime[monthWord]
		if !ok {
			return input, dateResult{}, fmt.Errorf("unknown month: %s", monthWord)
		}
		if day < 1 || day > 31 {
			return input, dateResult{}, fmt.Errorf("invalid day: %d", day)
		}

		year := now.Year()
		todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		candidate := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
		if candidate.Before(todayMidnight) {
			year++
		}
		return cut(input, []int{m[0], m[1]}), dateResult{Year: year, Month: month, Day: day, Found: true}, nil
	}
	return input, dateResult{}, nil
}

func newDateResult(t time.Time) dateResult {
	return dateResult{Year: t.Year(), Month: t.Month(), Day: t.Day(), Found: true}
}

func cut(s string, loc []int) string {
	return collapseSpaces(s[:loc[0]] + s[loc[1]:])
}
