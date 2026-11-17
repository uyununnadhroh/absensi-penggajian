import Link from "next/link";
import api from "../../lib/api";
import { durasi, jam, tglLabel, useLoad } from "../../lib/format";
import { Alert, Badge, Empty, Loading, PageHeader, Stat } from "../../components/ui";

export default function Dashboard() {
  const { data, error, loading, reload } = useLoad(() => api.dashboard(), []);
  const j = data?.jumlah;
  const d = data?.hari_ini;
  const order = { belum: 0, alpa: 1, hadir: 2, izin: 3, sakit: 3, cuti: 3, libur: 4 };
  const rows = [...(d?.rows || [])].sort((a, b) => (order[a.hasil.status] ?? 9) - (order[b.hasil.status] ?? 9) || a.karyawan.nama.localeCompare(b.karyawan.nama));

  return (
    <div>
      <PageHeader title="Dashboard" subtitle={d ? `${tglLabel(d.tanggal)}${d.libur ? ` · Libur: ${d.libur}` : !d.hari_kerja ? " · Hari libur" : ""}` : ""}>
        <button className="btn-secondary" onClick={reload}>Muat ulang</button>
      </PageHeader>
      <Alert>{error}</Alert>
      {loading && !data && <Loading />}
      {data && (
        <div className="flex flex-col gap-5">
          {data.pengajuan_menunggu > 0 && (
            <Alert kind="warn">
              Ada {data.pengajuan_menunggu} pengajuan izin/sakit/cuti menunggu persetujuan.{" "}
              <Link href="/admin/pengajuan" className="underline font-medium">Tinjau sekarang</Link>
            </Alert>
          )}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
            <Stat label="Karyawan aktif" value={j.karyawan} />
            <Stat label="Hadir" value={j.hadir} tone="green" note={j.belum_pulang ? `${j.belum_pulang} belum pulang` : ""} />
            <Stat label="Telat" value={j.telat} tone={j.telat ? "amber" : undefined} />
            <Stat label="Belum absen" value={j.belum} tone={j.belum ? "amber" : undefined} />
            <Stat label="Izin / sakit / cuti" value={j.izin + j.sakit + j.cuti} />
            <Stat label="Alpa" value={j.alpa} tone={j.alpa ? "red" : undefined} />
          </div>

          <div className="card overflow-x-auto">
            <div className="px-5 py-4 border-b border-line flex items-center justify-between">
              <h2 className="font-semibold">Kehadiran hari ini</h2>
              <Link href="/admin/absensi" className="text-sm text-brand-600 hover:underline">Ubah / koreksi →</Link>
            </div>
            {rows.length === 0 ? (
              <Empty>Belum ada karyawan. Tambahkan di menu Karyawan.</Empty>
            ) : (
              <table className="w-full">
                <thead className="bg-canvas border-b border-line">
                  <tr><th className="th">Karyawan</th><th className="th">Jadwal</th><th className="th">Status</th><th className="th">Masuk</th><th className="th">Pulang</th><th className="th">Telat</th></tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {rows.map(({ karyawan: k, absensi: a, hasil: h }) => (
                    <tr key={k.id}>
                      <td className="td"><p className="font-medium">{k.nama}</p><p className="text-xs text-ink-muted">{k.jabatan || k.nik}</p></td>
                      <td className="td tabular-nums text-ink-muted">{k.jam_masuk}–{k.jam_pulang}</td>
                      <td className="td"><Badge status={h.status} /></td>
                      <td className="td tabular-nums">{jam(a?.jam_masuk)}</td>
                      <td className="td tabular-nums">{jam(a?.jam_pulang)}</td>
                      <td className={`td tabular-nums ${h.telat_menit ? "text-red-600" : "text-ink-faint"}`}>{h.telat_menit ? durasi(h.telat_menit) : "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
