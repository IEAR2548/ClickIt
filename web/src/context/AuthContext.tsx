import * as api from "../api";
import { useState, type ReactNode } from "react";
import { AuthContext } from "./authContext";

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