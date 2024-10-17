<script>
    export let visible = false;
    export let onClose;
    export let analysisData = '';

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }

    $: if (visible) {
        setTimeout(() => {
            const analysisEl = document.getElementById('analysisPanel');
            if (analysisEl) {
                analysisEl.focus();
            }
        }, 0);
    }
</script>

{#if visible}
    <div class="analysis-panel" tabindex="0" id="analysisPanel" on:keydown={handleKeyDown}>
        <div class="close-icon" on:click={onClose}>×</div>
        <div class="analysis-content">
            <h3>Analysis</h3>
            <p>{analysisData}</p>
        </div>
    </div>
{/if}

<style>
    .analysis-panel {
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        max-height: 50vh;
        overflow-y: auto;
        background: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 16px;
        box-sizing: border-box;
        z-index: 5;
    }

    .close-icon {
        position: absolute;
        top: -6px;
        right: 4px;
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        transition: background-color 0.3s ease, opacity 0.3s ease;
    }

    .analysis-content {
        margin-top: 30px;
        font-size: 16px;
    }

    h3 {
        margin: 0 0 10px 0;
        font-size: 24px;
    }
</style>

