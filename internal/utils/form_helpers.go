package utils

import (
	"strconv"
	"time"
)

func StringToInt(s string) int {
	i, err := strconv.Atoi(s)

	if err != nil {
		return 0
	}
	return i
}

func StringToTime(s string) time.Time {
	t, err := time.Parse(time.DateTime, s)

	if err != nil {
		return time.Now().Add(24 * time.Hour)
	}
	return t
}
