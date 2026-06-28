import { defineConfig, lazyPlugins } from "vite-plus";

export default defineConfig({
  plugins: lazyPlugins(async () => {
    const { default: react } = await import("@vitejs/plugin-react");
    return [react()];
  }),
  build: {
    manifest: true,
    outDir: "dist",
    emptyOutDir: true,
    rolldownOptions: {
      input: "src/main.tsx",
    },
  },
  server: {
    // Allow the Go origin to load modules from the dev server.
    cors: { origin: "http://localhost:8080" },
  },
  // templates/ and static/ hold the legacy server-rendered Go-template + Lit UI;
  // dist/ is build output. None of it is SPA source, so keep the JS toolchain off it.
  fmt: {
    ignorePatterns: ["templates/**", "static/**", "dist/**", "node_modules/**"],
  },
  lint: {
    ignorePatterns: ["templates/**", "static/**", "dist/**", "node_modules/**"],
    jsPlugins: [{ name: "vite-plus", specifier: "vite-plus/oxlint-plugin" }],
    rules: { "vite-plus/prefer-vite-plus-imports": "error" },
    options: { typeAware: true, typeCheck: true },
  },
});
