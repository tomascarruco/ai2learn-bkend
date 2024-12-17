/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./web/**/*.{templ,html}"],
  theme: {
    extend: {
      maxWidth: {
        "2.5xl": "1350px",
      },
    },
  },
  plugins: [],
};
