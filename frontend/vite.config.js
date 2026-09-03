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
                if (warning.code === 'PLUGIN_WARNING' && warning.message?.includes('dynamic import will not move module')) return;
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
