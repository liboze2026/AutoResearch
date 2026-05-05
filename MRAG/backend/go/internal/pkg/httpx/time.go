package httpx

import "time"

func NowString(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}
