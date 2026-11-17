import { Badge } from "./ui";
import { bulanLabel, durasi, num, rp, tglPendek } from "../lib/format";

/** Rincian slip gaji satu karyawan (dipakai halaman karyawan & admin, bisa dicetak). */
export function Slip({ slip, bulan, ditutup, aturan }) {
  const { karyawan: k, rekap: r, rincian: p } = slip;
  
  // Untuk sistem per jam
  const isPerjam = k.tipe_gaji === "per_jam";
  const gajiLabel = isPerjam 
    ? `Gaji per jam (${r.total_jam_kerja?.toFixed(2) || 0} jam × ${rp(k.tarif_per_jam)})`
    : k.tipe_gaji === "harian" 
      ? `Upah harian (${num(r.hadir)} hari × ${rp(k.gaji_pokok)})`
      : "Gaji pokok";
  
  const pendapatan = [
    [gajiLabel, p.gaji_pokok],
    ["Tunjangan tetap", p.tunjangan_tetap],
    [`Uang makan (${num(r.hadir)} hari × ${rp(k.uang_makan)})`, p.uang_makan],
    [`Lembur (${durasi(r.lembur_menit)} × ${rp(k.tarif_lembur_per_jam)}/jam)`, p.lembur],
  ];
  const potongan = [
    ...(k.tipe_gaji === "bulanan" ? [[`Alpa (${num(r.alpa)} hari)`, p.potongan_alpa]] : []),
    [`Telat & pulang cepat (${durasi(r.telat_menit + r.pulang_cepat_menit)}${aturan ? ` × ${rp(aturan.potongan_telat_per_menit)}/menit` : ""})`, p.potongan_telat],
  ];

  return (
    <div className="card p-6 sm:p-8 print:border-0 print:p-0">
      <div className="flex flex-wrap justify-between gap-4 pb-5 border-b border-line">
        <div>
          <p className="text-xs uppercase tracking-wider text-ink-muted">Slip Gaji</p>
          <p className="text-xl font-semibold mt-1">{bulanLabel(bulan)}</p>
          {!ditutup && <p className="text-xs text-amber-700 mt-1">Perkiraan sementara — periode belum ditutup HRD</p>}
        </div>
        <div className="text-sm sm:text-right">
          <p className="font-semibold">{k.nama}</p>
          <p className="text-ink-muted">{k.nik}{k.jabatan ? ` · ${k.jabatan}` : ""}</p>
          <p className="text-ink-muted">
            Gaji {k.tipe_gaji === "per_jam" ? "per jam" : k.tipe_gaji}
            {k.jam_masuk && k.jam_pulang && ` · Jadwal ${k.jam_masuk}–${k.jam_pulang}`}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-3 sm:grid-cols-6 gap-3 py-5 border-b border-line text-center">
        {[
          ["Hari kerja", r.hari_kerja], ["Hadir", r.hadir], ["Izin", r.izin],
          ["Sakit", r.sakit], ["Cuti", r.cuti], ["Alpa", r.alpa],
        ].map(([l, v]) => (
          <div key={l}>
            <p className="text-lg font-semibold tabular-nums">{num(v)}</p>
            <p className="text-xs text-ink-muted">{l}</p>
          </div>
        ))}
      </div>
      
      {isPerjam && r.total_jam_kerja > 0 && (
        <div className="py-4 text-center border-b border-line">
          <p className="text-2xl font-bold text-brand-600 tabular-nums">{r.total_jam_kerja.toFixed(2)} jam</p>
          <p className="text-sm text-ink-muted mt-1">Total jam kerja bulan ini</p>
        </div>
      )}

      <div className="grid sm:grid-cols-2 gap-6 py-5">
        <Rows title="Pendapatan" rows={pendapatan} total={["Total pendapatan", p.total_pendapatan]} />
        <Rows title="Potongan" rows={potongan} total={["Total potongan", p.total_potongan]} negative />
      </div>

      <div className="flex items-center justify-between rounded-lg bg-brand-50 px-5 py-4">
        <span className="font-medium text-brand-900">Gaji bersih diterima</span>
        <span className="text-2xl font-semibold text-brand-700 tabular-nums">{rp(p.gaji_bersih)}</span>
      </div>
      {r.tanpa_absen_pulang > 0 && (
        <p className="text-xs text-amber-700 mt-3">{r.tanpa_absen_pulang} hari tanpa absen pulang: lembur hari itu tidak dihitung.</p>
      )}
    </div>
  );
}

function Rows({ title, rows, total, negative }) {
  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wider text-ink-muted mb-2">{title}</p>
      <div className="flex flex-col">
        {rows.map(([l, v]) => (
          <div key={l} className="flex justify-between gap-4 py-1.5 text-sm">
            <span className="text-ink-muted">{l}</span>
            <span className="tabular-nums">{negative && v > 0 ? `− ${rp(v)}` : rp(v)}</span>
          </div>
        ))}
        <div className="flex justify-between gap-4 pt-2 mt-1 border-t border-line text-sm font-semibold">
          <span>{total[0]}</span>
          <span className="tabular-nums">{rp(total[1])}</span>
        </div>
      </div>
    </div>
  );
}

/** Tabel absensi harian dalam sebulan. */
export function DailyTable({ days, karyawan }) {
  const shown = days.filter((d) => d.status !== "belum" || d.keterangan);
  const isPerjam = karyawan?.tipe_gaji === "per_jam";
  const hasJadwal = karyawan?.jam_masuk && karyawan?.jam_pulang;
  
  return (
    <div className="card overflow-x-auto">
      <table className="w-full">
        <thead className="bg-canvas border-b border-line">
          <tr>
            <th className="th">Tanggal</th>
            <th className="th">Status</th>
            <th className="th">Masuk</th>
            <th className="th">Pulang</th>
            <th className="th">Jam Kerja</th>
            {hasJadwal && <th className="th">Telat</th>}
            {hasJadwal && <th className="th">Pulang cepat</th>}
            {hasJadwal && <th className="th">Lembur</th>}
            <th className="th">Keterangan</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {shown.map((d) => (
            <tr key={d.tanggal} className={d.status === "libur" ? "bg-canvas/60" : ""}>
              <td className="td">{tglPendek(d.tanggal)}</td>
              <td className="td"><Badge status={d.status} /></td>
              <td className="td tabular-nums">{d.jam_masuk || "—"}</td>
              <td className="td tabular-nums">{d.jam_pulang || (d.tanpa_absen_pulang ? <span className="text-amber-700">belum</span> : "—")}</td>
              <td className="td tabular-nums font-medium text-brand-600">{d.jam_kerja ? `${d.jam_kerja.toFixed(2)} jam` : "—"}</td>
              {hasJadwal && <td className={`td tabular-nums ${d.telat_menit ? "text-red-600" : "text-ink-faint"}`}>{d.telat_menit ? durasi(d.telat_menit) : "—"}</td>}
              {hasJadwal && <td className={`td tabular-nums ${d.pulang_cepat_menit ? "text-amber-700" : "text-ink-faint"}`}>{d.pulang_cepat_menit ? durasi(d.pulang_cepat_menit) : "—"}</td>}
              {hasJadwal && <td className={`td tabular-nums ${d.lembur_menit ? "text-emerald-700" : "text-ink-faint"}`}>{d.lembur_menit ? durasi(d.lembur_menit) : "—"}</td>}
              <td className="td text-ink-muted max-w-[260px] truncate" title={d.keterangan}>{d.keterangan || ""}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
