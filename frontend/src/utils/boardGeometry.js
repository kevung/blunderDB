// Pure board geometry/parsing helpers, testable without two.js (drawing: boardScene.js; mouse:
// boardInteractions.js).

/**
 * Parse a move string (e.g. "24/23 13/11(2)", "bar/23", "6/off(4)") into a flat
 * list of `{ from, to, index }` moves. `bar` → 0, `off` → -1; the `(n)` suffix
 * expands to n moves. Empty/"bar"/"cannot move" strings yield [].
 */
export function parseMoveNotation(moveString) {
    if (!moveString || moveString === 'bar' || moveString.toLowerCase().includes('cannot move')) {
        return [];
    }

    const moves = [];
    // Split by spaces to get individual moves like "24/23" or "13/11(2)"
    const parts = moveString.trim().split(/\s+/);

    for (const part of parts) {
        // Match pattern like "24/23" or "24/23(2)" or "bar/23" or "6/off(4)"
        const match = part.match(/(\d+|bar)\/(\d+|off)(?:\((\d+)\))?/i);
        if (match) {
            const from = match[1].toLowerCase() === 'bar' ? 0 : parseInt(match[1]);
            const to = match[2].toLowerCase() === 'off' ? -1 : parseInt(match[2]);
            const count = match[3] ? parseInt(match[3]) : 1;

            for (let i = 0; i < count; i++) {
                moves.push({ from, to, index: i });
            }
        }
    }

    return moves;
}

/**
 * Mirror a position to bring the other player to the bottom: points i ↔ 25-i with colours
 * swapped, bearoff/scores swapped, on-roll and cube owner flipped. Returns a deep copy.
 */
export function mirrorPosition(pos) {
    const mirrored = JSON.parse(JSON.stringify(pos)); // Deep copy

    const tempPoints = [...mirrored.board.points];
    for (let i = 0; i < 26; i++) {
        mirrored.board.points[25 - i] = {
            color: tempPoints[i].color === -1 ? -1 : 1 - tempPoints[i].color,
            checkers: tempPoints[i].checkers
        };
    }

    [mirrored.board.bearoff[0], mirrored.board.bearoff[1]] = [mirrored.board.bearoff[1], mirrored.board.bearoff[0]];

    mirrored.player_on_roll = 1 - mirrored.player_on_roll;

    [mirrored.score[0], mirrored.score[1]] = [mirrored.score[1], mirrored.score[0]];

    if (mirrored.cube.owner !== -1) {
        mirrored.cube.owner = 1 - mirrored.cube.owner;
    }

    return mirrored;
}

/**
 * Compute both players' pip counts from a (display) position: color-0 checkers
 * count their point index, color-1 checkers count 25 - index.
 */
export function computePipCount(position) {
    let pipCount1 = 0;
    let pipCount2 = 0;

    position.board.points.forEach((point, index) => {
        if (point.color === 0) {
            pipCount1 += point.checkers * index;
        } else if (point.color === 1) {
            pipCount2 += point.checkers * (25 - index);
        }
    });

    return { pipCount1, pipCount2 };
}

/**
 * The board's fixed measurements for a `width` × `height` surface: every drawing function and hit
 * test derives from this one object, so they cannot disagree. 13 checkers wide (6 + bar + 6),
 * 11 tall, centred; a triangle is 5 checkers tall.
 */
export function boardMetrics(width, height, widthFactor) {
    const boardWidth = widthFactor * width;
    const boardHeight = (11 / 13) * boardWidth;
    const checkerSize = boardHeight / 11;
    return {
        width,
        height,
        boardWidth,
        boardHeight,
        checkerSize,
        triangleHeight: 5 * checkerSize,
        originX: width / 2,
        originY: height / 2
    };
}

/**
 * Convert a mouse event position into drawing coordinates. The canvas may be CSS-scaled
 * (interface zoom, side layout), so client pixels are normalised by drawing size / rendered box;
 * otherwise clicks drift with the scale.
 */
export function boardMouseToDrawing(clientX, clientY, rect, width, height) {
    const scaleX = rect.width > 0 ? width / rect.width : 1;
    const scaleY = rect.height > 0 ? height / rect.height : 1;
    return {
        x: (clientX - rect.left) * scaleX,
        y: (clientY - rect.top) * scaleY
    };
}

/**
 * Map drawing coordinates to the clicked point and checker slot, matching drawCheckers.
 * Returns { checkerPoint, checkerCount }; checkerPoint is -1 outside the board, 0/25 the bars.
 */
export function checkerPointAndCountAt(x_mouse, y_mouse, width, height, widthFactor, orientation) {
    const boardAspectFactor = 11 / 13;
    const boardWidth = widthFactor * width;
    const boardHeight = boardAspectFactor * boardWidth;
    const boardCheckerSize = boardHeight / 11;
    const boardOrigXpos = width / 2;
    const boardOrigYpos = height / 2;

    const x = Math.round((x_mouse - boardOrigXpos) / boardCheckerSize);
    const y = Math.round((y_mouse - boardOrigYpos) / boardCheckerSize);

    let checkerCount = 0;
    if (Math.abs(x) <= 6 && Math.abs(y) > 0 && Math.abs(y) <= 6) {
        if (Math.abs(y) == 0 || Math.abs(y) == 6) {
            checkerCount = 0;
        } else if (Math.abs(y) <= 5) {
            if (x != 0) {
                checkerCount = 6 - Math.abs(y);
            } else {
                checkerCount = Math.abs(y);
            }
        }

        let checkerPoint = 0;
        if (orientation == 'right') {
            if (y < 0) {
                if (x > 0) {
                    checkerPoint = 18 + x;
                } else if (x < 0) {
                    checkerPoint = 19 + x;
                } else {
                    checkerPoint = 25;
                }
            } else if (y > 0) {
                if (x > 0) {
                    checkerPoint = 7 - x;
                } else if (x < 0) {
                    checkerPoint = 6 - x;
                } else {
                    checkerPoint = 0;
                }
            }
        } else {
            if (y < 0) {
                if (x > 0) {
                    checkerPoint = 19 - x;
                } else if (x < 0) {
                    checkerPoint = 18 - x;
                } else {
                    checkerPoint = 25;
                }
            } else if (y > 0) {
                if (x > 0) {
                    checkerPoint = 6 + x;
                } else if (x < 0) {
                    checkerPoint = 7 + x;
                } else {
                    checkerPoint = 0;
                }
            }
        }

        return { checkerPoint, checkerCount };
    }
    return { checkerPoint: -1, checkerCount: 0 };
}
