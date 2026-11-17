import { useCallback, useEffect, useState } from "react";

export const rp = (n) => `Rp ${Math.round(Number(n) || 0).toLocaleString("id-ID")}`;
export const num = (n) => (Number(n) || 0).toLocaleString("id-ID");

/** 125 -> "2 j 5 m" */
export const durasi = (menit) => {
  const m = Math.round(Number(menit) || 0);
  if (m <= 0) return "0 m";
  const j = Math.floor(m / 60);
  return j ? `${j} j${m % 60 ? ` ${m % 60} m` : ""}` : `${m} m`;
};

const pad = (n) => String(n).padStart(2, "0");
export const ymd = (d = new Date()) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
export const thisMonth = () => ymd().slice(0, 7);

export function shiftMonth(ym, delta) {
  const [y, m] = ym.split("-").map(Number);
  const d = new Date(y, m - 1 + delta, 1);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}`;
}

export function shiftDay(day, delta) {
  const [y, m, d] = day.split("-").map(Number);
  return ymd(new Date(y, m - 1, d + delta));
}

export const bulanLabel = (ym) => {
  const [y, m] = ym.split("-").map(Number);
  return new Date(y, m - 1, 1).toLocaleDateString("id-ID", { month: "long", year: "numeric" });
};

export const tglLabel = (day, opts = { weekday: "long", day: "numeric", month: "long", year: "numeric" }) => {
  const [y, m, d] = day.split("-").map(Number);
  return new Date(y, m - 1, d).toLocaleDateString("id-ID", opts);
};

export const tglPendek = (day) => tglLabel(day, { weekday: "short", day: "numeric", month: "short" });

export const jam = (iso) => (iso ? new Date(iso).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" }) : "—");

export const HARI = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];

export const STATUS = {
  hadir: { label: "Hadir", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  izin: { label: "Izin", cls: "bg-sky-50 text-sky-700 border-sky-200" },
  sakit: { label: "Sakit", cls: "bg-violet-50 text-violet-700 border-violet-200" },
  cuti: { label: "Cuti", cls: "bg-indigo-50 text-indigo-700 border-indigo-200" },
  alpa: { label: "Alpa", cls: "bg-red-50 text-red-700 border-red-200" },
  libur: { label: "Libur", cls: "bg-slate-50 text-slate-500 border-slate-200" },
  belum: { label: "Belum absen", cls: "bg-amber-50 text-amber-700 border-amber-200" },
  menunggu: { label: "Menunggu", cls: "bg-amber-50 text-amber-700 border-amber-200" },
  disetujui: { label: "Disetujui", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  ditolak: { label: "Ditolak", cls: "bg-red-50 text-red-700 border-red-200" },
};

/** Muat data async; reload() mengulang. */
export function useLoad(fn, deps = []) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const run = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setData(await fn());
    } catch (e) {
      setError(e.message || "Gagal memuat data");
    } finally {
      setLoading(false);
    }
  }, deps);
  useEffect(() => { run(); }, [run]);
  return { data, error, loading, reload: run, setData };
}

export function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
