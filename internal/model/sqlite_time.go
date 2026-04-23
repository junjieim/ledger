package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type SQLiteTime struct {
	time.Time
}

func (t *SQLiteTime) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		parsed, err := parseSQLiteTime(v)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	case []byte:
		parsed, err := parseSQLiteTime(string(v))
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	case nil:
		t.Time = time.Time{}
		return nil
	default:
		return fmt.Errorf("unsupported SQLite time value %T", value)
	}
}

func (t SQLiteTime) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time.UTC().Format(time.RFC3339), nil
}

func parseSQLiteTime(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse SQLite time %q", raw)
}
