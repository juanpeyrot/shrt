package models

import "time"

type LinkStats struct {
	TotalClicks  int64          `json:"total_clicks"`
	LastClickAt  *time.Time     `json:"last_click_at,omitempty"`
	ClicksPerDay []DailyClicks  `json:"clicks_per_day"`
	TopReferers  []RefererCount `json:"top_referers"`
}

type DailyClicks struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
}

type RefererCount struct {
	Referer string `json:"referer"`
	Clicks  int64  `json:"clicks"`
}
