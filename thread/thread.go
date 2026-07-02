package thread

import "time"

type Thread struct {
	ID        int64
	BoardID   int64
	Title     string
	CreatedAt time.Time
	BumpedAt  time.Time
}
