// Les thèmes nommés (#286, fiche I.30).
//
// Un thème est une DONNÉE, pas une feuille de style : quatre listes de valeurs,
// appliquées aux variables CSS de `:root` et à la palette du plateau. Écrire
// chaque thème en CSS aurait dupliqué la liste des jetons autant de fois qu'il
// y a de thèmes, et un jeton ajouté un jour aurait manqué dans trois d'entre
// eux sans que rien ne le dise.
//
// # Pourquoi le plateau en fait partie
//
// L'ADR-0031 laisse volontairement la palette du plateau HORS du système de
// jetons, parce qu'elle est une préférence de l'utilisateur et non la chrome de
// l'interface. Cela reste vrai. Mais une interface sombre autour d'un plateau
// clair n'est pas un thème, c'est une moitié de thème : le plateau occupe
// l'essentiel de l'écran. Un thème propose donc AUSSI une palette de plateau,
// et l'utilisateur garde le dernier mot — l'onglet Couleurs continue de la
// régler, et son réglage survit au thème. Voir ADR-0038.

/** L'identifiant du thème qui suit le système. */
export const THEME_SYSTEM = 'system';

/**
 * Les jetons de couleur de l'interface. Ce sont EXACTEMENT ceux que
 * `style.css` déclare sous `:root` (ADR-0031) : un thème qui en oublierait un
 * laisserait la valeur du thème précédent, ce qui est la façon la plus sûre de
 * produire du texte illisible.
 */
export const UI_COLOR_TOKENS = ['--color-text', '--color-text-muted', '--color-border', '--color-surface', '--color-surface-alt', '--color-primary', '--color-danger'];

/**
 * @typedef {{ui: Record<string, string>, board: Record<string, string>}} Theme
 */

/** @type {Record<string, Theme>} */
export const THEMES = {
    // Le thème historique, celui que style.css déclare : il est ici pour
    // qu'aucun thème ne soit un cas particulier, et pour qu'on puisse y
    // revenir explicitement.
    light: {
        ui: {
            '--color-text': '#333333',
            '--color-text-muted': '#666666',
            '--color-border': '#cccccc',
            '--color-surface': '#ffffff',
            '--color-surface-alt': '#f5f5f5',
            '--color-primary': '#1976d2',
            '--color-danger': '#b3261e'
        },
        board: {
            background: '#f0f0f0',
            border: '#333333',
            point1: '#d9d9d9',
            point2: '#a6a6a6',
            checker1: '#333333',
            checker2: '#ffffff',
            dice: '#ffffff',
            diceDot: '#000000',
            cube: '#ffffff'
        }
    },

    // Les contrastes sont tenus au-dessus de 4,5:1 pour le texte, comme
    // l'ADR-0031 l'exige du thème clair : un thème sombre n'est pas une
    // dispense.
    dark: {
        ui: {
            '--color-text': '#e6e6e6',
            '--color-text-muted': '#a8a8a8',
            '--color-border': '#4a4a4a',
            '--color-surface': '#1e1e1e',
            '--color-surface-alt': '#2a2a2a',
            '--color-primary': '#6cb1ff',
            '--color-danger': '#ff6b60'
        },
        board: {
            background: '#2a2a2a',
            border: '#8a8a8a',
            point1: '#3d3d3d',
            point2: '#565656',
            checker1: '#111111',
            checker2: '#dcdcdc',
            dice: '#dcdcdc',
            diceDot: '#111111',
            cube: '#dcdcdc'
        }
    },

    // Contraste élevé : du noir sur du blanc, un seul accent, et des
    // frontières qui se voient. Ce n'est pas un thème « clair en plus dur » :
    // c'est celui qu'on choisit quand la nuance ne passe pas.
    contrast: {
        ui: {
            '--color-text': '#000000',
            '--color-text-muted': '#000000',
            '--color-border': '#000000',
            '--color-surface': '#ffffff',
            '--color-surface-alt': '#ededed',
            '--color-primary': '#0032a0',
            '--color-danger': '#a80000'
        },
        board: {
            background: '#ffffff',
            border: '#000000',
            point1: '#ffffff',
            point2: '#b0b0b0',
            checker1: '#000000',
            checker2: '#ffffff',
            dice: '#ffffff',
            diceDot: '#000000',
            cube: '#ffffff'
        }
    },

    // Imprimable : ce qui reste lisible une fois passé par une imprimante
    // noir et blanc, et ce qui ne gaspille pas d'encre. Le fond est blanc,
    // les aplats sont clairs, et l'accent est sombre plutôt que coloré.
    print: {
        ui: {
            '--color-text': '#111111',
            '--color-text-muted': '#444444',
            '--color-border': '#999999',
            '--color-surface': '#ffffff',
            '--color-surface-alt': '#ffffff',
            '--color-primary': '#333333',
            '--color-danger': '#000000'
        },
        board: {
            background: '#ffffff',
            border: '#000000',
            point1: '#ffffff',
            point2: '#d0d0d0',
            checker1: '#000000',
            checker2: '#ffffff',
            dice: '#ffffff',
            diceDot: '#000000',
            cube: '#ffffff'
        }
    }
};

/** Les identifiants proposés, `system` en tête. */
export const THEME_NAMES = [THEME_SYSTEM, ...Object.keys(THEMES)];

/**
 * Le thème effectivement à appliquer pour un choix donné. `system` suit
 * `prefers-color-scheme`, et retombe sur le thème clair quand la préférence
 * n'est pas exprimée — ce qui est le cas de la plupart des environnements de
 * bureau que ce WebView rencontre.
 * @param {string} name
 */
export function resolveTheme(name) {
    if (name !== THEME_SYSTEM) return THEMES[name] ? name : 'light';
    const prefersDark = typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(prefers-color-scheme: dark)').matches;
    return prefersDark ? 'dark' : 'light';
}

/**
 * Écrit les jetons d'un thème sur l'élément racine, et pose `data-theme` pour
 * qu'une règle CSS puisse s'y accrocher si un jour l'une en a besoin.
 * @param {string} name
 * @returns {Theme} le thème résolu, pour que l'appelant en tire la palette.
 */
export function applyThemeTokens(name) {
    const resolved = resolveTheme(name);
    const theme = THEMES[resolved];
    const root = document.documentElement;
    for (const token of UI_COLOR_TOKENS) {
        root.style.setProperty(token, theme.ui[token]);
    }
    root.dataset.theme = resolved;
    return theme;
}
