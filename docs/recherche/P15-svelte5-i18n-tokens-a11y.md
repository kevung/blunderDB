# Moderniser une application de bureau Svelte 5 + Vite 7 en WebView (Wails) : i18n paresseux, tokens de couleur, accessibilité et raccourcis clavier

## TL;DR
- **i18n** : passez au **chargement paresseux par locale** via `import.meta.glob` (lazy) ou `await import()`, avec une seule locale de repli incluse dans le bundle initial ; pour supprimer durablement les 65 % de poids, **Paraglide JS 2.x** (compilateur, tree-shaking par message, fonctionne en Vite pur sans SvelteKit) est le meilleur choix, en gardant à l'esprit qu'il embarque toutes les langues par défaut (le découpage par locale est expérimental).
- **Couleurs & thème** : adoptez un système de tokens à trois niveaux (primitifs → sémantiques → composants) au format **DTCG v2025.10** généré par **Style Dictionary** vers CSS *et* JS ; implémentez le thème sombre avec `color-scheme` + `light-dark()` (Baseline mai 2024) et une préférence persistée sur `<html data-theme>`, plus des thèmes `forced-colors` (crucial pour WebView2) et `@media print` ; verrouillez le tout avec **stylelint-declaration-strict-value** en CI.
- **Accessibilité & clavier** : remplacez les widgets maison par les patterns **ARIA APG** (Tabs à roving tabindex, table sémantique + tri par bouton plutôt que `role="grid"`), remplacez le focus trap maison par `<dialog>`+`showModal()`/`inert`, faites **échouer le build** sur les warnings a11y du compilateur Svelte (`warningFilter` + `svelte-check --fail-on-warnings`), et cessez de confisquer Tab globalement (violation potentielle des SC 2.1.1/2.1.2/2.1.4) : limitez la capture à un widget, offrez Échap/Ctrl+Tab, et rendez le raccourci désactivable/remappable.

## Key Findings

1. **Le lazy-loading est natif dans Vite** : `import.meta.glob('./locales/*.json')` produit par défaut des `() => import()` découpés en chunks séparés ; c'est le mécanisme central pour ne charger qu'une locale à la demande. Une seule locale de repli doit être importée statiquement pour éviter le « flash » de contenu non traduit.
2. **Paraglide JS 2.25.0** compile chaque message en fonction ESM tree-shakeable. Son benchmark officiel (opral/paraglide-js) revendique « up to 70% smaller i18n bundle sizes », avec des exemples concrets : **47 Ko avec Paraglide contre 205 Ko avec i18next**, et une propriété clé — le poids Paraglide reste stable quand le nombre total de messages croît (« using 100 messages shipped 47 KB with Paraglide whether the project had 200, 500, or 1,000 total messages. The i18next runtime bundle grew from 205 KB to 414 KB »). Il fonctionne avec « n'importe quelle app Vite » (pas besoin de SvelteKit). Limite majeure : il embarque **toutes** les langues par défaut ; le découpage par locale est encore expérimental.
3. **La spécification W3C Design Tokens (DTCG) a atteint sa première version stable, v2025.10, le 28 octobre 2025**, développée par « more than 20 editors and authors » représentant Adobe, Amazon, Google, Baidu, Sony, Microsoft, Meta, Sketch, Salesforce, Shopify, Figma, Framer, Cisco, Intuit, New York Times, GM, Disney, Pinterest, Tokens Studio, Penpot et zeroheight (communiqué W3C du 28 oct. 2025). Style Dictionary v4 supporte le format DTCG ; le support complet du format 2025.10 est en cours dans v5.
4. **`light-dark()` est Baseline « newly available » depuis le 13 mai 2024** — confirmé exactement par web.dev (« becomes Baseline Newly available as of May 13, 2024 »). Dates par moteur : Firefox 120 (21 nov. 2023), Chrome 123 (19 mars 2024), Edge 123 (22 mars 2024), Safari 17.5 (13 mai 2024, d'où la date Baseline). Donc sûr dans WebView2/WebKitGTK récents — mais à vérifier selon la version embarquée par Wails sur chaque OS. Le passage à « widely available » est attendu ultérieurement.
5. **Le `<dialog>` natif avec `showModal()` gère le piège du focus, Échap et le backdrop nativement** depuis 2022 — et par décision du W3C APA, il laisse volontairement le focus s'échapper vers le *chrome* du navigateur, ce qui est correct.
6. **Le compilateur Svelte 5 émet des warnings a11y** (préfixe `a11y_`) qu'on peut transformer en erreurs bloquantes via `compilerOptions.warningFilter` et `svelte-check`. Mais ces warnings ne couvrent PAS l'ordre de focus, le contraste, ni la correction sémantique dynamique.
7. **Les tests automatisés (axe-core) ne détectent qu'une partie des problèmes** : le README officiel d'axe-core (dequelabs/axe-core) indique « With axe-core, you can find on average 57% of WCAG issues automatically » — chiffre issu de l'étude Deque « The Automated Accessibility Coverage Report » (57,38 %, analyse de 2 000+ audits), à comparer à la baseline industrielle historique de 20-30 % citée par Deque. Le test manuel au clavier et au lecteur d'écran reste indispensable.
8. **Confisquer Tab globalement pour un raccourci applicatif est une anti-pattern** qui risque de violer plusieurs critères WCAG 2.2. Les applications de référence le résolvent par un mode basculable : la doc officielle VS Code précise « You can toggle Tab trapping with ⌃⇧M (Windows, Linux Ctrl+M)... you will see a Tab moves focus indicator in the Status Bar » (configurable via `editor.tabFocusMode`), plus F6/Shift+F6 pour naviguer entre régions.

## Details

### 1. Chargement paresseux des locales (i18n)

#### 1.1 `import.meta.glob` : les variantes exactes

La documentation officielle Vite décrit `import.meta.glob` comme la fonction d'import de multiples modules. Par défaut, **les fichiers correspondants sont chargés paresseusement (lazy) via dynamic import et découpés en chunks séparés au build** — exactement ce dont vous avez besoin pour vos 9 catalogues :

```js
// locales/index.js — un chunk par locale, chargé à la demande
const catalogues = import.meta.glob('./*.json'); 
// => { './fr.json': () => import('./fr.json'), './en.json': () => import('./en.json'), ... }

export async function loadCatalogue(locale) {
  const loader = catalogues[`./${locale}.json`];
  if (!loader) throw new Error(`Locale inconnue : ${locale}`);
  const mod = await loader();
  return mod.default; // le JSON est en export default
}
```

Options clés (doc Vite « Features ») :
- `{ eager: true }` : importe tout **synchroniquement** — à éviter ici, car cela ré-inline les 9 langues dans le bundle et reproduit votre problème actuel.
- `{ query: '?raw', import: 'default' }` : importe le contenu **brut comme chaîne** (utile pour vos fichiers d'aide HTML, voir §1.5). Attention : dans Vite récent, le point d'interrogation `?` est devenu obligatoire pour `raw`, `url`, `init` (l'ancienne option `as: 'raw'` est dépréciée au profit de `query: '?raw'`).
- `{ import: 'default' }` : n'importe que l'export par défaut (permet un meilleur tree-shaking quand combiné à `eager`).

#### 1.2 Découpage du bundle : `manualChunks` dans Vite 7

La configuration se fait via `build.rollupOptions.output.manualChunks`, sous **forme fonction** (recommandée) ou **forme objet** :

```js
// vite.config.js (Vite 7 / Rollup)
export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/node_modules/')) return 'vendor';
          // les locales sont déjà découpées par import.meta.glob ;
          // regrouper ici uniquement si nécessaire
        }
      }
    }
  }
});
```

**Point important pour Vite 7** : Vite 7 introduit l'intégration **Rolldown** (`rolldown-vite`). La doc officielle Vite 7 signale que, sous Rolldown, `manualChunks` est **supporté mais marqué déprécié**, au profit de `advancedChunks` (plus proche de `splitChunks` de webpack) :

```js
// Forme Rolldown (Vite 7) — plus fine
output: {
  advancedChunks: {
    groups: [{ name: 'vendor', test: /\/node_modules\// }]
  }
}
```
Signalez ceci dans votre choix : si vous restez sur le Rollup classique, `manualChunks` reste valable ; si vous adoptez Rolldown, migrez vers `advancedChunks`.

**Piège documenté** : Rollup peut placer des helpers partagés dans un chunk de locale, forçant son chargement anticipé. Dirigez explicitement les helpers connus vers `vendor`. `experimental.renderBuiltUrl` sert à réécrire les URLs d'assets (utile en WebView où le protocole/base peut différer, ex. `wails://` ou `file://`) ; le polyfill `modulepreload` et l'analyse `vite:build-import-analysis` insèrent automatiquement des `<link rel="modulepreload">` pour les chunks — en WebView embarquée, vérifiez que ce préchargement fonctionne avec le schéma d'URL de Wails, sinon désactivez-le via `build.modulePreload: false`.

#### 1.3 Repli synchrone et fallback en cascade

Pour éviter le flash de contenu non traduit, incluez **une seule** locale (la langue par défaut) dans le bundle initial via import statique, et gardez les autres en chunks paresseux :

```js
// i18n.svelte.js — état global de langue (Svelte 5)
import fr from './locales/fr.json';       // repli STATIQUE (dans le bundle initial)
const lazy = import.meta.glob('./locales/*.json'); // les autres, à la demande

let current = $state({ locale: 'fr', messages: fr });

// fallback en cascade fr-CA -> fr -> en
function resolveChain(locale) {
  const chain = [];
  const parts = locale.split('-');
  while (parts.length) { chain.push(parts.join('-')); parts.pop(); }
  if (!chain.includes('fr')) chain.push('fr'); // repli final
  return chain;
}

export const i18n = {
  get locale() { return current.locale; },
  get messages() { return current.messages; },
  async setLocale(locale) {
    for (const cand of resolveChain(locale)) {
      const loader = lazy[`./locales/${cand}.json`];
      if (loader) {
        const mod = await loader();
        current = { locale, messages: { ...fr, ...mod.default } }; // clés manquantes -> repli fr
        document.documentElement.lang = locale;
        return;
      }
    }
  }
};
```

La fusion `{ ...fr, ...mod.default }` gère les **clés manquantes** en retombant sur la locale de repli. Pour une fonction `t()`, ajoutez un repli par clé qui renvoie la clé elle-même si absente.

#### 1.4 Pièges Svelte 5 pour un store de langue

- **Règle « on ne peut pas réassigner un `$state` importé »** : la doc `$state` de Svelte confirme qu'on peut *déclarer* du `$state` dans un fichier `.svelte.js`/`.svelte.ts`, mais **on ne peut l'exporter que s'il n'est pas réassigné directement**. La solution idiomatique est un **objet avec getters/setters** (comme ci-dessus) ou un objet `$state({...})` dont on mute les propriétés — car Svelte enveloppe les objets `$state` dans un Proxy qui préserve la réactivité au travers des modules.
- **`$effect` ne s'exécute pas hors composant** : un `$effect` déclaré dans un module `.svelte.js` hors d'un composant ne tournera pas. Pour réagir au changement de langue en dehors d'un composant (ex. re-render d'un canvas), utilisez **`$effect.root`** (qui crée une portée d'effet manuelle à nettoyer soi-même) ou l'API `createSubscriber` (Svelte 5.7+) pour exposer une valeur réactive alimentée par une source externe.
- **Éviter les effets qui écrivent dans l'état** : ne mettez pas à jour `current` depuis un `$effect` déclenché par la lecture de `current` — cela crée des boucles. Préférez des fonctions asynchrones explicites (`setLocale`) déclenchées par l'interaction utilisateur.
- **`$derived` vs état asynchrone** : `$derived` est synchrone ; il ne peut pas « attendre » un `import()`. Gérez le chargement asynchrone dans `setLocale` (impératif) et exposez `messages` en `$state`, ou utilisez `{#await}` dans le composant.
- **Flash & `{#await}` vs pré-chargement** : pour éviter le flash, pré-chargez la locale cible **avant** de basculer l'affichage (attendre la résolution de `setLocale` avant de mettre à jour l'UI), plutôt que d'afficher un état intermédiaire via `{#await}`.

#### 1.5 Fichiers d'aide HTML volumineux par langue

Deux approches selon le besoin :
- **`?url`** : renvoie l'URL du fichier ; chargez-le à la demande via `fetch()`, ce qui évite de gonfler le bundle JS. Idéal pour de gros fichiers d'aide.
- **`?raw`** : inline le contenu comme chaîne dans le chunk. Plus rapide (pas de requête HTTP supplémentaire) mais augmente la taille du chunk.

Pour de gros fichiers d'aide par langue, préférez `?url` + `fetch` + mise en cache (Map en mémoire). **Sécurité** : si vous injectez ce HTML via `{@html ...}`, Svelte n'échappe rien — vous devez **sanitiser** (ex. avec DOMPurify) si le contenu n'est pas 100 % de confiance. Comme ce sont vos propres fichiers d'aide, le risque est limité, mais documentez la contrainte (pas d'inclusion de contenu tiers non sanitisé).

#### 1.6 Comparaison des bibliothèques i18n compatibles Svelte 5 (2026)

| Bibliothèque | Version 2026 | Tree-shaking par message | Lazy-loading par locale | Marche en Vite pur (sans SvelteKit) | Bascule de langue |
|---|---|---|---|---|---|
| **Paraglide JS** (`@inlang/paraglide-js`) | **2.25.0** | ✅ Oui (compilateur ; 47 Ko vs 205 Ko i18next, poids stable) | ⚠️ Par page/message oui ; par locale = expérimental (`experimentalPerLocaleBuild`, Vite seulement) ; toutes langues embarquées par défaut | ✅ Oui (« n'importe quelle app Vite », vanilla/Svelte) | `setLocale()` / `getLocale()` |
| **typesafe-i18n** | **5.27.1** (fév. 2026) | ⚠️ Limité (plugin rollup retiré en v5) | ✅ Oui (async + namespaces) | ✅ Oui (agnostique) | wrappers générés |
| **svelte-i18n** | **4.0.1** (~2023, figée) | ❌ Non | ✅ Oui (`register` async) | ✅ Oui (stores Svelte) | `locale.set()`, `$_()` |
| **sveltekit-i18n** | **2.4.2** (~2022, figée) | ❌ Non | ✅ Oui (par route) | ⚠️ Faible — couplé SvelteKit | `loadTranslations()` |
| **@formatjs/intl** | **4.1.19** | ❌ Non | Manuel (recréer l'objet intl) | ✅ Oui (cœur agnostique) | recréer `createIntl()` |
| **i18next** (+ `i18next-resources-to-backend`) | i18next **26.4.1** | ❌ Non | ✅ Oui (dynamic import ; peu ergonomique en Vite) | ✅ Oui (agnostique) | `changeLanguage()` |

**Recommandation** : **Paraglide JS** est le meilleur candidat pour votre stack (Svelte 5 + Vite pur), grâce au tree-shaking par message et à une intégration mono-plugin. Signalez la limite : il embarque toutes les langues par défaut ; pour un vrai découpage par locale, l'option `experimentalPerLocaleBuild` est réservée aux apps Vite (ce qui inclut votre cas) mais reste expérimentale. Si vous préférez rester en JSON runtime avec lazy-loading simple, **svelte-i18n** reste le plus simple mais est figé depuis ~2023 et n'est pas « runes-native ». **typesafe-i18n** (maintenu en 2026 sous nouvelle gouvernance `codingcommons`, dernière version 5.27.1 de février 2026, non déprécié) est une alternative typée. Évitez **sveltekit-i18n** (couplé à SvelteKit).

*Note de divergence* : le « jusqu'à 70 % » de Paraglide provient de son propre benchmark ; c'est un ordre de grandeur, pas une garantie universelle. La stabilité du poids avec le nombre de messages est en revanche la propriété structurellement garantie par l'approche compilateur.

### 2. Tokens de couleur et thème sombre

#### 2.1 Nommage : trois niveaux (three-tier)

La convention consensuelle en 2026 est le modèle à trois couches : **primitif** (`--blue-500`) → **sémantique** (`--color-action`) → **composant** (`--button-bg-primary`). Ce modèle est repris par Material Design 3, Radix Colors, Tailwind CSS v4 (`@theme`), Primer (GitHub), Carbon (IBM), Adobe Spectrum et l'US Web Design System. L'intérêt : un changement de valeur primitive se propage proprement à toutes les surfaces. Vos 108 couleurs en dur doivent être ramenées d'abord à une palette primitive réduite, puis mappées sémantiquement.

#### 2.2 Format DTCG et Style Dictionary : source unique de vérité

Le format **DTCG** (W3C Design Tokens Community Group) a atteint sa **première version stable v2025.10 le 28 octobre 2025** : un token a un `$value` et un `$type`, et peut référencer d'autres tokens par chemin — ce qui rend les tokens sémantiques portables. **Style Dictionary v4** supporte le DTCG « en première classe » ; la doc précise que le format 2025.10 le plus récent **n'est pas encore totalement supporté** (travail en cours dans **v5**). Signalez cette divergence : générez vos tokens vers **CSS custom properties ET JS/TS** depuis la même source DTCG, ce qui résout élégamment le problème du canvas (§2.6).

#### 2.3 Implémentation CSS : `color-scheme`, `light-dark()`, préférence persistée

`light-dark()` est **Baseline « newly available » depuis le 13 mai 2024** (Chrome/Edge 123, Firefox 120, Safari 17.5). Combinez-le avec une préférence utilisateur persistée :

```css
:root {
  color-scheme: light dark;
  --color-bg: light-dark(#ffffff, #121212);
  --color-text: light-dark(#1a1a1a, #f5f5f5);
  --color-text-secondary: light-dark(#595959, #b3b3b3); /* voir §2.4 contraste */
}
/* Override explicite par préférence utilisateur, appliqué sur <html data-theme> */
:root[data-theme="light"] { color-scheme: light; }
:root[data-theme="dark"]  { color-scheme: dark; }
```

**Éviter le FOUC/flash de thème** : la préférence doit être appliquée **synchroniquement avant le premier paint**, via un petit script inline dans le `<head>` qui lit `localStorage` et pose `data-theme` sur `<html>` — avant que Svelte ne monte. C'est le « flash of default theme (FODT) », analogue au FOUC.

```html
<!-- dans index.html, AVANT le script du module principal -->
<script>
  const t = localStorage.getItem('theme');
  if (t) document.documentElement.dataset.theme = t;
</script>
```

En Svelte 5, gérez la bascule via un état global `.svelte.js` (getters/setters, comme §1.4) et un `$effect` **dans un composant racine** pour persister dans `localStorage`.

#### 2.4 Contraste WCAG 2.2 AA à 11 px

Rappel des seuils : **4.5:1 pour texte normal**, 3:1 pour « large text » défini à **18pt/24px** (ou 14pt bold/18.66px). Votre texte secondaire à **11 px est donc du texte normal → 4.5:1 obligatoire** en AA. Beaucoup de gris secondaires échouent à ce seuil : vérifiez chaque token de texte, dans les deux thèmes. Mentionnez APCA (contraste perceptuel, pressenti pour WCAG 3) comme perspective, mais gardez WCAG 2.2 AA comme cible normative aujourd'hui. Outils de vérification : `@adobe/leonardo-contrast-colors` (génère des palettes à contraste garanti), les échelles Radix Colors (contraste pensé par pas), et `colorjs.io` pour calculer les ratios en CI.

#### 2.5 Thèmes supplémentaires : contraste élevé et impression

- **`forced-colors: active` (Windows High Contrast Mode)** — particulièrement pertinent car votre app tourne en **WebView2 sur Windows**. En mode forced-colors, le navigateur force certaines propriétés vers les **mots-clés système** (`Canvas`, `CanvasText`, `ButtonText`, `LinkText`, `Highlight`…). Bonne pratique (doc Firefox Source Docs) : redéfinissez vos custom properties dans un bloc `@media (forced-colors: active)` pour qu'elles pointent vers les couleurs système, sans réécrire chaque règle :

```css
@media (forced-colors: active) {
  :root {
    --color-bg: Canvas;
    --color-text: CanvasText;
    --color-action: LinkText;
  }
  /* les bordures deviennent visibles là où on s'appuyait sur box-shadow (forcé à none) */
  .card { border: 1px solid CanvasText; }
}
```
Note MDN importante : le navigateur choisit les couleurs système d'après la **sémantique HTML native**, pas d'après les rôles ARIA ajoutés (`role="button"` sur un `div` ne forcera pas `ButtonText`) — argument de plus pour du HTML sémantique. `forced-colors: active` s'accompagne de `prefers-contrast: forced` (extension récente).

- **Thème imprimable** : `@media print` avec fond blanc/texte noir forcés et `print-color-adjust: exact` (ou `economy`) selon que vous voulez préserver ou non les aplats de couleur.

```css
@media print {
  :root { --color-bg: #fff; --color-text: #000; }
  body { print-color-adjust: economy; }
}
```

#### 2.6 Canvas (two.js / 2D) qui doit lire les tokens

Un canvas ne peut pas utiliser directement les custom properties CSS. La technique documentée (Aaron Gustafson) est de lire les valeurs calculées :

```js
function importTheme() {
  const cs = getComputedStyle(document.documentElement);
  theme.fg = cs.getPropertyValue('--color-text').trim() || '#000';
  theme.bg = cs.getPropertyValue('--color-bg').trim() || '#fff';
}
importTheme();
// réagir au changement de thème système
matchMedia('(prefers-color-scheme: dark)').addEventListener('change', importTheme);
// réagir au changement de préférence utilisateur (data-theme sur <html>)
new MutationObserver(importTheme).observe(document.documentElement, {
  attributes: true, attributeFilter: ['data-theme']
});
```

**Coût de performance** : `getComputedStyle` force un reflow ; **mettez en cache** les valeurs (comme l'objet `theme`) et ne relisez qu'au changement de thème (événement `change` ou MutationObserver), jamais dans la boucle de rendu. **Alternative recommandée et plus propre** : générer les tokens en JS/TS depuis Style Dictionary (§2.2) et les importer directement dans le code canvas — source unique de vérité, zéro `getComputedStyle`, zéro reflow. Il existe aussi `@bramus/style-observer` (MutationObserver pour propriétés CSS calculées) si vous devez observer des variables sans attribut déclencheur.

#### 2.7 Outillage anti-couleurs-en-dur (CI)

Le plugin de référence est **`stylelint-declaration-strict-value`** (règle `scale-unlimited/declaration-strict-value`), qui impose l'usage d'une variable/fonction pour des propriétés données :

```js
// stylelint.config.js
export default {
  plugins: ['stylelint-declaration-strict-value'],
  overrides: [
    { files: ['**/*.svelte'], customSyntax: 'postcss-html' }
  ],
  rules: {
    'scale-unlimited/declaration-strict-value': [
      ['/color$/', 'fill', 'stroke', 'background-color', 'border-color'],
      { ignoreValues: ['transparent', 'currentColor', 'inherit'], ignoreVariables: false }
    ]
  }
};
```

Points clés : `ignoreVariables: false` interdit même une valeur littérale au profit d'une `var(...)` ; l'option `customSyntax: 'postcss-html'` (avec `stylelint-config-html`) permet de linter les blocs `<style>` dans les fichiers `.svelte`. Complétez avec la règle native `color-no-hex` et/ou `declaration-property-value-disallowed-list`. Pour les couleurs en dur dans le **JS/TS** (canvas, styles inline), ajoutez une règle ESLint `no-restricted-syntax` avec une regex `#[0-9a-fA-F]{3,8}`, ou un test unitaire qui parcourt les fichiers à la recherche de `#rrggbb`. Ajoutez enfin des **tests de régression visuelle** (Playwright screenshots) pour les deux thèmes.

### 3. Accessibilité (ARIA APG et Svelte)

#### 3.1 Pattern Tabs (APG)

Le pattern APG Tabs impose : conteneur `role="tablist"`, boutons `role="tab"` avec `aria-selected` et `aria-controls`, panneaux `role="tabpanel"` avec `aria-labelledby`. La navigation utilise un **roving tabindex** : seul l'onglet sélectionné a `tabindex="0"`, les autres `tabindex="-1"` ; les flèches déplacent le focus, Home/End vont au premier/dernier. **Activation automatique** (le panneau s'affiche au focus) recommandée uniquement si tout le contenu est déjà dans le DOM ; sinon **activation manuelle** (Espace/Entrée). Pour des onglets fermables, Delete supprime l'onglet.

```svelte
<script>
  let { tabs } = $props();            // [{ id, label, ... }]
  let selected = $state(tabs[0].id);
  let tabEls = $state([]);            // refs pour le roving tabindex

  function onKeydown(e, i) {
    const last = tabs.length - 1;
    let next = null;
    if (e.key === 'ArrowRight') next = i === last ? 0 : i + 1;
    else if (e.key === 'ArrowLeft') next = i === 0 ? last : i - 1;
    else if (e.key === 'Home') next = 0;
    else if (e.key === 'End') next = last;
    if (next !== null) {
      e.preventDefault();
      selected = tabs[next].id;       // activation automatique
      tabEls[next]?.focus();
    }
  }
</script>

<div role="tablist" aria-label="Sections">
  {#each tabs as tab, i}
    <button
      role="tab"
      id={`tab-${tab.id}`}
      bind:this={tabEls[i]}
      aria-selected={selected === tab.id}
      aria-controls={`panel-${tab.id}`}
      tabindex={selected === tab.id ? 0 : -1}
      onclick={() => (selected = tab.id)}
      onkeydown={(e) => onKeydown(e, i)}
    >{tab.label}</button>
  {/each}
</div>

{#each tabs as tab}
  <div role="tabpanel" id={`panel-${tab.id}`} aria-labelledby={`tab-${tab.id}`}
       hidden={selected !== tab.id} tabindex="0">
    {@render tab.content()}   <!-- snippet -->
  </div>
{/each}
```

#### 3.2 Table triable : préférer `<table>` sémantique + tri par bouton à `role="grid"`

C'est un point de consensus fort chez les experts (Adrian Roselli, Sarah Higley, échanges du groupe ARIA du W3C) : **une table triable ne doit PAS être un `role="grid"`**. Le `role="grid"` implique une navigation clavier bidimensionnelle cellule par cellule (flèches, sélection via `aria-selected`, roving tabindex ou `aria-activedescendant`) — « major overkill » pour une table de données statique, et source de confusion pour les utilisateurs de lecteurs d'écran quand il est mal appliqué. La règle « No ARIA is better than Bad ARIA » s'applique.

La bonne solution : une **`<table>` sémantique native**, avec `aria-sort` sur le `<th>` de la colonne triée et un `<button>` dans le `<th>` pour déclencher le tri :

```svelte
<script>
  let { rows, columns } = $props();
  let sortKey = $state(null);
  let sortDir = $state('none'); // 'ascending' | 'descending' | 'none'

  const sorted = $derived.by(() => {
    if (!sortKey || sortDir === 'none') return rows;
    const dir = sortDir === 'ascending' ? 1 : -1;
    return [...rows].sort((a, b) => (a[sortKey] > b[sortKey] ? 1 : -1) * dir);
  });

  function toggleSort(key) {
    if (sortKey !== key) { sortKey = key; sortDir = 'ascending'; }
    else sortDir = sortDir === 'ascending' ? 'descending' : 'ascending';
  }
</script>

<table>
  <thead>
    <tr>
      {#each columns as col}
        <th aria-sort={sortKey === col.key ? sortDir : 'none'} scope="col">
          <button onclick={() => toggleSort(col.key)}>
            {col.label}
            <span aria-hidden="true">{sortKey === col.key ? (sortDir === 'ascending' ? '▲' : '▼') : ''}</span>
          </button>
        </th>
      {/each}
    </tr>
  </thead>
  <tbody>
    {#each sorted as row}
      <tr>{#each columns as col}<td>{row[col.key]}</td>{/each}</tr>
    {/each}
  </tbody>
</table>
```

Réservez `role="grid"`/`treegrid` aux cas réellement interactifs (édition de cellules, sélection 2D, réorganisation par glisser-déposer). Pour de très grandes tables, `aria-rowcount`/`aria-rowindex` permettent d'annoncer la taille réelle malgré la virtualisation.

#### 3.3 Combobox avec autocomplétion (ARIA 1.2)

Le pattern moderne (APG Combobox, ARIA 1.2) : `role="combobox"` sur l'`<input>`, avec `aria-expanded`, `aria-controls` (pointant vers le listbox), et `aria-activedescendant` (pointant vers l'option visuellement focalisée). **Le focus DOM reste sur l'input** ; les flèches déplacent le focus *visuel* dans le listbox via `aria-activedescendant`. `aria-autocomplete` prend `list` (suggestions dans une liste), `both` (liste + complétion inline), ou `none`. Distinguez input **éditable** (le cas usuel, filtre) et **non-éditable** (sélection type `<select>` custom). Bug connu à tester : certaines combinaisons lecteur d'écran/navigateur annoncent mal `aria-activedescendant` — testez sur NVDA (WebView2) et l'AT concernée.

#### 3.4 `aria-live` bien délimité

Bonnes pratiques (MDN, APG, A11yPath) : **montez les régions live vides au démarrage** (dans le DOM initial), puis injectez le texte — sinon les lecteurs d'écran manquent l'annonce si l'élément est créé et rempli dans le même « tick ». Préférez les **rôles-raccourcis** : `role="status"` (= `aria-live="polite"`, pour la plupart des cas), `role="alert"` (= assertive, à réserver aux erreurs urgentes), `role="log"` (flux type chat, sans `aria-atomic`). **`assertive` est presque toujours une erreur** : il interrompt la lecture en cours. `aria-atomic="true"` fait relire toute la région (utile pour une horloge) ; `aria-atomic="false"` ne lit que ce qui change. Une règle ACT (SIA-R54) veut qu'une région `assertive` avec descendants soit `aria-atomic="true"`. Pattern robuste : deux régions uniques (une polite, une assertive) dans le layout, dont on swappe le `textContent`.

```svelte
<!-- monté une fois dans le composant racine -->
<div role="status" class="sr-only">{politeMessage}</div>
<div role="alert"  class="sr-only">{assertiveMessage}</div>
```

#### 3.5 Focus trap : préférer le natif

Votre `focusTrap` maison devrait être remplacé. Les experts (CSS-Tricks, Scott O'Hara, décision du W3C APA) recommandent le **`<dialog>` natif ouvert avec `showModal()`** : il fournit **nativement** le piège du focus (Tab cycle à l'intérieur), la fermeture par Échap, le backdrop et les bonnes sémantiques ARIA — sans bibliothèque. Point clé : le W3C APA a **décidé que `<dialog>` ne devait PAS piéger le focus du *chrome* du navigateur** (on peut Tab vers la barre d'adresse), ce qui est un comportement voulu offrant une échappatoire.

Pour un piège hors dialog (ex. un panneau), utilisez l'attribut **`inert`** sur le reste du document. Pour détecter la « focusabilité » réelle (éléments invisibles à ignorer), la méthode moderne est `element.checkVisibility({ visibilityProperty: true, contentVisibilityAuto: true, opacityProperty: true })`, complétée par les tests classiques (`offsetParent`, `getClientRects().length`, `display:none`, `visibility:hidden`, `content-visibility`, `inert`). Si vous avez besoin d'une bibliothèque (composants montés dynamiquement, contenu async), `focus-trap` + `tabbable` (David Clark) ou `a11y-dialog` restent les références — mais pour la plupart des cas, `<dialog>` suffit et supprime des centaines de lignes de code maison.

#### 3.6 Ce que `svelte-check` / `eslint-plugin-svelte` détectent (et pas)

Le compilateur Svelte 5 émet des warnings a11y (préfixe `a11y_`) listés dans « Compiler warnings ». Exemples pertinents : `a11y_click_events_have_key_events`, `a11y_no_static_element_interactions`, `a11y_role_has_required_aria_props`, `a11y_autofocus`, `a11y_label_has_associated_control`, `a11y_aria_activedescendant_has_tabindex`, `a11y_no_noninteractive_tabindex`, `a11y_missing_attribute`, `a11y_distracting_elements`, `a11y_no_accesskey`. `eslint-plugin-svelte` expose la règle `svelte/valid-compile` qui remonte ces mêmes warnings, et respecte le `warningFilter` de `svelte.config.js`. **`eslint-plugin-jsx-a11y` ne s'applique pas** (c'est pour JSX).

**Limites explicites** (Geoff Rich, « What Svelte's accessibility warnings won't tell you ») : le compilateur **ne détecte PAS** l'ordre de focus, le contraste des couleurs, ni la correction sémantique dynamique (ex. un `aria-expanded` jamais mis à jour). Ces warnings sont statiques et ne remplacent pas les tests runtime + manuels.

#### 3.7 Faire ÉCHOUER le build sur les warnings a11y

Deux leviers :

**a) Au build (vite-plugin-svelte)** via `compilerOptions.warningFilter` (introduit en Svelte 5) — mais `warningFilter` sert à *filtrer*, pas à faire échouer. Pour transformer en erreur, combinez avec `onwarn` du plugin ou levez une exception :

```js
// svelte.config.js
const a11yAsError = process.env.CI === 'true';
export default {
  onwarn(warning, defaultHandler) {
    if (a11yAsError && warning.code.startsWith('a11y_')) {
      throw new Error(`A11y (bloquant) : ${warning.code} — ${warning.message}`);
    }
    defaultHandler(warning);
  }
};
```
*Divergence à signaler* : un bug a été rapporté où `warningFilter` a cessé de fonctionner à partir de Svelte 5.10.0 (issue #14654) — vérifiez le comportement sur votre version exacte.

**b) En CI via `svelte-check`** : l'outil fournit `--threshold warning` (affiche warnings+erreurs) et `--compiler-warnings "code:error|ignore"` pour élever un warning précis en erreur, ainsi qu'un code de sortie non-nul. Exemple GitHub Actions :

```yaml
# .github/workflows/a11y.yml
name: a11y
on: [pull_request]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 22, cache: npm }
      - run: npm ci
      - run: npx svelte-check --threshold warning --compiler-warnings "a11y_click_events_have_key_events:error"
      - run: npx stylelint "src/**/*.{css,svelte}"
```

#### 3.8 Tests a11y automatisés

Pile 2026 : **axe-core** comme moteur, exposé via **`@axe-core/playwright`** (`AxeBuilder`) pour les tests end-to-end, ou **`vitest-axe`** pour les tests unitaires. Pour les composants Svelte 5, **`vitest-browser-svelte`** (mode navigateur de Vitest, via Playwright — pas jsdom) est la voie recommandée depuis 2025 ; `@testing-library/svelte` reste utilisable mais la communauté migre vers `vitest-browser-svelte`. **Attention** : Playwright 1.59 a retiré `@playwright/experimental-ct-svelte` (le component-testing Svelte de Playwright n'est plus supporté) — d'où l'importance de `vitest-browser-svelte`.

```js
import { render } from 'vitest-browser-svelte';
import { expect, test } from 'vitest';
import Tabs from './Tabs.svelte';

test('les onglets sont accessibles au clavier', async () => {
  const screen = render(Tabs, { tabs: sampleTabs });
  const first = screen.getByRole('tab', { selected: true });
  await first.element().focus();
  // ... simuler ArrowRight, vérifier aria-selected
});
```

**Limite majeure à documenter** : les outils automatiques ne détectent qu'une **partie** des problèmes. Le README officiel d'axe-core annonce « on average 57% of WCAG issues automatically » (étude Deque, 57,38 % sur 2 000+ audits), à comparer à la baseline industrielle historique de 20-30 %. Le test manuel au clavier et au lecteur d'écran (NVDA sur WebView2 Windows, VoiceOver sur WKWebView macOS, Orca sur WebKitGTK Linux) reste indispensable — et ces AT diffèrent selon le moteur WebView de Wails.

### 4. Raccourcis clavier vs navigation

#### 4.1 Pourquoi confisquer Tab globalement est un problème

Capter Tab au niveau global pour un raccourci applicatif entre en tension avec plusieurs critères WCAG 2.2 :
- **2.1.1 Keyboard** : tout doit rester opérable au clavier.
- **2.1.2 No Keyboard Trap** : l'utilisateur ne doit jamais être piégé — or détourner Tab peut empêcher de sortir d'une zone.
- **2.1.4 Character Key Shortcuts (niveau A)** : un raccourci à **caractère unique** doit pouvoir être **désactivé**, **remappé** (vers une combinaison avec modificateur), OU **actif seulement au focus** du composant. Tab n'est pas un « character key » au sens strict (c'est une touche non-imprimable), mais l'esprit du critère — et 2.1.1/2.1.2 — s'applique : un raccourci global qui casse la navigation standard est une régression d'accessibilité.
- **2.4.3 Focus Order**, **2.4.7 Focus Visible**, **2.4.11 Focus Not Obscured** : l'ordre et la visibilité du focus doivent rester cohérents.

**Recommandation** : ne captez JAMAIS Tab globalement. Limitez la capture à un **widget composite** précis (éditeur, grille) et seulement quand il a le focus. Évitez `role="application"` (il désactive le mode navigation des lecteurs d'écran et transfère TOUTES les touches à l'app — dangereux, à réserver à de vrais widgets type jeu/éditeur). Offrez toujours une **échappatoire** (Échap, ou Ctrl+Tab) et rendez le raccourci **désactivable/remappable** dans les préférences (satisfait 2.1.4).

#### 4.2 Comment les applications de référence gèrent cela

- **VS Code / Monaco Editor** : Tab insère une tabulation dans l'éditeur, mais un mode **« Tab moves focus »** basculable par **Ctrl+M** (Ctrl+Shift+M sur macOS) fait ressortir le focus ; un indicateur « Tab moves focus » s'affiche dans la barre d'état (configurable via `editor.tabFocusMode`). Pour naviguer entre parties de l'UI, **F6 / Shift+F6** (Focus Next/Previous Part). C'est le modèle à copier : Tab n'est capté que dans l'éditeur, jamais globalement, et un basculement explicite + indicateur visuel existe. À noter (doc VS Code) : les fichiers en lecture seule ne piègent jamais Tab, et pour les WebViews il est recommandé d'utiliser F6/Shift+F6.
- **CodeMirror 6** : Échap puis Tab pour sortir de l'éditeur.
- **Google Docs/Sheets** : mode compatible lecteur d'écran activable ; raccourcis dédiés.
- **Gmail** : raccourcis clavier **activables/désactivables** dans les paramètres, et actifs seulement dans la grille de messages (« x » sélectionne, « # » supprime — inactifs hors grille) — exemple canonique de conformité 2.1.4.
- **Slack** : **Ctrl+F6** pour naviguer entre régions.
- Navigateurs et applications natives Windows : **F6 / Ctrl+F6** pour cycler entre les régions/volets.

#### 4.3 Patterns à reprendre

- **Échap pour sortir d'un piège clavier** : dans tout widget qui capte des touches, Échap doit rendre le focus au conteneur parent / sortir du widget.
- **F6 / Ctrl+F6 pour cycler entre régions** : implémentez des **landmarks ARIA** (`<header>`, `<nav>`, `<main>`, `role="region"` avec `aria-label`) comme cibles, et un gestionnaire F6 qui déplace le focus vers le landmark suivant. C'est le modèle des navigateurs et des applications natives.
- **Indicateur visible** : comme VS Code, affichez l'état du mode (Tab capté vs Tab libre).
- **Remappage** : stockez les raccourcis dans les préférences (réactif via état global Svelte 5), permettez la désactivation.

## Recommandations

### Étape 1 — Fondations mesurables (semaines 1-2)
1. **Geler une baseline** : comptez les couleurs en dur (108), les warnings a11y du compilateur, et le poids du bundle i18n (65 %). Ce sont vos métriques de progrès.
2. **i18n lazy** : convertissez immédiatement les 9 imports statiques en `import.meta.glob('./locales/*.json')` lazy + 1 locale de repli statique. Gain rapide sur le bundle initial, sans changer de bibliothèque.
3. **Store de langue Svelte 5** : refactorez en objet `$state` avec getters/setters dans un `.svelte.js` (jamais de `$state` scalaire exporté-réassigné).

### Étape 2 — Tokens et thème (semaines 3-5)
4. **Source unique de vérité DTCG → Style Dictionary** (v4, en surveillant v5 pour le support complet 2025.10), générant CSS + JS/TS. Modèle three-tier (primitif/sémantique/composant).
5. **Migrer les 108 couleurs** vers des tokens sémantiques ; implémenter `color-scheme` + `light-dark()` + `data-theme` persisté + script anti-FOUC inline.
6. **Vérifier le contraste** de chaque token de texte (11 px = 4.5:1) dans les deux thèmes avec `colorjs.io`/Leonardo.
7. **Thèmes forced-colors (Windows/WebView2) et print** : blocs `@media` redéfinissant les tokens.
8. **Canvas** : consommer les tokens JS générés (pas de `getComputedStyle` en boucle) ; MutationObserver sur `data-theme` + `matchMedia` pour le re-render.

### Étape 3 — Accessibilité (semaines 6-9)
9. **Remplacer** les onglets maison par le pattern APG Tabs (roving tabindex), et la table triable par `<table>` sémantique + `aria-sort` + bouton dans `<th>` (PAS `role="grid"`).
10. **Supprimer le focusTrap maison** au profit de `<dialog>`+`showModal()` / `inert`.
11. **Régions `aria-live`** montées au démarrage (`role="status"` / `role="alert"`).
12. **Cesser de confisquer Tab globalement** : restreindre à un widget avec Échap + Ctrl+M-like + F6 entre landmarks + remappage/désactivation en préférences.

### Étape 4 — Verrouillage CI (continu)
13. **stylelint-declaration-strict-value** (`ignoreVariables: false`, `postcss-html` pour `.svelte`).
14. **svelte-check --threshold warning** + élévation ciblée de warnings a11y en erreurs.
15. **Tests** : `vitest-browser-svelte` + `@axe-core/playwright` ; tests de régression visuelle des deux thèmes.
16. **Budget de warnings (ratchet)** : voir ci-dessous.

### Budget de warnings en CI (stratégie du cliquet / ratchet)
Pour gérer la dette sans bloquer l'équipe :
- **Baseline gelée** : générez un instantané du nombre de warnings a11y/stylelint actuels par règle (ex. `svelte-check --output machine > baseline.json`).
- **Cliquet décroissant** : un script CI compare le nombre courant à la baseline ; **le build échoue si le total AUGMENTE**, réussit s'il stagne ou décroît. À chaque baisse, régénérez la baseline (elle ne peut que descendre).
- **Zéro sur les nouvelles règles** : pour toute règle nouvellement activée, seuil = 0 dès le départ (pas de dette autorisée).
- **Nouveaux fichiers stricts** : appliquez le seuil zéro aux fichiers modifiés/ajoutés dans la PR (diff-based), tout en tolérant la dette dans le legacy.

```yaml
# extrait GitHub Actions — cliquet
- run: node scripts/a11y-ratchet.mjs   # sort code !=0 si warnings > baseline
```
```js
// scripts/a11y-ratchet.mjs (principe)
import { readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
const baseline = JSON.parse(readFileSync('a11y-baseline.json', 'utf8')).total;
const out = execSync('npx svelte-check --output machine', { encoding: 'utf8' });
const current = (out.match(/WARNING/g) ?? []).length;
if (current > baseline) {
  console.error(`❌ Régression a11y : ${current} > baseline ${baseline}`);
  process.exit(1);
}
console.log(`✅ ${current} ≤ ${baseline}`);
```

### Seuils qui changeraient ces recommandations
- Si Wails embarque un **WebView2/WebKitGTK antérieur à ~2024**, `light-dark()` peut manquer → repli sur la double déclaration `@media (prefers-color-scheme)`.
- Si le **poids i18n reste critique après le lazy-loading JSON**, migrez vers Paraglide JS (tree-shaking par message).
- Si le découpage **par locale** devient indispensable et que l'option expérimentale de Paraglide vous bloque, restez sur JSON lazy (svelte-i18n/typesafe-i18n).

## Liste d'outils et versions (2026) et compatibilité

| Outil | Version 2026 | Notes de compatibilité |
|---|---|---|
| Vite | 7.x (option Rolldown) | `manualChunks` OK en Rollup ; `advancedChunks` sous Rolldown |
| Svelte | 5.x (runes) | `warningFilter` — vérifier régression ≥5.10.0 |
| svelte-check | courant | `--threshold`, `--compiler-warnings`, code sortie CI |
| eslint-plugin-svelte | courant | règle `svelte/valid-compile`, respecte `warningFilter` |
| stylelint + stylelint-declaration-strict-value | courant | `postcss-html` + `stylelint-config-html` pour `.svelte` |
| Style Dictionary | v4 (DTCG première classe) ; v5 (support 2025.10 en cours) | Générer CSS + JS/TS |
| DTCG spec | v2025.10 (stable, 28 oct. 2025) | `$value`/`$type`, références |
| axe-core / @axe-core/playwright | courant | moteur de test a11y (~57 % des critères) |
| Playwright | 1.59+ | ⚠️ retrait de `@playwright/experimental-ct-svelte` |
| vitest-browser-svelte | courant | testing Svelte 5 en vrai navigateur |
| focus-trap + tabbable | courant | fallback si `<dialog>` insuffisant |
| Paraglide JS (@inlang/paraglide-js) | 2.25.0 | compilateur, Vite pur, per-locale expérimental |
| typesafe-i18n | 5.27.1 (fév. 2026) | nouvelle gouvernance codingcommons |
| svelte-i18n | 4.0.1 (~2023) | figée, non runes-native |
| i18next / i18next-resources-to-backend | 26.4.1 / courant | lazy via dynamic import (clunky en Vite) |
| @formatjs/intl | 4.1.19 | cœur agnostique, sans binding Svelte |
| @bramus/style-observer | courant | observer les custom properties (canvas) |
| DOMPurify | courant | sanitiser le HTML d'aide injecté via `{@html}` |

## Caveats
- **Spécificités WebView (Wails)** : le moteur diffère par OS (WebView2/Chromium sur Windows, WebKitGTK sur Linux, WKWebView sur macOS). Le support de `light-dark()`, `checkVisibility()`, `content-visibility` et `forced-colors` dépend de la **version exacte** embarquée — testez sur chaque plateforme cible. Le DevTools complet n'est pas disponible partout (surtout WKWebView/WebKitGTK), ce qui complique le débogage a11y. Le préchargement `modulepreload` peut mal se comporter avec les schémas d'URL de Wails (`wails://`, `file://`) — vérifiez ou désactivez.
- **Données susceptibles d'évoluer** : les versions npm (Paraglide 2.25.0, i18next 26.4.1, @formatjs/intl 4.1.19, typesafe-i18n 5.27.1) sont des instantanés qui s'incrémentent souvent. Le support complet du DTCG 2025.10 dans Style Dictionary v5 était « en cours » à la date de rédaction.
- **Divergences signalées** : (1) le « jusqu'à 70 % » de Paraglide vient de son propre benchmark (l'exemple concret 47 Ko vs 205 Ko et la stabilité du poids avec le nombre de messages sont plus fiables que le pourcentage seul) ; (2) `warningFilter` aurait régressé à partir de Svelte 5.10.0 (issue #14654) — à valider sur votre version ; (3) la couverture des tests a11y automatiques (~57 % selon Deque/axe-core, 20-30 % en baseline historique) varie selon les études et la méthodologie.
- **Le focus trap natif de `<dialog>` laisse le focus atteindre le chrome du navigateur** — c'est voulu (décision W3C APA), pas un bug ; ne réintroduisez pas un trap maison pour « corriger » cela.
- **Aucun outil automatique ne remplace le test manuel** au clavier et au lecteur d'écran sur chaque moteur WebView.