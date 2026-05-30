package parser

import (
	"fmt"
	"regexp"
	"strconv"
)

const (
	defaultHour   = 9
	defaultMinute = 0
)

var reTime = regexp.MustCompile(`(?i)(?:в\s+)?(\d{1,2}):(\d{2})`)

type timeResult struct {
	Hour   int
	Minute int
	Found  bool
}

func extractTime(input string) (string, timeResult, error) {
	loc := reTime.FindStringSubmatchIndex(input)
	if loc == nil {
		return input, timeResult{Hour: defaultHour, Minute: defaultMinute}, nil
	}
	h, err := strconv.Atoi(input[loc[2]:loc[3]])
	if err != nil {
		return input, timeResult{}, fmt.Errorf("parse hour: %w", err)
	}
	m, err := strconv.Atoi(input[loc[4]:loc[5]])
	if err != nil {
		return input, timeResult{}, fmt.Errorf("parse minute: %w", err)
	}
	if h < 0 || h > 23 {
		return input, timeResult{}, fmt.Errorf("invalid hour: %d", h)
	}
	if m < 0 || m > 59 {
		return input, timeResult{}, fmt.Errorf("invalid minute: %d", m)
	}
	rest := input[:loc[0]] + input[loc[1]:]
	return collapseSpaces(rest), timeResult{Hour: h, Minute: m, Found: true}, nil
}
