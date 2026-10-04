import { useState, type FormEvent } from "react";
import * as api from "../api"

export function CreateLinkForm({ onCreated }: { onCreated: () => void }) {
    const [url, setUrl] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);

    async function handleSubmit(e: FormEvent) {
        e.preventDefault();
        setError(null);
        setSubmitting(true);
        try {
            await api.createLink(url);
            setUrl("");
            onCreated();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to create link");
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <form onSubmit={handleSubmit} className="create-form">
            <input
                type="url"
                placeholder="https://example.com/path"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                required
            />
            <button type="submit" disabled={submitting}>
                {submitting ? "Creating..." : "Create Link"}
            </button>
            {error && <p className="error">{error}</p>}
        </form>
    );
}