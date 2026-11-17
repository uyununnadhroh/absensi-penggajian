/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./pages/**/*.js", "./components/**/*.js", "./lib/**/*.js"],
  theme: {
    extend: {
      colors: {
        brand: { 50: "#eef6f5", 100: "#d5ebe8", 500: "#1f7a6e", 600: "#17665c", 700: "#11514a", 900: "#0b302c" },
        ink: { DEFAULT: "#1d2733", muted: "#5f6b7a", faint: "#8b96a3" },
        line: "#e3e8ee",
        canvas: "#f5f7f9",
      },
      fontFamily: { sans: ["Inter", "system-ui", "sans-serif"] },
    },
  },
  plugins: [],
};
