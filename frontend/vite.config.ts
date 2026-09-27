import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
export default defineConfig(() => ({
    plugins: [vue(), tailwindcss()],
    clearScreen: false,
    server: {
        host: "0.0.0.0",
        port: 5173,
        strictPort: true,
        proxy: { "/api": "http://127.0.0.1:8080" },
    },
}));
