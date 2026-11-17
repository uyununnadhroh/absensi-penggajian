import Head from "next/head";
import { useEffect, useState } from "react";
import "../styles/globals.css";
import Layout from "../components/Layout";
import { getToken, getUser, homeFor } from "../lib/session";

const PUBLIC = ["/login"];

export default function App({ Component, pageProps, router }) {
  const [ready, setReady] = useState(false);
  const [user, setUser] = useState(null);
  const isPublic = PUBLIC.includes(router.pathname) || router.pathname === "/404";

  // cek sesi & hak akses setiap pindah halaman
  useEffect(() => {
    const u = getUser();
    setUser(u);
    setReady(true);
    if (isPublic) return;
    if (!getToken() || !u) {
      router.replace("/login");
      return;
    }
    const adminPage = router.pathname.startsWith("/admin");
    const empPage = ["/absen", "/riwayat", "/pengajuan", "/slip"].includes(router.pathname);
    if ((adminPage && u.role !== "admin") || (empPage && u.role !== "karyawan") || router.pathname === "/") {
      router.replace(homeFor(u));
    }
  }, [router.pathname]); // eslint-disable-line react-hooks/exhaustive-deps

  const allowed =
    user &&
    getToken() &&
    router.pathname !== "/" &&
    !(router.pathname.startsWith("/admin") && user.role !== "admin");

  return (
    <>
      <Head>
        <title>Absensi &amp; Penggajian</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>
      {!ready ? null : isPublic ? (
        <Component {...pageProps} />
      ) : allowed ? (
        <Layout user={user}>
          <Component {...pageProps} user={user} />
        </Layout>
      ) : null}
    </>
  );
}
