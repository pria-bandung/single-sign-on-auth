package store

import "time"

// Timestamps are stored as RFC3339 text in UTC. Normalizing to UTC strips any
// monotonic clock reading and gives every row the same zone offset, so the
// "expires_at > now" text comparison is lexically equal to chronological order
// and stays correct across server restarts and timezone/DST changes.

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
