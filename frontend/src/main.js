import './style.css';
import { mount } from 'svelte';
import App from './App.svelte';
import { markStartup } from './utils/startupMarks.js';
import { initLanguage } from './i18n';
import { GetLanguage } from '../wailsjs/go/main/Config.js';
import { preloadDeferredComponents } from './components/lazyComponents.js';

// Every static import above has been evaluated by now: the gap from navigation is parse + eval.
markStartup('modules-evaluated');

// The dev server answers module by module, each import found only once its parent is
// parsed: a lazy panel's first open there is a waterfall of seconds on a busy machine, and
// the e2e suite runs there. Dev mounts with everything loaded, as before the split, the
// fetch overlapping the locale's; the production bundle keeps the panels off the first screen.
const devPreload = import.meta.env.DEV ? preloadDeferredComponents() : null;

// The persisted language before the first render: mounted in English, the window painted
// every label twice when the locale arrived. Bounded, so a backend slow to answer never
// leaves the window blank; App's onMount applies the language whatever happens here.
await Promise.race([
    // Through then(): a binding missing altogether throws synchronously, and must not stop the mount.
    Promise.resolve()
        .then(() => GetLanguage())
        .then(initLanguage)
        .catch(() => {}),
    new Promise((resolve) => setTimeout(resolve, 1000))
]);

await devPreload;

const app = mount(App, { target: document.getElementById('app') });
markStartup('mounted');

export default app;
