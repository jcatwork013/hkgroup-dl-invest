import type { Config } from "tailwindcss";

// HKGroup "Dược Liệu lên men" — dark forest green + gold + cream, elegant serif headings.
const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        forest: {
          950: "#091a10",
          900: "#0e2317",
          800: "#143020",
          700: "#1d4530",
          600: "#27613f",
        },
        gold: {
          300: "#e6cd86",
          400: "#d8b863",
          500: "#c9a24a",
          600: "#a9863a",
        },
        cream: "#f3eede",
        // green accent scale (kept for existing brand-* usages)
        brand: {
          50: "#ecf7f0",
          100: "#d2ecdc",
          200: "#a9d9bd",
          300: "#74bf95",
          400: "#3fa56e",
          500: "#1f7a4d",
          600: "#19663f",
          700: "#135030",
          800: "#0f3f26",
          900: "#0c3a22",
        },
      },
      fontFamily: {
        serif: ["var(--font-serif)", "ui-serif", "Georgia", "serif"],
        sans: ["var(--font-sans)", "ui-sans-serif", "system-ui", "sans-serif"],
      },
    },
  },
  plugins: [],
};

export default config;
