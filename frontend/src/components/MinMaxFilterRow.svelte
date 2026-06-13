<script>
    // A single min / max / range numeric filter row, factored out of the ~20
    // identical blocks in SearchPanel's filter builder. The five state values
    // are $bindable so the parent keeps owning them (two-way like the original
    // inline bind:value); `min`/`max` are the optional HTML numeric bounds.
    import { t } from '../i18n';

    let { option = $bindable(), minVal = $bindable(), maxVal = $bindable(), rangeMin = $bindable(), rangeMax = $bindable(), min = undefined, max = undefined } = $props();
</script>

<div class="minmax-controls">
    <label
        ><input type="radio" bind:group={option} value="min" />
        {$t('common.min')} <input type="number" bind:value={minVal} class="num-input" {min} {max} disabled={option !== 'min'} /></label
    ><label
        ><input type="radio" bind:group={option} value="max" />
        {$t('common.max')} <input type="number" bind:value={maxVal} class="num-input" {min} {max} disabled={option !== 'max'} /></label
    ><label
        ><input type="radio" bind:group={option} value="range" />
        {$t('common.range')}
        <input type="number" bind:value={rangeMin} class="num-input" {min} {max} disabled={option !== 'range'} />
        <input type="number" bind:value={rangeMax} class="num-input" {min} {max} disabled={option !== 'range'} /></label
    >
</div>

<style>
    /* Copied from SearchPanel.svelte (scoped CSS does not cross the component
       boundary); SearchPanel keeps its own copies for the non-converted blocks. */
    .minmax-controls {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    .minmax-controls label {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: 12px;
        user-select: none;
        -webkit-user-select: none;
    }
    .num-input {
        width: 60px;
        font-size: 12px;
        padding: 2px 3px;
    }
</style>
