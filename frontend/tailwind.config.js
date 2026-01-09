/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "../internal/handlers/templates/**/*.html",
    "../internal/handlers/**/*.go",
    "./src/**/*.ts",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
