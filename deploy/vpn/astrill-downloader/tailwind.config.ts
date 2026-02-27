import type { Config } from 'tailwindcss';

const config: Config = {
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Noto Sans SC', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      colors: {
        bg: '#0a0e1a',
        surface: '#111827',
        surface2: '#1a2236',
        border: '#253049',
      },
    },
  },
  plugins: [],
};

export default config;
