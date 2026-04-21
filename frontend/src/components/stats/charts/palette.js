/**
 * Shared colour palette for Stats panel charts.
 * Uses a restricted set of colours (UX principle: palette restreinte).
 */

// Primary colour for main data series
export const PRIMARY = '#1976d2';

// Secondary / accent colour
export const SECONDARY = '#757575';

// Neutral gridline colour
export const GRIDLINE = '#e0e0e0';

// Background colour for area fills (with alpha)
export const PRIMARY_ALPHA = 'rgba(25, 118, 210, 0.15)';

// Backgammon skill-grade band colours (from best to worst)
export const GRADE_COLORS = [
    '#2e7d32', // World class   (≤ 1.5 PR)
    '#558b2f', // Expert        (≤ 3.0)
    '#f57f17', // Intermediate  (≤ 5.0)
    '#e65100', // Amateur       (≤ 8.0)
    '#b71c1c' // Beginner      (> 8.0)
];

// Cube-action breakdown colours
export const CUBE_COLORS = {
    checker: PRIMARY,
    cube: '#7b1fa2'
};
