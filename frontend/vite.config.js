import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte({ compilerOptions: { runes: true } })],
  build: {
    // Desktop app shipped as a single bundle — chunk size warnings are irrelevant
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      onwarn(warning, warn) {
        // Suppress "dynamic import will not move module into another chunk" noise
        if (warning.code === 'PLUGIN_WARNING' && warning.message?.includes('dynamic import will not move module')) return;
        warn(warning);
      }
    }
  }
})
