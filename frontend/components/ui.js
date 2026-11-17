import { useEffect } from "react";
import { STATUS, bulanLabel, shiftMonth, thisMonth } from "../lib/format";

export function PageHeader({ title, subtitle, children }) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-3 mb-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        {subtitle && <p className="text-sm text-ink-muted mt-1">{subtitle}</p>}
      </div>
      {children && <div className="flex flex-wrap items-center gap-2 no-print">{children}</div>}
    </div>
  );
}

export function Badge({ status, children }) {
  const s = STATUS[status] || { label: status, cls: "bg-slate-50 text-slate-600 border-slate-200" };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-md border text-xs font-medium whitespace-nowrap ${s.cls}`}>
      {children || s.label}
    </span>
  );
}

export function Alert({ kind = "error", children }) {
  if (!children) return null;
  const cls = {
    error: "bg-red-50 border-red-200 text-red-700",
    info: "bg-sky-50 border-sky-200 text-sky-800",
    warn: "bg-amber-50 border-amber-200 text-amber-800",
    ok: "bg-emerald-50 border-emerald-200 text-emerald-800",
  }[kind];
  return <div className={`border rounded-lg px-4 py-3 text-sm ${cls}`}>{children}</div>;
}

export function Stat({ label, value, note, tone }) {
  const tones = { red: "text-red-600", amber: "text-amber-600", green: "text-emerald-600" };
  return (
    <div className="card p-4">
      <p className="text-xs text-ink-muted">{label}</p>
      <p className={`text-2xl font-semibold mt-1 tabular-nums ${tones[tone] || ""}`}>{value}</p>
      {note && <p className="text-xs text-ink-faint mt-1">{note}</p>}
    </div>
  );
}

export function Field({ label, hint, children }) {
  return (
    <label className="block">
      <span className="label">{label}</span>
      {children}
      {hint && <span className="block text-xs text-ink-faint mt-1">{hint}</span>}
    </label>
  );
}

export function Modal({ open, title, onClose, children, wide }) {
  useEffect(() => {
    if (!open) return;
    const esc = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", esc);
    return () => window.removeEventListener("keydown", esc);
  }, [open, onClose]);
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-start sm:items-center justify-center bg-black/40 p-3 overflow-y-auto no-print" onMouseDown={onClose}>
      <div
        className={`card w-full ${wide ? "max-w-3xl" : "max-w-lg"} shadow-xl my-6`}
        onMouseDown={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-line">
          <h2 className="font-semibold">{title}</h2>
          <button onClick={onClose} className="text-ink-faint hover:text-ink text-xl leading-none" aria-label="Tutup">×</button>
        </div>
        <div className="p-5">{children}</div>
      </div>
    </div>
  );
}

export function MonthPicker({ value, onChange }) {
  const next = shiftMonth(value, 1);
  return (
    <div className="flex items-center gap-1">
      <button className="btn-secondary w-10 px-0" onClick={() => onChange(shiftMonth(value, -1))} aria-label="Bulan sebelumnya">‹</button>
      <input type="month" className="input w-44" value={value} onChange={(e) => e.target.value && onChange(e.target.value)} aria-label={bulanLabel(value)} />
      <button className="btn-secondary w-10 px-0" disabled={next > thisMonth()} onClick={() => onChange(next)} aria-label="Bulan berikutnya">›</button>
    </div>
  );
}

export function Loading({ text = "Memuat…" }) {
  return <p className="text-sm text-ink-muted py-6">{text}</p>;
}

export function Empty({ children }) {
  return <p className="text-sm text-ink-muted text-center py-10">{children}</p>;
}
