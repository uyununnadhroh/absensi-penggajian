import { useState } from "react";
import api from "../../lib/api";
import { tglPendek, useLoad } from "../../lib/format";
import { Alert, Badge, Empty, Loading, PageHeader } from "../../components/ui";

const TABS = [["menunggu", "Menunggu"], ["disetujui", "Disetujui"], ["ditolak", "Ditolak"], ["", "Semua"]];
const JENIS = { izin: "Izin", sakit: "Sakit", cuti: "Cuti" };

export default function PengajuanAdmin() {
  const [tab, setTab] = useState("menunggu");
  const { data, error, loading, reload } = useLoad(() => api.leaves(tab), [tab]);
  const [busyId, setBusyId] = useState(0);
  const [msg, setMsg] = useState({ kind: "", text: "" });

  async function decide(l, action) {
    let catatan = "";
    if (action === "tolak") {
      catatan = prompt("Alasan menolak (opsional):", "") ?? null;
      if (catatan === null) return;
    }
    setBusyId(l.id);
    setMsg({ kind: "", text: "" });
    try {
      const res = await api.decideLeave(l.id, action, catatan);
      setMsg({ kind: "ok", text: action === "setujui"
        ? `Pengajuan ${l.nama} disetujui; ${res.hari_dicatat} hari kerja dicatat sebagai ${l.jenis}.`
        : `Pengajuan ${l.nama} ditolak.` });
      reload();
    } catch (err) {
      setMsg({ kind: "error", text: err.message });
    } finally {
      setBusyId(0);
    }
  }

  return (
    <div>
      <PageHeader title="Pengajuan Izin, Sakit & Cuti" subtitle="Hari yang disetujui otomatis tercatat di absensi (hari libur dilewati)." />
      <div className="flex gap-1 mb-4 border-b border-line">
        {TABS.map(([k, l]) => (
          <button key={k} onClick={() => setTab(k)}
            className={`px-4 h-10 text-sm -mb-px border-b-2 ${tab === k ? "border-brand-500 text-brand-700 font-medium" : "border-transparent text-ink-muted"}`}>{l}</button>
        ))}
      </div>
      <div className="flex flex-col gap-3 mb-4">
        <Alert>{error}</Alert>
        <Alert kind={msg.kind || "error"}>{msg.text}</Alert>
      </div>
      {loading && !data && <Loading />}
      {data && (
        <div className="card">
          {data.length === 0 ? <Empty>Tidak ada pengajuan.</Empty> : (
            <ul className="divide-y divide-line">
              {data.map((l) => (
                <li key={l.id} className="px-5 py-4 flex flex-col md:flex-row md:items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">{l.nama}</span>
                      <span className="text-xs text-ink-muted">{l.jabatan}</span>
                      <Badge status={l.status} />
                    </div>
                    <p className="text-sm mt-1">
                      <span className="font-medium">{JENIS[l.jenis]}</span> · {tglPendek(l.tanggal_mulai)}
                      {l.tanggal_selesai !== l.tanggal_mulai ? ` – ${tglPendek(l.tanggal_selesai)}` : ""}
                    </p>
                    <p className="text-sm text-ink-muted mt-0.5">{l.alasan}</p>
                    <p className="text-xs text-ink-faint mt-1">Diajukan {l.dibuat_pada}{l.diputus_pada ? ` · diputuskan ${l.diputus_pada}` : ""}{l.catatan_admin ? ` · ${l.catatan_admin}` : ""}</p>
                  </div>
                  {l.status === "menunggu" && (
                    <div className="flex gap-2 shrink-0">
                      <button className="btn-danger btn-sm" disabled={busyId === l.id} onClick={() => decide(l, "tolak")}>Tolak</button>
                      <button className="btn-primary btn-sm" disabled={busyId === l.id} onClick={() => decide(l, "setujui")}>Setujui</button>
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
