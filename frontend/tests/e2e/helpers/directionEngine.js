/**
 * Une Direction ouverte, pour les specs de budget de gestes.
 *
 * Nicomaque décide des appariements, des arbres et du classement ; les tests Go du paquet
 * `database` tiennent déjà ce qu'il produit, décision par décision. Le réécrire ici en
 * JavaScript en ferait une seconde version qui dériverait, donc ce mock ne le fait pas : il
 * tient seulement ce QU'UN GESTE DOIT FAIRE BOUGER À L'ÉCRAN (confirmer sort une proposition de
 * la file et met un match sur une table, saisir un résultat libère la table, inscrire un joueur
 * l'ajoute à la liste) — aucun appariement n'est calculé, les propositions sont posées d'avance.
 *
 * Pas de constantes (contrairement à `installWailsMock`) : un budget compte un enchaînement
 * (cliquer, voir le résultat, cliquer encore), et une constante rendrait la même file après
 * comme avant, si bien qu'un flux à cinq clics passerait pour un flux à un clic.
 */

/** Les quatre inscrits. Le nom diffère de l'identifiant, comme dans les tests Go. */
export const ENTRANTS = [
    { id: 'ha', name: 'Hugo Andrieu', club: 'Lyon', rating: 5 },
    { id: 'lb', name: 'Léa Bonnet', club: 'Lyon', rating: 4.5 },
    { id: 'mc', name: 'Marc Colin', club: 'Paris', rating: 6 },
    { id: 'nd', name: 'Nadia Dubois', club: 'Paris', rating: 3.5 }
];

/**
 * Une salle pleine : 76 inscrits, 14 tables toutes occupées, 48 joueurs restants en 24
 * propositions. C'est la hauteur de la file qui poussait la grille sous la ligne de
 * flottaison ; quatre joueurs sur quatre tables ne le montrent jamais.
 *
 * @param {number} n
 */
export function crowd(n) {
    return Array.from({ length: n }, (_, i) => ({ id: `s${i + 1}`, name: `Joueur ${String(i + 1).padStart(2, '0')}`, club: i % 2 ? 'Lyon' : 'Paris', rating: 3 + (i % 5) }));
}

/** Les options de l'état S2 : `installDirectionEngine(page, S2_HALL)`. */
export const S2_HALL = { entrants: crowd(76), tables: 14, running: 14 };

/** L'état S4 du vendredi : 25 inscrits, rondes 1 et 2 jouées, la ronde 3 proposée et rien en cours. */
export const S4_FRIDAY = { entrants: crowd(25), tables: 6, running: 0, rounds: 2 };

/**
 * Installe la Direction factice. À appeler APRÈS `installWailsMock` et avant `page.goto`.
 *
 * @param {import('@playwright/test').Page} page
 * @param {{directed?: boolean, entrants?: typeof ENTRANTS, tables?: number, running?: number, rounds?: number, phases?: Object[], locks?: Object[]}} [opts]
 *   `directed: false` part d'un tournoi non dirigé, pour mesurer le coût d'entrée depuis rien.
 *   `entrants`, `tables` et `running` changent la taille de la salle (défaut : les quatre
 *   inscrits, quatre tables, aucun match) ; `running` matchs sont lancés d'avance. `rounds`
 *   donne le nombre de rondes déjà lancées, `phases` le format, `locks` les verrous que
 *   l'aperçu de configuration renvoie (une phase dont le tirage est fait).
 */
export async function installDirectionEngine(page, opts = {}) {
    await page.addInitScript(
        ({ entrants, directed, tableCount, runningAtStart, roundsAtStart, phases, lockedPhases }) => {
            const db = window.go.database.Database;

            const TOURNAMENT_ID = 1;
            let CONFIG = {
                name: 'Open de Lyon',
                tables: { count: tableCount },
                phases: phases || [{ kind: 'swiss_lives', length: 7, lives: 2, mode: 'continuous', target: 0 }]
            };

            let exists = directed;
            let players = directed ? entrants.map((p) => ({ ...p })) : [];
            /** @type {Array<Object>} */
            let proposals = [];
            let running = [];
            let last = null;
            /** Les résultats saisis, dans l'ordre : ce que l'Historique relit et que le classement compte. */
            const results = [];
            let events = directed ? 1 : 0;
            /** Clos ou non : ce que « Clore » et « Rouvrir » font bouger à l'écran. */
            let finished = false;
            let seq = 0;

            /** Les propositions que le moteur ferait à partir des joueurs libres, deux à deux. */
            function pair() {
                const free = players.filter((p) => p.state !== 'withdrawn' && !running.some((m) => m.a === p.id || m.b === p.id));
                const out = [];
                for (let i = 0; i + 1 < free.length; i += 2) {
                    out.push({
                        kind: 'start_match',
                        phase: 0,
                        key: `k${i}`,
                        a: free[i].id,
                        b: free[i + 1].id,
                        length: 7,
                        table: 0,
                        label: { kind: 'swiss_group', losses: 0, match: 1 }
                    });
                }
                return out;
            }

            function nameOf(id) {
                const p = players.find((x) => x.id === id);
                return p ? p.name : id;
            }

            function view() {
                return {
                    tournamentId: TOURNAMENT_ID,
                    state: !exists ? 'draft' : running.length || last ? 'running' : 'draft',
                    engineVersion: 'v0.2.1',
                    outputDir: '',
                    config: CONFIG,
                    proposals,
                    warnings: [],
                    ranking: [],
                    players: players.map((p) => ({ id: p.id, name: p.name, club: p.club, rating: p.rating })),
                    running,
                    phase: 0,
                    finished,
                    eventCount: events
                };
            }

            function start(a, b, length, table) {
                seq += 1;
                events += 1;
                running.push({
                    ID: `m${seq}`,
                    A: a,
                    B: b,
                    a,
                    b,
                    Length: length || 7,
                    Table: table || running.length + 1,
                    Status: 1,
                    Label: { kind: 'swiss_group', losses: 0, match: 1 }
                });
                proposals = pair();
            }

            // Un tournoi déjà dirigé arrive avec sa file : c'est l'état dans lequel un directeur
            // ouvre sa Direction, et les budgets se comptent à partir de là.
            if (directed) proposals = pair();
            if (directed) {
                for (let i = 0; i < runningAtStart && proposals.length; i++) start(proposals[0].a, proposals[0].b, 7, 0);
            }

            db.ListDirections = () => Promise.resolve(exists ? [{ tournamentId: TOURNAMENT_ID, state: view().state, engineVersion: 'v0.2.1', outputDir: '', updatedAt: '' }] : []);
            db.HasDirection = () => Promise.resolve(exists);
            db.CreateDirection = () => {
                exists = true;
                events = 1;
                proposals = pair();
                return Promise.resolve(null);
            };
            db.DeleteDirection = () => {
                exists = false;
                return Promise.resolve(null);
            };
            db.GetDirection = () => Promise.resolve(exists ? view() : null);
            // La configuration enregistrée est gardée, et l'aperçu nomme ce qui sépare la
            // candidate de celle en vigueur — pour les tables hors service seulement : le
            // reste de la comparaison est tenu en Go.
            db.SetDirectionConfig = (_id, blob) => {
                CONFIG = JSON.parse(blob || '{}');
                events += 1;
                return Promise.resolve(null);
            };
            db.PreviewDirectionConfig = (_id, blob) => {
                const next = JSON.parse(blob || '{}');
                const list = (c) => (c?.tables?.unavailable || []).join(', ');
                const changes = list(CONFIG) === list(next) ? [] : [{ code: 'tablesUnavailable', phase: 0, from: list(CONFIG), to: list(next) }];
                // La consolante d'une phase, comparée comme le fait diffConfig en Go.
                (next.phases || []).forEach((ph, i) => {
                    const was = !!CONFIG.phases?.[i]?.consolation;
                    if (was !== !!ph.consolation) changes.push({ code: 'consolation', phase: i + 1, from: String(was), to: String(!!ph.consolation) });
                });
                // Les verrous posés par l'état : une phase dont le tirage est fait.
                return Promise.resolve({ changes, refusals: [], locks: lockedPhases, opened: 0, current: -1, started: false });
            };
            db.SetDirectionStrings = () => Promise.resolve(null);
            db.WriteDirectionPage = () => Promise.resolve('');
            // Les rondes déjà lancées : l'état S4 en a deux, sans aucun match en cours.
            db.DirectionRounds = () => Promise.resolve(roundsAtStart || (running.length ? 1 : 0));
            // La feuille d'une ronde annoncée : rien n'est lancé, rien n'est écrit au journal ;
            // le spec relit ce qui a été demandé dans window.__announcedSheets.
            window.__announcedSheets = [];
            db.WriteDirectionUpcomingSheet = (_id, announced) => {
                window.__announcedSheets.push({ announced, proposals: proposals.length, events });
                return Promise.resolve('/tmp/appariements-annonce.html');
            };
            db.WriteDirectionPairingSheet = () => Promise.resolve('/tmp/appariements.html');

            db.EnterParticipants = (_id, blob) => {
                // Le backend attribue un identifiant à qui n'en a pas (`participantID`) : sans
                // lui, deux inscrits sans id auraient la même clé dans la liste.
                for (const p of JSON.parse(blob || '[]')) players.push({ ...p, id: p.id || 'p' + (players.length + 1) });
                proposals = pair();
                events += 1;
                return Promise.resolve(null);
            };
            db.AddParticipant = (_id, name, club, rating) => {
                players.push({ id: 'p' + (players.length + 1), name, club, rating: Number(rating) || 0 });
                proposals = pair();
                events += 1;
                return Promise.resolve(view());
            };
            db.UpdateParticipant = () => Promise.resolve(view());
            db.WithdrawParticipant = (_id, participantId) => {
                const p = players.find((x) => x.id === participantId);
                if (p) p.state = 'withdrawn';
                proposals = pair();
                events += 1;
                return Promise.resolve(view());
            };
            db.ReinstateParticipant = (_id, participantId) => {
                const p = players.find((x) => x.id === participantId);
                if (!p || p.state !== 'withdrawn') return Promise.reject(new Error('not withdrawn'));
                delete p.state;
                proposals = pair();
                events += 1;
                return Promise.resolve(view());
            };
            db.Participants = () =>
                Promise.resolve(
                    players.map((p) => ({
                        id: p.id,
                        name: p.name,
                        club: p.club || '',
                        rating: p.rating || 0,
                        state: p.state || (running.some((m) => m.a === p.id || m.b === p.id) ? 'playing' : 'free'),
                        wins: 0,
                        losses: 0,
                        lives: 2,
                        opponents: [],
                        table: 0
                    }))
                );
            db.EntrySuggestions = () => Promise.resolve([]);
            db.FreeParticipants = () => Promise.resolve(players.filter((p) => p.state !== 'withdrawn' && !running.some((m) => m.a === p.id || m.b === p.id)).map((p) => ({ id: p.id, name: p.name })));

            db.ConfirmProposal = (_id, blob) => {
                const a = JSON.parse(blob || '{}');
                proposals = proposals.filter((p) => p.key !== a.key);
                if (a.kind === 'start_match') start(a.a, a.b, a.length, 0);
                return Promise.resolve(view());
            };
            db.ConfirmAllProposals = () => {
                const queue = proposals.slice();
                proposals = [];
                for (const a of queue) if (a.kind === 'start_match') start(a.a, a.b, a.length, 0);
                return Promise.resolve(view());
            };
            db.StartMatchManually = (_id, a, b, length, table) => {
                start(a, b, length, table);
                return Promise.resolve(view());
            };

            function finish(matchId, winner) {
                const i = running.findIndex((m) => m.ID === matchId);
                if (i < 0) return;
                const m = running[i];
                running.splice(i, 1);
                events += 1;
                last = {
                    matchId: m.ID,
                    a: m.a,
                    b: m.b,
                    aName: nameOf(m.a),
                    bName: nameOf(m.b),
                    winner,
                    winnerName: nameOf(winner),
                    correctable: true,
                    cancellable: false
                };
                results.push({ seq: events, matchId: m.ID, a: m.a, b: m.b, winner });
                proposals = pair();
            }

            db.EnterResult = (_id, matchId, winner) => {
                finish(matchId, winner);
                return Promise.resolve(view());
            };
            db.EnterForfeit = (_id, matchId, winner) => {
                finish(matchId, winner);
                return Promise.resolve(view());
            };
            // Corriger vise UN match, pas forcément le dernier : c'est tout le sens de corriger
            // depuis l'Historique.
            db.CorrectResult = (_id, matchId, winner) => {
                const r = results.find((x) => x.matchId === matchId);
                if (r) {
                    r.winner = winner;
                    events += 1;
                }
                if (last && last.matchId === matchId) {
                    last.winner = winner;
                    last.winnerName = nameOf(winner);
                }
                return Promise.resolve(view());
            };
            db.CancelMatch = () => Promise.resolve(view());
            db.MoveMatchToTable = () => Promise.resolve(view());
            db.LastDecision = () => Promise.resolve(last);
            db.FinishedMatches = () => Promise.resolve([]);
            db.CloseDirection = () => {
                finished = true;
                events += 1;
                return Promise.resolve(view());
            };
            db.ReopenDirection = () => {
                finished = false;
                events += 1;
                return Promise.resolve(view());
            };

            db.TableGrid = () =>
                Promise.resolve(
                    Array.from({ length: CONFIG.tables.count }, (_, i) => {
                        const t = i + 1;
                        const m = running.find((x) => x.Table === t);
                        if (!m) return (CONFIG.tables.unavailable || []).includes(t) ? { table: t, free: false, unavailable: true } : { table: t, free: true };
                        return {
                            table: t,
                            free: false,
                            matchId: m.ID,
                            a: m.a,
                            b: m.b,
                            aName: nameOf(m.a),
                            bName: nameOf(m.b),
                            length: m.Length,
                            elapsedSeconds: 60
                        };
                    })
                );

            db.Brackets = () => Promise.resolve([]);
            // Le classement compte les victoires, rien de plus : assez pour qu'une correction le
            // fasse bouger à l'écran. Le vrai, sans départage, est tenu en Go.
            db.Standings = () => {
                const wins = (id) => results.filter((r) => r.winner === id).length;
                const rows = players
                    .map((p) => ({ id: p.id, name: p.name, club: p.club || '', w: wins(p.id) }))
                    .sort((x, y) => y.w - x.w || x.name.localeCompare(y.name))
                    .map((p, i) => ({ id: p.id, name: p.name, club: p.club, rank: i + 1, shared: false, note: { kind: 'record', wins: p.w, losses: 0 } }));
                return Promise.resolve({ finished, pool: 0, retained: 0, payable: 0, entrants: players.length, sections: results.length ? [{ name: '', rows }] : [] });
            };
            // Le CSV du classement : le texte que « Copier » et « Enregistrer… » reçoivent
            // tous deux. Un texte qui dépend des résultats, pour qu'une copie périmée se voie.
            db.StandingsCSV = () => Promise.resolve('Phase;Rang;id;Joueur\r\n' + results.map((r, i) => `;${i + 1};${r.winner};${nameOf(r.winner)}\r\n`).join(''));
            // Le dialogue natif d'enregistrement : il rend le chemin choisi, et le spec relit ce
            // qui a été « écrit » dans window.__savedCSV.
            window.__savedCSV = [];
            window.go.gui.App.SaveCSV = (name, body) => {
                window.__savedCSV.push({ name, body });
                return Promise.resolve('/home/nadia/' + name);
            };
            // Le presse-papier, relu de même : Playwright n'y donne pas accès sans permission.
            window.__copied = [];
            try {
                Object.defineProperty(navigator, 'clipboard', {
                    configurable: true,
                    value: { writeText: (s) => (window.__copied.push(s), Promise.resolve()), readText: () => Promise.resolve(window.__copied.at(-1) || '') }
                });
            } catch {
                /* un navigateur qui refuse : les specs qui lisent __copied échoueront en le disant */
            }

            db.History = () =>
                Promise.resolve(
                    results.map((r) => ({
                        seq: r.seq,
                        kind: 'result',
                        time: '',
                        matchId: r.matchId,
                        a: r.a,
                        b: r.b,
                        aName: nameOf(r.a),
                        bName: nameOf(r.b),
                        winner: r.winner,
                        winnerName: nameOf(r.winner),
                        correctable: true
                    }))
                );
            db.Clock = () => Promise.resolve({ elapsedSeconds: 600, played: 0, running: running.length, minutesPerPoint: 8, plannedPerPoint: 8, slowMatches: 0, warnings: 0 });
            db.Slots = () => Promise.resolve([]);
            db.UnattachedMatches = () => Promise.resolve([]);

            // Le tournoi lui-même : le coût d'entrée se mesure depuis « nouveau tournoi », donc
            // la liste doit grandir quand on en crée un.
            const tournaments = directed ? [{ id: TOURNAMENT_ID, name: 'Open de Lyon', date: '2026-09-12', location: 'Lyon', matchCount: 0 }] : [];
            db.GetAllTournaments = () => Promise.resolve(tournaments.slice());
            db.CreateTournament = (name, date, location) => {
                tournaments.push({ id: TOURNAMENT_ID, name, date, location, matchCount: 0 });
                return Promise.resolve(TOURNAMENT_ID);
            };
            db.GetTournamentMatches = () => Promise.resolve([]);

            // L'annuaire. Un tournoi précédent, déjà dirigé, dont on reprend les
            // inscrits : c'est le geste que le budget mesure.
            const PREVIOUS_ID = 99;
            db.DirectorySources = () => Promise.resolve([{ tournamentId: PREVIOUS_ID, name: 'Open de Lyon, mars', date: '2026-03-14', entrants: entrants.length }]);
            db.DirectoryEntrants = () => Promise.resolve(entrants.map((p) => ({ name: p.name, club: p.club, rating: p.rating, entries: 1 })));
            db.Directory = () => Promise.resolve(entrants.map((p) => ({ name: p.name, club: p.club, rating: p.rating, entries: 1 })));
            db.DirectoryCSV = () => Promise.resolve('name,club,rating\n' + entrants.map((p) => `${p.name},${p.club},${p.rating}\n`).join(''));
            db.ParseDirectoryCSV = () => Promise.resolve({ rows: [], errors: [], skipped: [] });

            // Une place d'exemption libre. Le faux moteur ne tire aucun tableau : il
            // fournit la FORME que l'interface doit savoir montrer — où le retardataire entre —
            // et c'est ce que le budget mesure.
            let slots = [{ phase: 0, section: 'main', key: 'm-1-2', label: { kind: 'bracket_round', n: 1 } }];
            db.DirectionFreeSlots = () => Promise.resolve(slots.slice());
            db.AddParticipantAtSlot = (_id, name, club, rating, _section, key) => {
                slots = slots.filter((s) => s.key !== key);
                players.push({ id: 'p' + (players.length + 1), name, club, rating: Number(rating) || 0 });
                proposals = pair();
                events += 1;
                return Promise.resolve(view());
            };
            db.GetMatchesByTournament = () => Promise.resolve([]);
        },
        {
            entrants: opts.entrants || ENTRANTS,
            directed: opts.directed !== false,
            tableCount: opts.tables || 4,
            runningAtStart: opts.running || 0,
            roundsAtStart: opts.rounds || 0,
            phases: opts.phases || null,
            lockedPhases: opts.locks || []
        }
    );
}
