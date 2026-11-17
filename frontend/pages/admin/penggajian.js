import { useState } from "react";
import api from "../../lib/api";
import { bulanLabel, downloadBlob, durasi, rp, shiftMonth, thisMonth, useLoad } from "../../lib/format";
import { Alert, Empty, Loading, Modal, MonthPicker, PageHeader, Stat } from "../../components/ui";
import { DailyTable, Slip } from "../../components/Slip";

export default function Penggajian() {
  const [bulan, setBulan] = useState(shiftMonth(thisMonth(), -1)); // biasanya gaji bulan lalu yang diproses
  const { data, error, loading, reload } = useLoad(() => api.payroll(bulan), [bulan]);
  const [detail, setDetail] = useState(null);
  const [msg, setMsg] = useState({ kind: "", text: "" });
  const [busy, setBusy] = useState(false);
  const selesai = shiftMonth(bulan, 1) <= thisMonth();

  async function run(fn, okText) {
    setBusy(true);
    setMsg({ kind: "", text: "" });
    try {
      await fn();
      if (okText) setMsg({ kind: "ok", text: okText });
      reload();
    } catch (err) {
      setMsg({ kind: "error", text: err.message });
    } finally {
      setBusy(false);
    }
  }

  const close = () =>
    confirm(`Tutup periode ${bulanLabel(bulan)}? Hasil gaji disimpan dan absensi bulan ini dikunci.`) &&
    run(() => api.closePayroll(bulan), `Periode ${bulanLabel(bulan)} ditutup.`);
  const reopen = () =>
    confirm(`Buka kembali periode ${bulanLabel(bulan)}? Gaji akan dihitung ulang dengan data karyawan terbaru.`) &&
    run(() => api.reopenPayroll(bulan), `Periode ${bulanLabel(bulan)} dibuka kembali.`);
  const exportCsv = () => run(async () => downloadBlob(await api.exportPayroll(bulan), `gaji-${bulan}.csv`));

  const t = data?.total;

  return (
    <div>
      <PageHeader title="Penggajian" subtitle={bulanLabel(bulan)}>
        <MonthPicker value={bulan} onChange={setBulan} />
        <button className="btn-secondary" disabled={busy || !data?.karyawan?.length} onClick={exportCsv}>Unduh Excel (CSV)</button>
        {data && (data.ditutup
          ? <button className="btn-secondary" disabled={busy} onClick={reopen}>Buka kembali</button>
          : <button className="btn-primary" disabled={busy || !selesai} title={selesai ? "" : "Bisa ditutup setelah bulan berakhir"} onClick={close}>Tutup periode</button>)}
      </PageHeader>
      <div className="flex flex-col gap-3 mb-5">
        <Alert>{error}</Alert>
        <Alert kind={msg.kind || "error"}>{msg.text}</Alert>
        {data && (data.ditutup
          ? <Alert kind="ok">Periode sudah ditutup pada {new Date(data.ditutup_pada).toLocaleString("id-ID")}. Angka di bawah adalah hasil final.</Alert>
          : <Alert kind="info">Perhitungan sementara (langsung dari data absensi). {selesai ? "Tutup periode setelah semua data dicek supaya angka final dan absensi terkunci." : "Bulan ini masih berjalan."}</Alert>)}
      </div>
      {loading && !data && <Loading />}
      {data && (
        <div className="flex flex-col gap-5">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
            <Stat label="Karyawan" value={data.karyawan.length} />
            <Stat label="Total pendapatan" value={rp(t.total_pendapatan)} />
            <Stat label="Total potongan" value={rp(t.total_potongan)} tone={t.total_potongan ? "red" : undefined} />
            <Stat label="Total gaji dibayar" value={rp(t.gaji_bersih)} tone="green" />
          </div>
          <div className="card overflow-x-auto">
            {data.karyawan.length === 0 ? <Empty>Tidak ada karyawan pada periode ini.</Empty> : (
              <table className="w-full">
                <thead className="bg-canvas border-b border-line">
                  <tr>
                    <th className="th">Karyawan</th><th className="th text-right">Hadir</th><th className="th text-right">Izin/Sakit/Cuti</th>
                    <th className="th text-right">Alpa</th><th className="th text-right">Telat</th><th className="th text-right">Lembur</th>
                    <th className="th text-right">Pendapatan</th><th className="th text-right">Potongan</th><th className="th text-right">Gaji bersih</th><th className="th"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {data.karyawan.map((s) => {
                    const { karyawan: k, rekap: r, rincian: p } = s;
                    return (
                      <tr key={k.id} className="hover:bg-canvas/60">
                        <td className="td"><p className="font-medium">{k.nama}</p><p className="text-xs text-ink-muted">{k.nik} · gaji {k.tipe_gaji}</p></td>
                        <td className="td text-right tabular-nums">{r.hadir}/{r.hari_kerja}</td>
                        <td className="td text-right tabular-nums">{r.izin + r.sakit + r.cuti}</td>
                        <td className={`td text-right tabular-nums ${r.alpa ? "text-red-600 font-medium" : ""}`}>{r.alpa}</td>
                        <td className={`td text-right tabular-nums ${r.telat_menit ? "text-amber-700" : "text-ink-faint"}`}>{r.telat_menit ? durasi(r.telat_menit) : "—"}</td>
                        <td className={`td text-right tabular-nums ${r.lembur_menit ? "text-emerald-700" : "text-ink-faint"}`}>{r.lembur_menit ? durasi(r.lembur_menit) : "—"}</td>
                        <td className="td text-right tabular-nums">{rp(p.total_pendapatan)}</td>
                        <td className="td text-right tabular-nums text-red-600">{p.total_potongan ? `− ${rp(p.total_potongan)}` : "—"}</td>
                        <td className="td text-right tabular-nums font-semibold">{rp(p.gaji_bersih)}</td>
                        <td className="td text-right"><button className="btn-secondary btn-sm" onClick={() => setDetail(s)}>Slip</button></td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
          <p className="text-xs text-ink-faint">
            Aturan: toleransi telat {data.aturan.toleransi_telat_menit} menit · potongan telat/pulang cepat {rp(data.aturan.potongan_telat_per_menit)}/menit ·
            potongan alpa = gaji pokok ÷ {data.aturan.hari_kerja_sebulan} hari · lembur dihitung per {data.aturan.lembur_minimal_menit} menit.
          </p>
        </div>
      )}
      {detail && (
        <Modal open wide title={`Slip gaji ${detail.karyawan.nama}`} onClose={() => setDetail(null)}>
          <PrintableSlip slip={detail} data={data} />
        </Modal>
      )}
    </div>
  );
}

function PrintableSlip({ slip, data }) {
  const [tab, setTab] = useState("slip");
  const print = () => {
    const w = window.open("", "_blank", "width=900,height=1000");
    if (!w) return;
    const el = document.getElementById("slip-print");
    const css = [...document.querySelectorAll("style, link[rel=stylesheet]")].map((n) => n.outerHTML).join("");
    w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>Slip ${slip.karyawan.nama} ${data.bulan}</title>${css}</head><body style="padding:24px;background:#fff">${el.outerHTML}</body></html>`);
    w.document.close();
    setTimeout(() => { w.focus(); w.print(); }, 400);
  };
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex gap-1">
          {[["slip", "Slip gaji"], ["harian", "Absensi harian"]].map(([k, l]) => (
            <button key={k} onClick={() => setTab(k)} className={`px-3 h-8 rounded-md text-sm ${tab === k ? "bg-brand-50 text-brand-700 font-medium" : "text-ink-muted"}`}>{l}</button>
          ))}
        </div>
        {tab === "slip" && <button className="btn-secondary btn-sm" onClick={print}>Cetak / PDF</button>}
      </div>
      {tab === "slip"
        ? <div id="slip-print"><Slip slip={slip} bulan={data.bulan} ditutup={data.ditutup} aturan={data.aturan} /></div>
        : <DailyTable days={slip.harian} karyawan={slip.karyawan} />}
    </div>
  );
}
