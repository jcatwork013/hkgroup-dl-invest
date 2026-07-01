"use client";

import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";

export function Button({
  variant = "primary",
  className = "",
  children,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant }) {
  const base =
    "inline-flex items-center justify-center rounded-full px-5 py-2.5 text-sm font-medium leading-none transition disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-gold-500/40";
  const variants: Record<ButtonVariant, string> = {
    primary:
      "bg-gold-500 text-forest-950 hover:bg-gold-400 shadow-[0_8px_24px_-10px_rgba(201,162,74,.6)]",
    secondary:
      "border border-gold-500/40 text-cream hover:bg-white/5 hover:border-gold-500/70",
    danger: "bg-red-600/90 text-white hover:bg-red-600",
    ghost: "bg-transparent text-cream/70 hover:bg-white/5 hover:text-cream",
  };
  return (
    <button className={`${base} ${variants[variant]} ${className}`} {...props}>
      {children}
    </button>
  );
}

export function Input({
  label,
  className = "",
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { label?: string }) {
  return (
    <label className="block">
      {label && (
        <span className="mb-1.5 block text-sm font-medium text-cream/80">
          {label}
        </span>
      )}
      <input
        className={`w-full rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm text-cream placeholder:text-cream/35 focus:border-gold-500 focus:outline-none focus:ring-1 focus:ring-gold-500/50 ${className}`}
        {...props}
      />
    </label>
  );
}

export function Select({
  label,
  className = "",
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement> & { label?: string }) {
  return (
    <label className="block">
      {label && (
        <span className="mb-1.5 block text-sm font-medium text-cream/80">
          {label}
        </span>
      )}
      <select
        className={`w-full rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm text-cream focus:border-gold-500 focus:outline-none focus:ring-1 focus:ring-gold-500/50 [&>option]:bg-ink ${className}`}
        {...props}
      >
        {children}
      </select>
    </label>
  );
}

export function Textarea({
  label,
  className = "",
  ...props
}: TextareaHTMLAttributes<HTMLTextAreaElement> & { label?: string }) {
  return (
    <label className="block">
      {label && (
        <span className="mb-1.5 block text-sm font-medium text-cream/80">
          {label}
        </span>
      )}
      <textarea
        className={`w-full rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm text-cream placeholder:text-cream/35 focus:border-gold-500 focus:outline-none focus:ring-1 focus:ring-gold-500/50 ${className}`}
        {...props}
      />
    </label>
  );
}

export function Card({
  children,
  className = "",
  title,
}: {
  children: ReactNode;
  className?: string;
  title?: string;
}) {
  return (
    <div
      className={`rounded-2xl border border-white/10 bg-white/[0.04] p-5 shadow-[0_20px_60px_-30px_rgba(0,0,0,.7)] backdrop-blur-sm ${className}`}
    >
      {title && (
        <h3 className="mb-3 text-sm font-semibold text-cream/80">{title}</h3>
      )}
      {children}
    </div>
  );
}

export function Stat({
  label,
  value,
  hint,
}: {
  label: string;
  value: ReactNode;
  hint?: string;
}) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.04] p-5 shadow-[0_20px_60px_-30px_rgba(0,0,0,.7)] backdrop-blur-sm">
      <p className="text-xs font-medium uppercase tracking-wide text-cream/45">
        {label}
      </p>
      <p className="mt-2 font-serif text-3xl font-semibold text-gold-400">
        {value}
      </p>
      {hint && <p className="mt-1 text-xs text-cream/40">{hint}</p>}
    </div>
  );
}

export function Badge({
  children,
  tone = "slate",
}: {
  children: ReactNode;
  tone?: "slate" | "green" | "yellow" | "red" | "blue";
}) {
  const tones: Record<string, string> = {
    slate: "bg-white/10 text-cream/75",
    green: "bg-brand-500/20 text-brand-200",
    yellow: "bg-gold-500/20 text-gold-300",
    red: "bg-red-500/20 text-red-300",
    blue: "bg-gold-500/20 text-gold-300",
  };
  return (
    <span
      className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${tones[tone]}`}
    >
      {children}
    </span>
  );
}

export function statusTone(
  status: string
): "slate" | "green" | "yellow" | "red" | "blue" {
  switch (status) {
    case "approved":
    case "paid":
    case "reconciled":
      return "green";
    case "pending":
    case "unverified":
      return "yellow";
    case "rejected":
    case "void":
    case "cancelled":
      return "red";
    case "signed":
    case "open":
    case "reserved":
      return "blue";
    default:
      return "slate";
  }
}

// statusLabel maps every backend status enum (investment / commission / withdrawal / contract /
// KYC / reservation / offering) to enterprise-grade Vietnamese for display.
export function statusLabel(status: string): string {
  switch (status) {
    case "pending":
      return "Chờ duyệt";
    case "reconciled":
      return "Đã đối soát";
    case "approved":
      return "Đã duyệt";
    case "paid":
      return "Đã chi trả";
    case "rejected":
      return "Đã từ chối";
    case "draft":
      return "Bản nháp";
    case "signed":
      return "Đã ký";
    case "void":
      return "Đã vô hiệu";
    case "unverified":
      return "Chưa xác minh";
    case "verified":
      return "Đã xác minh";
    case "reserved":
      return "Đã giữ chỗ";
    case "cancelled":
      return "Đã hủy";
    case "open":
      return "Đang mở";
    case "closed":
      return "Đã đóng";
    default:
      return status;
  }
}

export function ErrorText({ children }: { children: ReactNode }) {
  if (!children) return null;
  return (
    <p className="rounded-lg border border-red-500/25 bg-red-500/10 px-3 py-2 text-sm text-red-300">
      {children}
    </p>
  );
}

export function Spinner() {
  return (
    <div className="flex items-center justify-center py-10 text-sm text-cream/55">
      Đang tải...
    </div>
  );
}

// Decorative gold eyebrow used above section/hero headings.
export function Eyebrow({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.2em] text-gold-400">
      <span className="h-px w-6 bg-gold-500/70" />
      {children}
    </span>
  );
}
