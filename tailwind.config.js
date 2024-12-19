/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./web/**/*.{templ,html}"],
  theme: {
    extend: {
      fadeIn: {
        "0%": { opacity: 0 },
        "100%": { opacity: 1 },
      },
    },
  },
  darkMode: "class",
  plugins: [],
};
