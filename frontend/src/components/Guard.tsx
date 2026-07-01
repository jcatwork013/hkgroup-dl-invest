"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import { useAuth } from "./AuthContext";
import { Spinner } from "./ui";

export default function Guard({
  children,
  requireRole,
}: {
  children: ReactNode;
  requireRole?: "investor" | "admin" | "saler";
}) {
  const { user, ready } = useAuth();
  const router = useRouter();

  const home = (role?: string) => (role === "admin" ? "/admin" : role === "saler" ? "/sales/orders" : "/dashboard");

  useEffect(() => {
    if (!ready) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    if (requireRole && user.role !== requireRole) {
      router.replace(home(user.role));
    }
  }, [ready, user, requireRole, router]);

  if (!ready || !user) return <Spinner />;
  if (requireRole && user.role !== requireRole) return <Spinner />;

  return <>{children}</>;
}
