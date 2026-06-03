<script>
    import { trapFocus } from '../utils/focusTrap.js';
    import { t, language, setLanguage, LOCALES, LANGUAGE_LABELS } from '../i18n';
    import { boardColorsStore, setBoardColor, resetBoardColors } from '../stores/boardColorsStore';

    let { visible = false, onClose } = $props();

    // Board colour settings, in display order. Each maps a store key to a label.
    const COLOR_SETTINGS = [
        { key: 'background', labelKey: 'config.colorBackground' },
        { key: 'border', labelKey: 'config.colorBorder' },
        { key: 'point1', labelKey: 'config.colorPoint1' },
        { key: 'point2', labelKey: 'config.colorPoint2' },
        { key: 'checker1', labelKey: 'config.colorChecker1' },
        { key: 'checker2', labelKey: 'config.colorChecker2' },
        { key: 'dice', labelKey: 'config.colorDice' },
        { key: 'diceDot', labelKey: 'config.colorDiceDot' },
        { key: 'cube', labelKey: 'config.colorCube' }
    ];

    function onLanguageChange(event) {
        setLanguage(event.currentTarget.value);
    }

    function onColorChange(key, event) {
        setBoardColor(key, event.currentTarget.value);
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }
</script>

{#if visible}
    <div class="modal-overlay" onclick={onClose} onkeydown={handleKeyDown} role="dialog" aria-modal="true" aria-label={$t('config.title')} use:trapFocus>
        <div class="modal-content" onclick={(e) => e.stopPropagation()}>
            <div class="close-button" onclick={onClose}>×</div>
            <h2>{$t('config.title')}</h2>

            <div class="setting-row">
                <label for="config-language">{$t('config.language')}</label>
                <select id="config-language" class="setting-select" value={$language} onchange={onLanguageChange}>
                    {#each LOCALES as code (code)}
                        <option value={code}>{LANGUAGE_LABELS[code]}</option>
                    {/each}
                </select>
            </div>

            <div class="section-title">{$t('config.colors')}</div>
            {#each COLOR_SETTINGS as setting (setting.key)}
                <div class="setting-row">
                    <label for={`config-color-${setting.key}`}>{$t(setting.labelKey)}</label>
                    <input id={`config-color-${setting.key}`} type="color" class="setting-color" value={$boardColorsStore[setting.key]} oninput={(e) => onColorChange(setting.key, e)} />
                </div>
            {/each}
            <div class="setting-row reset-row">
                <button class="secondary-button" onclick={resetBoardColors}>{$t('config.resetColors')}</button>
            </div>

            <div class="modal-buttons">
                <button class="primary-button" onclick={onClose}>{$t('common.close')}</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }

    .modal-content {
        background-color: white;
        padding: 1rem;
        border-radius: 4px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        z-index: 1000;
        width: 90%;
        max-width: 360px;
        max-height: 80vh;
        overflow-y: auto;
        position: relative;
        display: flex;
        flex-direction: column;
        text-align: center;
    }

    h2 {
        margin: 0 0 1rem;
        font-size: 1.25rem;
    }

    .close-button {
        position: absolute;
        top: 8px;
        right: 8px;
        font-size: 1.5rem;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        z-index: 10;
        transition:
            background-color 0.3s ease,
            opacity 0.3s ease;
    }

    .setting-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        margin: 8px 0;
        text-align: left;
    }

    .setting-row label {
        font-weight: 500;
    }

    .setting-select {
        flex: 0 0 auto;
        min-width: 160px;
        padding: 8px;
        border: 1px solid #ccc;
        border-radius: 4px;
        box-sizing: border-box;
        font-size: 15px;
        background-color: white;
    }

    .setting-select:focus {
        outline: none;
        border-color: #6c757d;
        box-shadow: 0 0 5px rgba(108, 117, 125, 0.5);
    }

    .section-title {
        margin: 16px 0 4px;
        padding-top: 12px;
        border-top: 1px solid #eee;
        font-weight: 600;
        text-align: left;
    }

    .setting-color {
        flex: 0 0 auto;
        width: 48px;
        height: 28px;
        padding: 0;
        border: 1px solid #ccc;
        border-radius: 4px;
        background: none;
        cursor: pointer;
    }

    .reset-row {
        justify-content: flex-end;
    }

    .secondary-button {
        padding: 6px 12px;
        border: 1px solid #ccc;
        border-radius: 4px;
        background-color: #f5f5f5;
        cursor: pointer;
        font-size: 14px;
    }

    .secondary-button:hover {
        background-color: #e9e9e9;
    }

    .modal-buttons {
        margin-top: 10px;
        display: flex;
        justify-content: center;
        gap: 10px;
    }

    .modal-buttons button {
        padding: 8px 14px;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-size: 15px;
    }

    .primary-button {
        background-color: #6c757d;
        color: white;
    }

    .primary-button:hover {
        background-color: #5a6268;
    }
</style>
