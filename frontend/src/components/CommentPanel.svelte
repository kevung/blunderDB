<script>
    import { tick } from 'svelte';
    import { logger } from '../utils/logger.js';
    let { visible = false } = $props();

    import { currentPositionIndexStore } from '../stores/uiStore';
    import { positionStore } from '../stores/positionStore';
    import { GetCommentsByPosition, SearchComments, LoadAnalysis, LoadPosition, AddComment, UpdateCommentEntry, TrashCommentEntry, Tags, RecommendedTags } from '../../wailsjs/go/database/Database.js';
    import { MODAL, openModal } from '../stores/uiStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { t } from '../i18n';
    import { formatDateTime } from '../utils/format.js';

    let allComments = $state([]);
    let searchQuery = $state('');
    let displayedComments = $state([]);
    let feedEl;
    let promptEl;
    let editingCommentId = $state(null);
    let editingText = $state('');
    let promptText = $state('');

    // Autocomplétion des tags (#265, fiche I.9). Un tag est un `#mot` dans la
    // prose : rien ne le déclare, et la suggestion ne déclare rien non plus.
    // Elle propose ce que CETTE base utilise déjà — pour qu'on réécrive
    // `#backgame` comme la dernière fois plutôt que `#back-game` — et, à
    // défaut, le vocabulaire recommandé, qui vient de la littérature et pas
    // d'un calcul de blunderDB.
    /** @type {string[]} */
    let tagVocabulary = $state([]);
    /** @type {string[]} */
    let tagSuggestions = $state([]);
    let tagSuggestionIndex = $state(0);
    let tagWordStart = -1;

    async function loadTagVocabulary() {
        try {
            const [used, recommended] = await Promise.all([Tags(), RecommendedTags()]);
            // Un objet plutôt qu'un Set : la règle svelte/prefer-svelte-reactivity
            // interdit un Set mutable ici, et rien de tout ceci n'a besoin
            // d'être réactif — c'est une déduplication locale, jetée à la
            // ligne suivante.
            /** @type {Record<string, boolean>} */
            const seen = {};
            const out = [];
            for (const entry of used || []) {
                if (!seen[entry.tag]) {
                    seen[entry.tag] = true;
                    out.push(entry.tag);
                }
            }
            for (const tag of recommended || []) {
                if (!seen[tag]) {
                    seen[tag] = true;
                    out.push(tag);
                }
            }
            tagVocabulary = out;
        } catch (error) {
            logger.error('could not read the tag vocabulary:', error);
            tagVocabulary = [];
        }
    }

    /**
     * Met à jour les suggestions d'après le mot en cours de frappe. Le mot est
     * délimité par des espaces : c'est la même délimitation que la recherche
     * par tag, donc ce qui est proposé ici est ce qui sera trouvé là.
     */
    function refreshTagSuggestions() {
        const el = promptEl;
        if (!el) return;
        const caret = el.selectionStart ?? promptText.length;
        const before = promptText.slice(0, caret);
        const start = Math.max(before.lastIndexOf(' '), before.lastIndexOf('\n')) + 1;
        const word = before.slice(start);
        if (!word.startsWith('#') || word.length < 1) {
            tagSuggestions = [];
            return;
        }
        tagWordStart = start;
        const prefix = word.toLowerCase();
        tagSuggestions = tagVocabulary.filter((t) => t.startsWith(prefix) && t !== prefix).slice(0, 8);
        tagSuggestionIndex = 0;
    }

    /** @param {string} tag */
    function applyTagSuggestion(tag) {
        const el = promptEl;
        const caret = el?.selectionStart ?? promptText.length;
        promptText = promptText.slice(0, tagWordStart) + tag + ' ' + promptText.slice(caret);
        tagSuggestions = [];
        void tick().then(() => {
            if (!el) return;
            const pos = tagWordStart + tag.length + 1;
            el.focus();
            el.setSelectionRange(pos, pos);
        });
    }

    $effect(() => {
        if (visible) {
            loadComments();
            loadTagVocabulary();
            if (promptEl) promptEl.focus();
        }
    });
    // Reload comments when displayed position changes
    $effect(() => {
        if (visible && $positionStore && $positionStore.id) {
            loadComments();
        }
    });
    $effect(() => {
        if (searchQuery.trim()) {
            filterComments(searchQuery.trim());
        } else {
            displayedComments = allComments;
        }
    });

    // Provenance badge (#263). A comment carries where its text came from:
    // 'user' (typed here), 'xg'/'gnubg'/'bgf' (lifted out of an imported file)
    // or 'unknown' (written before the column existed). The user's own notes
    // get no badge — they are the norm, and marking every one of them would be
    // noise; only a note somebody else wrote is worth naming.
    function originLabel(origin) {
        switch (origin) {
            case 'xg':
                return $t('comment.originXG');
            case 'gnubg':
                return $t('comment.originGnuBG');
            case 'bgf':
                return $t('comment.originBGF');
            case 'unknown':
                return $t('comment.originUnknown');
            default:
                return '';
        }
    }

    function originTitle(origin) {
        switch (origin) {
            case 'xg':
                return $t('comment.originTitleXG');
            case 'gnubg':
                return $t('comment.originTitleGnuBG');
            case 'bgf':
                return $t('comment.originTitleBGF');
            case 'unknown':
                return $t('comment.originTitleUnknown');
            default:
                return '';
        }
    }

    async function loadComments() {
        // Only update allComments here; the search $effect below owns
        // displayedComments. Reading allComments in the same synchronous effect
        // pass that writes it (the no-DB / position-id-0 case takes no await,
        // so it stays synchronous) made the effect read-and-write the same
        // state, triggering an infinite update loop (effect_update_depth_exceeded).
        try {
            const pos = $positionStore;
            if (pos && pos.id) {
                allComments = (await GetCommentsByPosition(pos.id)) || [];
            } else {
                allComments = [];
            }
        } catch (error) {
            logger.error('Error loading comments:', error);
            allComments = [];
        }
    }

    async function filterComments(q) {
        try {
            displayedComments = (await SearchComments(q)) || [];
        } catch (_error) {
            displayedComments = [];
        }
    }

    async function navigateToComment(comment) {
        try {
            const position = await LoadPosition(comment.positionId);
            if (position) {
                positionStore.set(position);
                currentPositionIndexStore.set(-1);
                try {
                    const analysis = await LoadAnalysis(comment.positionId);
                    if (analysis) {
                        analysisStore.set(analysis);
                    }
                } catch (_e) {
                    /* ignored */
                }
                selectedMoveStore.set(null);
            }
        } catch (error) {
            logger.error('Error navigating to comment position:', error);
        }
    }

    async function addNewComment() {
        const pos = $positionStore;
        if (!pos || !pos.id) return;
        const text = promptText.trim();
        if (!text) return;
        try {
            await AddComment(pos.id, text);
            promptText = '';
            await loadComments();
            // Scroll feed to top (newest first)
            if (feedEl) feedEl.scrollTop = 0;
        } catch (error) {
            logger.error('Error adding comment:', error);
        }
    }

    function handlePromptKeyDown(event) {
        // La liste de suggestions capte d'abord les touches qui la
        // concernent — et seulement celles-là : tout le reste continue vers
        // la saisie, y compris Entrée quand aucune suggestion n'est ouverte.
        if (tagSuggestions.length > 0) {
            if (event.key === 'ArrowDown') {
                event.preventDefault();
                event.stopPropagation();
                tagSuggestionIndex = (tagSuggestionIndex + 1) % tagSuggestions.length;
                return;
            }
            if (event.key === 'ArrowUp') {
                event.preventDefault();
                event.stopPropagation();
                tagSuggestionIndex = (tagSuggestionIndex - 1 + tagSuggestions.length) % tagSuggestions.length;
                return;
            }
            if (event.key === 'Tab' || (event.key === 'Enter' && !event.shiftKey)) {
                event.preventDefault();
                event.stopPropagation();
                applyTagSuggestion(tagSuggestions[tagSuggestionIndex]);
                return;
            }
            if (event.key === 'Escape') {
                event.preventDefault();
                event.stopPropagation();
                tagSuggestions = [];
                return;
            }
        }
        if (event.key === 'Enter' && !event.shiftKey) {
            event.stopPropagation();
            event.preventDefault();
            addNewComment();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            event.currentTarget.blur();
        }
    }

    async function startEditComment(comment) {
        editingCommentId = comment.id;
        editingText = comment.text;
        // Move focus into the edit textarea so keystrokes go to the field and
        // don't leak to global keyboard shortcuts (board navigation, etc.).
        await tick();
        const el = document.querySelector('.msg-edit-input');
        if (el) {
            el.focus();
            el.setSelectionRange(el.value.length, el.value.length);
        }
    }

    async function saveEditedComment(comment) {
        editingCommentId = null;
        if (editingText !== comment.text) {
            try {
                await UpdateCommentEntry(comment.id, editingText);
                await loadComments();
            } catch (error) {
                logger.error('Error saving edited comment:', error);
            }
        }
    }

    function handleEditKeyDown(event, comment) {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.stopPropagation();
            event.preventDefault();
            saveEditedComment(comment);
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            editingCommentId = null;
        }
    }

    async function deleteComment(comment, event) {
        event.stopPropagation();
        try {
            // Through the trash (#285): restorable from the `trash` command.
            await TrashCommentEntry(comment.id);
            await loadComments();
        } catch (error) {
            logger.error('Error deleting comment:', error);
        }
    }

    // Reactive formatter: depends on $t so labels re-render on language change.
    let formatDate = $derived((dateStr) => {
        if (!dateStr) return '';
        try {
            // SQLite CURRENT_TIMESTAMP returns "YYYY-MM-DD HH:MM:SS" which some
            // JS engines can't parse. Replace the space with 'T' and add 'Z' for UTC.
            const normalized = dateStr.includes('T') ? dateStr : dateStr.replace(' ', 'T') + 'Z';
            const d = new Date(normalized);
            const now = new Date();
            const diffMs = now.getTime() - d.getTime();
            const diffMin = Math.floor(diffMs / 60000);
            if (diffMin < 1) return $t('comment.justNow');
            if (diffMin < 60) return $t('comment.minutesAgo', { n: diffMin });
            const diffHr = Math.floor(diffMin / 60);
            if (diffHr < 24) return $t('comment.hoursAgo', { n: diffHr });
            return formatDateTime(d, { day: '2-digit', month: '2-digit', year: '2-digit', hour: '2-digit', minute: '2-digit' });
        } catch {
            return '';
        }
    });

    function handleSearchKeyDown(event) {
        if (event.key === 'Escape') {
            event.stopPropagation();
            event.currentTarget.blur();
        }
    }
</script>

<div class="comment-panel">
    <!-- Search bar -->
    <div class="search-strip">
        <span class="search-icon">⌕</span>
        <input type="text" bind:value={searchQuery} placeholder={$t('comment.searchPlaceholder')} onkeydown={handleSearchKeyDown} class="search-input" />
        {#if searchQuery}
            <button
                class="clear-btn"
                onclick={() => {
                    searchQuery = '';
                }}>×</button
            >
        {/if}
    </div>

    <!-- Message feed -->
    <div class="feed" bind:this={feedEl}>
        {#if displayedComments.length === 0}
            <div class="empty-msg">{searchQuery.trim() ? $t('comment.noMatches') : $t('comment.noComments')}</div>
        {:else}
            {#each displayedComments as comment (comment.id)}
                {#if editingCommentId === comment.id}
                    <div class="msg editing">
                        <textarea class="msg-edit-input" bind:value={editingText} onkeydown={(e) => handleEditKeyDown(e, comment)} onblur={() => saveEditedComment(comment)} rows="2"></textarea>
                    </div>
                {:else}
                    <div class="msg" role="button" tabindex="-1" onclick={() => navigateToComment(comment)} onkeydown={() => {}}>
                        <div class="msg-header">
                            <span class="msg-date"
                                >{comment.modifiedAt && comment.modifiedAt !== comment.createdAt
                                    ? formatDate(comment.modifiedAt) + ' ' + $t('comment.editedSuffix')
                                    : formatDate(comment.createdAt)}</span
                            >
                            <!-- Provenance (#263). Only shown for a note the user did NOT
                                 write: their own comments are the norm, and a badge on every
                                 one of them would be noise. -->
                            {#if originLabel(comment.origin)}
                                <span class="msg-origin" title={originTitle(comment.origin)}>{originLabel(comment.origin)}</span>
                            {/if}
                        </div>
                        <div class="msg-text">{comment.text}</div>
                        <div class="msg-footer">
                            <button
                                class="msg-action msg-edit"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    (() => startEditComment(comment))();
                                }}
                                title={$t('common.edit')}>✎</button
                            >
                            <button
                                class="msg-action msg-delete"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    ((e) => deleteComment(comment, e))(e);
                                }}
                                title={$t('common.delete')}>×</button
                            >
                        </div>
                    </div>
                {/if}
            {/each}
        {/if}
    </div>

    <!-- Prompt -->
    <div class="prompt">
        {#if tagSuggestions.length > 0}
            <ul class="tag-suggestions" role="listbox" aria-label={$t('tags.suggestionsLabel')}>
                {#each tagSuggestions as tag, i (tag)}
                    <li>
                        <button type="button" class="tag-suggestion" class:selected={i === tagSuggestionIndex} onmousedown={(e) => e.preventDefault()} onclick={() => applyTagSuggestion(tag)}>
                            {tag}
                        </button>
                    </li>
                {/each}
            </ul>
        {/if}
        <textarea
            id="commentTextArea"
            bind:this={promptEl}
            bind:value={promptText}
            placeholder={$t('comment.promptPlaceholder')}
            onkeydown={handlePromptKeyDown}
            oninput={refreshTagSuggestions}
            onblur={() => (tagSuggestions = [])}
            rows="2"></textarea>
        <button type="button" class="tag-vocabulary-button" onclick={() => openModal(MODAL.TAGS)} title={$t('tags.title')}>#</button>
    </div>
</div>

<style>
    .tag-suggestions {
        position: absolute;
        bottom: 100%;
        left: 0;
        list-style: none;
        margin: 0 0 0.2em 0;
        padding: 0.2em;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        z-index: 10;
        max-height: 12em;
        overflow-y: auto;
    }

    .tag-suggestion {
        border: none;
        background: none;
        cursor: pointer;
        padding: 0.1em 0.4em;
        width: 100%;
        text-align: left;
    }

    .tag-suggestion.selected {
        background: var(--color-surface-alt);
        font-weight: 600;
    }

    .tag-vocabulary-button {
        cursor: pointer;
        padding: 0 0.5em;
    }

    .comment-panel {
        height: 100%;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        background-color: white;
        font-size: var(--font-size-base);
    }

    /* Search strip */
    .search-strip {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 3px 8px;
        border-bottom: 1px solid #eee;
        flex-shrink: 0;
        background: #fafafa;
    }
    .search-icon {
        color: #aaa;
        font-size: var(--font-size-base);
        flex-shrink: 0;
    }
    .search-input {
        flex: 1;
        border: none;
        outline: none;
        background: transparent;
        font-size: var(--font-size-small);
        color: var(--color-text);
        padding: 2px 0;
    }
    .clear-btn {
        border: none;
        background: none;
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: var(--font-size-base);
        padding: 0 2px;
        line-height: 1;
    }
    .clear-btn:hover {
        color: var(--color-text);
    }

    /* Feed */
    .feed {
        flex: 1;
        overflow-y: auto;
        padding: 4px 8px;
    }

    .msg {
        padding: 6px 10px;
        margin-bottom: 4px;
        background: #f0f2f8;
        border-radius: 10px 10px 10px 2px;
        cursor: pointer;
        transition: background 0.1s;
        position: relative;
    }
    .msg:hover {
        background: #e4e8f2;
    }
    .msg.editing {
        background: #fefce8;
        cursor: default;
        border-radius: 6px;
    }

    .msg-text {
        font-size: var(--font-size-base);
        color: var(--color-text);
        white-space: pre-wrap;
        word-break: break-word;
        line-height: 1.35;
        text-align: left;
    }

    .msg-header {
        display: flex;
        align-items: baseline;
        gap: var(--space-1);
        margin-bottom: 2px;
    }
    .msg-date {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        font-style: italic;
    }
    .msg-origin {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        padding: 0 var(--space-1);
        margin-left: var(--space-1);
        white-space: nowrap;
    }
    .msg-footer {
        display: flex;
        align-items: center;
        justify-content: flex-end;
        gap: 4px;
        margin-top: 3px;
    }
    .msg-action {
        border: none;
        background: none;
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: var(--font-size-title);
        padding: 2px 6px;
        line-height: 1;
        border-radius: 4px;
        transition:
            background 0.1s,
            color 0.1s;
    }
    .msg-edit:hover {
        color: #4a90d9;
        background: rgba(74, 144, 217, 0.12);
    }
    .msg-delete:hover {
        color: #c55;
        background: rgba(204, 85, 85, 0.12);
    }

    .msg-edit-input {
        width: 100%;
        box-sizing: border-box;
        padding: 4px 6px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        font-size: var(--font-size-base);
        font-family: inherit;
        line-height: 1.35;
        resize: none;
        outline: none;
    }
    .msg-edit-input:focus {
        border-color: #4a90d9;
    }

    .empty-msg {
        text-align: center;
        color: #bbb;
        padding: 20px;
        font-size: var(--font-size-small);
        font-style: italic;
    }

    /* Prompt */
    .prompt {
        flex-shrink: 0;
        border-top: 1px solid #eee;
        padding: 4px 8px;
        background: #fafafa;
        /* La liste de suggestions se pose au-dessus de la saisie. */
        position: relative;
        display: flex;
        align-items: flex-start;
        gap: 0.3em;
    }
    .prompt textarea {
        width: 100%;
        box-sizing: border-box;
        padding: 6px 8px;
        border: 1px solid #ddd;
        border-radius: 6px;
        outline: none;
        resize: none;
        background: white;
        font-size: var(--font-size-base);
        line-height: 1.35;
        font-family: inherit;
    }
    .prompt textarea:focus {
        border-color: #aab;
    }
</style>
