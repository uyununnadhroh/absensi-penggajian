import Link from "next/link";
export default function NotFound() {
  return (
    <div className="min-h-full grid place-items-center p-6 text-center">
      <div>
        <p className="text-5xl font-semibold">404</p>
        <p className="text-ink-muted mt-2">Halaman tidak ditemukan.</p>
        <Link href="/" className="btn-primary mt-6">Kembali</Link>
      </div>
    </div>
  );
}
