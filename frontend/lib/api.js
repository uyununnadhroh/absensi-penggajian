import axios from "axios";
import { getToken, clearSession } from "./session";

/* Semua panggilan ke backend lewat sini: token otomatis, respons = body, error berbahasa Indonesia. */
const client = axios.create({
  baseURL: (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080").replace(/\/$/, ""),
  timeout: 30000,
});

client.interceptors.request.use((config) => {
  const t = getToken();
  if (t) config.headers.Authorization = `Bearer ${t}`;
  return config;
});

client.interceptors.response.use(
  (res) => res.data,
  async (err) => {
    if (!err.response) {
      return Promise.reject(new Error("Tidak dapat terhubung ke server. Pastikan backend sedang berjalan."));
    }
    let { status, data } = err.response;
    if (typeof Blob !== "undefined" && data instanceof Blob) {
      try { data = JSON.parse(await data.text()); } catch { data = null; }
    }
    if (status === 401 && err.config?.url !== "/api/login") {
      clearSession();
      if (typeof window !== "undefined") window.location.assign("/login");
    }
    const e = new Error(data?.error || `Permintaan gagal (${status})`);
    e.status = status;
    return Promise.reject(e);
  }
);

const api = {
  login: (email, password) => client.post("/api/login", { email, password }),
  me: () => client.get("/api/me"),
  changePassword: (password_lama, password_baru) => client.put("/api/me/password", { password_lama, password_baru }),

  // karyawan
  myToday: () => client.get("/api/my/today"),
  checkIn: () => client.post("/api/my/check-in"),
  checkOut: () => client.post("/api/my/check-out"),
  myPayroll: (bulan) => client.get(`/api/my/payroll?bulan=${bulan}`),
  myLeaves: () => client.get("/api/my/leaves"),
  createLeave: (body) => client.post("/api/my/leaves", body),
  cancelLeave: (id) => client.delete(`/api/my/leaves/${id}`),

  // admin
  dashboard: () => client.get("/api/dashboard"),
  employees: () => client.get("/api/employees"),
  createEmployee: (body) => client.post("/api/employees", body),
  updateEmployee: (id, body) => client.put(`/api/employees/${id}`, body),
  attendance: (tanggal) => client.get(`/api/attendance?tanggal=${tanggal}`),
  saveAttendance: (body) => client.put("/api/attendance", body),
  leaves: (status = "") => client.get(`/api/leaves?status=${status}`),
  decideLeave: (id, action, catatan = "") => client.post(`/api/leaves/${id}/${action}`, { catatan }),
  holidays: (tahun) => client.get(`/api/holidays?tahun=${tahun}`),
  saveHoliday: (body) => client.post("/api/holidays", body),
  deleteHoliday: (tanggal) => client.delete(`/api/holidays/${tanggal}`),
  settings: () => client.get("/api/settings"),
  saveSettings: (body) => client.put("/api/settings", body),
  payroll: (bulan) => client.get(`/api/payroll?bulan=${bulan}`),
  exportPayroll: (bulan) => client.get(`/api/payroll/export?bulan=${bulan}`, { responseType: "blob" }),
  closePayroll: (bulan) => client.post(`/api/payroll/${bulan}/close`),
  reopenPayroll: (bulan) => client.delete(`/api/payroll/${bulan}/close`),
};

export default api;
