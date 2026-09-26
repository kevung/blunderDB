/**
 * helpers/gestureCount.js — compter les gestes qu'une spec émet, pour tenir
 * les budgets d'ux.md §4. `countGestures` remplace `page.keyboard.press` et
 * `page.mouse.click` le temps d'un bloc et rend `{ keys, clicks, total }`.
 *
 * `locator.click()` ne passe pas par `page.mouse.click` : un clic compté se
 * fait par `g.click(locator)`. Le remplacement de `page.mouse.click` sert aux
 * clics à la coordonnée (le damier).
 *
 * Seul le NOMBRE de gestes est compté : un navigateur sans tête ne mesure pas
 * le temps d'un utilisateur, les secondes se déduisent des constantes d'ux.md §1.
 */

/**
 * @typedef {Object} GestureDriver
 * @property {(key: string) => Promise<void>} press        une touche
 * @property {(target: any) => Promise<void>} click        un clic sur un locator
 * @property {(from: {x: number, y: number}, to: {x: number, y: number}) => Promise<void>} drag
 *           un glissé, appuyer ici et lâcher là — UN geste de souris, le
 *           P B B d'un pas joué au plateau (ux.md §4.1)
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
        drag: async (from, to) => {
            tally.clicks += 1;
            await mouse.move(from.x, from.y);
            await mouse.down();
            await mouse.move(to.x, to.y, { steps: 4 });
            await mouse.up();
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
