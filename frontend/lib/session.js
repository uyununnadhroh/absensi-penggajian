// Sesi login disimpan di localStorage (token + data user).
const KEY_TOKEN = "absensi_token";
const KEY_USER = "absensi_user";

const safe = (fn, fallback) => {
  try { return fn(); } catch { return fallback; }
};

export const getToken = () => (typeof window === "undefined" ? "" : safe(() => localStorage.getItem(KEY_TOKEN) || "", ""));

export const getUser = () =>
  typeof window === "undefined" ? null : safe(() => JSON.parse(localStorage.getItem(KEY_USER) || "null"), null);

export const saveSession = (token, user) =>
  safe(() => {
    localStorage.setItem(KEY_TOKEN, token);
    localStorage.setItem(KEY_USER, JSON.stringify(user));
  });

export const clearSession = () =>
  safe(() => {
    localStorage.removeItem(KEY_TOKEN);
    localStorage.removeItem(KEY_USER);
  });

export const homeFor = (user) => (user?.role === "admin" ? "/admin" : "/absen");
