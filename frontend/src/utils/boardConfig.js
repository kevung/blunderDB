// La configuration de dessin du plateau, en UN seul endroit : Board.svelte et
// les diagrammes du rapport (rendus hors écran avec les mêmes fonctions de
// scène) partagent ce littéral pour ne jamais faire diverger deux palettes.

/**
 * Une configuration NEUVE à chaque appel : l'appelant la mute (applyPalette), et deux plateaux
 * partageant l'objet partageraient leurs couleurs.
 */
export function defaultBoardConfig() {
    return {
        widthFactor: 0.75,
        orientation: 'right',
        fill: '#f0f0f0',
        stroke: '#333333',
        linewidth: 3,
        triangle: {
            fill1: '#d9d9d9',
            fill2: '#a6a6a6',
            stroke: '#333333',
            linewidth: 1.3
        },
        label: {
            size: 20,
            distanceToBoard: 0.3
        },
        checker: {
            sizeFactor: 0.97,
            colors: ['#333333', '#ffffff'],
            linewidth: 2.5
        },
        dice: {
            fill: '#ffffff',
            dot: '#000000'
        },
        cube: {
            fill: '#ffffff'
        }
    };
}

/**
 * Applique la palette de l'utilisateur à une configuration, en place.
 * @param {ReturnType<typeof defaultBoardConfig>} cfg
 * @param {Record<string, string>} colors
 */
export function applyPalette(cfg, colors) {
    if (!colors) return cfg;
    cfg.fill = colors.background;
    cfg.stroke = colors.border;
    cfg.triangle.fill1 = colors.point1;
    cfg.triangle.fill2 = colors.point2;
    cfg.triangle.stroke = colors.border;
    cfg.checker.colors = [colors.checker1, colors.checker2];
    cfg.dice.fill = colors.dice;
    cfg.dice.dot = colors.diceDot;
    cfg.cube.fill = colors.cube;
    return cfg;
}
