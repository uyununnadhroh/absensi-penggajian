import { useMemo, useState } from "react";
import api from "../../lib/api";
import { rp, useLoad } from "../../lib/format";
import { Alert, Badge, Empty, Field, Loading, Modal, PageHeader } from "../../components/ui";

const EMPTY = {
  nik: "", nama: "", email: "", password: "", jabatan: "", telepon: "", tanggal_masuk: "",
  tipe_gaji: "per_jam", gaji_pokok: 0, tarif_per_jam: 15000, tunjangan_tetap: 0, uang_makan: 0, tarif_lembur_per_jam: 0,
  jam_masuk: "", jam_pulang: "", aktif: true,
};

export default function Karyawan() {
  const { data, error, loading, reload } = useLoad(() => api.employees(), []);
  const [q, setQ] = useState("");
  const [showInactive, setShowInactive] = useState(false);
  const [edit, setEdit] = useState(null);

  const list = useMemo(() => {
    const s = q.toLowerCase();
    return (data || []).filter((e) => (showInactive || e.aktif) && (!s || `${e.nama} ${e.nik} ${e.jabatan} ${e.email}`.toLowerCase().includes(s)));
  }, [data, q, showInactive]);

  return (
    <div>
      <PageHeader title="Karyawan" subtitle="Data karyawan, akun login, jadwal kerja, dan skema gaji.">
        <button className="btn-primary" onClick={() => setEdit({ ...EMPTY })}>+ Tambah karyawan</button>
      </PageHeader>
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <input className="input max-w-xs" placeholder="Cari nama, NIK, jabatan…" value={q} onChange={(e) => setQ(e.target.value)} />
        <label className="flex items-center gap-2 text-sm text-ink-muted">
          <input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} /> Tampilkan nonaktif
        </label>
      </div>
      <Alert>{error}</Alert>
      {loading && !data && <Loading />}
      {data && (
        <div className="card overflow-x-auto">
          {list.length === 0 ? <Empty>{data.length ? "Tidak ada yang cocok." : "Belum ada karyawan."}</Empty> : (
            <table className="w-full">
              <thead className="bg-canvas border-b border-line">
                <tr><th className="th">Karyawan</th><th className="th">Email login</th><th className="th">Jadwal</th><th className="th">Skema gaji</th><th className="th">Uang makan</th><th className="th">Lembur / jam</th><th className="th"></th></tr>
              </thead>
              <tbody className="divide-y divide-line">
                {list.map((e) => (
                  <tr key={e.id} className={e.aktif ? "" : "opacity-60"}>
                    <td className="td"><p className="font-medium">{e.nama} {!e.aktif && <Badge status="ditolak">Nonaktif</Badge>}</p><p className="text-xs text-ink-muted">{e.nik}{e.jabatan ? ` · ${e.jabatan}` : ""}</p></td>
                    <td className="td text-ink-muted">{e.email}</td>
                    <td className="td tabular-nums">{e.jam_masuk && e.jam_pulang ? `${e.jam_masuk}–${e.jam_pulang}` : <span className="text-ink-faint">—</span>}</td>
                    <td className="td tabular-nums">
                      {e.tipe_gaji === "per_jam" ? (
                        <>{rp(e.tarif_per_jam)}<span className="text-ink-muted text-xs"> /jam</span></>
                      ) : (
                        <>{rp(e.gaji_pokok)}<span className="text-ink-muted text-xs"> /{e.tipe_gaji === "harian" ? "hari" : "bulan"}</span></>
                      )}
                    </td>
                    <td className="td tabular-nums">{rp(e.uang_makan)}</td>
                    <td className="td tabular-nums">{rp(e.tarif_lembur_per_jam)}</td>
                    <td className="td text-right"><button className="btn-secondary btn-sm" onClick={() => setEdit({ ...e, password: "" })}>Ubah</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
      {edit && <EmployeeForm initial={edit} onClose={() => setEdit(null)} onSaved={() => { setEdit(null); reload(); }} />}
    </div>
  );
}

function EmployeeForm({ initial, onClose, onSaved }) {
  const isNew = !initial.id;
  const [f, setF] = useState(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const set = (k, numeric) => (e) => setF({ ...f, [k]: numeric ? Number(e.target.value) : e.target.value });

  async function submit(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      if (isNew) await api.createEmployee(f);
      else await api.updateEmployee(f.id, f);
      onSaved();
    } catch (err) {
      setError(err.message);
      setBusy(false);
    }
  }

  const money = (k, label, hint) => (
    <Field label={label} hint={hint}>
      <div className="relative">
        <span className="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-ink-faint">Rp</span>
        <input type="number" min={0} step={1000} className="input pl-9" value={f[k]} onChange={set(k, true)} />
      </div>
    </Field>
  );

  return (
    <Modal open wide title={isNew ? "Tambah karyawan" : `Ubah ${initial.nama}`} onClose={onClose}>
      <form onSubmit={submit} className="flex flex-col gap-6">
        <section className="grid sm:grid-cols-2 gap-4">
          <Field label="Nama lengkap"><input className="input" required value={f.nama} onChange={set("nama")} /></Field>
          <Field label="NIK / nomor karyawan"><input className="input" required value={f.nik} onChange={set("nik")} /></Field>
          <Field label="Jabatan"><input className="input" value={f.jabatan} onChange={set("jabatan")} /></Field>
          <Field label="No. telepon"><input className="input" value={f.telepon} onChange={set("telepon")} /></Field>
          <Field label="Tanggal mulai bekerja" hint="Hari sebelum tanggal ini tidak dihitung alpa; gaji bulan pertama prorata."><input type="date" className="input" value={f.tanggal_masuk} onChange={set("tanggal_masuk")} /></Field>
        </section>

        <section>
          <p className="text-sm font-semibold mb-3">Akun login</p>
          <div className="grid sm:grid-cols-2 gap-4">
            <Field label="Email"><input type="email" className="input" required value={f.email} onChange={set("email")} /></Field>
            <Field label={isNew ? "Kata sandi awal" : "Reset kata sandi"} hint={isNew ? "Minimal 8 karakter. Berikan ke karyawan." : "Kosongkan bila tidak diganti."}>
              <input type="text" className="input" required={isNew} minLength={8} value={f.password} onChange={set("password")} autoComplete="new-password" />
            </Field>
          </div>
        </section>

        {f.tipe_gaji !== "per_jam" && (
          <section>
            <p className="text-sm font-semibold mb-3">Jadwal kerja</p>
            <p className="text-xs text-ink-muted mb-3">Untuk sistem per jam, jadwal tidak diperlukan. Jam kerja dihitung dari absen masuk sampai pulang.</p>
            <div className="grid grid-cols-2 gap-4">
              <Field label="Jam masuk"><input type="time" className="input" value={f.jam_masuk} onChange={set("jam_masuk")} /></Field>
              <Field label="Jam pulang" hint="Lebih kecil dari jam masuk = shift malam."><input type="time" className="input" value={f.jam_pulang} onChange={set("jam_pulang")} /></Field>
            </div>
          </section>
        )}

        <section>
          <p className="text-sm font-semibold mb-3">Skema gaji</p>
          <div className="grid grid-cols-3 gap-2 mb-4 max-w-2xl">
            {[["per_jam", "Per Jam"], ["harian", "Harian"], ["bulanan", "Bulanan"]].map(([k, l]) => (
              <button type="button" key={k} onClick={() => setF({ ...f, tipe_gaji: k })}
                className={`h-10 rounded-lg border text-sm ${f.tipe_gaji === k ? "border-brand-500 bg-brand-50 text-brand-700 font-medium" : "border-line"}`}>{l}</button>
            ))}
          </div>
          <div className="grid sm:grid-cols-2 gap-4">
            {f.tipe_gaji === "per_jam" ? (
              money("tarif_per_jam", "Tarif per jam", "Gaji = Total jam kerja × Tarif per jam")
            ) : (
              money("gaji_pokok", f.tipe_gaji === "harian" ? "Upah per hari hadir" : "Gaji pokok per bulan",
                f.tipe_gaji === "harian" ? "Dikali jumlah hari hadir." : "Dipotong per hari alpa (gaji ÷ hari kerja sebulan).")
            )}
            {money("tunjangan_tetap", "Tunjangan tetap per bulan")}
            {money("uang_makan", "Uang makan / transport per hari hadir")}
            {money("tarif_lembur_per_jam", "Tarif lembur per jam")}
          </div>
        </section>

        {!isNew && (
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={f.aktif} onChange={(e) => setF({ ...f, aktif: e.target.checked })} />
            Karyawan aktif <span className="text-ink-muted">(nonaktif = tidak bisa login dan tidak masuk rekap gaji berikutnya)</span>
          </label>
        )}

        <Alert>{error}</Alert>
        <div className="flex justify-end gap-2">
          <button type="button" className="btn-secondary" onClick={onClose}>Batal</button>
          <button className="btn-primary" disabled={busy}>{busy ? "Menyimpan…" : "Simpan"}</button>
        </div>
      </form>
    </Modal>
  );
}
