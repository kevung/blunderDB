// Le point d'entrée du front web : ni magasin global, ni routeur, ni i18n (périmètre ADR-0039).
// Les MÊMES jetons que l'application (ADR-0031, ADR-0008) ; `tokens.css` et non `style.css`,
// pour ne pas embarquer les @font-face (police japonaise) que la page n'affiche pas.
import '../tokens.css';
import { mount } from 'svelte';
import WebApp from './WebApp.svelte';

mount(WebApp, { target: document.getElementById('app') });
