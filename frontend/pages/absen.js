import { useEffect, useState } from "react";
import Link from "next/link";
import api from "../lib/api";
import { durasi, jam, tglLabel, thisMonth, useLoad } from "../lib/format";
import { Alert, Badge, Loading, PageHeader, Stat } from "../components/ui";

/** Jam berjalan mengikuti waktu server (bukan jam HP karyawan). */
function useServerClock(serverTime) {
  const [now, setNow] = useState(null);
  useEffect(() => {
    if (!serverTime) return;
    const offset = new Date(serverTime).getTime() - Date.now();
    const tick = () => setNow(new Date(Date.now() + offset));
    tick();
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [serverTime]);
  return now;
}

export default function Absen({ user }) {
  const today = useLoad(() => api.myToday(), []);
  const rekap = useLoad(() => api.myPayroll(thisMonth()), []);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  const clock = useServerClock(today.data?.server_time);
  const v = today.data;

  async function act(kind) {
    setBusy(true);
    setError("");
    setDone("");
    try {
      const res = kind === "in" ? await api.checkIn() : await api.checkOut();
      today.setData(res);
      setDone(kind === "in" ? "Absen masuk berhasil dicatat." : "Absen pulang berhasil dicatat. Hati-hati di jalan!");
      rekap.reload();
    } catch (e) {
      setError(e.message);
      today.reload();
    } finally {
      setBusy(false);
    }
  }

  const r = rekap.data?.slip?.rekap;
  const a = v?.absensi;
  const h = v?.hasil;

  return (
    <div>
      <PageHeader title={`Halo, ${user.nama.split(" ")[0]}`} subtitle={v ? tglLabel(v.tanggal) : ""} />
      {today.loading && !v && <Loading />}
      <Alert>{today.error}</Alert>

      {v && (
        <div className="grid lg:grid-cols-5 gap-5">
          <div className="card p-6 sm:p-8 lg:col-span-3 flex flex-col items-center text-center gap-5">
            <div>
              <p className="text-5xl sm:text-6xl font-semibold tabular-nums tracking-tight">
                {clock ? clock.toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit", second: "2-digit" }) : "--.--.--"}
              </p>
              <p className="text-sm text-ink-muted mt-2">
                Jadwal kerja {v.jadwal_masuk} – {v.jadwal_pulang}
                {v.libur ? ` · Libur: ${v.libur}` : !v.hari_kerja ? " · Hari libur" : ""}
              </p>
            </div>

            <p className="text-sm text-ink">{v.pesan}</p>

            <div className="grid grid-cols-2 gap-3 w-full max-w-md">
              <button className="btn-primary h-14 text-base" disabled={busy || !v.bisa_masuk} onClick={() => act("in")}>
                Absen Masuk
              </button>
              <button className="btn h-14 text-base bg-ink text-white hover:bg-black" disabled={busy || !v.bisa_pulang} onClick={() => act("out")}>
                Absen Pulang
              </button>
            </div>
            <div className="w-full max-w-md flex flex-col gap-2">
              <Alert>{error}</Alert>
              <Alert kind="ok">{done}</Alert>
            </div>

            {v.absen_terbuka && (
              <Alert kind="warn">
                Shift tanggal {v.absen_terbuka.tanggal} belum absen pulang (masuk {jam(v.absen_terbuka.jam_masuk)}).
              </Alert>
            )}

            <div className="grid grid-cols-3 w-full max-w-md border-t border-line pt-5 text-sm">
              <div>
                <p className="text-ink-muted text-xs">Masuk</p>
                <p className="font-semibold tabular-nums mt-0.5">{jam(a?.jam_masuk)}</p>
              </div>
              <div>
                <p className="text-ink-muted text-xs">Pulang</p>
                <p className="font-semibold tabular-nums mt-0.5">{jam(a?.jam_pulang)}</p>
              </div>
              <div>
                <p className="text-ink-muted text-xs">Status</p>
                <div className="mt-0.5">
                  {a ? <Badge status={a.status}>{h?.telat_menit ? `Telat ${durasi(h.telat_menit)}` : undefined}</Badge> : <Badge status="belum" />}
                </div>
              </div>
            </div>
          </div>

          <div className="lg:col-span-2 flex flex-col gap-4">
            <div className="grid grid-cols-2 gap-3">
              <Stat label="Hadir bulan ini" value={r ? r.hadir : "—"} note={r ? `dari ${r.hari_kerja} hari kerja` : ""} />
              <Stat label="Alpa" value={r ? r.alpa : "—"} tone={r?.alpa ? "red" : undefined} />
              <Stat label="Total telat" value={r ? durasi(r.telat_menit) : "—"} note={r ? `${r.telat} kali` : ""} tone={r?.telat_menit ? "amber" : undefined} />
              <Stat label="Total lembur" value={r ? durasi(r.lembur_menit) : "—"} tone={r?.lembur_menit ? "green" : undefined} />
            </div>
            <div className="card p-4 text-sm flex flex-col gap-2">
              <p className="font-medium">Tidak bisa masuk kerja?</p>
              <p className="text-ink-muted">Ajukan izin, sakit, atau cuti supaya tidak dihitung alpa.</p>
              <Link href="/pengajuan" className="btn-secondary self-start mt-1">Buat pengajuan</Link>
            </div>
            {rekap.data && (
              <p className="text-xs text-ink-faint">
                Toleransi telat {rekap.data.aturan.toleransi_telat_menit} menit. Absen masuk dibuka mulai pukul {v.absen_masuk_mulai}.
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
