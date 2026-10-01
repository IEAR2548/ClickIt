import { useState, type FormEvent } from "react";
import * as api from "../api";
import { useNavigate, Link } from "react-router-dom";

export function RegisterPage() {
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    const navigate = useNavigate();

    async function handleSubmit(e: FormEvent) {
        e.preventDefault();
        setError(null);
        setLoading(true);
        try {
            await api.register(email, password);
            navigate("/login");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Registration failed");
        } finally {
            setLoading(false);
        }
    }

    return (
        <div className="auth-page">
            <form onSubmit={handleSubmit} className="auth-form">
                <h1>Register</h1>
                {error && <p className="error">{error}</p>}
                <label>
                    Email
                    <input
                        type="email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        required
                    />
                </label>
                <label>
                    Password
                    <input
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                    />
                </label>
                <button type="submit" disabled={loading}>
                    {loading ? "Creating account..." : "Register"}
                </button>
                <p>
                    Already have an account? <Link to="/login">login</Link>
                </p>
            </form>
        </div>
    );

}