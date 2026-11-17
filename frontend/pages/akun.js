import { useState } from "react";
import api from "../lib/api";
import { useLoad } from "../lib/format";
import { Alert, Field, PageHeader } from "../components/ui";

export default function Akun({ user }) {
  const me = useLoad(() => api.me(), []);
  const [lama, setLama] = useState("");
  const [baru, setBaru] = useState("");
  const [ulang, setUlang] = useState("");
  const [msg, setMsg] = useState({ kind: "", text: "" });
  const [busy, setBusy] = useState(false);
  const e = me.data?.employee;

  async function submit(ev) {
    ev.preventDefault();
    if (baru !== ulang) {
      setMsg({ kind: "error", text: "Konfirmasi kata sandi baru tidak sama." });
      return;
    }
    setBusy(true);
    try {
      await api.changePassword(lama, baru);
      setMsg({ kind: "ok", text: "Kata sandi berhasil diganti." });
      setLama(""); setBaru(""); setUlang("");
    } catch (err) {
      setMsg({ kind: "error", text: err.message });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="max-w-2xl">
      <PageHeader title="Akun Saya" />
      <div className="card p-5 mb-5 grid sm:grid-cols-2 gap-4 text-sm">
        <Info label="Nama" value={user.nama} />
        <Info label="Email" value={user.email} />
        {e && (
          <>
            <Info label="NIK" value={e.nik} />
            <Info label="Jabatan" value={e.jabatan || "—"} />
            <Info label="Jadwal kerja" value={`${e.jam_masuk} – ${e.jam_pulang}`} />
            <Info label="Tipe gaji" value={e.tipe_gaji === "harian" ? "Harian" : "Bulanan"} />
          </>
        )}
      </div>
      <form onSubmit={submit} className="card p-5 flex flex-col gap-4">
        <h2 className="font-semibold">Ganti kata sandi</h2>
        <Field label="Kata sandi lama"><input type="password" className="input" required value={lama} onChange={(ev) => setLama(ev.target.value)} autoComplete="current-password" /></Field>
        <Field label="Kata sandi baru" hint="Minimal 8 karakter."><input type="password" className="input" required minLength={8} value={baru} onChange={(ev) => setBaru(ev.target.value)} autoComplete="new-password" /></Field>
        <Field label="Ulangi kata sandi baru"><input type="password" className="input" required value={ulang} onChange={(ev) => setUlang(ev.target.value)} autoComplete="new-password" /></Field>
        <Alert kind={msg.kind || "error"}>{msg.text}</Alert>
        <button className="btn-primary self-start" disabled={busy}>{busy ? "Menyimpan…" : "Simpan"}</button>
      </form>
    </div>
  );
}

function Info({ label, value }) {
  return (
    <div>
      <p className="text-xs text-ink-muted">{label}</p>
      <p className="font-medium mt-0.5">{value}</p>
    </div>
  );
}
