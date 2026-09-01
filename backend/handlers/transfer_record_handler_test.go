package handlers

import (
	"testing"
	"time"
)

func TestParseTransferRecordDate(t *testing.T) {
	start, err := parseTransferRecordDate("2026-09-01", false)
	if err != nil || start == nil {
		t.Fatalf("parse start date: value=%v error=%v", start, err)
	}
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Fatalf("start date must begin at midnight: %v", start)
	}

	end, err := parseTransferRecordDate("2026-09-01", true)
	if err != nil || end == nil {
		t.Fatalf("parse end date: value=%v error=%v", end, err)
	}
	wantEnd := start.Add(24*time.Hour - time.Nanosecond)
	if !end.Equal(wantEnd) {
		t.Fatalf("end date = %v, want %v", end, wantEnd)
	}

	if value, err := parseTransferRecordDate("", false); err != nil || value != nil {
		t.Fatalf("empty date = %v, %v; want nil, nil", value, err)
	}
	if _, err := parseTransferRecordDate("not-a-date", false); err == nil {
		t.Fatal("invalid date should return an error")
	}
}
