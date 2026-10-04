import { useCallback, useEffect, useState } from "react";
import type { LinkItem } from "../types";
import { useAuth } from "../context/useAuth";
import * as api from "../api"
import { CreateLinkForm } from "../components/CreateLinkForm";
import { StatsPanel } from "../components/StatsPanel";

export function DashboardPage() {
    const [links, setLinks] = useState<LinkItem[]>([]);
    const [selected, setSelected] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);
    const { logout } = useAuth();

    const fetchLinks = useCallback(async () => {
        try {
            const data = await api.listLinks();
            setLinks(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to fetch links");
        }
    }, []);

    useEffect(() => {
        let isMounted = true;
        api.listLinks()
            .then((data) => {
                if (isMounted) setLinks(data);
            })
            .catch((err) => {
                if (isMounted) setError(err instanceof Error ? err.message : "Failed to fetch links");
            });

        return () => {
            isMounted = false;
        };
    }, []);

    return (
        <div className="dashboard">
            <header>
                <h1>ClickIt Dashboard</h1>
                <button onClick={logout}>Log out</button>
            </header>

            <CreateLinkForm onCreated={fetchLinks} />

            {error && <p className="error">{error}</p>}

            <div className="dashboard-body">
                <ul className="link-list">
                    {links.length == 0 && <li>No links yet, create one above :</li>}
                    {links.map((link) => {
                        const shortUrl = `${api.API_BASE}/${link.short_code}`;
                        return (
                            <li
                                key={link.short_code}
                                className={link.short_code === selected ? "selected" : ""}
                                onClick={() => setSelected(link.short_code)}
                            >
                                <div className="link-row">
                                    <span className="code">{shortUrl}</span>
                                    <div className="link-actions">
                                        <button
                                            type="button"
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                navigator.clipboard.writeText(shortUrl);
                                            }}
                                        >
                                            Copy
                                        </button>

                                        <a
                                            href={shortUrl}
                                            target="_blank"
                                            rel="noreferrer"
                                            onClick={(e) => e.stopPropagation()}
                                        >
                                            Open
                                        </a>
                                    </div>
                                </div>
                                <span className="long-url">{link.long_url}</span>
                            </li>
                        );
                    })}
                </ul>

                {selected && <StatsPanel shortCode={selected} />}
            </div>
        </div>
    );
}