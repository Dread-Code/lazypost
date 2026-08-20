import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  site: 'https://dread-code.github.io',
  base: '/lazypost/',
  vite: {
    plugins: [tailwindcss()],
  },
});
