/**
 * helpers/gestureCount.js — compter les gestes qu'une spec émet.
 *
 * `tasks/transcription/ux.md` §4 chiffre chaque flux en GESTES : trois touches
 * pour le meilleur coup joué, un clic pour un jet donné au triangle, `2k + 1`
 * touches pour un camp corrigé k Actions en arrière. Un budget qui n'est pas
 * compté n'est pas tenu : il suffit d'un clic d'armement de plus, ou d'un
 * aller-retour clavier ↔ souris oublié, pour qu'il se perde sans que rien ne
 * rougisse.
 *
 * Playwright ne compte rien de tout cela — d'où ce fichier. `countGestures`
 * remplace `page.keyboard.press` et `page.mouse.click` le temps d'un bloc, et
 * rend `{ keys, clicks, total }`.
 *
 * ## Le clic passe par le compteur, jamais par le locator
 *
 * `locator.click()` ne descend PAS par `page.mouse.click` : il parle au pilote
 * directement, et aucun remplacement de `page.mouse` ne le verrait. Un clic
 * compté se fait donc par `g.click(locator)`, qui compte puis délègue. Le
 * remplacement de `page.mouse.click` reste, lui, pour les rares clics à la
 * coordonnée (le damier), et pour qu'un `page.keyboard.press` écrit
 * naturellement dans le bloc soit compté sans qu'on y pense.
 *
 * ## Ce qui n'est pas compté, et pourquoi
 *
 * Le temps. Le budget d'ux.md est en secondes (K = 0,28 s, P + 2 B pour un
 * clic), et un navigateur sans tête ne mesure pas le temps d'un utilisateur :
 * il mesure celui d'un pilote. Les secondes restent une vérification manuelle,
 * comme la fiche T2.1 le dit du triangle ; ce qui se tient ici est le NOMBRE
 * de gestes, dont les secondes se déduisent par les constantes de §1.
 */

/**
 * @typedef {Object} GestureDriver
 * @property {(key: string) => Promise<void>} press        une touche
 * @property {(target: any) => Promise<void>} click        un clic sur un locator
 * @property {() => {keys: number, clicks: number, total: number}} count  le compte courant
 */

/**
 * Compte les gestes émis par `run`.
 *
 * @param {import('@playwright/test').Page} page
 * @param {(g: GestureDriver) => Promise<void>} run
 * @returns {Promise<{keys: number, clicks: number, total: number}>}
 */
export async function countGestures(page, run) {
    const tally = { keys: 0, clicks: 0 };
    const snapshot = () => ({ ...tally, total: tally.keys + tally.clicks });

    const keyboard = page.keyboard;
    const mouse = page.mouse;
    const pressOriginal = keyboard.press.bind(keyboard);
    const clickOriginal = mouse.click.bind(mouse);

    keyboard.press = async (...args) => {
        tally.keys += 1;
        return pressOriginal(...args);
    };
    mouse.click = async (...args) => {
        tally.clicks += 1;
        return clickOriginal(...args);
    };

    /** @type {GestureDriver} */
    const driver = {
        press: (key) => page.keyboard.press(key),
        click: async (target) => {
            tally.clicks += 1;
            await (typeof target === 'string' ? page.locator(target) : target).click();
        },
        count: snapshot
    };

    try {
        await run(driver);
    } finally {
        keyboard.press = pressOriginal;
        mouse.click = clickOriginal;
    }
    return snapshot();
}
