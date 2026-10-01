import * as api from "../api";
import { createContext, useContext, useState, type ReactNode } from "react";

interface AuthContextValue {
    isAuthenticated: boolean;
    login: (email: string, password: string) => Promise<void>;
    logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [isAuthenticated, setIsAuthenticated] = useState(() => api.getToken() !== null);

    async function login(email: string, password: string) {
        const token = await api.login(email, password);
        api.setToken(token);
        setIsAuthenticated(true);
    }

    function logout() {
        api.clearToken();
        setIsAuthenticated(false);
    }

    return (
        <AuthContext.Provider value={{ isAuthenticated, login, logout }}>
            {children}
        </AuthContext.Provider>
    )
}

export function useAuth(): AuthContextValue {
    const ctx = useContext(AuthContext);
    if (!ctx) {
        throw new Error("useAuth must be used within an AuthProvider");
    }
    return ctx;
}