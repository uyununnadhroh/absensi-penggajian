import Link from "next/link";
import { useRouter } from "next/router";
import { useState } from "react";
import { clearSession } from "../lib/session";

const MENU = {
  karyawan: [
    { href: "/absen", label: "Absen Hari Ini", icon: "M12 6v6l4 2M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z" },
    { href: "/riwayat", label: "Riwayat Absensi", icon: "M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z" },
    { href: "/pengajuan", label: "Izin / Sakit / Cuti", icon: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6ZM14 2v6h6M9 15h6M9 11h2" },
    { href: "/slip", label: "Slip Gaji", icon: "M2 7h20v12a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V7ZM16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2M12 12v4M10 14h4" },
  ],
  admin: [
    { href: "/admin", label: "Dashboard", icon: "M3 3h7v9H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 16h7v5H3z" },
    { href: "/admin/absensi", label: "Data Absensi", icon: "M8 2v4M16 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2ZM9 16l2 2 4-4" },
    { href: "/admin/pengajuan", label: "Pengajuan Izin", icon: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6ZM14 2v6h6M9 15l2 2 4-4" },
    { href: "/admin/penggajian", label: "Penggajian", icon: "M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" },
    { href: "/admin/karyawan", label: "Karyawan", icon: "M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" },
    { href: "/admin/pengaturan", label: "Pengaturan", icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-2.82 1.17V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-2.82-1.17l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 3.2 14H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.17-2.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 10 3.2V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 2.82 1.17l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 20.8 10H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" },
  ],
};

function Icon({ d }) {
  return (
    <svg viewBox="0 0 24 24" className="size-[18px] shrink-0" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={d} />
    </svg>
  );
}

export default function Layout({ user, children }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const items = [...(MENU[user.role] || []), { href: "/akun", label: "Akun Saya", icon: "M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z" }];
  const active = (href) => router.pathname === href || (href !== "/admin" && router.pathname.startsWith(href + "/"));

  const logout = () => {
    clearSession();
    router.replace("/login");
  };

  const nav = (
    <nav className="flex flex-col gap-1">
      {items.map((it) => (
        <Link
          key={it.href}
          href={it.href}
          onClick={() => setOpen(false)}
          className={`flex items-center gap-3 h-10 px-3 rounded-lg text-sm transition-colors ${
            active(it.href) ? "bg-brand-500 text-white font-medium" : "text-brand-100 hover:bg-white/10"
          }`}
        >
          <Icon d={it.icon} />
          {it.label}
        </Link>
      ))}
    </nav>
  );

  const profile = (
    <div className="border-t border-white/10 pt-4 flex flex-col gap-3">
      <div className="flex items-center gap-3 px-1">
        <div className="size-9 rounded-full bg-brand-500 text-white grid place-items-center text-sm font-semibold shrink-0">
          {(user.nama || "?").trim().charAt(0).toUpperCase()}
        </div>
        <div className="min-w-0">
          <p className="text-sm text-white font-medium truncate">{user.nama}</p>
          <p className="text-xs text-brand-100/70">{user.role === "admin" ? "Admin / HRD" : "Karyawan"}</p>
        </div>
      </div>
      <button onClick={logout} className="h-9 rounded-lg border border-white/20 text-sm text-white hover:bg-white/10">Keluar</button>
    </div>
  );

  return (
    <div className="min-h-full md:flex">
      {/* sidebar desktop */}
      <aside className="hidden md:flex no-print flex-col justify-between w-60 shrink-0 bg-brand-900 p-4 h-screen sticky top-0">
        <div className="flex flex-col gap-6">
          <Brand />
          {nav}
        </div>
        {profile}
      </aside>

      {/* header mobile */}
      <header className="md:hidden no-print sticky top-0 z-40 bg-brand-900 px-4 h-14 flex items-center justify-between">
        <Brand />
        <button onClick={() => setOpen(!open)} className="text-white p-2 -mr-2" aria-label="Menu" aria-expanded={open}>
          <svg viewBox="0 0 24 24" className="size-6" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d={open ? "M6 6l12 12M18 6 6 18" : "M4 6h16M4 12h16M4 18h16"} /></svg>
        </button>
      </header>
      {open && (
        <div className="md:hidden no-print fixed inset-x-0 top-14 z-40 bg-brand-900 p-4 flex flex-col gap-4 shadow-xl">
          {nav}
          {profile}
        </div>
      )}

      <main className="flex-1 min-w-0 px-4 py-6 sm:px-8 sm:py-8 max-w-6xl">{children}</main>
    </div>
  );
}

function Brand() {
  return (
    <div className="flex items-center gap-2.5">
      <div className="size-8 rounded-lg bg-brand-500 grid place-items-center">
        <svg viewBox="0 0 24 24" className="size-5 text-white" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 11l3 3L22 4M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" /></svg>
      </div>
      <div className="leading-tight">
        <p className="text-white text-sm font-semibold">Absensi</p>
        <p className="text-brand-100/70 text-[11px]">&amp; Penggajian</p>
      </div>
    </div>
  );
}
