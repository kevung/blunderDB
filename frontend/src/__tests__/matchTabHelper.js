// Shows a tab of the match sheet the way a click on it does.
import { fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

/**
 * @param {HTMLElement} container
 * @param {string} id
 */
export async function openTab(container, id) {
    const tab = /** @type {HTMLElement} */ (container.querySelector(`[data-testid="match-tab-${id}"]`));
    await fireEvent.click(tab);
    await tick();
}
