package post_test

import "time"

// testNow is the fixed instant these tests build aggregates at, so nothing
// depends on the wall clock.
var testNow = time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
