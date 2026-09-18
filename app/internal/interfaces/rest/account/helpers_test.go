package account_test

import (
	"errors"
	"time"
)

var (
	testNow = time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	errBoom = errors.New("boom: connection reset by peer")
)
