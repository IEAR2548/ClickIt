export interface CreatedLink {
    short_code: string;
    short_url: string;
}

export interface LinkItem {
    short_code: string;
    long_url: string;
    created_at: string;
}

export interface DailyClicks {
    date: string;
    clicks: number;
}

export interface ReferrerCount {
    referrer: string;
    clicks: number;
}

export interface AnalyticsSummary {
    short_code: string;
    total_clicks: number;
    clicks_by_day: DailyClicks[];
    top_referrers: ReferrerCount[];
}