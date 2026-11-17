import { useState } from "react";
import api from "../lib/api";
import { bulanLabel, durasi, thisMonth, useLoad } from "../lib/format";
import { Alert, Loading, MonthPicker, PageHeader, Stat } from "../components/ui";
import { DailyTable } from "../components/Slip";

export default function Riwayat() {
  const [bulan, setBulan] = useState(thisMonth());
  const { data, error, loading } = useLoad(() => api.myPayroll(bulan), [bulan]);
  const slip = data?.slip;
  const r = slip?.rekap;

  return (
    <div>
      <PageHeader title="Riwayat Absensi" subtitle={bulanLabel(bulan)}>
        <MonthPicker value={bulan} onChange={setBulan} />
      </PageHeader>
      <Alert>{error}</Alert>
      {loading && !data && <Loading />}
      {data && !slip && <Alert kind="info">Belum ada data absensi untuk bulan ini.</Alert>}
      {r && (
        <div className="flex flex-col gap-5">
          <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-3">
            <Stat label="Hari kerja" value={r.hari_kerja} />
            <Stat label="Hadir" value={r.hadir} />
            <Stat label="Izin" value={r.izin} />
            <Stat label="Sakit" value={r.sakit} />
            <Stat label="Cuti" value={r.cuti} />
            <Stat label="Alpa" value={r.alpa} tone={r.alpa ? "red" : undefined} />
            <Stat label="Lembur" value={durasi(r.lembur_menit)} tone={r.lembur_menit ? "green" : undefined} />
          </div>
          {(r.telat_menit > 0 || r.pulang_cepat_menit > 0) && (
            <Alert kind="warn">
              Telat {r.telat} kali (total {durasi(r.telat_menit)}){r.pulang_cepat_menit ? `, pulang cepat total ${durasi(r.pulang_cepat_menit)}` : ""}.
            </Alert>
          )}
          <DailyTable days={slip.harian} karyawan={slip.karyawan} />
        </div>
      )}
    </div>
  );
}
