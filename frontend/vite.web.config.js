import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// La construction du front web (#295, fiche J.5).
//
// Une seconde entrée, pas une seconde chaîne d'outils : le front web réutilise
// le dessinateur de plateau de l'application (ADR-0039), donc il est bâti par
// le même Vite avec le même compilateur Svelte. Le résultat est versionné dans
// internal/server/webui/dist et embarqué dans le binaire — même discipline que
// les paquets d'aide (ADR-0034), openapi.yaml et le client Python.
//
// `base` est relatif parce que la page est servie sous /app/ : des chemins
// absolus la rendraient dépendante de l'endroit où le démon la monte.
export default defineConfig({
    plugins: [svelte({ compilerOptions: { runes: true } })],
    base: './',
    build: {
        outDir: '../internal/server/webui/dist',
        emptyOutDir: true,
        rollupOptions: { input: 'web.html' }
    }
});
