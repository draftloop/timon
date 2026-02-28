package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var durationRegexp = regexp.MustCompile(`(\d+)(mo|[ywd])`)

func ParseDuration(s string) (time.Duration, error) {
	s = strings.ReplaceAll(s, " ", "")
	units := map[string]int{"y": 365 * 24, "mo": 30 * 24, "w": 7 * 24, "d": 24}
	converted := durationRegexp.ReplaceAllStringFunc(s, func(match string) string {
		sub := durationRegexp.FindStringSubmatch(match)
		val, _ := strconv.Atoi(sub[1])
		if h, ok := units[sub[2]]; ok {
			return fmt.Sprintf("%dh", val*h)
		}
		return match
	})
	t, err := time.ParseDuration(converted)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	return t, nil
}

func HumanDuration(d time.Duration) string {
	units := []struct {
		div    time.Duration
		symbol string
	}{
		{365 * 24 * time.Hour, "y"},
		{30 * 24 * time.Hour, "mo"},
		{7 * 24 * time.Hour, "w"},
		{24 * time.Hour, "d"},
		{time.Hour, "h"},
		{time.Minute, "m"},
		{time.Second, "s"},
		{time.Millisecond, "ms"},
	}

	var p []string
	for _, u := range units {
		if d < u.div {
			continue
		}
		major := d / u.div
		p = append(p, fmt.Sprintf("%d%s", major, u.symbol))
		if len(p) == 2 {
			break
		}
		d -= major * u.div
	}
	if len(p) == 0 {
		p = append(p, "0s")
	}
	return strings.Join(p, " ")
}
