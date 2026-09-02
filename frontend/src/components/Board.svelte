<script>
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { positionStore, matchContextStore } from '../stores/positionStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore'; // Import analysisStore and selectedMoveStore
    import { isResponseCubeAction } from '../utils/cubeAction.js';
    import { parseMoveNotation, mirrorPosition, boardMetrics, boardMouseToDrawing, checkerPointAndCountAt } from '../utils/boardGeometry.js';
    import { EXCLUDE_EMPTY, drawStaticScene, drawDynamicScene, drawFrame } from '../utils/boardScene.js';
    import { onMount, onDestroy } from 'svelte';
    import Two from 'two.js';
    import { get } from 'svelte/store';
    import { statusBarModeStore, isAnyModalOpen, activeModal, MODAL, showPipcountStore, activeTabStore } from '../stores/uiStore';
    import { searchStructureModeStore, searchOfferedCubeStore } from '../stores/searchExcludePositionStore';
    import { boardColorsStore } from '../stores/boardColorsStore';
    import { sendPositionToEval } from '../services/positionService.js';
    import ContextMenu from './ContextMenu.svelte';

    // Read-only mirrors of stores — always current when read inside drawing/handler functions
    let mode = $derived($statusBarModeStore);
    let showTakePoint2Modal = $derived($activeModal === MODAL.TAKE_POINT_2);
    let showTakePoint4Modal = $derived($activeModal === MODAL.TAKE_POINT_4);
    // openPanels/PANEL.COMMENT is never set: the 'comments' tab has driven no
    // PANEL since the tabHandler.js refactor (applyTabPanels only wires
    // matches/stats/tournaments/collections there), so that flag reads as
    // permanently closed. The comment panel is the CommentPanel instance
    // TabbedPanel mounts, one-to-one with the active tab (see that file's
    // {#if} — same fix as AnalysisPanel's stuck selectedMoveStore).
    let showComment = $derived($activeTabStore === 'comments');
    let showPipcount = $derived($showPipcountStore);

    let canvasCfg = {
        aspectFactor: 0.72
    };

    let boardCfg = {
        widthFactor: 0.75, // Increase widthFactor to make the board take up more space
        orientation: 'right',
        fill: '#f0f0f0', // Light grey background
        stroke: '#333333', // Dark grey border
        linewidth: 3,
        triangle: {
            fill1: '#d9d9d9', // Light grey
            fill2: '#a6a6a6', // Slightly darker grey for balanced contrast
            stroke: '#333333',
            linewidth: 1.3 // Changed linewidth to 1
        },
        label: {
            size: 20,
            distanceToBoard: 0.3
        },
        checker: {
            sizeFactor: 0.97,
            colors: ['#333333', '#ffffff'], // Dark grey and white checkers
            linewidth: 2.5 // Added linewidth property and set to 2
        },
        dice: {
            fill: '#ffffff', // White dice face
            dot: '#000000' // Dice pips
        },
        cube: {
            fill: '#ffffff' // White doubling cube face
        }
    };

    let two;
    let canvas;
    let width;
    let height;
    let unsubscribeBoardRedrawTriggers;
    let startMousePos = null;
    // Manual double-click tracking for the Except "must be empty" marker. Native
    // 'dblclick' is unreliable here because each click redraws (recreates) the
    // two.js shapes, so the two clicks land on different DOM nodes.
    let lastExceptClick = null;
    let cubePosition = { x: 0, y: 0, size: 0 }; // where the cube was last drawn (hit-testing)
    let previousDice = get(positionStore).dice; // Save previous dice values

    // enterEPCMode() (positionService.js) always starts the Eval panel's
    // position with dice = [0, 0]; without this, the first die click of an
    // Eval session would restore whatever previousDice was left over from an
    // unrelated position seen earlier, instead of starting from a clean roll.
    $effect(() => {
        if (mode === 'EPC') {
            previousDice = [0, 0];
        }
    });

    // selectedMove is read inside drawBoard() via this derived; the actual
    // redraw is triggered (coalesced) by scheduleRedraw() — see
    // subscribeBoardRedrawTriggers() below.
    let selectedMove = $derived($selectedMoveStore);

    // ── Redraw coalescing ──────────────────────────────────────────────────
    // Several independent triggers can each ask for a board redraw within the
    // same tick (a position navigation touches positionStore, resets
    // selectedMoveStore, and often analysisStore reloads right after). Before
    // this, each trigger called drawBoard() directly and a single navigation
    // could rebuild the whole two.js scene (two.clear() + ~100 SVG nodes
    // destroyed/recreated) 3-4 times. scheduleRedraw() sets a dirty flag and
    // asks for a single requestAnimationFrame; by the time it fires, every
    // store involved has already settled its final value (Svelte's reactive
    // updates and the .subscribe() callbacks below all run synchronously,
    // long before the next animation frame), so drawBoard() — which always
    // re-reads every store itself — paints the fully-settled state exactly
    // once per frame.
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

    // Svelte 5 invariant exception (CLAUDE.md): four stores each mark the
    // board dirty for unrelated reasons (position navigation, a move picked
    // in the analysis panel, analysis reloading, the take/pass "offered cube"
    // toggle) and drawBoard() re-reads every one of them regardless of which
    // one changed, so an $effect per store would buy nothing over a single
    // grouped subscription — it would just multiply the places carrying this
    // exception by four. Grouping them here keeps it to one documented spot.
    // positionStore's callback additionally carries a business rule that must
    // run before scheduleRedraw() fires: reset the selected move only on a
    // *real* navigation (position id change), not on every store tick (board
    // edits, analysis refresh). That rule stays inside the subscription
    // rather than becoming its own $effect — Board.svelte has no rendering
    // test, and the ordering between "reset selectedMoveStore" and "read
    // selectedMoveStore at draw time" is exactly the kind of thing a render
    // regression would hide; the synchronous .subscribe() ordering here is
    // unambiguous, an $effect's ordering relative to this one would not be
    // obviously so.
    function subscribeBoardRedrawTriggers() {
        let previousPositionId = null;
        const unsubPosition = positionStore.subscribe(() => {
            const position = get(positionStore);
            // Only clear selected move when position ID actually changes (real navigation)
            // Don't clear it on board redraws or analysis updates
            if (position.id !== previousPositionId) {
                selectedMoveStore.set(null);
                previousPositionId = position.id;
            }
            scheduleRedraw();
        });

        // Redraw when a move is selected/hovered in the analysis panel so its
        // arrows appear (or clear) on the board.
        const unsubSelectedMove = selectedMoveStore.subscribe(() => scheduleRedraw());

        // Redraw when analysis loads/changes so the offered cube (take/pass
        // decisions) appears for the displayed position outside match mode.
        const unsubAnalysis = analysisStore.subscribe(() => scheduleRedraw());

        // Redraw when the take/pass "offered cube" mode toggles so the cube moves
        // between its centered (offered) and owner positions immediately.
        const unsubOfferedCube = searchOfferedCubeStore.subscribe(() => scheduleRedraw());

        return () => {
            unsubPosition();
            unsubSelectedMove();
            unsubAnalysis();
            unsubOfferedCube();
        };
    }

    // Apply the user-customisable palette to boardCfg and redraw. boardCfg is a
    // plain object read imperatively by the draw functions, so we mutate it in
    // place and trigger a redraw whenever the colours change (and once the
    // two.js canvas exists).
    $effect(() => {
        const colors = $boardColorsStore;
        boardCfg.fill = colors.background;
        boardCfg.stroke = colors.border;
        boardCfg.triangle.fill1 = colors.point1;
        boardCfg.triangle.fill2 = colors.point2;
        boardCfg.triangle.stroke = colors.border;
        boardCfg.checker.colors = [colors.checker1, colors.checker2];
        boardCfg.dice.fill = colors.dice;
        boardCfg.dice.dot = colors.diceDot;
        boardCfg.cube.fill = colors.cube;
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
    function handleMouseDown(event) {
        event.preventDefault(); // Prevent text or element selection
        // Blur any focused text field when clicking the board
        if (document.activeElement && document.activeElement.matches('input, textarea, [contenteditable]')) {
            /** @type {HTMLElement} */ (document.activeElement).blur();
        }
        if (mode !== 'EDIT' && mode !== 'EPC') return;

        // Normalise into drawing coordinates: the canvas may be CSS-scaled
        // (interface zoom, side layout), so raw client pixels drift — at 90 %
        // scale a click on point 1 used to land on point 2.
        const rect = canvas.getBoundingClientRect();
        const { x: mouseX, y: mouseY } = boardMouseToDrawing(event.clientX, event.clientY, rect, width, height);

        startMousePos = {
            x: mouseX,
            y: mouseY,
            button: event.button
        };
    }

    function handleMouseMove(event) {
        event.preventDefault(); // Prevent text or element selection
        if (mode !== 'EDIT' && mode !== 'EPC') return;
    }

    function handleMouseUp(event) {
        event.preventDefault(); // Prevent text or element selection
        if (mode !== 'EDIT' && mode !== 'EPC') return;

        const rect = canvas.getBoundingClientRect();
        const norm = boardMouseToDrawing(event.clientX, event.clientY, rect, width, height);
        const endMousePos = {
            x: norm.x,
            y: norm.y,
            button: event.button
        };

        // In the Except structure, a quick second click on the same point blocks it
        // (must be empty). Detected manually because native 'dblclick' is unreliable
        // when redraws recreate the shapes between the two clicks.
        if (mode === 'EDIT' && get(searchStructureModeStore) === 'exclude') {
            const { checkerPoint } = getCheckerPointAndCount(endMousePos.x, endMousePos.y, 0);
            if (checkerPoint >= 1 && checkerPoint <= 24) {
                const isMarker = get(positionStore).board.points[checkerPoint]?.color === EXCLUDE_EMPTY;
                const now = Date.now();
                if (!isMarker && lastExceptClick && lastExceptClick.point === checkerPoint && now - lastExceptClick.time < 450) {
                    lastExceptClick = null;
                    setEmptyExcludeMarker(checkerPoint);
                    return;
                }
                // A click on a blocked point unblocks it (see updateCheckerPositionByPoint);
                // don't let it seed a double-click that would immediately re-block.
                lastExceptClick = isMarker ? null : { point: checkerPoint, time: now };
            } else {
                lastExceptClick = null;
            }
        }

        fillCheckersBetween(startMousePos, endMousePos);
    }

    function fillCheckersBetween(startPos, endPos) {
        const startChecker = getCheckerPointAndCount(startPos.x, startPos.y, startPos.button);
        const endChecker = getCheckerPointAndCount(endPos.x, endPos.y, startPos.button);

        const maxCheckers = Math.max(startChecker.checkerCount, endChecker.checkerCount);

        const startPoint = Math.min(startChecker.checkerPoint, endChecker.checkerPoint);
        const endPoint = Math.max(startChecker.checkerPoint, endChecker.checkerPoint);

        for (let point = startPoint; point <= endPoint; point++) {
            updateCheckerPositionByPoint(point, maxCheckers, startPos.button);
        }
    }

    function getCheckerPointAndCount(x_mouse, y_mouse, _button) {
        // Pure mapping shared with the unit tests (boardGeometry.js), which
        // assert it against the drawCheckers() formulas.
        return checkerPointAndCountAt(x_mouse, y_mouse, width, height, boardCfg.widthFactor, boardCfg.orientation);
    }

    function updateCheckerPositionByPoint(checkerPoint, checkerCount, button) {
        // The Eval panel (EPC mode) merges race analysis and gammonNet position
        // evaluation (ADR-0012): it is not race-only, so the whole board is
        // editable, exactly like EDIT mode — colour follows the mouse button
        // (left → Black, right → White) on every point, not just the home
        // boards. A position outside the race domain simply gets no race
        // verdict; the Go side already handles that gracefully.

        // A single click on a blocked ("must be empty") Except point unblocks it
        // (back to empty) so checkers can be edited again.
        if (get(positionStore).board.points[checkerPoint]?.color === EXCLUDE_EMPTY) {
            positionStore.update((pos) => {
                pos.board.points = pos.board.points.map((p, i) => (i === checkerPoint ? { checkers: 0, color: -1 } : p));
                return pos;
            });
            return;
        }
        const color = checkerPoint === 0 || checkerPoint === 25 ? (checkerPoint === 0 ? 1 : 0) : button === 2 ? 1 : 0;

        // When editing a search structure (Search tab), a point may carry up to 15
        // checkers and the per-colour total is NOT capped at 15 — a pattern can ask
        // for e.g. 3 checkers on each of 1-6 ("closed board without a spare"). For a
        // real position being edited, keep the 15-per-colour cap.
        const isSearchStructure = get(activeTabStore) === 'search';

        positionStore.update((pos) => {
            // Cap total checkers of this color at 15 (only for real positions).
            const totalOtherPoints = pos.board.points.reduce((acc, point, idx) => {
                if (idx !== checkerPoint && point.color === color) return acc + point.checkers;
                return acc;
            }, 0);
            const effectiveMaxPerPoint = isSearchStructure ? 15 : 15 - totalOtherPoints;
            if (effectiveMaxPerPoint <= 0) return pos;

            pos.board.points = pos.board.points.map((point, index) => {
                if (index === checkerPoint) {
                    if (point.checkers >= 5 && point.color === color) {
                        // Only add more checkers if clicked on the 5th checker
                        if (checkerCount === 5) {
                            return {
                                ...point,
                                checkers: Math.min(point.checkers + 1, effectiveMaxPerPoint)
                            };
                        } else {
                            return {
                                ...point,
                                checkers: Math.min(checkerCount, effectiveMaxPerPoint),
                                color: color
                            };
                        }
                    } else {
                        return {
                            ...point,
                            checkers: Math.min(checkerCount, effectiveMaxPerPoint),
                            color: color
                        };
                    }
                }
                return point;
            });

            // Set color to -1 if no checkers on a point
            pos.board.points = pos.board.points.map((point) => {
                if (point.checkers === 0) {
                    return {
                        ...point,
                        color: -1
                    };
                }
                return point;
            });

            return pos;
        });

        const position = get(positionStore);
        const player1Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 0 ? point.checkers : 0), 0);
        const player2Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 1 ? point.checkers : 0), 0);
        // A search structure can exceed 15 checkers per colour; clamp bearoff at 0
        // (it is irrelevant to structure search anyway).
        position.board.bearoff[0] = Math.max(0, 15 - player1Checkers);
        position.board.bearoff[1] = Math.max(0, 15 - player2Checkers);

        positionStore.update((pos) => {
            pos.board.bearoff = [position.board.bearoff[0], position.board.bearoff[1]];
            return pos;
        });
    }

    // setEmptyExcludeMarker flags a point of the "Except" structure as must-be-empty
    // (sentinel colour EXCLUDE_EMPTY). Used by the double-click handler.
    function setEmptyExcludeMarker(checkerPoint) {
        positionStore.update((pos) => {
            pos.board.points = pos.board.points.map((point, index) => (index === checkerPoint ? { checkers: 1, color: EXCLUDE_EMPTY } : point));
            return pos;
        });
    }

    function resizeBoard() {
        const container = canvas.parentElement;
        const containerWidth = container.clientWidth;
        const containerHeight = container.clientHeight;
        // Fit board within both width and height, maintaining aspect ratio
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
        // The measurement above must stay synchronous (it reads the live
        // container box), but the actual repaint is coalesced: a burst of
        // 'resize' events (window drag, panel toggle) must repaint at most
        // once per animation frame, not once per event.
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

    // EPC's own "clear" target: a blank board the user can build up from
    // scratch, but with the defaults enterEPCMode() itself uses — money
    // score (not a 7-away match), no dice in progress — rather than EDIT's
    // search-flavoured 7-7/3-1. See the grilling session that settled this:
    // reusing resetBoard() verbatim would inject an arbitrary match score
    // into a panel whose race table otherwise reads as money by default.
    function resetEPCBoard() {
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

    function handleDoubleClick(event) {
        if (mode !== 'EDIT' && mode !== 'EPC') return;

        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;

        // Note: the Except "must be empty" marker is handled by manual double-click
        // detection in handleMouseUp (native 'dblclick' is unreliable here because
        // redraws recreate the shapes between the two clicks).

        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardWidth = boardCfg.widthFactor * width;
        const boardHeight = (11 / 13) * boardWidth;

        // Check if the click is outside of the board
        if (mouseX < boardOrigXpos - boardWidth / 2 || mouseX > boardOrigXpos + boardWidth / 2 || mouseY < boardOrigYpos - boardHeight / 2 || mouseY > boardOrigYpos + boardHeight / 2) {
            if (mode === 'EPC') resetEPCBoard();
            else resetBoard();
        }
    }

    function handleDoublingCubeClick(event) {
        if (mode !== 'EDIT' && mode !== 'EPC') return;

        // Same normalisation as checker clicks: the canvas may be CSS-scaled.
        const rect = canvas.getBoundingClientRect();
        const { x: mouseX, y: mouseY } = boardMouseToDrawing(event.clientX, event.clientY, rect, width, height);

        // Check if the click is within the doubling cube
        if (
            mouseX >= cubePosition.x - cubePosition.size / 2 &&
            mouseX <= cubePosition.x + cubePosition.size / 2 &&
            mouseY >= cubePosition.y - cubePosition.size / 2 &&
            mouseY <= cubePosition.y + cubePosition.size / 2
        ) {
            positionStore.update((pos) => {
                // EPC mode: only the owner matters (money equities are in units
                // of the current cube), so clicks cycle the owner and pin the
                // value — centered (face 1) → bottom owns (face 2) → top owns
                // (face 2) → centered; right-click cycles backwards.
                if (mode === 'EPC') {
                    const cycle = [-1, 0, 1];
                    const dir = event.button === 2 ? -1 : 1;
                    const cur = cycle.indexOf(pos.cube.owner === undefined ? -1 : pos.cube.owner);
                    const next = cycle[(cur + dir + 3) % 3];
                    pos.cube.owner = next;
                    pos.cube.value = next === -1 ? 0 : 1;
                    return pos;
                }
                // Take/pass search: the cube is an offered (centered) cube. Edit its
                // value while keeping it centered (owner -1); an offered cube is at
                // least a double, so value stays ≥ 1 (displayed ≥ 2).
                if (get(searchOfferedCubeStore) && pos.decision_type === 1) {
                    if (event.button === 0) {
                        pos.cube.value = Math.min(pos.cube.value + 1, 6);
                    } else if (event.button === 2) {
                        pos.cube.value = Math.max(pos.cube.value - 1, 1);
                    }
                    pos.cube.owner = -1;
                    return pos;
                }
                if (pos.cube.owner === -1) {
                    pos.cube.value = Math.min(pos.cube.value + 1, 6);
                    pos.cube.owner = event.button === 0 ? 0 : 1;
                } else if (pos.cube.owner === 0) {
                    if (event.button === 0) {
                        pos.cube.value = Math.min(pos.cube.value + 1, 6);
                    } else if (event.button === 2) {
                        pos.cube.value = Math.max(pos.cube.value - 1, 0);
                    }
                } else if (pos.cube.owner === 1) {
                    if (event.button === 0) {
                        pos.cube.value = Math.max(pos.cube.value - 1, 0);
                    } else if (event.button === 2) {
                        pos.cube.value = Math.min(pos.cube.value + 1, 6);
                    }
                }

                if (pos.cube.value === 0) {
                    pos.cube.owner = -1;
                }

                return pos;
            });
        }
    }

    function handleRectangleAndDiceClick(event) {
        if (mode !== 'EDIT' && mode !== 'EPC') return;

        // Same normalisation as checker clicks: the canvas may be CSS-scaled.
        const rect = canvas.getBoundingClientRect();
        const { x: mouseX, y: mouseY } = boardMouseToDrawing(event.clientX, event.clientY, rect, width, height);

        logger.log('Rectangle or Dice click detected at:', mouseX, mouseY); // Debug log

        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardWidth = boardCfg.widthFactor * width;
        const boardCheckerSize = ((11 / 13) * (boardCfg.widthFactor * width)) / 11;
        const boardHeight = (11 / 13) * boardWidth;
        const gap = 1.2 * boardCheckerSize;

        const bearoff1Xpos = boardOrigXpos + boardWidth / 2 + gap;
        const bearoff1Ypos = boardOrigYpos + boardHeight / 2 - 3.7 * boardCheckerSize;
        const score1Ypos = boardOrigYpos + boardHeight / 2 + 0.2 * boardCheckerSize;

        const bearoff2Xpos = boardOrigXpos + boardWidth / 2 + gap;
        const bearoff2Ypos = boardOrigYpos - boardHeight / 2 + 3.7 * boardCheckerSize;
        const score2Ypos = boardOrigYpos - boardHeight / 2 - 0.2 * boardCheckerSize;

        // Exclude score areas from player rectangle detection to avoid overlap with handleScoreClick
        const scoreHeight = 0.5 * boardCheckerSize;
        const isInsideScore1 =
            mouseX >= bearoff1Xpos - 0.75 * boardCheckerSize && mouseX <= bearoff1Xpos + 0.75 * boardCheckerSize && mouseY >= score1Ypos - scoreHeight / 2 && mouseY <= score1Ypos + scoreHeight / 2;
        const isInsideScore2 =
            mouseX >= bearoff2Xpos - 0.75 * boardCheckerSize && mouseX <= bearoff2Xpos + 0.75 * boardCheckerSize && mouseY >= score2Ypos - scoreHeight / 2 && mouseY <= score2Ypos + scoreHeight / 2;

        const isInsideTopPlayerRectangle =
            !isInsideScore1 &&
            mouseX >= bearoff1Xpos - 0.75 * boardCheckerSize &&
            mouseX <= bearoff1Xpos + 0.75 * boardCheckerSize &&
            mouseY >= Math.min(bearoff1Ypos, score1Ypos) &&
            mouseY <= Math.max(bearoff1Ypos, score1Ypos);

        const isInsideBottomPlayerRectangle =
            !isInsideScore2 &&
            mouseX >= bearoff2Xpos - 0.75 * boardCheckerSize &&
            mouseX <= bearoff2Xpos + 0.75 * boardCheckerSize &&
            mouseY >= Math.min(bearoff2Ypos, score2Ypos) &&
            mouseY <= Math.max(bearoff2Ypos, score2Ypos);

        const diceGap = 0.325 * boardCheckerSize;
        const diceSize = 0.7 * boardCheckerSize;
        const diceXpos = boardOrigXpos + boardWidth / 2 + 2 * diceGap;
        const diceYpos = get(positionStore).player_on_roll === 0 ? boardOrigYpos + 0.5 * boardHeight - 1.5 * boardCheckerSize : boardOrigYpos - 0.5 * boardHeight + 1.5 * boardCheckerSize;

        let isInsideDie1 = false;
        let isInsideDie2 = false;

        for (let index = 0; index < 2; index++) {
            const dieXpos = diceXpos + index * (diceSize + diceGap);
            if (mouseX >= dieXpos - diceSize / 2 && mouseX <= dieXpos + diceSize / 2 && mouseY >= diceYpos - diceSize / 2 && mouseY <= diceYpos + diceSize / 2) {
                if (index === 0) {
                    isInsideDie1 = true;
                } else {
                    isInsideDie2 = true;
                }
            }
        }

        if (isInsideDie1 || isInsideDie2) {
            logger.log('Die clicked'); // Debug log
        }

        // EPC mode shares this whole handler with EDIT mode: the Eval panel's
        // evaluation volet needs real dice to show candidate moves (no dice
        // means a cube verdict instead, EPCPanel.svelte) — a player's
        // rectangle clearing the dice to show the cube decision, then a die
        // click restoring/bumping them for a move decision, is exactly EDIT
        // mode's own toggle.
        positionStore.update((pos) => {
            if (isInsideTopPlayerRectangle && !isInsideDie1 && !isInsideDie2) {
                logger.log("Top player's rectangle clicked"); // Debug log
                pos.player_on_roll = 0;
                pos.decision_type = 1; // Set decision type to doubling cube
                previousDice = pos.dice; // Save previous dice values
                pos.dice = [0, 0];
                logger.log('Updated decision_type to 1 for top player'); // Debug log
            } else if (isInsideBottomPlayerRectangle && !isInsideDie1 && !isInsideDie2) {
                logger.log("Bottom player's rectangle clicked"); // Debug log
                pos.player_on_roll = 1;
                pos.decision_type = 1; // Set decision type to doubling cube
                pos.dice = [0, 0];
                logger.log('Updated decision_type to 1 for bottom player'); // Debug log
            } else if (isInsideDie1) {
                logger.log('Die 1 clicked'); // Debug log
                pos.decision_type = 0;
                pos.dice = previousDice; // Restore previous dice values
                if (event.button === 0) {
                    pos.dice[0] = (pos.dice[0] % 6) + 1; // Left click to increase
                } else if (event.button === 2) {
                    pos.dice[0] = pos.dice[0] === 1 ? 6 : pos.dice[0] - 1; // Right click to decrease
                }
            } else if (isInsideDie2) {
                logger.log('Die 2 clicked'); // Debug log
                pos.decision_type = 0;
                pos.dice = previousDice; // Restore previous dice values
                if (event.button === 0) {
                    pos.dice[1] = (pos.dice[1] % 6) + 1; // Left click to increase
                } else if (event.button === 2) {
                    pos.dice[1] = pos.dice[1] === 1 ? 6 : pos.dice[1] - 1; // Right click to decrease
                }
            }

            logger.log('Updated dice values:', pos.dice); // Debug log
            logger.log('Updated position store:', pos); // Log the updated position store
            return pos;
        });
    }

    function handleScoreClick(event) {
        if (mode !== 'EDIT' && mode !== 'EPC') return;

        // Same normalisation as checker clicks: the canvas may be CSS-scaled.
        const rect = canvas.getBoundingClientRect();
        const { x: mouseX, y: mouseY } = boardMouseToDrawing(event.clientX, event.clientY, rect, width, height);

        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardWidth = boardCfg.widthFactor * width;
        const boardCheckerSize = ((11 / 13) * (boardCfg.widthFactor * width)) / 11;
        const boardHeight = (11 / 13) * boardWidth; // Define boardHeight

        const score1Xpos = boardOrigXpos + boardWidth / 2 + 1.2 * boardCheckerSize;
        const score1Ypos = boardOrigYpos + boardHeight / 2 + 0.2 * boardCheckerSize;
        const score2Xpos = boardOrigXpos + boardWidth / 2 + 1.2 * boardCheckerSize;
        const score2Ypos = boardOrigYpos - boardHeight / 2 - 0.2 * boardCheckerSize;

        const scoreWidth = 1.5 * boardCheckerSize;
        const scoreHeight = 0.5 * boardCheckerSize;

        // Check if the click is inside the top player's green rectangle
        if (mouseX >= score1Xpos - scoreWidth / 2 && mouseX <= score1Xpos + scoreWidth / 2 && mouseY >= score1Ypos - scoreHeight / 2 && mouseY <= score1Ypos + scoreHeight / 2) {
            positionStore.update((pos) => {
                if (event.button === 0) {
                    pos.score[0] = Math.max(pos.score[0] - 1, -1); // Decrement score
                } else if (event.button === 2) {
                    pos.score[0] = Math.min(pos.score[0] + 1, 99); // Increment score, max 99
                }
                if (pos.score[0] === -1) {
                    pos.score[1] = -1; // Set other player's score to unlimited
                } else if (pos.score[1] === -1) {
                    // Leaving money by editing this side alone must not strand the
                    // position at [N, -1] — an away score with no opponent away
                    // score is not a valid match state (EPC's own money default is
                    // [-1, -1], the only state where a lone -1 is meaningful).
                    pos.score[1] = pos.score[0];
                }
                return pos;
            });
        }

        // Check if the click is inside the bottom player's green rectangle
        if (mouseX >= score2Xpos - scoreWidth / 2 && mouseX <= score2Xpos + scoreWidth / 2 && mouseY >= score2Ypos - scoreHeight / 2 && mouseY <= score2Ypos + scoreHeight / 2) {
            positionStore.update((pos) => {
                if (event.button === 0) {
                    pos.score[1] = Math.max(pos.score[1] - 1, -1); // Decrement score
                } else if (event.button === 2) {
                    pos.score[1] = Math.min(pos.score[1] + 1, 99); // Increment score, max 99
                }
                if (pos.score[1] === -1) {
                    pos.score[0] = -1; // Set other player's score to unlimited
                } else if (pos.score[0] === -1) {
                    pos.score[0] = pos.score[1]; // Leaving money: see the mirrored comment above
                }
                return pos;
            });
        }
    }

    function logCanvasSize() {
        const actualWidth = canvas.clientWidth;
        const actualHeight = canvas.clientHeight;
        logger.log('Actual canvas width: ', actualWidth, 'Actual canvas height: ', actualHeight);
        logger.log('Two.js width: ', two.width, 'Two.js height: ', two.height);
    }

    function setBoardOrientation(orientation) {
        boardCfg.orientation = orientation;
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
        if ((mode !== 'EDIT' && mode !== 'EPC') || showTakePoint2Modal || showTakePoint4Modal) return; // Disable shortcuts when TakePoint2Modal or TakePoint4Modal is open

        if (event.key === 'Backspace' && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
            event.preventDefault();
            if (mode === 'EPC') resetEPCBoard();
            else resetBoard();
        }
    }

    onMount(() => {
        canvas = document.getElementById('backgammon-board');
        const params = { width: window.innerWidth, height: window.innerHeight };
        two = new Two(params).appendTo(canvas);

        // Set the width and height based on the actual container dimensions
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

        canvas.addEventListener('mousedown', handleMouseDown);
        canvas.addEventListener('mousemove', handleMouseMove);
        canvas.addEventListener('mouseup', handleMouseUp);
        canvas.addEventListener('dblclick', handleDoubleClick);
        canvas.addEventListener('mousedown', handleDoublingCubeClick);
        canvas.addEventListener('mousedown', handleRectangleAndDiceClick);
        canvas.addEventListener('mousedown', handleScoreClick);
        canvas.addEventListener('contextmenu', handleContextMenu);
        // No direct drawBoard() here: subscribeBoardRedrawTriggers() below
        // fires each subscription once on the spot (svelte/store calls the
        // callback synchronously), which schedules the first paint on the
        // next animation frame — before the browser paints the mounted DOM.
        // A synchronous draw on top of it built the whole scene twice.
        window.addEventListener('resize', resizeBoard);
        window.addEventListener('keydown', handleOrientationChange);
        window.addEventListener('keydown', handleKeyDown);

        unsubscribeBoardRedrawTriggers = subscribeBoardRedrawTriggers();

        logCanvasSize();
        window.addEventListener('resize', logCanvasSize);
    });

    onDestroy(() => {
        canvas.removeEventListener('mousedown', handleMouseDown);
        canvas.removeEventListener('mousemove', handleMouseMove);
        canvas.removeEventListener('mouseup', handleMouseUp);
        canvas.removeEventListener('dblclick', handleDoubleClick);
        canvas.removeEventListener('mousedown', handleDoublingCubeClick);
        canvas.removeEventListener('mousedown', handleRectangleAndDiceClick);
        canvas.removeEventListener('mousedown', handleScoreClick);
        canvas.removeEventListener('contextmenu', handleContextMenu);
        window.removeEventListener('resize', resizeBoard);
        window.removeEventListener('resize', logCanvasSize);
        window.removeEventListener('keydown', handleOrientationChange);
        window.removeEventListener('keydown', handleKeyDown);
        if (unsubscribeBoardRedrawTriggers) unsubscribeBoardRedrawTriggers();
        // A redraw can be pending (rAF already requested) at the moment the
        // component is torn down; cancel it so drawBoard() never runs against
        // a detached two.js instance.
        if (redrawFrameId !== null) cancelAnimationFrame(redrawFrameId);
    });

    // ── Board context menu ─────────────────────────────────────────────────
    // Right-clicking the board opens actions on the position it shows, but
    // ONLY in the modes where the right button is otherwise idle. In EDIT and
    // EPC the right button already means "place the other colour's checker"
    // (see handleMouseDown), so the menu stays out of their way.
    let boardMenu = $state(null);

    function handleContextMenu(event) {
        event.preventDefault(); // no native menu, in every mode
        if (mode === 'EDIT' || mode === 'EPC') return;
        if (get(isAnyModalOpen)) return;
        boardMenu = {
            x: event.clientX,
            y: event.clientY,
            items: [
                {
                    label: $t('board.menu.evaluate'),
                    // The position AS DISPLAYED, not the stored record: in a
                    // match with player 2 on roll the board is mirrored, and
                    // the Eval panel must open on the board the user is
                    // actually looking at.
                    onClick: () => sendPositionToEval(getDisplayPosition())
                }
            ]
        };
    }

    // Helper function to get the position to display
    //
    // STORAGE: All positions are stored normalized with player_on_roll = 0
    //          (player on roll is always the bottom player in stored positions)
    //
    // NORMAL MODE: Player on roll should always be at the bottom.
    //   - Stored positions already have player_on_roll = 0, so they display correctly.
    //   - If editing and player_on_roll = 1, mirror for display so player on roll is at bottom.
    //
    // MATCH MODE: Player 1 is always at the bottom, Player 2 at top.
    //   - Positions are stored normalized (player_on_roll = 0)
    //   - MatchMovePosition.player_on_roll tells us who was actually on roll in the match
    //   - If Player 2 was on roll (player_on_roll = 1), we need to mirror the stored position
    //     so that Player 1 appears at bottom and Player 2 (who was actually on roll) appears at top
    function getDisplayPosition() {
        const position = get(positionStore);
        const matchCtx = get(matchContextStore);

        // In EPC mode, always use position as-is (player_on_roll is always 0)
        if (mode === 'EPC') {
            return position;
        }

        // In EDIT mode, show position as-is so editing coordinates match the display.
        // The mirroring (if player2 is on roll) is handled at search time instead.
        if (mode === 'EDIT') {
            return position;
        }

        // In match mode, check who was actually on roll
        if (matchCtx && matchCtx.isMatchMode && matchCtx.movePositions.length > 0) {
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            if (currentMovePos && currentMovePos.player_on_roll === 1) {
                // Player 2 was on roll - mirror the position so Player 1 stays at bottom
                // but the dice show on Player 2's side (top)
                return mirrorPosition(position);
            }
            return position;
        }

        // In normal mode, if player_on_roll is 1, mirror so player on roll is at bottom
        if (position.player_on_roll === 1) {
            return mirrorPosition(position);
        }

        return position;
    }

    // Whether the board is shown from player 2's side: in match mode when
    // player 2 is on roll for the current move (the stored position is then
    // mirrored for display), otherwise when the displayed position itself has
    // player 2 on roll (an edited position before it is saved).
    function isPlayer2Perspective(displayPosition) {
        const matchCtx = get(matchContextStore);
        if (matchCtx && matchCtx.isMatchMode && matchCtx.movePositions.length > 0) {
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            return !!currentMovePos && currentMovePos.player_on_roll === 1;
        }
        return displayPosition.player_on_roll === 1;
    }

    // A take/pass (response) decision: the cube has been offered to the
    // player on roll and is drawn in the middle of the board rather than at
    // its owner spot. The signal is the played cube action — in match mode
    // the current move's action, otherwise the analysis' recorded played
    // actions. While editing, the offered cube is shown only when the user is
    // explicitly building a take/pass search — never from stale analysis of
    // a previously viewed position.
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

    // The selected move's checkers, in the display position's point numbers.
    // Move notation uses the stored (normalised) numbering; in match mode with
    // player 2 on roll the display is mirrored (point i → 25 - i), so the
    // arrows must be too.
    function selectedMoveArrows() {
        const moves = parseMoveNotation(selectedMove);
        if (moves.length === 0) return moves;
        const matchCtx = get(matchContextStore);
        if (matchCtx && matchCtx.isMatchMode && matchCtx.movePositions.length > 0) {
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            if (currentMovePos && currentMovePos.player_on_roll === 1) {
                return moves.map((m) => ({ ...m, from: m.from === -1 ? -1 : 25 - m.from, to: m.to === -1 ? -1 : 25 - m.to }));
            }
        }
        return moves;
    }

    export function drawBoard() {
        if (!two) return; // Safety check

        two.clear();
        const geom = boardMetrics(width, height, boardCfg.widthFactor);
        const position = getDisplayPosition();
        logger.log('drawBoard', width, height, 'decision_type:', position.decision_type);

        drawStaticScene(two, geom, boardCfg, isPlayer2Perspective(position));
        cubePosition = drawDynamicScene(two, geom, boardCfg, position, {
            offeredCube: isOfferedCube(position),
            showPipcount,
            moves: selectedMoveArrows()
        });
        // Outline on top so its linewidth is not eaten by the checkers.
        drawFrame(two, geom, boardCfg);

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
