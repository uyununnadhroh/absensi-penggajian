import { useState } from "react";
import api from "../../lib/api";
import { durasi, jam, shiftDay, tglLabel, useLoad, ymd } from "../../lib/format";
import { Alert, Badge, Empty, Field, Loading, Modal, PageHeader } from "../../components/ui";

const hhmm = (iso) => (iso ? new Date(iso).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit" }) : "");

export default function DataAbsensi() {
  const [tanggal, setTanggal] = useState(ymd());
  const { data, error, loading, reload } = useLoad(() => api.attendance(tanggal), [tanggal]);
  const [edit, setEdit] = useState(null);

  return (
    <div>
      <PageHeader title="Data Absensi" subtitle={`${tglLabel(tanggal)}${data?.libur ? ` · Libur: ${data.libur}` : data && !data.hari_kerja ? " · Hari libur" : ""}`}>
        <button className="btn-secondary w-10 px-0" onClick={() => setTanggal(shiftDay(tanggal, -1))} aria-label="Hari sebelumnya">‹</button>
        <input type="date" className="input w-44" max={ymd()} value={tanggal} onChange={(e) => e.target.value && setTanggal(e.target.value)} />
        <button className="btn-secondary w-10 px-0" disabled={tanggal >= ymd()} onClick={() => setTanggal(shiftDay(tanggal, 1))} aria-label="Hari berikutnya">›</button>
      </PageHeader>
      <Alert>{error}</Alert>
      {data?.ditutup && <div className="mb-4"><Alert kind="info">Periode gaji bulan ini sudah ditutup. Absensi terkunci; buka kembali periode di menu Penggajian untuk mengubah.</Alert></div>}
      {loading && !data && <Loading />}
      {data && (
        <div className="card overflow-x-auto">
          {data.rows.length === 0 ? <Empty>Tidak ada karyawan aktif pada tanggal ini.</Empty> : (
            <table className="w-full">
              <thead className="bg-canvas border-b border-line">
                <tr>
                  <th className="th">Karyawan</th><th className="th">Status</th><th className="th">Masuk</th><th className="th">Pulang</th>
                  <th className="th">Telat</th><th className="th">Pulang cepat</th><th className="th">Lembur</th><th className="th">Catatan</th><th className="th"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {data.rows.map((row) => {
                  const { karyawan: k, absensi: a, hasil: h } = row;
                  return (
                    <tr key={k.id}>
                      <td className="td"><p className="font-medium">{k.nama}</p><p className="text-xs text-ink-muted">{k.nik} · {k.jam_masuk}–{k.jam_pulang}</p></td>
                      <td className="td"><Badge status={h.status} /></td>
                      <td className="td tabular-nums">{jam(a?.jam_masuk)}</td>
                      <td className="td tabular-nums">{a?.jam_pulang ? jam(a.jam_pulang) : h.tanpa_absen_pulang ? <span className="text-amber-700">belum</span> : "—"}</td>
                      <td className={`td tabular-nums ${h.telat_menit ? "text-red-600" : "text-ink-faint"}`}>{h.telat_menit ? durasi(h.telat_menit) : "—"}</td>
                      <td className={`td tabular-nums ${h.pulang_cepat_menit ? "text-amber-700" : "text-ink-faint"}`}>{h.pulang_cepat_menit ? durasi(h.pulang_cepat_menit) : "—"}</td>
                      <td className={`td tabular-nums ${h.lembur_menit ? "text-emerald-700" : "text-ink-faint"}`}>{h.lembur_menit ? durasi(h.lembur_menit) : "—"}</td>
                      <td className="td text-ink-muted max-w-[200px] truncate" title={a?.catatan}>
                        {a?.catatan}{a?.sumber === "admin" && <span className="text-xs text-ink-faint"> (diubah admin)</span>}
                      </td>
                      <td className="td text-right">
                        {!data.ditutup && <button className="btn-secondary btn-sm" onClick={() => setEdit(row)}>Ubah</button>}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>
      )}
      {edit && <EditModal row={edit} tanggal={tanggal} onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload(); }} />}
    </div>
  );
}

function EditModal({ row, tanggal, onClose, onSaved }) {
  const a = row.absensi;
  const [f, setF] = useState({
    status: a?.status || "hadir",
    jam_masuk: hhmm(a?.jam_masuk) || row.karyawan.jam_masuk,
    jam_pulang: hhmm(a?.jam_pulang) || (a ? "" : row.karyawan.jam_pulang),
    catatan: a?.catatan || "",
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const set = (k) => (e) => setF({ ...f, [k]: e.target.value });

  async function save(status) {
    setBusy(true);
    setError("");
    try {
      await api.saveAttendance({ employee_id: row.karyawan.id, tanggal, ...f, status });
      onSaved();
    } catch (err) {
      setError(err.message);
      setBusy(false);
    }
  }

  return (
    <Modal open title={`${row.karyawan.nama} · ${tglLabel(tanggal, { day: "numeric", month: "long", year: "numeric" })}`} onClose={onClose}>
      <form onSubmit={(e) => { e.preventDefault(); save(f.status); }} className="flex flex-col gap-4">
        <Field label="Status">
          <select className="input" value={f.status} onChange={set("status")}>
            <option value="hadir">Hadir</option><option value="izin">Izin</option><option value="sakit">Sakit</option>
            <option value="cuti">Cuti</option><option value="alpa">Alpa</option>
          </select>
        </Field>
        {f.status === "hadir" && (
          <div className="grid grid-cols-2 gap-3">
            <Field label="Jam masuk"><input type="time" className="input" required value={f.jam_masuk} onChange={set("jam_masuk")} /></Field>
            <Field label="Jam pulang" hint="Kosongkan bila belum pulang."><input type="time" className="input" value={f.jam_pulang} onChange={set("jam_pulang")} /></Field>
          </div>
        )}
        <Field label="Catatan (alasan koreksi)"><input className="input" maxLength={255} value={f.catatan} onChange={set("catatan")} /></Field>
        <Alert>{error}</Alert>
        <div className="flex flex-wrap justify-between gap-2">
          {a ? <button type="button" className="btn-danger" disabled={busy} onClick={() => confirm("Hapus catatan absensi hari ini? Hari kerja tanpa absensi dihitung alpa.") && save("")}>Hapus catatan</button> : <span />}
          <div className="flex gap-2">
            <button type="button" className="btn-secondary" onClick={onClose}>Batal</button>
            <button className="btn-primary" disabled={busy}>{busy ? "Menyimpan…" : "Simpan"}</button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
