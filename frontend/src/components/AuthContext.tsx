"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import {
  clearAuth,
  getStoredUser,
  setStoredUser,
  setTokens,
} from "@/lib/auth";
import type { AuthResponse, User } from "@/lib/types";

interface AuthContextValue {
  user: User | null;
  ready: boolean;
  setSession: (res: AuthResponse) => void;
  updateUser: (user: User) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    setUser(getStoredUser());
    setReady(true);
  }, []);

  const setSession = useCallback((res: AuthResponse) => {
    setTokens(res.tokens);
    setStoredUser(res.user);
    setUser(res.user);
  }, []);

  const updateUser = useCallback((u: User) => {
    setStoredUser(u);
    setUser(u);
  }, []);

  const logout = useCallback(() => {
    clearAuth();
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{ user, ready, setSession, updateUser, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
