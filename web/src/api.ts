import type { CreatedLink, LinkItem, AnalyticsSummary } from "./types";

export const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:8080";
const TOKEN_KEY = "clickit_token";

export function getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
    localStorage.removeItem(TOKEN_KEY);
}

function authHeaders(): HeadersInit {
    const token = getToken();
    return token ? { Authorization: `Bearer ${token}` } : {};
}

async function handle<T>(res: Response): Promise<T> {
    if (!res.ok) {
        const message = await res.text();
        throw new Error(message || `Request failed with status ${res.status}`);
    }
    if (res.status === 204) {
        return undefined as T;
    }
    const text = await res.text();
    if (!text || !text.trim()) {
        return undefined as T;
    }
    return JSON.parse(text) as T;
}

export async function register(email: string, password: string): Promise<void> {
    const res = await fetch(`${API_BASE}/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
    });
    return handle<void>(res);
}

export async function login(email: string, password: string): Promise<string> {
    const res = await fetch(`${API_BASE}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password })
    });
    const data = await handle<{ token: string }>(res);
    return data.token;
}

export async function listLinks(): Promise<LinkItem[]> {
    const res = await fetch(`${API_BASE}/links`, { headers: authHeaders() });
    return handle<LinkItem[]>(res);
}

export async function createLink(url: string): Promise<CreatedLink> {
    const res = await fetch(`${API_BASE}/shorten`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ url })
    });
    return handle<CreatedLink>(res)
}

export async function getStats(code: string, days = 30): Promise<AnalyticsSummary> {
    const res = await fetch(`${API_BASE}/api/stats/${code}?days=${days}`, {
        headers: authHeaders(),
    });
    return handle<AnalyticsSummary>(res);
}