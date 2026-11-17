import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import api from "../lib/api";
import { getToken, getUser, homeFor, saveSession } from "../lib/session";
import { Alert } from "../components/ui";

export default function Login() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [show, setShow] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const u = getUser();
    if (getToken() && u) router.replace(homeFor(u));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function submit(e) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const res = await api.login(email, password);
      saveSession(res.token, res.user);
      router.replace(homeFor(res.user));
    } catch (err) {
      setError(err.message);
      setBusy(false);
    }
  }

  return (
    <div className="min-h-full grid lg:grid-cols-2">
      <div className="hidden lg:flex flex-col justify-between bg-brand-900 p-12 text-white">
        <div className="flex items-center gap-3">
          <div className="size-10 rounded-xl bg-brand-500 grid place-items-center">
            <svg viewBox="0 0 24 24" className="size-6" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 11l3 3L22 4M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" /></svg>
          </div>
          <span className="font-semibold">Absensi &amp; Penggajian</span>
        </div>
        <div className="max-w-md">
          <h1 className="text-4xl font-semibold leading-tight">Absen tepat waktu, gaji dihitung otomatis.</h1>
          <ul className="mt-8 flex flex-col gap-4 text-brand-100">
            <li>• Karyawan absen masuk &amp; pulang langsung dari HP.</li>
            <li>• Telat, pulang cepat, dan lembur tercatat dengan jam server.</li>
            <li>• Pengajuan izin, sakit, dan cuti disetujui HRD secara online.</li>
            <li>• Rekap gaji bulanan dan slip gaji siap cetak.</li>
          </ul>
        </div>
        <p className="text-xs text-brand-100/60">Sistem internal perusahaan</p>
      </div>

      <div className="flex items-center justify-center p-6">
        <form onSubmit={submit} className="w-full max-w-sm flex flex-col gap-5">
          <div>
            <h2 className="text-2xl font-semibold">Masuk</h2>
            <p className="text-sm text-ink-muted mt-1">Gunakan email dan kata sandi dari HRD.</p>
          </div>
          <Alert>{error}</Alert>
          <label className="block">
            <span className="label">Email</span>
            <input className="input" type="email" autoComplete="username" required value={email} onChange={(e) => setEmail(e.target.value)} />
          </label>
          <label className="block">
            <span className="label">Kata sandi</span>
            <div className="relative">
              <input className="input pr-16" type={show ? "text" : "password"} autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
              <button type="button" onClick={() => setShow(!show)} className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-ink-muted px-2 py-1">
                {show ? "Sembunyikan" : "Lihat"}
              </button>
            </div>
          </label>
          <button className="btn-primary h-11" disabled={busy}>{busy ? "Memeriksa…" : "Masuk"}</button>
        </form>
      </div>
    </div>
  );
}
