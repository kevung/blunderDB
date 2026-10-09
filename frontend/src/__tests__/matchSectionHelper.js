// Opens a folded section of the match sheet the way a click on its summary does.
import { fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

/**
 * @param {HTMLElement} container
 * @param {string} id
 */
export async function openSection(container, id) {
    const section = /** @type {HTMLDetailsElement} */ (container.querySelector(`[data-testid="match-section-${id}"]`));
    section.open = true;
    await fireEvent(section, new Event('toggle'));
    await tick();
}
