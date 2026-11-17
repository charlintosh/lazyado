package api

import (
	"encoding/json"
	"fmt"
	"time"
)

var gitTimeLayouts = []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"}

type gitTime struct {
	time.Time
}

func (t *gitTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("parsing time %s: %w", b, err)
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	for _, layout := range gitTimeLayouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("parsing time %q", s)
}
