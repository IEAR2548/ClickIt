package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type querier interface {
	execer
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type AnalyticsSummary struct {
	ShortCode    string          `json:"short_code"`
	TotalClicks  int64           `json:"total_clicks"`
	ClicksByDay  []DailyClicks   `json:"clicks_by_day"`
	TopReferrers []ReferrerCount `json:"top_referrers"`
}

type DailyClicks struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
}

type ReferrerCount struct {
	Referrer string `json:"referrer"`
	Clicks   int64  `json:"clicks"`
}

type AnalyticsReader interface {
	Summary(ctx context.Context, code string, days int) (*AnalyticsSummary, error)
}

var _ AnalyticsReader = (*AnalyticsStore)(nil)

type AnalyticsStore struct {
	db querier
}

func (a *AnalyticsStore) Summary(ctx context.Context, code string, days int) (*AnalyticsSummary, error) {
	summary := AnalyticsSummary{
		ShortCode:    code,
		ClicksByDay:  []DailyClicks{},
		TopReferrers: []ReferrerCount{},
	}

	if err := a.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM clicks WHERE short_code = $1`, code,
	).Scan(&summary.TotalClicks); err != nil {
		return &AnalyticsSummary{}, fmt.Errorf("total clicks: %w", err)
	}

	dailyRows, err := a.db.Query(ctx,
		`WITH days AS (
			SELECT generate_series(
				date_trunc('day', now()) - ($2 * INTERVAL '1 day'),
				date_trunc('day', now()),
				'1 day'::interval
			)::date AS day
		)
		SELECT d.day, COUNT(c.id) AS clicks
		FROM days d
		LEFT JOIN clicks c 
			ON date_trunc('day', c.created_at) = d.day
			AND c.short_code = $1
		GROUP BY d.day
		ORDER BY d.day
		`, code, days)
	if err != nil {
		return &AnalyticsSummary{}, fmt.Errorf("clicks by day: %w", err)
	}
	defer dailyRows.Close()

	for dailyRows.Next() {
		var day time.Time
		var clicks int64
		if err := dailyRows.Scan(&day, &clicks); err != nil {
			return &AnalyticsSummary{}, fmt.Errorf("scan daily row: %w", err)
		}
		summary.ClicksByDay = append(summary.ClicksByDay, DailyClicks{
			Date:   day.Format("2006-01-02"),
			Clicks: clicks,
		})
	}

	if err := dailyRows.Err(); err != nil {
		return &AnalyticsSummary{}, fmt.Errorf("iterate daily rows: %w", err)
	}

	refRows, err := a.db.Query(ctx,
		`SELECT COALESCE(NULLIF(referrer, ''), 'direct') AS referrer, COUNT(*) AS clicks
		FROM clicks
		WHERE short_code = $1
		GROUP BY 1
		ORDER BY clicks DESC
		LIMIT 5
		`, code)
	if err != nil {
		return &AnalyticsSummary{}, fmt.Errorf("top referrer: %w", err)
	}
	defer refRows.Close()

	for refRows.Next() {
		var rc ReferrerCount
		if err := refRows.Scan(&rc.Referrer, &rc.Clicks); err != nil {
			return &AnalyticsSummary{}, fmt.Errorf("scan referrer row: %w", err)
		}
		summary.TopReferrers = append(summary.TopReferrers, rc)
	}
	if err := refRows.Err(); err != nil {
		return &AnalyticsSummary{}, fmt.Errorf("iterate referrer rows: %w", err)
	}

	return &summary, nil
}
