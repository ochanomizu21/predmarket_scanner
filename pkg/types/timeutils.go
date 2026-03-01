package types

import (
	"strconv"
	"time"
)

func ParseTimestamp(ts string) (int64, error) {
	return strconv.ParseInt(ts, 10, 64)
}

func TimestampToTime(ts int64) time.Time {
	seconds := ts / 1000
	nanos := (ts % 1000) * 1000000
	return time.Unix(seconds, nanos)
}
