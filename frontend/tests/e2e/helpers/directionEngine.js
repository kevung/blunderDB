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
export const S4_FRIDAY = { entrants: crowd(25), tables: 6, running: 0, rounds: 2, seats: true };

/**
 * L'état S3 du samedi, 20 h : le speed (32 inscrits) s'ouvre dans la salle de 14 tables où le
 * principal joue encore sur les tables 1 à 7. Les deux épreuves sont dans une Rencontre.
 */
export const S3_SATURDAY_20H = {
    entrants: crowd(32),
    tables: 14,
    seats: true,
    room: { id: 5, name: 'Festival de Lyon', sister: 'Principal', busy: [1, 2, 3, 4, 5, 6, 7] }
};

/**
 * Installe la Direction factice. À appeler APRÈS `installWailsMock` et avant `page.goto`.
 *
 * @param {import('@playwright/test').Page} page
 * @param {{directed?: boolean, entrants?: typeof ENTRANTS, tables?: number, running?: number, rounds?: number, seats?: boolean, phases?: Object[], locks?: Object[], unavailable?: number[]}} [opts]
 *   `directed: false` part d'un tournoi non dirigé, pour mesurer le coût d'entrée depuis rien.
 *   `entrants`, `tables` et `running` changent la taille de la salle (défaut : les quatre
 *   inscrits, quatre tables, aucun match) ; `running` matchs sont lancés d'avance. `rounds`
 *   donne le nombre de rondes déjà lancées, `phases` le format, `locks` les verrous que
 *   l'aperçu de configuration renvoie (une phase dont le tirage est fait). `seats` donne aux
 *   propositions les tables libres, et aux autres la raison `waiting_table`, comme le moteur ;
 *   sans lui, la file ignore la salle.
 */
export async function installDirectionEngine(page, opts = {}) {
    await page.addInitScript(
        ({ entrants, directed, tableCount, runningAtStart, roundsAtStart, seats, phases, lockedPhases, room, unavailable }) => {
            const db = window.go.database.Database;

            const TOURNAMENT_ID = 1;
            let CONFIG = {
                name: 'Open de Lyon',
                tables: { count: tableCount, ...(unavailable.length ? { unavailable } : {}) },
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
                // Les tables de l'épreuve sœur et celles hors service ne sont pas à donner.
                const busy = new Set([...running.map((m) => m.Table), ...(room ? room.busy : []), ...(CONFIG.tables.unavailable || [])]);
                const freeTables = Array.from({ length: tableCount }, (_, i) => i + 1).filter((t) => !busy.has(t));
                for (let i = 0; i + 1 < free.length; i += 2) {
                    const table = seats ? freeTables.shift() || 0 : 0;
                    out.push({
                        kind: 'start_match',
                        phase: 0,
                        key: `k${i}`,
                        a: free[i].id,
                        b: free[i + 1].id,
                        length: 7,
                        table,
                        ...(seats && !table ? { reason: 'waiting_table' } : {}),
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
                    engineVersion: 'v0.3.0',
                    outputDir: '',
                    config: CONFIG,
                    proposals,
                    warnings: [],
                    ranking: [],
                    players: players.map((p) => ({ id: p.id, name: p.name, club: p.club, rating: p.rating })),
                    running,
                    phase: 0,
                    finished,
                    eventCount: events,
                    rencontreId: room ? room.id : 0,
                    busyTables: room ? room.busy.slice() : []
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

            db.ListDirections = () => Promise.resolve(exists ? [{ tournamentId: TOURNAMENT_ID, state: view().state, engineVersion: 'v0.3.0', outputDir: '', updatedAt: '' }] : []);
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
            // Un id qui n'est pas celui de l'épreuve tenue ici : `room.dual` (D6.4) donne une
            // vraie deuxième Direction, plate, pour les specs d'onglets ; sinon, inconnu.
            db.GetDirection = (id) => {
                if (room && room.dual && id === 2) return Promise.resolve(room.dual);
                if (id && id !== TOURNAMENT_ID) return Promise.resolve(null);
                return Promise.resolve(exists ? view() : null);
            };
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
                const refusals = [];
                // La consolante d'une phase, comparée comme le fait diffConfig en Go ; celle d'une
                // phase tirée est refusée, comme le fait CheckConfig dans le moteur.
                (next.phases || []).forEach((ph, i) => {
                    const was = !!CONFIG.phases?.[i]?.consolation;
                    if (was === !!ph.consolation) return;
                    const change = { code: 'consolation', phase: i + 1, from: String(was), to: String(!!ph.consolation) };
                    changes.push(change);
                    const lock = (lockedPhases || []).find((l) => l.phase === i + 1 && l.locked);
                    if (lock && !refusals.length) refusals.push({ ...change, reason: lock.reason });
                });
                return Promise.resolve({ changes, refusals, locks: lockedPhases, opened: 0, current: -1, started: false });
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
            // Absenter/revenir (D7.1) : un joueur absent garde son rang et ses vies, il quitte
            // seulement l'appariement — donc `pair()` n'est pas relancé sur `withdrawn`.
            db.MakeParticipantAbsent = (_id, participantId, until, round) => {
                const p = players.find((x) => x.id === participantId);
                if (!p) return Promise.reject(new Error('no such participant'));
                if (!until && !round) return Promise.reject(new Error('needs until or round'));
                p.state = 'absent';
                p.absentUntil = until || '';
                p.absentRound = round || 0;
                events += 1;
                return Promise.resolve(view());
            };
            db.MakeParticipantAvailable = (_id, participantId) => {
                const p = players.find((x) => x.id === participantId);
                if (p && p.state === 'absent') {
                    delete p.state;
                    delete p.absentUntil;
                    delete p.absentRound;
                }
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
                        table: 0,
                        absentUntil: p.absentUntil || '',
                        absentRound: p.absentRound || 0
                    }))
                );
            db.EntrySuggestions = () => Promise.resolve([]);
            db.FreeParticipants = () => Promise.resolve(players.filter((p) => p.state !== 'withdrawn' && !running.some((m) => m.a === p.id || m.b === p.id)).map((p) => ({ id: p.id, name: p.name })));

            db.ConfirmProposal = (_id, blob) => {
                const a = JSON.parse(blob || '{}');
                proposals = proposals.filter((p) => p.key !== a.key);
                if (a.kind === 'start_match') start(a.a, a.b, a.length, a.table || 0);
                return Promise.resolve(view());
            };
            db.ConfirmAllProposals = () => {
                const queue = proposals.slice();
                proposals = [];
                // Un match sans table reste dans la file, comme dans ConfirmAllProposals.
                for (const a of queue) if (a.kind === 'start_match' && !a.reason) start(a.a, a.b, a.length, a.table);
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
            // Comme le service : une table occupée échange, une table hors service est refusée.
            window.__moves = [];
            db.MoveMatchToTable = (_id, matchId, table) => {
                const m = running.find((x) => x.ID === matchId);
                if ((CONFIG.tables.unavailable || []).includes(table)) return Promise.reject(new Error(`direction: table ${table} is out of service`));
                if (m && m.Table !== table) {
                    const o = running.find((x) => x.Table === table && x !== m);
                    if (o) o.Table = m.Table;
                    m.Table = table;
                    events += 1;
                    window.__moves.push([matchId, table]);
                }
                return Promise.resolve(view());
            };
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

            // La Rencontre : un geste de salle est compté, pour qu'une spec vérifie qu'il ne se
            // déclare qu'une fois.
            window.__roomGestures = [];
            function rencontre() {
                return {
                    id: room.id,
                    name: room.name,
                    tables: tableCount,
                    tournamentIds: [TOURNAMENT_ID, 2],
                    room: { tables: tableCount, unavailable: (CONFIG.tables.unavailable || []).slice(), breaks: [] },
                    members: [
                        { tournamentId: 2, name: room.sister, state: 'running' },
                        { tournamentId: TOURNAMENT_ID, name: 'Speed', state: 'draft' }
                    ]
                };
            }
            db.ListRencontres = () => Promise.resolve(room ? [rencontre()] : []);
            db.GetRencontre = (id) => Promise.resolve(room && room.id === id ? rencontre() : null);
            db.SetRencontreTableOutOfService = (_id, table, out) => {
                window.__roomGestures.push({ table, out });
                const rest = (CONFIG.tables.unavailable || []).filter((x) => x !== table);
                CONFIG = { ...CONFIG, tables: { ...CONFIG.tables, unavailable: out ? [...rest, table] : rest } };
                events += 1;
                proposals = pair();
                return Promise.resolve(rencontre());
            };

            db.TableGrid = () =>
                Promise.resolve(
                    Array.from({ length: CONFIG.tables.count }, (_, i) => {
                        const t = i + 1;
                        const m = running.find((x) => x.Table === t);
                        if (!m && (CONFIG.tables.unavailable || []).includes(t)) return { table: t, free: false, unavailable: true };
                        if (!m && room && room.busy.includes(t)) return { table: t, free: false, elsewhere: room.sister };
                        if (!m) return { table: t, free: true };
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

            // L'index de la recherche rapide : les joueurs de l'épreuve, ses tables et celles de la
            // sœur, comme le Go les rend (épreuves, joueurs, tables, matchs).
            function searchIndex() {
                const tid = TOURNAMENT_ID;
                const epreuve = CONFIG.name;
                const out = [{ kind: 'epreuve', tournamentId: tid, epreuve, name: epreuve }];
                if (room) out.push({ kind: 'epreuve', tournamentId: 2, epreuve: room.sister, name: room.sister });
                const playing = (p) => running.some((m) => m.a === p.id || m.b === p.id);
                const row = (p) => {
                    const m = running.find((x) => x.a === p.id || x.b === p.id);
                    return {
                        kind: 'player',
                        tournamentId: tid,
                        epreuve,
                        name: p.name,
                        playerId: p.id,
                        club: p.club || '',
                        state: p.state || (m ? 'playing' : 'free'),
                        table: m ? m.Table : 0,
                        opponent: m ? nameOf(m.a === p.id ? m.b : m.a) : ''
                    };
                };
                out.push(...players.filter(playing).map(row), ...players.filter((p) => !playing(p)).map(row));
                for (let t = 1; t <= CONFIG.tables.count; t++) {
                    const m = running.find((x) => x.Table === t);
                    if (m) out.push({ kind: 'table', tournamentId: tid, epreuve, name: nameOf(m.a) + ' – ' + nameOf(m.b), table: t, state: 'running' });
                    else if (room && room.busy.includes(t)) out.push({ kind: 'table', tournamentId: 2, epreuve: room.sister, name: 'Joueur X – Joueur Y', table: t, state: 'running' });
                    else out.push({ kind: 'table', tournamentId: 0, table: t, state: (CONFIG.tables.unavailable || []).includes(t) ? 'unavailable' : 'free' });
                }
                for (const m of running) out.push({ kind: 'match', tournamentId: tid, epreuve, name: nameOf(m.a) + ' – ' + nameOf(m.b), table: m.Table, matchId: m.ID, state: 'running' });
                return out;
            }
            db.DirectionSearchIndex = () => Promise.resolve(searchIndex());
            db.RencontreSearchIndex = () => Promise.resolve(searchIndex());

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
            seats: !!opts.seats,
            phases: opts.phases || null,
            lockedPhases: opts.locks || [],
            room: opts.room || null,
            unavailable: opts.unavailable || []
        }
    );
}
