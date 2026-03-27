<script>
    export let visible = false;
    export let onClose;

    import { currentPositionIndexStore } from '../stores/uiStore';
    import { positionStore } from '../stores/positionStore';
    import { GetCommentsByPosition, SearchComments, LoadComment, LoadAnalysis, LoadPosition, AddComment, UpdateCommentEntry, DeleteCommentEntry } from '../../wailsjs/go/main/Database.js';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';

    let allComments = [];
    let searchQuery = '';
    let displayedComments = [];
    let feedEl;
    let editingCommentId = null;
    let editingText = '';
    let promptText = '';

    $: if (visible) {
        loadComments();
    }

    // Reload comments when displayed position changes
    $: if (visible && $positionStore && $positionStore.id) {
        loadComments();
    }

    $: {
        if (searchQuery.trim()) {
            filterComments(searchQuery.trim());
        } else {
            displayedComments = allComments;
        }
    }

    async function loadComments() {
        try {
            const pos = $positionStore;
            if (pos && pos.id) {
                allComments = await GetCommentsByPosition(pos.id) || [];
            } else {
                allComments = [];
            }
            if (!searchQuery.trim()) displayedComments = allComments;
        } catch (error) {
            console.error('Error loading comments:', error);
            allComments = [];
            displayedComments = [];
        }
    }

    async function filterComments(q) {
        try {
            displayedComments = await SearchComments(q) || [];
        } catch (error) {
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
                } catch (e) {}
                selectedMoveStore.set(null);
            }
        } catch (error) {
            console.error('Error navigating to comment position:', error);
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
            console.error('Error adding comment:', error);
        }
    }

    function handlePromptKeyDown(event) {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.stopPropagation();
            event.preventDefault();
            addNewComment();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            event.currentTarget.blur();
        }
    }

    function startEditComment(comment) {
        editingCommentId = comment.id;
        editingText = comment.text;
    }

    async function saveEditedComment(comment) {
        editingCommentId = null;
        if (editingText !== comment.text) {
            try {
                await UpdateCommentEntry(comment.id, editingText);
                await loadComments();
            } catch (error) {
                console.error('Error saving edited comment:', error);
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
            await DeleteCommentEntry(comment.id);
            await loadComments();
        } catch (error) {
            console.error('Error deleting comment:', error);
        }
    }

    function formatDate(dateStr) {
        if (!dateStr) return '';
        try {
            // SQLite CURRENT_TIMESTAMP returns "YYYY-MM-DD HH:MM:SS" which some
            // JS engines can't parse. Replace the space with 'T' and add 'Z' for UTC.
            const normalized = dateStr.includes('T') ? dateStr : dateStr.replace(' ', 'T') + 'Z';
            const d = new Date(normalized);
            const now = new Date();
            const diffMs = now.getTime() - d.getTime();
            const diffMin = Math.floor(diffMs / 60000);
            if (diffMin < 1) return 'just now';
            if (diffMin < 60) return `${diffMin}m ago`;
            const diffHr = Math.floor(diffMin / 60);
            if (diffHr < 24) return `${diffHr}h ago`;
            return d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit', year: '2-digit' }) + ' ' + d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
        } catch { return ''; }
    }

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
        <input
            type="text"
            bind:value={searchQuery}
            placeholder="Search comments…"
            on:keydown={handleSearchKeyDown}
            class="search-input"
        />
        {#if searchQuery}
            <button class="clear-btn" on:click={() => { searchQuery = ''; }}>×</button>
        {/if}
    </div>

    <!-- Message feed -->
    <div class="feed" bind:this={feedEl}>
        {#if displayedComments.length === 0}
            <div class="empty-msg">{searchQuery.trim() ? 'No matches' : 'No comments yet'}</div>
        {:else}
            {#each displayedComments as comment}
                {#if editingCommentId === comment.id}
                    <div class="msg editing">
                        <textarea
                            class="msg-edit-input"
                            bind:value={editingText}
                            on:keydown={(e) => handleEditKeyDown(e, comment)}
                            on:blur={() => saveEditedComment(comment)}
                            rows="2"
                        ></textarea>
                    </div>
                {:else}
                    <div class="msg" role="button" tabindex="-1" on:click={() => navigateToComment(comment)} on:keydown={() => {}}>
                        <div class="msg-header">
                            <span class="msg-date">{comment.modifiedAt && comment.modifiedAt !== comment.createdAt ? formatDate(comment.modifiedAt) + ' (edited)' : formatDate(comment.createdAt)}</span>
                        </div>
                        <div class="msg-text">{comment.text}</div>
                        <div class="msg-footer">
                            <button class="msg-action msg-edit" on:click|stopPropagation={() => startEditComment(comment)} title="Edit">✎</button>
                            <button class="msg-action msg-delete" on:click|stopPropagation={(e) => deleteComment(comment, e)} title="Delete">×</button>
                        </div>
                    </div>
                {/if}
            {/each}
        {/if}
    </div>

    <!-- Prompt -->
    <div class="prompt">
        <textarea
            id="commentTextArea"
            bind:value={promptText}
            placeholder="Comment on current position…"
            on:keydown={handlePromptKeyDown}
            rows="2"
        ></textarea>
    </div>
</div>

<style>
    .comment-panel {
        height: 100%;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        background-color: white;
        font-size: 12px;
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
    .search-icon { color: #aaa; font-size: 13px; flex-shrink: 0; }
    .search-input {
        flex: 1;
        border: none;
        outline: none;
        background: transparent;
        font-size: 11px;
        color: #333;
        padding: 2px 0;
    }
    .clear-btn {
        border: none;
        background: none;
        color: #999;
        cursor: pointer;
        font-size: 13px;
        padding: 0 2px;
        line-height: 1;
    }
    .clear-btn:hover { color: #333; }

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
    .msg:hover { background: #e4e8f2; }
    .msg.editing { background: #fefce8; cursor: default; border-radius: 6px; }

    .msg-text {
        font-size: 12px;
        color: #333;
        white-space: pre-wrap;
        word-break: break-word;
        line-height: 1.35;
        text-align: left;
    }

    .msg-header {
        margin-bottom: 2px;
    }
    .msg-date {
        font-size: 9px;
        color: #888;
        font-style: italic;
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
        color: transparent;
        cursor: pointer;
        font-size: 13px;
        padding: 0 2px;
        line-height: 1;
        transition: color 0.1s;
    }
    .msg:hover .msg-action { color: #bbb; }
    .msg-edit:hover { color: #4a90d9 !important; }
    .msg-delete:hover { color: #c55 !important; }

    .msg-edit-input {
        width: 100%;
        box-sizing: border-box;
        padding: 4px 6px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 12px;
        font-family: inherit;
        line-height: 1.35;
        resize: none;
        outline: none;
    }
    .msg-edit-input:focus { border-color: #4a90d9; }

    .empty-msg {
        text-align: center;
        color: #bbb;
        padding: 20px;
        font-size: 11px;
        font-style: italic;
    }

    /* Prompt */
    .prompt {
        flex-shrink: 0;
        border-top: 1px solid #eee;
        padding: 4px 8px;
        background: #fafafa;
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
        font-size: 13px;
        line-height: 1.35;
        font-family: inherit;
    }
    .prompt textarea:focus {
        border-color: #aab;
    }
</style>