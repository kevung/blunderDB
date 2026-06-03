// Guided-tour definitions (data only). The tour engine (tourService.js) turns
// these into driver.js steps, resolving the i18n keys at run time.
//
// Each step targets a stable DOM element via a `data-tour="..."` attribute (so
// it does not depend on translated labels or on the two.js canvas internals).
// A step without `element` is shown as a centered popover. Steps whose target
// is absent from the DOM are skipped automatically.
//
// To add a new tour, append an entry here and add its i18n keys to every locale
// (the locale-parity test enforces this). No engine change is needed.

export const TOURS = [
    {
        id: 'general',
        titleKey: 'tour.general.title',
        descKey: 'tour.general.desc',
        steps: [
            { titleKey: 'tour.general.welcome.title', bodyKey: 'tour.general.welcome.body' },
            { element: '[data-tour="toolbar"]', titleKey: 'tour.general.toolbar.title', bodyKey: 'tour.general.toolbar.body', side: 'bottom', align: 'start' },
            { element: '[data-tour="board"]', titleKey: 'tour.general.board.title', bodyKey: 'tour.general.board.body', side: 'top' },
            { element: '[data-tour="panels"]', titleKey: 'tour.general.panels.title', bodyKey: 'tour.general.panels.body', side: 'top' },
            { element: '[data-tour="statusbar"]', titleKey: 'tour.general.commandline.title', bodyKey: 'tour.general.commandline.body', side: 'top' },
            { element: '[data-tour="help"]', titleKey: 'tour.general.help.title', bodyKey: 'tour.general.help.body', side: 'left' },
            { titleKey: 'tour.general.done.title', bodyKey: 'tour.general.done.body' }
        ]
    },
    {
        id: 'search',
        titleKey: 'tour.search.title',
        descKey: 'tour.search.desc',
        steps: [
            { titleKey: 'tour.search.intro.title', bodyKey: 'tour.search.intro.body' },
            // Switch to the Search tab so the actual search panel is highlighted.
            { element: '[data-tour="panels"]', activateTab: 'search', titleKey: 'tour.search.tab.title', bodyKey: 'tour.search.tab.body', side: 'top' },
            { element: '[data-tour="board"]', titleKey: 'tour.search.structure.title', bodyKey: 'tour.search.structure.body', side: 'top' },
            { titleKey: 'tour.search.commandline.title', bodyKey: 'tour.search.commandline.body' }
        ]
    },
    {
        id: 'matches',
        titleKey: 'tour.matches.title',
        descKey: 'tour.matches.desc',
        steps: [
            { titleKey: 'tour.matches.intro.title', bodyKey: 'tour.matches.intro.body' },
            { element: '[data-tour="panels"]', activateTab: 'matches', titleKey: 'tour.matches.tab.title', bodyKey: 'tour.matches.tab.body', side: 'top' },
            { titleKey: 'tour.matches.review.title', bodyKey: 'tour.matches.review.body' }
        ]
    },
    {
        id: 'tournaments',
        titleKey: 'tour.tournaments.title',
        descKey: 'tour.tournaments.desc',
        steps: [
            { titleKey: 'tour.tournaments.intro.title', bodyKey: 'tour.tournaments.intro.body' },
            { element: '[data-tour="panels"]', activateTab: 'tournaments', titleKey: 'tour.tournaments.tab.title', bodyKey: 'tour.tournaments.tab.body', side: 'top' },
            { titleKey: 'tour.tournaments.drill.title', bodyKey: 'tour.tournaments.drill.body' }
        ]
    }
];

export function getTourById(id) {
    return TOURS.find((t) => t.id === id);
}
