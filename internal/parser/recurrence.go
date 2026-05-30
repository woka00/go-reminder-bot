package parser

import (
	"regexp"
	"strings"
)

var weekdayRuToEn = map[string]string{
	"понедельник": "monday",
	"вторник":     "tuesday",
	"среду":       "wednesday",
	"среда":       "wednesday",
	"четверг":     "thursday",
	"пятницу":     "friday",
	"пятница":     "friday",
	"субботу":     "saturday",
	"суббота":     "saturday",
	"воскресенье": "sunday",
}

var (
	reEveryWeekday = regexp.MustCompile(`(?i)кажд(?:ый|ое|ую)\s+(понедельник|вторник|среду|среда|четверг|пятницу|пятница|субботу|суббота|воскресенье)`)
	reEveryWeek    = regexp.MustCompile(`(?i)кажд(?:ый|ое|ую)\s+неделю`)
	reEveryDay     = regexp.MustCompile(`(?i)кажд(?:ый|ое|ую)\s+день`)
)

type recurrenceResult struct {
	Recurrence    *string
	RecurrenceDay *string
}

func extractRecurrence(input string) (string, recurrenceResult) {
	if m := reEveryWeekday.FindStringSubmatchIndex(input); m != nil {
		rec := "weekly"
		day := weekdayRuToEn[strings.ToLower(input[m[2]:m[3]])]
		rest := input[:m[0]] + input[m[1]:]
		return collapseSpaces(rest), recurrenceResult{Recurrence: &rec, RecurrenceDay: &day}
	}
	if loc := reEveryWeek.FindStringIndex(input); loc != nil {
		rec := "weekly"
		rest := input[:loc[0]] + input[loc[1]:]
		return collapseSpaces(rest), recurrenceResult{Recurrence: &rec}
	}
	if loc := reEveryDay.FindStringIndex(input); loc != nil {
		rec := "daily"
		rest := input[:loc[0]] + input[loc[1]:]
		return collapseSpaces(rest), recurrenceResult{Recurrence: &rec}
	}
	return input, recurrenceResult{}
}
