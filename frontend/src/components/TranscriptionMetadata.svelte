<!--
  TranscriptionMetadata — l'en-tête du brouillon, saisissable à tout moment.

  Les champs de `fonctionnel.md` §1.1 marqués « plus tard » : les deux noms,
  l'événement, le lieu, la ronde, la date, le transcripteur, le tournoi
  auquel le Match sera rattaché à l'enregistrement. Rien n'y est bloquant
  (T3.1) : un brouillon sans noms s'enregistre et s'exporte, avec des en-têtes
  vides, exactement comme un import en produirait.

  Le volet est un CLIENT du moteur, comme le panneau qui le porte (ADR-0045
  règle 9) : il ne dérive rien, il envoie deux gestes.

  — `set_header` écrit la partie descriptive de l'en-tête, et elle seule : la
    longueur, les règles de session et le `match_id` qu'un premier
    enregistrement a posé lui survivent (transcript.mergeHeader). Il part au
    COMMIT d'un champ — sa validation, sa perte de focus — et non à chaque
    frappe : chaque geste est une écriture de ligne et un Replay complet.

  — `swap_players` échange les noms, donne toutes les Actions au camp d'en
    face et retourne le plateau. Le geste ne touche à aucune Action : c'est le
    même match, lu de l'autre côté.

  La longueur du match est ici aussi (T3.2), parce que c'est le même en-tête :
  la changer en cours de transcription relance le Replay entier — le score away
  de chaque position, la partie Crawford, le référentiel (argent si 0) et la
  marque « au-delà de la fin » sur les Actions postérieures à la victoire. Rien
  n'est jamais supprimé. Passer en argent fait apparaître les règles de session
  (Jacoby, beaver), et repasser en match les masque sans les perdre : elles
  restent dans le document, où aucune Position ne les lit (ADR-0028).

  Les deux noms et le tournoi passent par EntityAutocomplete, le même widget
  que les panneaux Matchs et Tournois : la liste des joueurs et celle des
  tournois de la base y sont filtrées, au clavier comme à la souris, et un nom
  absent de la liste reste tel qu'il a été tapé.
-->
<script>
    import EntityAutocomplete from './EntityAutocomplete.svelte';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { GetAllPlayerNames, GetAllTournaments } from '../../wailsjs/go/database/Database.js';

    let {
        /** L'en-tête du document, tel que le moteur vient de le renvoyer. */
        header = {},
        /** Envoie un geste au moteur ; le panneau tient la file. */
        apply = async () => {},
        /** Le panneau est occupé : les champs restent lisibles, pas modifiables. */
        busy = false
    } = $props();

    let players = $state([]);
    let tournaments = $state([]);

    // Ce que les champs montrent. Une copie locale, et non l'en-tête lui-même :
    // un champ en cours de frappe ne doit pas être réécrit par le document que
    // le geste précédent a renvoyé, et un en-tête qui change ailleurs (annuler,
    // inverser les joueurs) doit malgré tout se voir ici.
    let form = $state(blank());
    let synced = '';

    function blank() {
        return { player1: '', player2: '', event: '', location: '', round: '', date: '', transcriber: '', tournament: '' };
    }

    /**
     * La date de l'en-tête, en `YYYY-MM-DD` pour un `<input type="date">`.
     *
     * Go sérialise un `time.Time` en RFC 3339 et n'omet jamais un zéro
     * (`omitempty` ne dit rien d'une structure) : l'an 1 est la date vide.
     */
    function dateFieldOf(value) {
        if (typeof value !== 'string' || value.length < 10) return '';
        return value.startsWith('0001-01-01') ? '' : value.slice(0, 10);
    }

    /** L'inverse : un jour saisi devient l'instant RFC 3339 que Go relit. */
    function dateHeaderOf(day) {
        return day ? `${day}T00:00:00Z` : undefined;
    }

    function tournamentNameOf(id) {
        if (!id) return '';
        return tournaments.find((row) => row.id === id)?.name ?? '';
    }

    /** L'identifiant du tournoi dont le nom a été tapé, ou null. */
    function tournamentIdOf(name) {
        const wanted = (name ?? '').trim().toLowerCase();
        if (!wanted) return null;
        return tournaments.find((row) => (row.name ?? '').trim().toLowerCase() === wanted)?.id ?? null;
    }

    // La signature de l'en-tête reçu : tant qu'elle ne change pas, les champs
    // sont ceux de l'utilisateur et personne ne les réécrit.
    let incoming = $derived(
        JSON.stringify([
            header?.player1 ?? '',
            header?.player2 ?? '',
            header?.event ?? '',
            header?.location ?? '',
            header?.round ?? '',
            header?.date ?? '',
            header?.transcriber ?? '',
            tournamentNameOf(header?.tournament_id)
        ])
    );

    $effect(() => {
        const key = incoming;
        if (key === synced) return;
        synced = key;
        form = {
            player1: header?.player1 ?? '',
            player2: header?.player2 ?? '',
            event: header?.event ?? '',
            location: header?.location ?? '',
            round: header?.round ?? '',
            date: dateFieldOf(header?.date),
            transcriber: header?.transcriber ?? '',
            tournament: tournamentNameOf(header?.tournament_id)
        };
    });

    // La longueur est un texte, comme dans le formulaire de création : `0` — la
    // partie d'argent — est la seule réponse qu'un champ numérique lié à un
    // nombre rendrait indiscernable d'un champ vide.
    let lengthField = $state('');
    let lengthSynced = null;
    let matchLength = $derived(header?.match_length ?? 0);
    let isMoney = $derived(matchLength === 0);

    $effect(() => {
        const length = matchLength;
        if (length === lengthSynced) return;
        lengthSynced = length;
        lengthField = String(length);
    });

    async function loadLists() {
        try {
            const [names, tours] = await Promise.all([GetAllPlayerNames(), GetAllTournaments()]);
            players = names ?? [];
            tournaments = tours ?? [];
        } catch (err) {
            logger.error('Failed to load the players and tournaments of the metadata pane:', err);
        }
    }

    $effect(() => {
        loadLists();
    });

    /** Écrit l'en-tête descriptif ; la longueur et le `match_id` n'y sont pas. */
    function commit() {
        if (busy) return;
        const next = {
            player1: form.player1.trim(),
            player2: form.player2.trim(),
            event: form.event.trim(),
            location: form.location.trim(),
            round: form.round.trim(),
            transcriber: form.transcriber.trim(),
            date: dateHeaderOf(form.date)
        };
        const tournamentId = tournamentIdOf(form.tournament);
        if (tournamentId) next.tournament_id = tournamentId;
        apply({ Kind: 'set_header', Header: next });
    }

    /**
     * Change la longueur du match. Le Replay entier suit, dans le moteur : le
     * volet ne dérive ni score, ni Crawford, ni fin de match.
     *
     * Les règles de session partent avec elle, telles que le document les
     * porte : c'est ce qui les rend au retour d'un aller en match.
     */
    function commitLength() {
        if (busy) return;
        const text = lengthField.trim();
        if (!/^\d+$/.test(text)) {
            lengthField = String(matchLength);
            return;
        }
        const length = Number(text);
        if (length === matchLength) return;
        applyLength(length, header?.jacoby ?? false, header?.beaver ?? false);
    }

    /** Les règles de la session : elles ne se posent qu'en argent (ADR-0028). */
    function commitRules(jacoby, beaver) {
        if (busy) return;
        applyLength(matchLength, jacoby, beaver);
    }

    function applyLength(length, jacoby, beaver) {
        apply({ Kind: 'set_length', HasLength: true, MatchLength: length, HasRules: true, Jacoby: jacoby, Beaver: beaver });
    }

    /**
     * Un item choisi dans la liste. EntityAutocomplete prévient AVANT d'écrire
     * le nom dans le champ (`pick` appelle `onSelect` puis remplit `value`), si
     * bien qu'un commit branché tel quel sur `onSelect` enverrait le texte
     * d'avant le clic. Le volet écrit donc le nom lui-même, puis commet.
     */
    function pickInto(field, name) {
        form[field] = String(name ?? '');
        commit();
    }

    function swapPlayers() {
        if (busy) return;
        apply({ Kind: 'swap_players' });
    }
</script>

<div class="metadata" data-testid="transcription-metadata">
    <div class="row">
        <label class="field">
            <span>{$t('transcription.player1')}</span>
            <EntityAutocomplete
                bind:value={form.player1}
                items={players}
                label={(name) => String(name)}
                variant="field"
                placeholder={$t('transcription.unnamed')}
                onSelect={(name) => pickInto('player1', name)}
                onSubmit={commit}
                onDismiss={commit}
            />
        </label>
        <label class="field">
            <span>{$t('transcription.player2')}</span>
            <EntityAutocomplete
                bind:value={form.player2}
                items={players}
                label={(name) => String(name)}
                variant="field"
                placeholder={$t('transcription.unnamed')}
                onSelect={(name) => pickInto('player2', name)}
                onSubmit={commit}
                onDismiss={commit}
            />
        </label>
        <button class="meta-btn" type="button" onclick={swapPlayers} disabled={busy} title={$t('transcription.swapPlayersTooltip')}>
            {$t('transcription.swapPlayers')}
        </button>
    </div>

    <div class="row">
        <label class="field">
            <span>{$t('transcription.eventLabel')}</span>
            <input type="text" bind:value={form.event} onchange={commit} disabled={busy} />
        </label>
        <label class="field">
            <span>{$t('transcription.locationLabel')}</span>
            <input type="text" bind:value={form.location} onchange={commit} disabled={busy} />
        </label>
        <label class="field narrow">
            <span>{$t('transcription.roundLabel')}</span>
            <input type="text" bind:value={form.round} onchange={commit} disabled={busy} />
        </label>
        <label class="field narrow">
            <span>{$t('transcription.dateLabel')}</span>
            <input type="date" bind:value={form.date} onchange={commit} disabled={busy} />
        </label>
    </div>

    <div class="row">
        <label class="field">
            <span>{$t('transcription.transcriberLabel')}</span>
            <input type="text" bind:value={form.transcriber} onchange={commit} disabled={busy} />
        </label>
        <label class="field">
            <span>{$t('transcription.tournamentLabel')}</span>
            <EntityAutocomplete
                bind:value={form.tournament}
                items={tournaments}
                variant="field"
                placeholder={$t('transcription.tournamentNone')}
                onSelect={(entry) => pickInto('tournament', entry?.name)}
                onSubmit={commit}
                onDismiss={commit}
            />
        </label>
        <label class="field narrow">
            <span>{$t('transcription.matchLength')}</span>
            <input type="text" inputmode="numeric" bind:value={lengthField} onchange={commitLength} disabled={busy} title={$t('transcription.lengthTooltip')} />
        </label>
        {#if isMoney}
            <!-- Les règles de la session n'existent qu'en argent : elles sont
                 posées sur chaque Position, jamais sur une Action (ADR-0028). -->
            <label class="rule">
                <input type="checkbox" checked={header?.jacoby ?? false} onchange={(e) => commitRules(e.currentTarget.checked, header?.beaver ?? false)} disabled={busy} />
                {$t('transcription.jacoby')}
            </label>
            <label class="rule">
                <input type="checkbox" checked={header?.beaver ?? false} onchange={(e) => commitRules(header?.jacoby ?? false, e.currentTarget.checked)} disabled={busy} />
                {$t('transcription.beaver')}
            </label>
        {/if}
    </div>

    <p class="hint">{$t('transcription.metadataHint')}</p>
    <p class="hint">{$t('transcription.lengthHint')}</p>
</div>

<style>
    .metadata {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
    }

    .row {
        display: flex;
        flex-wrap: wrap;
        align-items: flex-end;
        gap: var(--space-2);
    }

    .field {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        flex: 1 1 12em;
        min-width: 8em;
        color: var(--color-text-muted);
    }

    .field.narrow {
        flex: 0 1 7em;
        min-width: 5em;
    }

    .field input {
        padding: var(--space-1);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
    }

    .rule {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
        color: var(--color-text);
    }

    .meta-btn {
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .meta-btn:hover:not(:disabled) {
        background: var(--color-surface-alt);
    }

    .meta-btn:disabled {
        color: var(--color-text-muted);
        cursor: default;
    }

    .hint {
        margin: 0;
        color: var(--color-text-muted);
    }
</style>
