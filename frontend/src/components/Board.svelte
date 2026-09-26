<script>
    import { t, tMsg } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { positionStore, matchContextStore } from '../stores/positionStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { isResponseCubeAction } from '../utils/cubeAction.js';
    import { parseMoveNotation, mirrorPosition, boardMetrics } from '../utils/boardGeometry.js';
    import { layerOf, drawStaticScene, drawDynamicScene, drawFrame } from '../utils/boardScene.js';
    import { defaultBoardConfig, applyPalette } from '../utils/boardConfig.js';
    import { attachBoardInteractions } from '../utils/boardInteractions.js';
    import { rankNeighboursOfCurrentPosition } from '../services/rankService.js';
    import { onMount, onDestroy } from 'svelte';
    import Two from 'two.js';
    import { get } from 'svelte/store';
    import { statusBarModeStore, isAnyModalOpen, activeModal, MODAL, pipcountVisibleStore, activeTabStore } from '../stores/uiStore';
    import { subscribeBoardRedrawTriggers as subscribeSharedRedrawTriggers } from '../services/boardRedraw.js';
    import { boardIsMirrored, labelsFlipped, screenOfModelPoint, screenOfNotationPoint } from '../services/boardOrientation.js';
    import { searchStructureModeStore, searchOfferedCubeStore } from '../stores/searchExcludePositionStore';
    import { boardColorsStore } from '../stores/boardColorsStore';
    import { sendPositionToEval } from '../services/positionService.js';
    import { copyBoardWithAnalysisImage, exportBoardImage } from '../services/clipboardService.js';
    import { setStatusBarMessage } from '../services/databaseService.js';
    import { viewStore } from '../stores/viewStore.js';
    import * as anki from '../services/ankiService.js';
    import { ankiDecksStore } from '../stores/ankiStore.js';
    import { quizPlayStore, quizPlaySourcesStore, quizPlayTargetsStore } from '../stores/quizPlayStore.js';
    import { transcriptionCubeRequestStore, transcriptionBoardSwapStore } from '../stores/transcriptionStore.js';
    import { resetBoardPlay } from '../services/transcriptionPlay.js';
    import ContextMenu from './ContextMenu.svelte';

    // Read-only mirrors of stores — always current when read inside drawing/handler functions
    let mode = $derived($statusBarModeStore);
    let showTakePoint2Modal = $derived($activeModal === MODAL.TAKE_POINT_2);
    let showTakePoint4Modal = $derived($activeModal === MODAL.TAKE_POINT_4);
    // No PANEL entry for the comment tab: CommentPanel is mounted by TabbedPanel per active tab.
    let showComment = $derived($activeTabStore === 'comments');
    // Préférence, surchargée pendant une question de Pions (uiStore.pipcountVisibleStore) ;
    // le redessin vient de boardRedraw.js.
    let showPipcount = $derived($pipcountVisibleStore);

    let canvasCfg = {
        aspectFactor: 0.72
    };

    // Partagée avec les diagrammes du rapport (utils/boardConfig.js) : une seule palette.
    let boardCfg = defaultBoardConfig();

    let two;
    let canvas;
    let width;
    let height;
    let unsubscribeBoardRedrawTriggers;
    let detachInteractions;
    let cubePosition = { x: 0, y: 0, size: 0 }; // where the cube was last drawn (hit-testing)
    let previousDice = get(positionStore).dice; // Save previous dice values

    // enterEvalMode() starts with dice [0, 0]; reset so the first die click
    // does not restore a stale previousDice.
    $effect(() => {
        if (mode === 'EVAL') {
            previousDice = [0, 0];
        }
    });

    // Read by drawBoard(); redraw is triggered by scheduleRedraw().
    let selectedMove = $derived($selectedMoveStore);

    // Redraw coalescing: several stores can dirty the board in one tick.
    // scheduleRedraw() sets a flag and requests one animation frame; by then
    // every store has settled, and drawBoard() (which re-reads them all)
    // paints once per frame.
    let redrawScheduled = false;
    let redrawFrameId = null;
    function scheduleRedraw() {
        if (redrawScheduled) return;
        redrawScheduled = true;
        redrawFrameId = requestAnimationFrame(() => {
            redrawScheduled = false;
            redrawFrameId = null;
            if (two && canvas) drawBoard();
        });
    }

    // Svelte 5 store-rule exception (CLAUDE.md): drawBoard() re-reads every
    // store whichever changed, so one grouped subscription beats an $effect
    // per store. positionStore's callback also resets the selected move on a
    // real navigation (id change) BEFORE the redraw; synchronous .subscribe()
    // makes that ordering unambiguous, an $effect's would not be.
    function subscribeBoardRedrawTriggers() {
        let previousPositionId = null;
        const unsubPosition = positionStore.subscribe(() => {
            const position = get(positionStore);
            // Only on a real navigation (id change), not on edits or analysis refresh.
            if (position.id !== previousPositionId) {
                selectedMoveStore.set(null);
                previousPositionId = position.id;
            }
            scheduleRedraw();
        });

        // Les autres déclencheurs : liste nommée et testée dans services/boardRedraw.js.
        const unsubShared = subscribeSharedRedrawTriggers(scheduleRedraw);

        return () => {
            unsubPosition();
            unsubShared();
        };
    }

    // boardCfg is a plain object read imperatively: mutate in place, then redraw.
    $effect(() => {
        applyPalette(boardCfg, $boardColorsStore);
        invalidateStaticLayer(); // triangles, bar and frame carry the palette
        if (two && canvas) scheduleRedraw();
    });

    let boardDescription = $derived.by(() => {
        const pos = $positionStore;
        if (!pos || !pos.board || !pos.board.points) return $t('board.label');
        let pip1 = 0;
        let pip2 = 0;
        pos.board.points.forEach((point, index) => {
            if (point.color === 0) pip1 += point.checkers * index;
            else if (point.color === 1) pip2 += point.checkers * (25 - index);
        });
        const roller = pos.player_on_roll === 0 ? $t('board.player1') : $t('board.player2');
        return $t('board.description', { pip1, pip2, roller });
    });
    function resizeBoard() {
        const container = canvas.parentElement;
        const containerWidth = container.clientWidth;
        const containerHeight = container.clientHeight;
        const heightFromWidth = containerWidth * canvasCfg.aspectFactor;
        if (heightFromWidth <= containerHeight) {
            width = containerWidth;
            height = heightFromWidth;
        } else {
            height = containerHeight;
            width = containerHeight / canvasCfg.aspectFactor;
        }
        two.width = width;
        two.height = height;
        two.renderer.setSize(width, height);
        invalidateStaticLayer(); // every coordinate depends on the size
        // Measure synchronously, repaint coalesced (a resize burst paints once per frame).
        scheduleRedraw();
    }

    function resetBoard() {
        positionStore.update((pos) => {
            pos.board.points.forEach((point) => (point.checkers = 0));
            pos.board.bearoff = [15, 15]; // Reset bearoff
            pos.cube.value = 0; // Set cube in the middle
            pos.cube.owner = -1; // Reset cube owner
            pos.score = [7, 7]; // Reset score to 7 away for both players
            pos.dice = [3, 1]; // Set dice to 3 and 1
            pos.decision_type = 0; // Checker decision
            pos.player_on_roll = 0; // Player on roll is below
            return pos;
        });
    }

    // Eval's "clear": blank board with enterEvalMode()'s defaults (money, no
    // dice), not resetBoard()'s 7-away score, which the race table would read.
    function resetEvalBoard() {
        positionStore.update((pos) => {
            pos.board.points.forEach((point) => (point.checkers = 0));
            pos.board.bearoff = [15, 15];
            pos.cube.value = 0;
            pos.cube.owner = -1;
            pos.score = [-1, -1]; // money
            pos.dice = [0, 0]; // no move in progress
            pos.decision_type = 0;
            pos.player_on_roll = 0;
            return pos;
        });
    }

    function logCanvasSize() {
        const actualWidth = canvas.clientWidth;
        const actualHeight = canvas.clientHeight;
        logger.log('Actual canvas width: ', actualWidth, 'Actual canvas height: ', actualHeight);
        logger.log('Two.js width: ', two.width, 'Two.js height: ', two.height);
    }

    function setBoardOrientation(orientation) {
        boardCfg.orientation = orientation;
        invalidateStaticLayer(); // labels and bearoff side move
        scheduleRedraw();
    }

    function handleOrientationChange(event) {
        const isAnyModalOpenVal = get(isAnyModalOpen);
        if (isAnyModalOpenVal || showComment) return; // Disable orientation change when any modal or comment panel is open
        if (event.ctrlKey && event.key === 'ArrowLeft') {
            setBoardOrientation('left');
        } else if (event.ctrlKey && event.key === 'ArrowRight') {
            setBoardOrientation('right');
        }
    }

    function handleKeyDown(event) {
        if ((mode !== 'EDIT' && mode !== 'EVAL') || showTakePoint2Modal || showTakePoint4Modal) return; // Disable shortcuts when TakePoint2Modal or TakePoint4Modal is open

        if (event.key === 'Backspace' && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
            event.preventDefault();
            if (mode === 'EVAL') resetEvalBoard();
            else resetBoard();
        }
    }

    onMount(() => {
        canvas = document.getElementById('backgammon-board');
        const params = { width: window.innerWidth, height: window.innerHeight };
        two = new Two(params).appendTo(canvas);

        const container = canvas.parentElement;
        const containerWidth = container.clientWidth;
        const containerHeight = container.clientHeight;
        const heightFromWidth = containerWidth * canvasCfg.aspectFactor;
        if (heightFromWidth <= containerHeight) {
            width = containerWidth;
            height = heightFromWidth;
        } else {
            height = containerHeight;
            width = containerHeight / canvasCfg.aspectFactor;
        }
        two.width = width;
        two.height = height;
        two.renderer.setSize(width, height);

        // boardInteractions.js reads live state through getters: no re-attach on redraw.
        detachInteractions = attachBoardInteractions(canvas, {
            getMode: () => mode,
            getSize: () => ({ width, height }),
            cfg: boardCfg,
            getCubeBox: () => cubePosition,
            stores: {
                position: positionStore,
                structureMode: searchStructureModeStore,
                activeTab: activeTabStore,
                offeredCube: searchOfferedCubeStore,
                anyModalOpen: isAnyModalOpen,
                quizPlay: quizPlayStore,
                // Transcription : le plateau pose la demande, le panneau décide.
                transcriptionCube: transcriptionCubeRequestStore
            },
            // Plateau retourné : point cliqué → point du modèle, 25 - p.
            quizDisplayMirrored: () => displayMirrored(),
            // Côté de sortie du camp au trait : 0 en bas, 1 en haut (en
            // transcription le joueur 2 sort par le haut).
            quizBearoffSide: () => getDisplayPosition().player_on_roll,
            resetQuizPlay: () => quizPlayStore.update((s) => (s ? resetBoardPlay(s, get(positionStore)) : s)),
            getPreviousDice: () => previousDice,
            setPreviousDice: (dice) => (previousDice = dice),
            reset: () => (mode === 'EVAL' ? resetEvalBoard() : resetBoard()),
            openContextMenu,
            logger
        });
        // No direct drawBoard(): the subscriptions fire synchronously and
        // schedule the first paint; drawing here too would build it twice.
        window.addEventListener('resize', resizeBoard);
        window.addEventListener('keydown', handleOrientationChange);
        window.addEventListener('keydown', handleKeyDown);

        unsubscribeBoardRedrawTriggers = subscribeBoardRedrawTriggers();

        logCanvasSize();
        window.addEventListener('resize', logCanvasSize);
    });

    onDestroy(() => {
        if (detachInteractions) detachInteractions();
        window.removeEventListener('resize', resizeBoard);
        window.removeEventListener('resize', logCanvasSize);
        window.removeEventListener('keydown', handleOrientationChange);
        window.removeEventListener('keydown', handleKeyDown);
        if (unsubscribeBoardRedrawTriggers) unsubscribeBoardRedrawTriggers();
        // Cancel a pending rAF so drawBoard() never hits a detached two.js.
        if (redrawFrameId !== null) cancelAnimationFrame(redrawFrameId);
    });

    // Board context menu; gating (not in EDIT/EVAL, not over a modal) is boardInteractions.js's.
    let boardMenu = $state(null);

    function openContextMenu({ x, y }) {
        const items = [
            {
                label: $t('board.menu.evaluate'),
                // The position as displayed (possibly mirrored), not the stored record.
                onClick: () => sendPositionToEval(getDisplayPosition())
            },
            {
                label: $t('board.menu.evaluateMirror'),
                onClick: () => sendPositionToEval(mirrorPosition(getDisplayPosition()))
            },
            {
                label: $t('board.menu.copyImageWithAnalysis'),
                onClick: () => copyBoardWithAnalysisImage()
            },
            // Un fichier (souvent vectoriel) plutôt qu'une copie.
            {
                label: $t('board.menu.saveImageSVG'),
                onClick: () => exportBoardImage('svg')
            },
            {
                label: $t('board.menu.saveImagePNG'),
                onClick: () => exportBoardImage('png')
            },
            {
                label: $t('board.menu.newView'),
                onClick: () => viewStore.addView()
            }
        ];

        // Anki cards are keyed by position id: nothing to add for a scratch board (id 0).
        const position = get(positionStore);
        if (position?.id) {
            items.push(...ankiDeckMenuItems(position.id));
            // « Positions voisines » (ADR-0043) ; pas pour un plateau brouillon (id 0).
            items.push({
                label: $t('board.menu.neighbours'),
                onClick: () => rankNeighboursOfCurrentPosition()
            });
        }

        boardMenu = { x, y, items };

        // Decks load only once the Anki tab is visited: fetch here, non-blocking,
        // and append the deck items when it resolves.
        if (position?.id && get(ankiDecksStore).length === 0) {
            anki.loadDecks()
                .then(() => {
                    if (boardMenu?.x !== x || boardMenu?.y !== y) return; // menu closed/reopened meanwhile
                    const extra = ankiDeckMenuItems(position.id);
                    if (extra.length > 0) boardMenu = { ...boardMenu, items: [...boardMenu.items, ...extra] };
                })
                .catch((err) => logger.error('Failed to load Anki decks for the board context menu:', err));
        }
    }

    function ankiDeckMenuItems(positionId) {
        return get(ankiDecksStore).map((deck) => ({
            label: $t('board.menu.addToAnkiDeck', { deck: deck.name }),
            onClick: () => addPositionToAnkiDeck(deck, positionId)
        }));
    }

    async function addPositionToAnkiDeck(deck, positionId) {
        try {
            await anki.addPositionToDeck(deck.id, positionId);
            setStatusBarMessage(tMsg('status.positionAddedToDeck', { deck: deck.name }));
        } catch (err) {
            logger.error('Failed to add the position to the Anki deck:', err);
            setStatusBarMessage(tMsg('status.failedAddToDeck', { err }));
        }
    }

    // Le sens du plateau est décidé une fois (services/boardOrientation.js) pour
    // les pions, les clics et les flèches.
    function displayIsMirrored(position) {
        return boardIsMirrored({
            mode,
            position,
            matchContext: get(matchContextStore),
            transcriptionSwap: get(transcriptionBoardSwapStore)
        });
    }

    // La position telle qu'elle est DESSINÉE.
    function getDisplayPosition() {
        const stored = get(positionStore);
        // Quiz : seul le damier suit le coup en cours ; le reste vient de la position.
        const play = get(quizPlayStore);
        const position = play ? { ...stored, board: play.board } : stored;
        return displayIsMirrored(position) ? mirrorPosition(position) : position;
    }

    // Le miroir en vigueur ; le coup en cours n'y entre pas.
    function displayMirrored() {
        return displayIsMirrored(get(positionStore));
    }

    // Points numérotés depuis le camp du joueur 2 ? Distinct du miroir : en
    // transcription le joueur 1 reste en bas, on renumérote sans retourner.
    function isPlayer2Perspective(displayPosition) {
        return labelsFlipped(displayPosition);
    }

    // Take/pass decision: the offered cube is drawn mid-board. Signalled by the
    // played cube action (match move, else analysis); in EDIT only for an
    // explicit take/pass search, never from stale analysis.
    function isOfferedCube(position) {
        if (position.decision_type !== 1) return false;
        if (mode === 'EDIT') return get(searchOfferedCubeStore) === true;
        const matchCtx = get(matchContextStore);
        if (matchCtx && matchCtx.isMatchMode && matchCtx.movePositions.length > 0) {
            const mp = matchCtx.movePositions[matchCtx.currentIndex];
            return !!mp && isResponseCubeAction(mp.cube_action);
        }
        const ana = get(analysisStore);
        const acts = (ana && ana.playedCubeActions) || [];
        return acts.some(isResponseCubeAction);
    }

    // Points offerts par le coup en cours, en numéros affichés : les pas de
    // `LegalMoves` sont absolus, convertis par le même `mirrored` que le clic.
    function playHighlights(mirrored) {
        const play = get(quizPlayStore);
        if (!play) return {};
        const shown = (point) => screenOfModelPoint(point, mirrored);
        // Le point choisi est toujours marqué, même hors des règles (ADR-0052).
        const picked = play.selected === null || play.selected === undefined ? [] : [play.selected];
        // Départs allumés seulement coup engagé : sinon presque tous le sont (21 jets).
        const engaged = play.steps.length > 0 || picked.length > 0;
        return {
            sources: engaged ? [...new Set([...$quizPlaySourcesStore, ...picked])].map(shown) : [],
            targets: [...$quizPlayTargetsStore].map(shown),
            selected: play.selected === null || play.selected === undefined ? null : shown(play.selected)
        };
    }

    // Flèches : la notation est dans la numérotation du camp au trait, donc
    // `flip` (joueur dessiné en haut), pas `mirrored` — ils divergent en transcription.
    function selectedMoveArrows(flipped) {
        const moves = parseMoveNotation(selectedMove);
        if (moves.length === 0 || !flipped) return moves;
        return moves.map((m) => ({ ...m, from: screenOfNotationPoint(m.from, true), to: screenOfNotationPoint(m.to, true) }));
    }

    // Static layer (triangles, labels, bar, outline) is rebuilt only on size,
    // orientation, palette or numbering change; scheduleRedraw() refills only
    // the dynamic group. Board.redraw.test.js counts two.clear() as a static
    // rebuild and two.update() as a paint.
    let staticLayer = null; // triangles, labels, bar — null = must be rebuilt
    let dynamicLayer = null; // emptied and refilled on every redraw
    let staticFlip = null; // the label side staticLayer was built for

    function invalidateStaticLayer() {
        staticLayer = null;
    }

    function rebuildStaticLayers(geom, flip) {
        two.clear();
        staticLayer = two.makeGroup();
        dynamicLayer = two.makeGroup();
        const frameLayer = two.makeGroup(); // above the checkers so the outline keeps its linewidth
        drawStaticScene(layerOf(two, staticLayer), geom, boardCfg, flip);
        drawFrame(layerOf(two, frameLayer), geom, boardCfg);
        staticFlip = flip;
    }

    export function drawBoard() {
        if (!two) return; // Safety check

        const geom = boardMetrics(width, height, boardCfg.widthFactor);
        const position = getDisplayPosition();
        // `mirrored` convertit un point absolu, `flip` un point de notation (boardOrientation.js).
        const flip = isPlayer2Perspective(position);
        const mirrored = displayMirrored();
        logger.log('drawBoard', width, height, 'decision_type:', position.decision_type);

        if (!staticLayer || staticFlip !== flip) rebuildStaticLayers(geom, flip);
        dynamicLayer.remove(dynamicLayer.children);
        cubePosition = drawDynamicScene(layerOf(two, dynamicLayer), geom, boardCfg, position, {
            offeredCube: isOfferedCube(position),
            showPipcount,
            play: playHighlights(mirrored),
            moves: selectedMoveArrows(flip)
        });

        two.update();
    }
</script>

<div class="canvas-container">
    <div id="backgammon-board" class="full-size-board" role="img" aria-label={boardDescription}></div>
    {#if boardMenu}
        <ContextMenu x={boardMenu.x} y={boardMenu.y} items={boardMenu.items} onClose={() => (boardMenu = null)} />
    {/if}
</div>

<style>
    .canvas-container {
        width: 100%;
        height: 100%;
        display: flex;
        justify-content: center;
        align-items: center;
        margin: 0;
        padding: 0;
        overflow: hidden;
    }

    #backgammon-board {
        max-width: 100%;
        max-height: 100%;
        box-sizing: border-box;
        padding: 0;
        margin: 0;
        user-select: none;
    }
</style>
