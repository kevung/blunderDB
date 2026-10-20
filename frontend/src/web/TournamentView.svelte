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

    let html = $state('');
    let error = $state('');
    let updatedAt = $state('');
    let etag = '';

    async function refresh() {
        try {
            const res = await callConditional('directions.pageHtml', { tournamentId }, etag);
            error = '';
            if (!res.changed) return;
            etag = res.etag;
            html = res.body?.html || '';
            updatedAt = new Date().toLocaleTimeString();
        } catch (e) {
            error = String(e.message || e);
        }
    }

    $effect(() => {
        etag = '';
        html = '';
        refresh();
        const timer = setInterval(refresh, POLL_MS);
        return () => clearInterval(timer);
    });
</script>

{#if error}
    <p class="error">{error}</p>
{/if}
{#if html}
    <!-- La page vient du moteur mais porte des noms de joueurs saisis : elle est isolée dans un
         cadre sandbox sans aucun droit (ni script, ni origine), si bien que rien de ce qu'elle
         contient ne peut atteindre cette page ni le démon. -->
    <iframe class="tournament" title="Tournoi" sandbox="" srcdoc={html}></iframe>
    <p class="note">Lecture seule · mise à jour {updatedAt}</p>
{:else if !error}
    <p class="note">Chargement du tournoi…</p>
{/if}

<style>
    .tournament {
        width: 100%;
        height: calc(100vh - 5rem);
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
