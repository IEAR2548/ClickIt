import { useEffect, useState } from "react";
import type { AnalyticsSummary } from "../types";
import * as api from "../api";
import { LineChart, Line, XAxis, YAxis, Tooltip, CartesianGrid, ResponsiveContainer } from "recharts";

export function StatsPanel({ shortCode }: { shortCode: string }) {
    const [summary, setSummary] = useState<AnalyticsSummary | null>(null);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        let cancelled = false;

        api.getStats(shortCode)
            .then((data) => {
                if (!cancelled) setSummary(data);
            })
            .catch((err) => {
                if (!cancelled) setError(err instanceof Error ? err.message : "Failed to load stats");
            });

        return () => {
            cancelled = true;
        };
    }, [shortCode]);

    if (error) return <p className="error">{error}</p>;
    if (!summary) return <p>Loading stats...</p>;

    return (
        <div className="stats-panel">
            <h3>Stats for /{summary.short_code}</h3>
            <p className="total-clicks">{summary.total_clicks} clicks</p>

            <ResponsiveContainer width="100%" height={200}>
                <LineChart data={summary.clicks_by_day || []}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" tick={{ fontSize: 12 }} />
                    <YAxis allowDecimals={false} width={30} />
                    <Tooltip />
                    <Line type="monotone" dataKey="clicks" stroke="#8884d8" strokeWidth={2} dot={false} />
                </LineChart>
            </ResponsiveContainer>

            <h4>Top referrers</h4>
            {!summary.top_referrers || summary.top_referrers.length === 0 ? (
                <p>No referrer data yet</p>
            ) : (
                <ul className="referrer-list">
                    {summary.top_referrers.map((r) => (
                        <li key={r.referrer}>
                            <span>{r.referrer}</span>
                            <span>{r.clicks}</span>
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
}