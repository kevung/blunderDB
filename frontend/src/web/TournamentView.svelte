<script>
    // Le quatrième verbe du front web, en lecture seule (ADR-0097) : afficher un tournoi. La page
    // montrée est celle que la Direction écrit dans son dossier d'affichage (arbre, appariements,
    // résultats, classement), rendue par le moteur ; le front n'a pas de second dessinateur de
    // tournoi. Elle est sondée par lecture conditionnelle : un 304 ne coûte rien.
    import { callConditional } from './api.js';

    /** @type {{ tournamentId: number }} */
    let { tournamentId } = $props();

    // Assez court pour qu'un résultat saisi apparaisse avant que le joueur ait rangé son
    // téléphone, assez long pour qu'une salle de trente téléphones ne charge pas le portable du
    // directeur.
    const POLL_MS = 10000;

    // La page du moteur est pensée pour un écran large : sur un téléphone, ses tableaux
    // défileraient avec le cadre entier. Ils défilent seuls, le reste se replie. Son rafraîchissement
    // propre est retiré : le cadre est sondé d'ici, et la balise est refusée dans un cadre isolé.
    const FIT_CSS = 'table{display:block;max-width:100%;overflow-x:auto}img,svg{max-width:100%;height:auto}body{overflow-wrap:anywhere}';

    /** @param {string} page */
    function fitted(page) {
        const body = page.replace(/<meta[^>]+http-equiv=["']?refresh["']?[^>]*>/gi, '');
        const style = `<style>${FIT_CSS}</style>`;
        return /<\/head>/i.test(body) ? body.replace(/<\/head>/i, `${style}</head>`) : style + body;
    }

    /** @param {Date} d */
    const clock = (d) => d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });

    let html = $state('');
    // Ce que dit la page quand elle ne peut pas montrer le tournoi : un texte clair, jamais le
    // message de la machine.
    let problem = $state('');
    // L'heure du dernier sondage réussi, même sans changement : une page calme se distingue
    // d'une page figée.
    let checkedAt = $state(/** @type {Date | null} */ (null));
    let etag = '';
    /** @type {HTMLIFrameElement | null} */
    let frame = $state(null);
    let frameHeight = $state(0);

    /** @param {any} e */
    function explain(e) {
        if (e?.readable) return e.message;
        if (e?.status === 404 || e?.status === 410) return "Ce tournoi n'est pas affiché ici.";
        if (typeof e?.status === 'number') return "L'affichage n'a pas pu être chargé.";
        return checkedAt ? `Affichage interrompu : données de ${clock(checkedAt)}.` : "Affichage indisponible : l'appareil qui le sert ne répond pas.";
    }

    async function refresh() {
        try {
            const res = await callConditional('directions.pageHtml', { tournamentId }, etag);
            problem = '';
            checkedAt = new Date();
            if (!res.changed) return;
            etag = res.etag;
            html = fitted(res.body?.html || '');
        } catch (e) {
            problem = explain(e);
        }
    }

    // Le cadre prend la hauteur de son contenu : un seul défilement, celui de la page. Il peut lire
    // le document parce que le cadre n'a pas l'origine séparée mais n'exécute aucun script.
    function fit() {
        const doc = frame?.contentDocument;
        if (doc?.documentElement) frameHeight = doc.documentElement.scrollHeight;
    }

    $effect(() => {
        etag = '';
        html = '';
        if (!tournamentId) {
            problem = "Ce tournoi n'est pas affiché ici.";
            return undefined;
        }
        refresh();
        const timer = setInterval(refresh, POLL_MS);
        return () => clearInterval(timer);
    });
</script>

<svelte:window onresize={fit} />

{#if problem}
    <p class="error" role="alert">{problem}</p>
{/if}
{#if html}
    <!-- La page vient du moteur mais porte des noms de joueurs saisis : elle est isolée dans un
         cadre sandbox sans droit d'exécution (aucun script), si bien que rien de ce qu'elle
         contient ne peut agir sur cette page ni sur le démon. -->
    <iframe class="tournament" title="Tournoi" sandbox="allow-same-origin" srcdoc={html} bind:this={frame} onload={fit} style:height={frameHeight ? `${frameHeight}px` : undefined}></iframe>
    {#if checkedAt}
        <p class="note">Lecture seule · vérifié à {clock(checkedAt)}</p>
    {/if}
{:else if !problem}
    <p class="note">Chargement du tournoi…</p>
{/if}

<style>
    .tournament {
        display: block;
        width: 100%;
        min-height: 12rem;
        border: 1px solid var(--color-border);
        /* La page du moteur porte ses propres couleurs, pensées sur fond clair. */
        background: white;
    }

    .error {
        color: var(--color-danger);
    }

    .note {
        color: var(--color-text-muted);
    }
</style>
