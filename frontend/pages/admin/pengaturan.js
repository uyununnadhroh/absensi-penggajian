import { useEffect, useState } from "react";
import api from "../../lib/api";
import { HARI, tglLabel, useLoad } from "../../lib/format";
import { Alert, Empty, Field, Loading, PageHeader } from "../../components/ui";

const FIELDS = [
  ["toleransi_telat_menit", "Toleransi telat (menit)", "Masuk sampai jadwal + toleransi belum dihitung telat. Lewat dari itu, telat dihitung penuh dari jam jadwal."],
  ["potongan_telat_per_menit", "Potongan telat & pulang cepat (Rp per menit)", "Isi 0 bila tidak ada potongan."],
  ["hari_kerja_sebulan", "Hari kerja sebulan (pembagi gaji)", "Potongan 1 hari alpa = gaji pokok bulanan ÷ angka ini."],
  ["lembur_minimal_menit", "Lembur dihitung per (menit)", "Contoh 30: lembur 75 menit dibayar 60 menit."],
  ["awal_absen_menit", "Absen masuk dibuka (menit sebelum jadwal)", "Mencegah absen terlalu awal."],
];

export default function Pengaturan() {
  return (
    <div className="max-w-3xl">
      <PageHeader title="Pengaturan" subtitle="Aturan absensi, perhitungan gaji, dan hari libur." />
      <div className="flex flex-col gap-5">
        <Aturan />
        <HariLibur />
      </div>
    </div>
  );
}

function Aturan() {
  const { data, error, loading } = useLoad(() => api.settings(), []);
  const [f, setF] = useState(null);
  const [msg, setMsg] = useState({ kind: "", text: "" });
  const [busy, setBusy] = useState(false);
  useEffect(() => { if (data) setF(data); }, [data]);

  const toggleDay = (d) => {
    const set = new Set(f.hari_libur_mingguan);
    set.has(d) ? set.delete(d) : set.add(d);
    setF({ ...f, hari_libur_mingguan: [...set].sort() });
  };

  async function save(e) {
    e.preventDefault();
    setBusy(true);
    setMsg({ kind: "", text: "" });
    try {
      const body = { hari_libur_mingguan: f.hari_libur_mingguan };
      FIELDS.forEach(([k]) => { body[k] = Number(f[k]); });
      setF(await api.saveSettings(body));
      setMsg({ kind: "ok", text: "Pengaturan disimpan. Berlaku untuk perhitungan periode yang belum ditutup." });
    } catch (err) {
      setMsg({ kind: "error", text: err.message });
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={save} className="card p-5 flex flex-col gap-5">
      <h2 className="font-semibold">Aturan absensi &amp; gaji</h2>
      <Alert>{error}</Alert>
      {loading && !f && <Loading />}
      {f && (
        <>
          <div className="grid sm:grid-cols-2 gap-4">
            {FIELDS.map(([k, l, hint]) => (
              <Field key={k} label={l} hint={hint}>
                <input type="number" min={0} className="input" required value={f[k]} onChange={(e) => setF({ ...f, [k]: e.target.value })} />
              </Field>
            ))}
          </div>
          <Field label="Hari libur rutin setiap minggu">
            <div className="flex flex-wrap gap-2">
              {HARI.map((h, i) => (
                <button type="button" key={h} onClick={() => toggleDay(i)}
                  className={`h-9 px-3 rounded-lg border text-sm ${f.hari_libur_mingguan.includes(i) ? "border-brand-500 bg-brand-50 text-brand-700 font-medium" : "border-line"}`}>{h}</button>
              ))}
            </div>
          </Field>
          <Alert kind={msg.kind || "error"}>{msg.text}</Alert>
          <button className="btn-primary self-start" disabled={busy}>{busy ? "Menyimpan…" : "Simpan aturan"}</button>
        </>
      )}
    </form>
  );
}

function HariLibur() {
  const [tahun, setTahun] = useState(new Date().getFullYear());
  const { data, error, loading, reload } = useLoad(() => api.holidays(tahun), [tahun]);
  const [form, setForm] = useState({ tanggal: "", keterangan: "" });
  const [msg, setMsg] = useState("");

  async function add(e) {
    e.preventDefault();
    setMsg("");
    try {
      await api.saveHoliday(form);
      setForm({ tanggal: "", keterangan: "" });
      reload();
    } catch (err) {
      setMsg(err.message);
    }
  }

  async function remove(t) {
    if (!confirm(`Hapus hari libur ${tglLabel(t)}?`)) return;
    try {
      await api.deleteHoliday(t);
      reload();
    } catch (err) {
      setMsg(err.message);
    }
  }

  return (
    <div className="card">
      <div className="px-5 py-4 border-b border-line flex items-center justify-between gap-3">
        <div>
          <h2 className="font-semibold">Hari libur nasional &amp; perusahaan</h2>
          <p className="text-xs text-ink-muted mt-0.5">Tidak dihitung hari kerja, jadi tidak ada alpa.</p>
        </div>
        <select className="input w-28" value={tahun} onChange={(e) => setTahun(Number(e.target.value))}>
          {[-1, 0, 1].map((d) => { const y = new Date().getFullYear() + d; return <option key={y} value={y}>{y}</option>; })}
        </select>
      </div>
      <form onSubmit={add} className="px-5 py-4 border-b border-line grid sm:grid-cols-[180px_1fr_auto] gap-3 items-end">
        <Field label="Tanggal"><input type="date" className="input" required value={form.tanggal} onChange={(e) => setForm({ ...form, tanggal: e.target.value })} /></Field>
        <Field label="Keterangan"><input className="input" required maxLength={120} placeholder="mis. Hari Raya Idul Fitri" value={form.keterangan} onChange={(e) => setForm({ ...form, keterangan: e.target.value })} /></Field>
        <button className="btn-primary">Tambah</button>
      </form>
      <div className="px-5 pt-3"><Alert>{error || msg}</Alert></div>
      {loading && !data && <div className="px-5"><Loading /></div>}
      {data?.length === 0 && <Empty>Belum ada hari libur di {tahun}.</Empty>}
      <ul className="divide-y divide-line">
        {data?.map((h) => (
          <li key={h.tanggal} className="px-5 py-3 flex items-center justify-between gap-3 text-sm">
            <div><p className="font-medium">{h.keterangan}</p><p className="text-ink-muted text-xs">{tglLabel(h.tanggal)}</p></div>
            <button className="btn-danger btn-sm" onClick={() => remove(h.tanggal)}>Hapus</button>
          </li>
        ))}
      </ul>
    </div>
  );
}
