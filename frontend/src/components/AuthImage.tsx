"use client";

import { useEffect, useState } from "react";
import { API_URL } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";

export default function AuthImage({ path, alt, className = "" }: { path?: string; alt: string; className?: string }) {
  const [src, setSrc] = useState("");
  const [err, setErr] = useState(false);

  useEffect(() => {
    if (!path) return;
    let url = "";
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch(`${API_URL}${path}`, {
          headers: { Authorization: `Bearer ${getAccessToken() ?? ""}` },
        });
        if (!res.ok) throw new Error();
        const blob = await res.blob();
        if (cancelled) return;
        url = URL.createObjectURL(blob);
        setSrc(url);
      } catch {
        if (!cancelled) setErr(true);
      }
    })();
    return () => { cancelled = true; if (url) URL.revokeObjectURL(url); };
  }, [path]);

  if (!path || err) return <div className={`flex items-center justify-center rounded-lg bg-white/5 text-xs text-cream/40 ${className}`}>Không có ảnh</div>;
  if (!src) return <div className={`animate-pulse rounded-lg bg-white/5 ${className}`} />;
  /* eslint-disable-next-line @next/next/no-img-element */
  return <img src={src} alt={alt} className={`rounded-lg object-cover ${className}`} />;
}
