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
            linewidth: 2,
        },
        label: {
            size: 25,
            distanceToBoard: 0.4,
        },
        checker: {
            sizeFactor: 0.97,
            colors: ["black", "white"]
        }
    };

    let two;
    let canvas;
    let width = window.innerWidth;
    let height = canvasCfg.aspectFactor * width;
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
        width = window.innerWidth * 1.0;
        height = canvasCfg.aspectFactor * width;
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

    onMount(() => {
        canvas = document.getElementById("backgammon-board");
        const elem = canvas;
        const params = { width, height };
        two = new Two(params).appendTo(elem);

        canvas.addEventListener("mousedown", handleMouseDown);
        canvas.addEventListener("mousemove", handleMouseMove);
        canvas.addEventListener("mouseup", handleMouseUp);
        canvas.addEventListener("contextmenu", event => event.preventDefault());
        canvas.addEventListener("dblclick", handleDoubleClick);
        drawBoard();
        window.addEventListener("resize", resizeBoard);

        unsubscribe = positionStore.subscribe(() => {
            drawBoard();
            console.log("positionStore: ", get(positionStore));
        });
    });

    onDestroy(() => {
        canvas.removeEventListener("mousedown", handleMouseDown);
        canvas.removeEventListener("mousemove", handleMouseMove);
        canvas.removeEventListener("mouseup", handleMouseUp);
        canvas.removeEventListener("contextmenu", event => event.preventDefault());
        canvas.removeEventListener("dblclick", handleDoubleClick);
        window.removeEventListener("resize", resizeBoard);
        if (unsubscribe) unsubscribe();
    });

    function drawDoublingCube() {
        const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardWidth = boardCfg.widthFactor * width;

        // Get the value for the doubling cube
        const cubeValue = get(positionStore).cube.value;
        const doublingCubeTextValue = Math.pow(2, cubeValue);

        // draw doubling cube on the left side of the board with a small gap
        const doublingCubeSize = boardCheckerSize;
        const gap = 0.3 * boardCheckerSize;
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
        doublingCube.linewidth = 5; // Further increased linewidth
        const doublingCubeText = two.makeText(doublingCubeTextValue.toString(), doublingCubeXpos, doublingCubeYpos);
        doublingCubeText.size = 30; // Checker size
        doublingCubeText.alignment = "center";
        doublingCubeText.baseline = "middle";
        doublingCubeText.translation.set(doublingCubeXpos, doublingCubeYpos + 0.05 * doublingCubeSize); // Center the text
    }

    function drawBoard() {
        two.clear();

        const boardAspectFactor = 11 / 13;
        const boardWidth = boardCfg.widthFactor * width;
        const boardHeight = boardAspectFactor * boardWidth;
        const boardCheckerSize = boardHeight / 11;
        const boardTriangleHeight = 5 * boardCheckerSize;
        const boardTriangleWidth = 1.0 * boardCheckerSize;
        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;

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
                    checker.linewidth = boardCfg.triangle.linewidth;
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
                        checker.linewidth = boardCfg.triangle.linewidth;
                    }
                }
            });
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
        bar.linewidth = 5; // Further increased linewidth

        drawCheckers();
        drawDoublingCube();

        // draw board outline on top to ensure consistent linewidth
        const board = two.makeRectangle(
            boardOrigXpos,
            boardOrigYpos,
            boardWidth,
            boardHeight,
        );
        board.fill = "transparent"; // No fill to avoid covering other elements
        board.stroke = boardCfg.stroke;
        board.linewidth = 5; // Further increased linewidth
        
        two.update();
    }
</script>

<div class="canvas-container">
    <div id="backgammon-board"></div>
</div>

<style>
    body,
    html {
        height: 100%;
        margin: 0;
        display: flex;
        justify-content: center;
        align-items: center;
        flex-direction: column;
    }

    .canvas-container {
        display: flex;
        justify-content: center;
        align-items: center;
        margin-bottom: 0px;
    }

    #backgammon-board {
        width: 100%;
        height: 100%;
        border: 2px solid #000;
    }
</style>
