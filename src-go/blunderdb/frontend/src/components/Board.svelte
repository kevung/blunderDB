<script>
    import { positionStore } from "../stores/positionStore";
    import { onMount, onDestroy } from "svelte";
    import Two from "two.js";
    import { get } from 'svelte/store';

    export let mode;
    
    let canvasCfg = {
        aspectFactor: 0.63,
    };

    let boardCfg = {
        widthFactor: 0.63,
        orientation: "right",
        fill: "white",
        stroke: "black",
        linewidth: 3,
        triangle: {
            fill1: "white",
            fill2: "rgb(208, 208, 208)",
            stroke: "black",
            linewidth: 1.3, // Changed linewidth to 1
        },
        label: {
            size: 25,
            distanceToBoard: 0.4,
        },
        checker: {
            sizeFactor: 0.97,
            colors: ["black", "white"],
            linewidth: 2.5 // Added linewidth property and set to 2
        }
    };

    let two;
    let canvas;
    let width;
    let height;
    let unsubscribe;
    let isMouseDown = false;
    let startMousePos = null;

    function handleMouseDown(event) {
        if (mode !== "EDIT") return;

        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;

        // Check if the click is in the middle of the bar
        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
        if (Math.abs(mouseX - boardOrigXpos) < boardCheckerSize / 2 && Math.abs(mouseY - boardOrigYpos) < boardCheckerSize / 2) {
            positionStore.update(pos => {
                pos.board.points[0].checkers = 0;
                pos.board.points[25].checkers = 0;
                return pos;
            });
            return;
        }

        isMouseDown = true;
        startMousePos = {
            x: mouseX,
            y: mouseY,
            button: event.button
        };
    }

    function handleMouseMove(event) {
        if (mode !== "EDIT" || !isMouseDown) return;

        const rect = canvas.getBoundingClientRect();
        const currentMousePos = {
            x: event.clientX - rect.left,
            y: event.clientY - rect.top
        };

        fillCheckersBetween(startMousePos, currentMousePos);
    }

    function handleMouseUp(event) {
        if (mode !== "EDIT" || !isMouseDown) return;

        isMouseDown = false;
        const rect = canvas.getBoundingClientRect();
        const endMousePos = {
            x: event.clientX - rect.left,
            y: event.clientY - rect.top,
            button: event.button
        };

        fillCheckersBetween(startMousePos, endMousePos);
    }

    function fillCheckersBetween(startPos, endPos) {
        const startChecker = getCheckerPointAndCount(startPos.x, startPos.y, startPos.button);
        const endChecker = getCheckerPointAndCount(endPos.x, endPos.y, endPos.button);

        const maxCheckers = Math.max(startChecker.checkerCount, endChecker.checkerCount);

        const startPoint = Math.min(startChecker.checkerPoint, endChecker.checkerPoint);
        const endPoint = Math.max(startChecker.checkerPoint, endChecker.checkerPoint);

        for (let point = startPoint; point <= endPoint; point++) {
            updateCheckerPositionByPoint(point, maxCheckers, startPos.button);
        }
    }

    function getCheckerPointAndCount(x_mouse, y_mouse, button) {
        const boardAspectFactor = 11 / 13;
        const boardWidth = boardCfg.widthFactor * width;
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
            if (boardCfg.orientation == "right") {
                if (y < 0) {
                    if (x > 0) {
                        checkerPoint = 18 + x;
                    } else if (x < 0) {
                        checkerPoint = 19 + x;
                    } else {
                        checkerPoint = 25;
                    }
                } else {
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
                } else {
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
        return { checkerPoint: 0, checkerCount: 0 };
    }

    function updateCheckerPositionByPoint(checkerPoint, checkerCount, button) {
        const color = (checkerPoint === 0 || checkerPoint === 25) ? (checkerPoint === 0 ? 1 : 0) : (button === 2 ? 1 : 0);

        positionStore.update(pos => {
            pos.board.points = pos.board.points.map((point, index) => {
                if (index === checkerPoint) {
                    return {
                        ...point,
                        checkers: checkerCount,
                        color: color
                    };
                }
                return point;
            });
            return pos;
        });

        const position = get(positionStore);
        const player1Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 0 ? point.checkers : 0), 0);
        const player2Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 1 ? point.checkers : 0), 0);
        position.board.bearoff[0] = 15 - player1Checkers;
        position.board.bearoff[1] = 15 - player2Checkers;

        positionStore.update(pos => {
            pos.board.bearoff = [position.board.bearoff[0], position.board.bearoff[1]];
            return pos;
        });
    }

    function resizeBoard() {
        const actualWidth = canvas.clientWidth;
        const actualHeight = actualWidth * canvasCfg.aspectFactor;
        width = actualWidth;
        height = actualHeight;
        two.width = width;
        two.height = height;
        two.renderer.setSize(width, height);
        drawBoard();
        two.update();
    }

    function handleDoubleClick(event) {
        if (mode !== "EDIT") return;

        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;

        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardWidth = boardCfg.widthFactor * width;
        const boardHeight = (11 / 13) * boardWidth;

        // Check if the click is outside of the board
        if (mouseX < boardOrigXpos - boardWidth / 2 || mouseX > boardOrigXpos + boardWidth / 2 ||
            mouseY < boardOrigYpos - boardHeight / 2 || mouseY > boardOrigYpos + boardHeight / 2) {
            positionStore.update(pos => {
                pos.board.points.forEach(point => point.checkers = 0);
                return pos;
            });
        }
    }

    function logCanvasSize() {
        const actualWidth = canvas.clientWidth;
        const actualHeight = canvas.clientHeight;
        console.log("Actual canvas width: ", actualWidth, "Actual canvas height: ", actualHeight);
        console.log("Two.js width: ", two.width, "Two.js height: ", two.height);
    }

    onMount(() => {
        canvas = document.getElementById("backgammon-board");
        const params = { width: window.innerWidth, height: window.innerHeight };
        two = new Two(params).appendTo(canvas);

        // Set the width and height based on the actual canvas dimensions after appending
        const actualWidth = canvas.clientWidth;
        const actualHeight = actualWidth * canvasCfg.aspectFactor;
        width = actualWidth;
        height = actualHeight;
        two.width = width;
        two.height = actualHeight;
        two.renderer.setSize(width, height);

        canvas.addEventListener("mousedown", handleMouseDown);
        canvas.addEventListener("mousemove", handleMouseMove);
        canvas.addEventListener("mouseup", handleMouseUp);
        canvas.addEventListener("dblclick", handleDoubleClick);
        drawBoard();
        window.addEventListener("resize", resizeBoard);

        unsubscribe = positionStore.subscribe(() => {
            drawBoard();
            console.log("positionStore: ", get(positionStore));
        });

        logCanvasSize();
        window.addEventListener("resize", logCanvasSize);
    });

    onDestroy(() => {
        canvas.removeEventListener("mousedown", handleMouseDown);
        canvas.removeEventListener("mousemove", handleMouseMove);
        canvas.removeEventListener("mouseup", handleMouseUp);
        canvas.removeEventListener("dblclick", handleDoubleClick);
        window.removeEventListener("resize", resizeBoard);
        window.removeEventListener("resize", logCanvasSize);
        if (unsubscribe) unsubscribe();
    });

    function drawBoard() {
        two.clear();

        const boardAspectFactor = 11 / 13;
        const boardWidth = boardCfg.widthFactor * width;
        const boardHeight = boardAspectFactor * boardWidth;
        const boardCheckerSize = boardHeight / 11;
        const boardTriangleHeight = 5 * boardCheckerSize;
        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        console.log("width: ", width, "height: ", height);
        console.log("boardOrigXpos: ", boardOrigXpos, "boardOrigYpos: ", boardOrigYpos);
        console.log("two.width: ", two.width, "two.height: ", two.height);
        function createTriangle(x, y, flip) {
            if (flip == false) {
                const triangle = two.makePath(
                    x,
                    y,
                    x + boardCheckerSize,
                    y,
                    x + 0.5 * boardCheckerSize,
                    y + 5 * boardCheckerSize,
                );
                triangle.stroke = boardCfg.triangle.stroke;
                triangle.linewidth = boardCfg.triangle.linewidth;
                return triangle;
            } else {
                const triangle = two.makePath(
                    x,
                    y + boardTriangleHeight,
                    x + boardCheckerSize,
                    y + boardTriangleHeight,
                    x + 0.5 * boardCheckerSize,
                    y + boardTriangleHeight - 5 * boardCheckerSize,
                );

                triangle.stroke = boardCfg.triangle.stroke;
                triangle.linewidth = boardCfg.triangle.linewidth;
                return triangle;
            }
        }

        function createQuadrant(x, y, flip) {
            let quadrant = two.makeGroup();
            for (let i = 0; i < 6; i++) {
                const offsetX = x + i * boardCheckerSize;
                const offsetY = y;
                const t = createTriangle(offsetX, offsetY, flip);
                if (i % 2 == 1) {
                    t.fill = boardCfg.triangle.fill1;
                } else {
                    t.fill = boardCfg.triangle.fill2;
                }

                //invert color
                if (flip) {
                    if (i % 2 == 1) {
                        t.fill = boardCfg.triangle.fill2;
                    } else {
                        t.fill = boardCfg.triangle.fill1;
                    }
                }

                quadrant.add(t);
            }
            return quadrant;
        }

        function createLabels() {
            let labels = two.makeGroup();
            for (let i = 0; i < 6; i++) {
                const x = boardOrigXpos + (6 - i) * boardCheckerSize;
                const y =
                    boardOrigYpos +
                    0.5 * boardHeight +
                    boardCfg.label.distanceToBoard * boardCheckerSize;
                const t = two.makeText((i + 1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = "center";
                t.baseline = "top";
                labels.add(t);
            }
            for (let i = 6; i < 12; i++) {
                const x = boardOrigXpos - (i - 5) * boardCheckerSize;
                const y =
                    boardOrigYpos +
                    0.5 * boardHeight +
                    boardCfg.label.distanceToBoard * boardCheckerSize;
                const t = two.makeText((i + 1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = "center";
                t.baseline = "top";
                labels.add(t);
            }
            for (let i = 12; i < 18; i++) {
                const x = boardOrigXpos + (i - 18) * boardCheckerSize;
                const y =
                    boardOrigYpos -
                    0.5 * boardHeight -
                    boardCfg.label.distanceToBoard * boardCheckerSize;
                const t = two.makeText((i + 1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = "center";
                t.baseline = "middle";
                labels.add(t);
            }
            for (let i = 18; i < 24; i++) {
                const x = boardOrigXpos + (i - 17) * boardCheckerSize;
                const y =
                    boardOrigYpos -
                    0.5 * boardHeight -
                    boardCfg.label.distanceToBoard * boardCheckerSize;
                const t = two.makeText((i + 1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = "center";
                t.baseline = "middle";
                labels.add(t);
            }
            return labels;
        }

        function drawCheckers() {
            const position = get(positionStore);
            position.board.points.forEach((point, index) => {
                let x, yBase;
                if (index === 0) {
                    x = boardOrigXpos;
                    yBase = boardOrigYpos + 0.5 * boardCheckerSize;
                } else if (index === 25) {
                    x = boardOrigXpos;
                    yBase = boardOrigYpos - 0.5 * boardCheckerSize;
                } else if (index <= 6) {
                    x = boardOrigXpos + (7 - index) * boardCheckerSize;
                    yBase = boardOrigYpos + 0.5 * boardHeight;
                } else if (index <= 12) {
                    x = boardOrigXpos - (index - 6) * boardCheckerSize;
                    yBase = boardOrigYpos + 0.5 * boardHeight;
                } else if (index <= 18) {
                    x = boardOrigXpos - (19 - index) * boardCheckerSize;
                    yBase = boardOrigYpos - 0.5 * boardHeight;
                } else {
                    x = boardOrigXpos + (index - 18) * boardCheckerSize;
                    yBase = boardOrigYpos - 0.5 * boardHeight;
                }
                for (let i = 0; i < point.checkers; i++) {
                    const y = yBase + (index !== 0 && index <= 12 || index === 25 ? -1 : 1) * (i + 0.5) * boardCfg.checker.sizeFactor * boardCheckerSize;
                    const checker = two.makeCircle(x, y, boardCfg.checker.sizeFactor * boardCheckerSize / 2);
                    checker.fill = boardCfg.checker.colors[point.color];
                    checker.stroke = boardCfg.triangle.stroke;
                    checker.linewidth = boardCfg.checker.linewidth; // Use checker linewidth
                }
            });

            // Draw checkers on the bar above the bar
            position.board.points.forEach((point, index) => {
                if (index === 0 || index === 25) {
                    let x = boardOrigXpos;
                    let yBase = index === 0 ? boardOrigYpos + 0.5 * boardCheckerSize : boardOrigYpos - 0.5 * boardCheckerSize;
                    for (let i = 0; i < point.checkers; i++) {
                        const y = yBase + (index === 0 ? 1 : -1) * (i + 0.5) * boardCfg.checker.sizeFactor * boardCheckerSize;
                        const checker = two.makeCircle(x, y, boardCfg.checker.sizeFactor * boardCheckerSize / 2);
                        checker.fill = boardCfg.checker.colors[point.color];
                        checker.stroke = boardCfg.triangle.stroke;
                        checker.linewidth = boardCfg.checker.linewidth; // Use checker linewidth
                    }
                }
            });
        }

        function drawDoublingCube() {
            const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
            const boardOrigXpos = width / 2;
            const boardOrigYpos = height / 2;
            const boardWidth = boardCfg.widthFactor * width;

            // Get the value for the doubling cube
            const cubeValue = get(positionStore).cube.value;
            const doublingCubeTextValue = Math.pow(2, cubeValue);

            // draw doubling cube on the left side of the board with a small gap
            const doublingCubeSize = 1.25 * boardCheckerSize;
            const gap = 0.4 * boardCheckerSize;
            const doublingCubeXpos = boardOrigXpos - boardWidth / 2 - doublingCubeSize / 2 - gap;
            const doublingCubeYpos = boardOrigYpos;
            const doublingCube = two.makeRectangle(
                doublingCubeXpos,
                doublingCubeYpos,
                doublingCubeSize,
                doublingCubeSize,
            );
            doublingCube.fill = "white";
            doublingCube.stroke = "black";
            doublingCube.linewidth = 3.5; // Further increased linewidth
            const doublingCubeText = two.makeText(doublingCubeTextValue.toString(), doublingCubeXpos, doublingCubeYpos);
            doublingCubeText.size = 34; // Checker size
            doublingCubeText.alignment = "center";
            doublingCubeText.baseline = "middle";
            doublingCubeText.translation.set(doublingCubeXpos, doublingCubeYpos + 0.05 * doublingCubeSize); // Center the text
        }

        function computePipCount() {
            const position = get(positionStore);
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

        function drawPipCounts() {

            const { pipCount1, pipCount2 } = computePipCount();

            const boardOrigXpos = width / 2;
            const boardOrigYpos = height / 2;
            const boardWidth = boardCfg.widthFactor * width;
            const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
            const gap = 1.4 * boardCheckerSize;

            const pipCountText1 = `pip: ${pipCount1}`;
            const pipCountText2 = `pip: ${pipCount2}`;

            const pipCount1Xpos = boardOrigXpos - boardWidth / 2 - 1.2 * boardCheckerSize;
            const pipCount1Ypos = boardOrigYpos + boardHeight / 2 + 0.47 * boardCheckerSize;

            const pipCount2Xpos = boardOrigXpos - boardWidth / 2 - 1.2 * boardCheckerSize;
            const pipCount2Ypos = boardOrigYpos - boardHeight / 2 - 0.47 * boardCheckerSize;

            const pipCountText1Element = two.makeText(pipCountText1, pipCount1Xpos, pipCount1Ypos);
            pipCountText1Element.size = 20;
            pipCountText1Element.alignment = "center";
            pipCountText1Element.baseline = "middle";
            pipCountText1Element.weight = "bold";

            const pipCountText2Element = two.makeText(pipCountText2, pipCount2Xpos, pipCount2Ypos);
            pipCountText2Element.size = 20;
            pipCountText2Element.alignment = "center";
            pipCountText2Element.baseline = "middle";
            pipCountText2Element.weight = "bold";
        }

        function drawBearoff() {
            const bearoff1 = get(positionStore).board.bearoff[0];
            const bearoff2 = get(positionStore).board.bearoff[1];
            const boardOrigXpos = width / 2;
            const boardOrigYpos = height / 2;
            const boardWidth = boardCfg.widthFactor * width;
            const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
            const gap = 1.2 * boardCheckerSize;

            const bearoffText1 = `(${bearoff1} OFF)`;
            const bearoffText2 = `(${bearoff2} OFF)`;

            const bearoff1Xpos = boardOrigXpos + boardWidth / 2 + gap;
            const bearoff1Ypos = boardOrigYpos + boardHeight / 2 - 3.7 * boardCheckerSize;

            const bearoff2Xpos = boardOrigXpos + boardWidth / 2 + gap;
            const bearoff2Ypos = boardOrigYpos - boardHeight / 2 + 3.7 * boardCheckerSize;

            const bearoffText1Element = two.makeText(bearoffText1, bearoff1Xpos, bearoff1Ypos);
            bearoffText1Element.size = 20;
            bearoffText1Element.alignment = "center";
            bearoffText1Element.baseline = "middle";

            const bearoffText2Element = two.makeText(bearoffText2, bearoff2Xpos, bearoff2Ypos);
            bearoffText2Element.size = 20;
            bearoffText2Element.alignment = "center";
            bearoffText2Element.baseline = "middle";
        }

        function drawDice() {
            const position = get(positionStore);
            const playerOnRoll = position.player_on_roll;
            const dice = position.dice;
            const decisionType = position.decision_type;

            const boardOrigXpos = width / 2;
            const boardOrigYpos = height / 2;
            const boardWidth = boardCfg.widthFactor * width;
            const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
            const gap = 0.3 * boardCheckerSize;
            const diceSize = 0.95 * boardCheckerSize;

            const diceXpos = boardOrigXpos + boardWidth / 2 + 3 * gap;
            const diceYpos = playerOnRoll === 0 ? boardOrigYpos + 0.5 * boardHeight - 1.5 * boardCheckerSize : boardOrigYpos - 0.5 * boardHeight + 1.5 * boardCheckerSize;

            dice.forEach((die, index) => {
                const dieXpos = diceXpos + index * (diceSize + gap);
                const dieElement = two.makeRectangle(dieXpos, diceYpos, diceSize, diceSize);
                dieElement.fill = "white";
                dieElement.stroke = "black";
                dieElement.linewidth = 3.5;

                if (decisionType === 0) {
                    // Draw dots for traditional dice
                    const dotPositions = [
                        [],
                        [[0, 0]],
                        [[-0.7, -0.7], [0.7, 0.7]],
                        [[-0.7, -0.7], [0, 0], [0.7, 0.7]],
                        [[-0.7, -0.7], [0.7, -0.7], [-0.7, 0.7], [0.7, 0.7]],
                        [[-0.7, -0.7], [0.7, -0.7], [0, 0], [-0.7, 0.7], [0.7, 0.7]],
                        [[-0.7, -0.7], [0.7, -0.7], [-0.7, 0], [0.7, 0], [-0.7, 0.7], [0.7, 0.7]]
                    ];

                    dotPositions[die].forEach(([dx, dy]) => {
                        const dot = two.makeCircle(dieXpos + dx * diceSize / 3, diceYpos + dy * diceSize / 3, diceSize / 12);
                        dot.fill = "black";
                    });
                }
            });
        }

        function drawScores() {
            const boardOrigXpos = width / 2;
            const boardOrigYpos = height / 2;
            const boardWidth = boardCfg.widthFactor * width;
            const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;

            const score1 = get(positionStore).score[0];
            const score2 = get(positionStore).score[1];         

            const scoreText1 = score1 === 1 ? "crawford" : score1 === 0 ? "post" : score1 === -1 ? "unlimited" : `${score1} away`;
            const scoreText2 = score2 === 1 ? "crawford" : score2 === 0 ? "post" : score2 === -1 ? "unlimited" : `${score2} away`;

            const score1Xpos = boardOrigXpos + boardWidth / 2 + 1.2 * boardCheckerSize;
            const score1Ypos = boardOrigYpos + boardHeight / 2 + 0.47 * boardCheckerSize;

            const score2Xpos = boardOrigXpos + boardWidth / 2 + 1.2 * boardCheckerSize;
            const score2Ypos = boardOrigYpos - boardHeight / 2 - 0.47 * boardCheckerSize;

            const scoreText1Element = two.makeText(scoreText1, score1Xpos, score1Ypos - (score1 === 0 ? 10 : 0));
            scoreText1Element.size = 20;
            scoreText1Element.alignment = "center";
            scoreText1Element.baseline = "middle";
            scoreText1Element.weight = "bold";
            if (score1 === 0) {
                const scoreText1Element2 = two.makeText("crawford", score1Xpos, score1Ypos + 10);
                scoreText1Element2.size = 20;
                scoreText1Element2.alignment = "center";
                scoreText1Element2.baseline = "middle";
                scoreText1Element2.weight = "bold";
            }

            const scoreText2Element = two.makeText(scoreText2, score2Xpos, score2Ypos - (score2 === 0 ? 10 : 0));
            scoreText2Element.size = 20;
            scoreText2Element.alignment = "center";
            scoreText2Element.baseline = "middle";
            scoreText2Element.weight = "bold";
            if (score2 === 0) {
                const scoreText2Element2 = two.makeText("crawford", score2Xpos, score2Ypos + 10);
                scoreText2Element2.size = 20;
                scoreText2Element2.alignment = "center";
                scoreText2Element2.baseline = "middle";
                scoreText2Element2.weight = "bold";
            }
        }

        const labels = createLabels();

        const quadrant4 = createQuadrant(
            boardOrigXpos + 0.5 * boardCheckerSize,
            boardOrigYpos - boardTriangleHeight - 0.5 * boardCheckerSize,
            false,
        );

        const quadrant3 = createQuadrant(
            boardOrigXpos - 0.5 * boardWidth,
            boardOrigYpos - boardTriangleHeight - 0.5 * boardCheckerSize,
            false,
        );

        const quadrant2 = createQuadrant(
            boardOrigXpos - 0.5 * boardWidth,
            boardOrigYpos + 0.5 * boardCheckerSize,
            true,
        );

        const quadrant1 = createQuadrant(
            boardOrigXpos + 0.5 * boardCheckerSize,
            boardOrigYpos + 0.5 * boardCheckerSize,
            true,
        );

        // draw bar first to ensure checkers on the bar are drawn above it
        const bar = two.makeRectangle(
            boardOrigXpos,
            boardOrigYpos,
            boardCheckerSize,
            boardHeight,
        );
        bar.fill = boardCfg.fill;
        bar.stroke = boardCfg.stroke;
        bar.linewidth = 3.5; // Changed linewidth to 3.5

        drawCheckers();
        drawDoublingCube();
        drawScores();
        drawBearoff();        
        drawPipCounts();
        drawDice();

        // draw board outline on top to ensure consistent linewidth
        const board = two.makeRectangle(
            boardOrigXpos,
            boardOrigYpos,
            boardWidth,
            boardHeight,
        );
        board.fill = "transparent"; // No fill to avoid covering other elements
        board.stroke = boardCfg.stroke;
        board.linewidth = 3.5;
        
        two.update();
    }
</script>

<div class="canvas-container">
    <div id="backgammon-board" class="full-size-board"></div>
</div>

<style>
    body,
    html {
        height: 100%;
        width: 100%;
        margin: 0;
        display: flex;
        justify-content: center;
        align-items: center;
        flex-direction: column;
    }

    .canvas-container {
        width: 100%;
        height: 100%;
        display: flex;
        justify-content: center;
        align-items: center;
        margin: 0; /* Remove margin */
        padding: 0; /* Remove padding */
    }

    #backgammon-board {
        width: 100%;
        height: auto; /* Maintain aspect ratio */
        max-height: 100%; /* Ensure the board fits within the available height */
        box-sizing: border-box;
        padding: 0;
        border: 1px solid black; /* Add border for debugging */
        margin: 0; /* Remove margin */
    }
</style>
