import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// https://vitejs.dev/config/
export default defineConfig({
  base: "/runner",
  server: {
    proxy: {
      "/runner/api": "http://127.0.0.1:1337",
    },
  },
  plugins: [vue()],
});
