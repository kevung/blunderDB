import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [svelte({ compilerOptions: { runes: true } })],
    build: {
        chunkSizeWarningLimit: 1000,
        rollupOptions: {
            onwarn(warning, warn) {
                // Suppress "dynamic import will not move module into another chunk" noise
                // (help/en.js and locales/en.json are both the static fallback AND
                // reachable through the lazy loaders — expected, not a bug). The
                // warning code varies by bundler version (Rollup vs Vite 8's Rolldown),
                // so match on the message rather than pinning one code.
                if (warning.message?.includes('dynamic import will not move module')) return;
                warn(warning);
            },
            output: {
                manualChunks(id) {
                    // Split the heaviest, rarely-changing third-party deps out of the
                    // main chunk (#207) so it caches independently of app code. Locale
                    // JSON/help HTML and driver.js already get their own chunks for
                    // free, one per dynamic import() — this only needs to cover the
                    // remaining statically-imported vendor weight (two.js, the board
                    // renderer, is on screen from first paint so it stays eager).
                    if (id.includes('node_modules/two.js')) return 'two';
                }
            }
        }
    }
});
