/**
 * Svelte action: shows `node` inside `target` while one is given, back where it was rendered
 * otherwise. The node stays in its component's tree — props, bind:this and teardown are its
 * owner's — only its place in the DOM changes. A <video> moved within one task keeps playing;
 * an <iframe> reloads, which its owner has to account for.
 *
 * @param {HTMLElement} node
 * @param {HTMLElement | null} target
 */
export function portal(node, target) {
    const home = /** @type {Node} */ (node.parentNode);
    /** @param {HTMLElement | null} next */
    function place(next) {
        const host = next ?? home;
        if (host && node.parentNode !== host) host.appendChild(node);
    }
    place(target);
    return {
        /** @param {HTMLElement | null} next */
        update(next) {
            place(next);
        },
        destroy() {
            node.remove();
        }
    };
}
