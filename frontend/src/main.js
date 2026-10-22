import './style.css';
import { mount } from 'svelte';
import App from './App.svelte';
import { markStartup } from './utils/startupMarks.js';
import { startupLanguage } from './i18n';
import { GetLanguage } from '../wailsjs/go/main/Config.js';
import { preloadDeferredComponents } from './components/lazyComponents.js';

// Every static import above has been evaluated by now: the gap from navigation is parse + eval.
markStartup('modules-evaluated');

// The dev server answers module by module, each import found only once its parent is
// parsed: a lazy panel's first open there is a waterfall of seconds on a busy machine, and
// the e2e suite runs there. Dev starts fetching every panel now, without holding the mount
// on it; the production bundle keeps the panels off the first screen until `ready`. The
// `deferred-loaded` mark tells the e2e suite when a panel opens without waiting on the server.
if (import.meta.env.DEV) {
    // The Direction view is its own chunk, fetched by App on first display; it joins the wait.
    Promise.all([preloadDeferredComponents({ immediate: true }), import('./components/direction/DirectionView.svelte').catch(() => {})]).then(() => markStartup('deferred-loaded'));
}

// Mounted in English, the window painted every label twice when the locale arrived; App's
// onMount applies the language whatever happens here.
await startupLanguage(() => GetLanguage());

const app = mount(App, { target: document.getElementById('app') });
markStartup('mounted');

export default app;
