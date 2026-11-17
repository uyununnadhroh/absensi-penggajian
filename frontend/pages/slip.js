import { useState } from "react";
import api from "../lib/api";
import { bulanLabel, thisMonth, useLoad } from "../lib/format";
import { Alert, Loading, MonthPicker, PageHeader } from "../components/ui";
import { Slip } from "../components/Slip";

export default function SlipGaji() {
  const [bulan, setBulan] = useState(thisMonth());
  const { data, error, loading } = useLoad(() => api.myPayroll(bulan), [bulan]);

  return (
    <div>
      <PageHeader title="Slip Gaji" subtitle={bulanLabel(bulan)}>
        <MonthPicker value={bulan} onChange={setBulan} />
        {data?.slip && <button className="btn-secondary" onClick={() => window.print()}>Cetak / PDF</button>}
      </PageHeader>
      <Alert>{error}</Alert>
      {loading && !data && <Loading />}
      {data && !data.slip && <Alert kind="info">Belum ada slip gaji untuk bulan ini.</Alert>}
      {data?.slip && <Slip slip={data.slip} bulan={data.bulan} ditutup={data.ditutup} aturan={data.aturan} />}
    </div>
  );
}
