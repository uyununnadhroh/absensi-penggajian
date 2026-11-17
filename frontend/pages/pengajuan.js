import { useState } from "react";
import api from "../lib/api";
import { tglPendek, useLoad, ymd } from "../lib/format";
import { Alert, Badge, Empty, Field, Loading, PageHeader } from "../components/ui";

const JENIS = { izin: "Izin", sakit: "Sakit", cuti: "Cuti" };

export default function Pengajuan() {
  const list = useLoad(() => api.myLeaves(), []);
  const [form, setForm] = useState({ jenis: "izin", tanggal_mulai: ymd(), tanggal_selesai: ymd(), alasan: "" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value, ...(k === "tanggal_mulai" && f.tanggal_selesai < e.target.value ? { tanggal_selesai: e.target.value } : {}) }));

  async function submit(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setDone("");
    try {
      await api.createLeave(form);
      setDone("Pengajuan terkirim dan menunggu persetujuan HRD.");
      setForm((f) => ({ ...f, alasan: "" }));
      list.reload();
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }

  async function cancel(id) {
    if (!confirm("Batalkan pengajuan ini?")) return;
    try {
      await api.cancelLeave(id);
      list.reload();
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div>
      <PageHeader title="Izin, Sakit & Cuti" subtitle="Hari yang disetujui tidak dihitung alpa." />
      <div className="grid lg:grid-cols-5 gap-5 items-start">
        <form onSubmit={submit} className="card p-5 lg:col-span-2 flex flex-col gap-4">
          <h2 className="font-semibold">Buat pengajuan</h2>
          <Field label="Jenis">
            <div className="grid grid-cols-3 gap-2">
              {Object.entries(JENIS).map(([k, l]) => (
                <button type="button" key={k} onClick={() => setForm((f) => ({ ...f, jenis: k }))}
                  className={`h-10 rounded-lg border text-sm ${form.jenis === k ? "border-brand-500 bg-brand-50 text-brand-700 font-medium" : "border-line"}`}>
                  {l}
                </button>
              ))}
            </div>
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Dari tanggal"><input type="date" className="input" required value={form.tanggal_mulai} onChange={set("tanggal_mulai")} /></Field>
            <Field label="Sampai tanggal"><input type="date" className="input" required min={form.tanggal_mulai} value={form.tanggal_selesai} onChange={set("tanggal_selesai")} /></Field>
          </div>
          <Field label="Alasan" hint={form.jenis === "sakit" ? "Simpan surat dokter untuk diserahkan ke HRD." : ""}>
            <textarea className="input h-24 py-2" required maxLength={255} value={form.alasan} onChange={set("alasan")} />
          </Field>
          <Alert>{error}</Alert>
          <Alert kind="ok">{done}</Alert>
          <button className="btn-primary" disabled={busy}>{busy ? "Mengirim…" : "Kirim pengajuan"}</button>
        </form>

        <div className="card lg:col-span-3">
          <div className="px-5 py-4 border-b border-line font-semibold">Pengajuan saya</div>
          {list.loading && !list.data && <div className="px-5"><Loading /></div>}
          <div className="px-5"><Alert>{list.error}</Alert></div>
          {list.data?.length === 0 && <Empty>Belum ada pengajuan.</Empty>}
          <ul className="divide-y divide-line">
            {list.data?.map((l) => (
              <li key={l.id} className="px-5 py-4 flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{JENIS[l.jenis]}</span>
                    <Badge status={l.status} />
                  </div>
                  <p className="text-sm text-ink-muted mt-0.5">
                    {tglPendek(l.tanggal_mulai)}{l.tanggal_selesai !== l.tanggal_mulai ? ` – ${tglPendek(l.tanggal_selesai)}` : ""} · {l.alasan}
                  </p>
                  {l.catatan_admin && <p className="text-xs text-ink-muted mt-1">Catatan HRD: {l.catatan_admin}</p>}
                </div>
                {l.status === "menunggu" && (
                  <button className="btn-danger btn-sm self-start" onClick={() => cancel(l.id)}>Batalkan</button>
                )}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}
