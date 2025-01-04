<script>
    import { positionStore } from "../stores/positionStore";
    import { onMount, onDestroy } from "svelte";
    import Two from "two.js";
    import { get } from 'svelte/store';
    import { statusBarModeStore, showSearchModalStore, showCommentStore, showMetadataModalStore, showTakePoint2ModalStore, showTakePoint4ModalStore } from '../stores/uiStore'; // Import showTakePoint2ModalStore and showTakePoint4ModalStore

    let mode;
    let showSearchModal = false;
    let showComment = false;
    let showMetadataModal = false; // Add showMetadataModal variable
    let showTakePoint2Modal = false; // Add showTakePoint2Modal variable
    let showTakePoint4Modal = false; // Add showTakePoint4Modal variable
    statusBarModeStore.subscribe(value => {
        mode = value;
    });
    showSearchModalStore.subscribe(value => {
        showSearchModal = value;
    });
    showCommentStore.subscribe(value => {
        showComment = value;
    });
    showMetadataModalStore.subscribe(value => {
        showMetadataModal = value;
    });
    showTakePoint2ModalStore.subscribe(value => {
        showTakePoint2Modal = value;
    });
    showTakePoint4ModalStore.subscribe(value => {
        showTakePoint4Modal = value;
    });
    
    let canvasCfg = {
        aspectFactor: 0.72,
    };

    let boardCfg = {
        widthFactor: 0.75, // Increase widthFactor to make the board take up more space
        orientation: "right",
        fill: "#f0f0f0", // Light grey background
        stroke: "#333333", // Dark grey border
        linewidth: 3,
        triangle: {
            fill1: "#d9d9d9", // Light grey
            fill2: "#a6a6a6", // Slightly darker grey for balanced contrast
            stroke: "#333333",
            linewidth: 1.3, // Changed linewidth to 1
        },
        label: {
            size: 20,
            distanceToBoard: 0.30,
        },
        checker: {
            sizeFactor: 0.97,
            colors: ["#333333", "#ffffff"], // Dark grey and white checkers
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
    let cubePosition = { x: 0, y: 0 };
    let previousDice = get(positionStore).dice; // Save previous dice values

    function handleMouseDown(event) {
        event.preventDefault(); // Prevent text or element selection
        if (mode !== "EDIT") return;

        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;

        isMouseDown = true;
        startMousePos = {
            x: mouseX,
            y: mouseY,
            button: event.button
        };
    }

    function handleMouseMove(event) {
        event.preventDefault(); // Prevent text or element selection
        if (mode !== "EDIT") return;

        // No longer dynamically filling checkers during mouse move
    }

    function handleMouseUp(event) {
        event.preventDefault(); // Prevent text or element selection
        if (mode !== "EDIT") return;

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
        const endChecker = getCheckerPointAndCount(endPos.x, endPos.y, startPos.button);

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

    function updateCheckerPositionByPoint(checkerPoint, checkerCount, button) {
        const color = (checkerPoint === 0 || checkerPoint === 25) ? (checkerPoint === 0 ? 1 : 0) : (button === 2 ? 1 : 0);

        positionStore.update(pos => {
            pos.board.points = pos.board.points.map((point, index) => {
                if (index === checkerPoint) {
                    if (point.checkers >= 5 && point.color === color) {
                        // Only add more checkers if clicked on the 5th checker
                        if (checkerCount === 5) {
                            return {
                                ...point,
                                checkers: point.checkers + 1
                            };
                        } else {
                            return {
                                ...point,
                                checkers: Math.min(checkerCount, 5),
                                color: color
                            };
                        }
                    } else {
                        return {
                            ...point,
                            checkers: Math.min(checkerCount, 5),
                            color: color
                        };
                    }
                }
                return point;
            });

            // Set color to -1 if no checkers on a point
            pos.board.points = pos.board.points.map(point => {
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

    function resetBoard() {
        positionStore.update(pos => {
            pos.board.points.forEach(point => point.checkers = 0);
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
            resetBoard();
        }
    }

    function handleDoublingCubeClick(event) {
        if (mode !== "EDIT") return;

        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;

        // Check if the click is within the doubling cube
        if (mouseX >= cubePosition.x - cubePosition.size / 2 && mouseX <= cubePosition.x + cubePosition.size / 2 &&
            mouseY >= cubePosition.y - cubePosition.size / 2 && mouseY <= cubePosition.y + cubePosition.size / 2) {
            positionStore.update(pos => {
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
        if (mode !== "EDIT") return;

        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;

        console.log("Rectangle or Dice click detected at:", mouseX, mouseY); // Debug log

        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardWidth = boardCfg.widthFactor * width;
        const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
        const boardHeight = (11 / 13) * boardWidth;
        const gap = 1.2 * boardCheckerSize;

        const bearoff1Xpos = boardOrigXpos + boardWidth / 2 + gap;
        const bearoff1Ypos = boardOrigYpos + boardHeight / 2 - 3.7 * boardCheckerSize;
        const score1Ypos = boardOrigYpos + boardHeight / 2 + 0.2 * boardCheckerSize;

        const bearoff2Xpos = boardOrigXpos + boardWidth / 2 + gap;
        const bearoff2Ypos = boardOrigYpos - boardHeight / 2 + 3.7 * boardCheckerSize;
        const score2Ypos = boardOrigYpos - boardHeight / 2 - 0.2 * boardCheckerSize;

        const isInsideTopPlayerRectangle = mouseX >= bearoff1Xpos - 0.75 * boardCheckerSize && mouseX <= bearoff1Xpos + 0.75 * boardCheckerSize &&
            mouseY >= Math.min(bearoff1Ypos, score1Ypos) && mouseY <= Math.max(bearoff1Ypos, score1Ypos);

        const isInsideBottomPlayerRectangle = mouseX >= bearoff2Xpos - 0.75 * boardCheckerSize && mouseX <= bearoff2Xpos + 0.75 * boardCheckerSize &&
            mouseY >= Math.min(bearoff2Ypos, score2Ypos) && mouseY <= Math.max(bearoff2Ypos, score2Ypos);

        const rectangle1Xpos = bearoff1Xpos;
        const rectangle1Ypos = (bearoff1Ypos + score1Ypos) / 2;
        const rectangle2Xpos = bearoff2Xpos;
        const rectangle2Ypos = (bearoff2Ypos + score2Ypos) / 2;
        const rectangleWidth = 1.5 * boardCheckerSize;
        const rectangleHeight1 = Math.abs(bearoff1Ypos - score1Ypos);
        const rectangleHeight2 = Math.abs(bearoff2Ypos - score2Ypos);

        const diceGap = 0.325 * boardCheckerSize;
        const diceSize = 0.7 * boardCheckerSize;
        const diceXpos = boardOrigXpos + boardWidth / 2 + 2 * diceGap;
        const diceYpos = get(positionStore).player_on_roll === 0 ? boardOrigYpos + 0.5 * boardHeight - 1.5 * boardCheckerSize : boardOrigYpos - 0.5 * boardHeight + 1.5 * boardCheckerSize;
        
        let isInsideDie1 = false;
        let isInsideDie2 = false;

        for (let index = 0; index < 2; index++) {
            const dieXpos = diceXpos + index * (diceSize + diceGap);
            if (mouseX >= dieXpos - diceSize / 2 && mouseX <= dieXpos + diceSize / 2 &&
                mouseY >= diceYpos - diceSize / 2 && mouseY <= diceYpos + diceSize / 2) {
                if (index === 0) {
                    isInsideDie1 = true;
                } else {
                    isInsideDie2 = true;
                }
            }
        }

        if (isInsideDie1 || isInsideDie2) {
            console.log("Die clicked"); // Debug log
        }

        positionStore.update(pos => {
            if (isInsideTopPlayerRectangle && !isInsideDie1 && !isInsideDie2) {
                console.log("Top player's rectangle clicked"); // Debug log
                pos.player_on_roll = 0;
                pos.decision_type = 1; // Set decision type to doubling cube
                previousDice = pos.dice; // Save previous dice values
                pos.dice = [0, 0];
                console.log("Updated decision_type to 1 for top player"); // Debug log
            } else if (isInsideBottomPlayerRectangle && !isInsideDie1 && !isInsideDie2) {
                console.log("Bottom player's rectangle clicked"); // Debug log
                pos.player_on_roll = 1;
                pos.decision_type = 1; // Set decision type to doubling cube
                pos.dice = [0, 0];
                console.log("Updated decision_type to 1 for bottom player"); // Debug log
            } else if (isInsideDie1) {
                console.log("Die 1 clicked"); // Debug log
                pos.decision_type = 0;
                pos.dice = previousDice; // Restore previous dice values
                if (event.button === 0) {
                    pos.dice[0] = (pos.dice[0] % 6) + 1; // Left click to increase
                } else if (event.button === 2) {
                    pos.dice[0] = (pos.dice[0] === 1 ? 6 : pos.dice[0] - 1); // Right click to decrease
                }
            } else if (isInsideDie2) {
                console.log("Die 2 clicked"); // Debug log
                pos.decision_type = 0;
                pos.dice = previousDice; // Restore previous dice values
                if (event.button === 0) {
                    pos.dice[1] = (pos.dice[1] % 6) + 1; // Left click to increase
                } else if (event.button === 2) {
                    pos.dice[1] = (pos.dice[1] === 1 ? 6 : pos.dice[1] - 1); // Right click to decrease
                }
            }

            console.log("Updated dice values:", pos.dice); // Debug log
            console.log("Updated position store:", pos); // Log the updated position store
            return pos;
        });
    }

    function handleScoreClick(event) {
        if (mode !== "EDIT") return;

        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;

        const boardOrigXpos = width / 2;
        const boardOrigYpos = height / 2;
        const boardWidth = boardCfg.widthFactor * width;
        const boardCheckerSize = (11 / 13) * (boardCfg.widthFactor * width) / 11;
        const boardHeight = (11 / 13) * boardWidth; // Define boardHeight

        const score1Xpos = boardOrigXpos + boardWidth / 2 + 1.2 * boardCheckerSize;
        const score1Ypos = boardOrigYpos + boardHeight / 2 + 0.2 * boardCheckerSize;
        const score2Xpos = boardOrigXpos + boardWidth / 2 + 1.2 * boardCheckerSize;
        const score2Ypos = boardOrigYpos - boardHeight / 2 - 0.2 * boardCheckerSize;

        const scoreWidth = 1.5 * boardCheckerSize;
        const scoreHeight = 0.5 * boardCheckerSize;

        // Check if the click is inside the top player's green rectangle
        if (mouseX >= score1Xpos - scoreWidth / 2 && mouseX <= score1Xpos + scoreWidth / 2 &&
            mouseY >= score1Ypos - scoreHeight / 2 && mouseY <= score1Ypos + scoreHeight / 2) {
            positionStore.update(pos => {
                if (event.button === 0) {
                    pos.score[0] = Math.max(pos.score[0] - 1, -1); // Decrement score
                } else if (event.button === 2) {
                    pos.score[0] = Math.min(pos.score[0] + 1, 99); // Increment score, max 99
                }
                if (pos.score[0] === -1) {
                    pos.score[1] = -1; // Set other player's score to unlimited
                }
                return pos;
            });
        }

        // Check if the click is inside the bottom player's green rectangle
        if (mouseX >= score2Xpos - scoreWidth / 2 && mouseX <= score2Xpos + scoreWidth / 2 &&
            mouseY >= score2Ypos - scoreHeight / 2 && mouseY <= score2Ypos + scoreHeight / 2) {
            positionStore.update(pos => {
                if (event.button === 0) {
                    pos.score[1] = Math.max(pos.score[1] - 1, -1); // Decrement score
                } else if (event.button === 2) {
                    pos.score[1] = Math.min(pos.score[1] + 1, 99); // Increment score, max 99
                }
                if (pos.score[1] === -1) {
                    pos.score[0] = -1; // Set other player's score to unlimited
                }
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

    function setBoardOrientation(orientation) {
        boardCfg.orientation = orientation;
        drawBoard();
    }

    function handleOrientationChange(event) {
        if (showComment) return; // Disable orientation change when comment panel is open
        if (event.ctrlKey && event.key === 'ArrowLeft') {
            setBoardOrientation("left");
        } else if (event.ctrlKey && event.key === 'ArrowRight') {
            setBoardOrientation("right");
        }
    }

    function handleKeyDown(event) {
        if (mode !== "EDIT" || showSearchModal || showMetadataModal || showTakePoint2Modal || showTakePoint4Modal) return; // Disable shortcuts when metadata modal, TakePoint2Modal, or TakePoint4Modal is open

        if (event.key === "Backspace" && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
            event.preventDefault();
            resetBoard();
        }
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
        canvas.addEventListener("mousedown", handleDoublingCubeClick);
        canvas.addEventListener("mousedown", handleRectangleAndDiceClick);
        canvas.addEventListener("mousedown", handleScoreClick);
        canvas.addEventListener("contextmenu", event => event.preventDefault()); // Deactivate contextual menu
        drawBoard();
        window.addEventListener("resize", resizeBoard);
        window.addEventListener("keydown", handleOrientationChange);
        window.addEventListener("keydown", handleKeyDown);

        unsubscribe = positionStore.subscribe(() => {
            drawBoard();
            const position = get(positionStore);
            console.log("positionStore.subscribe - decision_type: ", position.decision_type); // Debug log
            console.log("positionStore: ", position);
        });

        logCanvasSize();
        window.addEventListener("resize", logCanvasSize);
    });

    onDestroy(() => {
        canvas.removeEventListener("mousedown", handleMouseDown);
        canvas.removeEventListener("mousemove", handleMouseMove);
        canvas.removeEventListener("mouseup", handleMouseUp);
        canvas.removeEventListener("dblclick", handleDoubleClick);
        canvas.removeEventListener("mousedown", handleDoublingCubeClick);
        canvas.removeEventListener("mousedown", handleRectangleAndDiceClick);
        canvas.removeEventListener("mousedown", handleScoreClick);
        canvas.removeEventListener("contextmenu", event => event.preventDefault()); // Remove event listener
        window.removeEventListener("resize", resizeBoard);
        window.removeEventListener("resize", logCanvasSize);
        window.removeEventListener("keydown", handleOrientationChange);
        window.removeEventListener("keydown", handleKeyDown);
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

        const position = get(positionStore);
        console.log("drawBoard - decision_type: ", position.decision_type); // Debug log

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
            const position = get(positionStore);
            const flip = position.player_on_roll === 1;

            if (boardCfg.orientation === "right") {
                for (let i = 0; i < 6; i++) {
                    const x = boardOrigXpos + (6 - i) * boardCheckerSize;
                    const y = boardOrigYpos + 0.5 * boardHeight + boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? 24 - i : i + 1).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "top";
                    labels.add(t);
                }
                for (let i = 6; i < 12; i++) {
                    const x = boardOrigXpos - (i - 5) * boardCheckerSize;
                    const y = boardOrigYpos + 0.5 * boardHeight + boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? 24 - i : i + 1).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "top";
                    labels.add(t);
                }
                for (let i = 12; i < 18; i++) {
                    const x = boardOrigXpos + (i - 18) * boardCheckerSize;
                    const y = boardOrigYpos - 0.5 * boardHeight - boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? 24 - i : i + 1).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "middle";
                    labels.add(t);
                }
                for (let i = 18; i < 24; i++) {
                    const x = boardOrigXpos + (i - 17) * boardCheckerSize;
                    const y = boardOrigYpos - 0.5 * boardHeight - boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? 24 - i : i + 1).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "middle";
                    labels.add(t);
                }
            } else if (boardCfg.orientation === "left") {
                for (let i = 0; i < 6; i++) {
                    const x = boardOrigXpos - (6 - i) * boardCheckerSize;
                    const y = boardOrigYpos - 0.5 * boardHeight - boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? i + 1 : 24 - i).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "middle";
                    labels.add(t);
                }
                for (let i = 6; i < 12; i++) {
                    const x = boardOrigXpos + (i - 5) * boardCheckerSize;
                    const y = boardOrigYpos - 0.5 * boardHeight - boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? i + 1 : 24 - i).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "middle";
                    labels.add(t);
                }
                for (let i = 12; i < 18; i++) {
                    const x = boardOrigXpos - (i - 18) * boardCheckerSize;
                    const y = boardOrigYpos + 0.5 * boardHeight + boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? i + 1 : 24 - i).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "top";
                    labels.add(t);
                }
                for (let i = 18; i < 24; i++) {
                    const x = boardOrigXpos - (i - 17) * boardCheckerSize;
                    const y = boardOrigYpos + 0.5 * boardHeight + boardCfg.label.distanceToBoard * boardCheckerSize;
                    const t = two.makeText((flip ? i + 1 : 24 - i).toString(), x, y);
                    t.size = boardCfg.label.size;
                    t.alignment = "center";
                    t.baseline = "top";
                    labels.add(t);
                }
            }
            return labels;
        }

        function drawCheckers() {
            const position = get(positionStore);
            position.board.points.forEach((point, index) => {
                let x, yBase;
                if (boardCfg.orientation === "right") {
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
                } else if (boardCfg.orientation === "left") {
                    if (index === 0) {
                        x = boardOrigXpos;
                        yBase = boardOrigYpos + 0.5 * boardCheckerSize;
                    } else if (index === 25) {
                        x = boardOrigXpos;
                        yBase = boardOrigYpos - 0.5 * boardCheckerSize;
                    } else if (index <= 6) {
                        x = boardOrigXpos - (7 - index) * boardCheckerSize;
                        yBase = boardOrigYpos + 0.5 * boardHeight;
                    } else if (index <= 12) {
                        x = boardOrigXpos + (index - 6) * boardCheckerSize;
                        yBase = boardOrigYpos + 0.5 * boardHeight;
                    } else if (index <= 18) {
                        x = boardOrigXpos + (19 - index) * boardCheckerSize;
                        yBase = boardOrigYpos - 0.5 * boardHeight;
                    } else {
                        x = boardOrigXpos - (index - 18) * boardCheckerSize;
                        yBase = boardOrigYpos - 0.5 * boardHeight;
                    }
                }
                const checkersToDraw = Math.min(point.checkers, 5);
                for (let i = 0; i < checkersToDraw; i++) {
                    const y = yBase + (index !== 0 && index <= 12 || index === 25 ? -1 : 1) * (i + 0.5) * boardCfg.checker.sizeFactor * boardCheckerSize;
                    const checker = two.makeCircle(x, y, boardCfg.checker.sizeFactor * boardCheckerSize / 2);
                    checker.fill = boardCfg.checker.colors[point.color];
                    checker.stroke = boardCfg.triangle.stroke;
                    checker.linewidth = boardCfg.checker.linewidth; // Use checker linewidth
                    if (i === 4 && point.checkers > 5) {
                        const text = two.makeText(point.checkers.toString(), x, y);
                        text.size = 20; // Ensure consistent text size
                        text.alignment = "center";
                        text.baseline = "middle";
                        text.weight = "bold"; // Ensure consistent text weight
                        if (point.color === 0) {
                            text.fill = "#ffffff"; // Contrast color for black checker
                        } else if (point.color === 1) {
                            text.fill = "#333333"; // Contrast color for white checker
                        }
                    }
                }
            });

            // Draw checkers on the bar above the bar
            position.board.points.forEach((point, index) => {
                if (index === 0 || index === 25) {
                    let x = boardOrigXpos;
                    let yBase = index === 0 ? boardOrigYpos + 0.5 * boardCheckerSize : boardOrigYpos - 0.5 * boardCheckerSize;
                    const checkersToDraw = Math.min(point.checkers, 5);
                    for (let i = 0; i < checkersToDraw; i++) {
                        const y = yBase + (index === 0 ? 1 : -1) * (i + 0.5) * boardCfg.checker.sizeFactor * boardCheckerSize;
                        const checker = two.makeCircle(x, y, boardCfg.checker.sizeFactor * boardCheckerSize / 2);
                        checker.fill = boardCfg.checker.colors[point.color];
                        checker.stroke = boardCfg.triangle.stroke;
                        checker.linewidth = boardCfg.checker.linewidth; // Use checker linewidth
                        if (i === 4 && point.checkers > 5) {
                            const text = two.makeText(point.checkers.toString(), x, y);
                            text.size = 20; // Ensure consistent text size
                            text.alignment = "center";
                            text.baseline = "middle";
                            text.weight = "bold"; // Ensure consistent text weight
                            if (point.color === 0) {
                                text.fill = "#ffffff"; // Contrast color for black checker
                            } else if (point.color === 1) {
                                text.fill = "#333333"; // Contrast color for white checker
                            }
                        }
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
            const position = get(positionStore);
            const cubeValue = position.cube.value;
            const doublingCubeTextValue = Math.pow(2, cubeValue);

            // Determine the position of the doubling cube based on its owner
            const doublingCubeSize = 0.9 * boardCheckerSize; // Reduce the size of the doubling cube
            const gap = 0.75 * boardCheckerSize;

            if (position.cube.owner === -1) {
                cubePosition.x = boardOrigXpos - boardWidth / 2 - doublingCubeSize / 2 - gap;
                cubePosition.y = boardOrigYpos;
            } else if (position.cube.owner === 0) {
                cubePosition.x = boardOrigXpos - boardWidth / 2 - doublingCubeSize / 2 - gap;
                cubePosition.y = boardOrigYpos + 0.5 * boardHeight - 1.5 * boardCheckerSize;
            } else if (position.cube.owner === 1) {
                cubePosition.x = boardOrigXpos - boardWidth / 2 - doublingCubeSize / 2 - gap;
                cubePosition.y = boardOrigYpos - 0.5 * boardHeight + 1.5 * boardCheckerSize;
            }
            cubePosition.size = doublingCubeSize;

            const doublingCube = two.makeRectangle(
                cubePosition.x,
                cubePosition.y,
                doublingCubeSize,
                doublingCubeSize,
            );
            doublingCube.fill = "#ffffff"; // White doubling cube
            doublingCube.stroke = "#333333"; // Dark grey border
            doublingCube.linewidth = 2.5; // Adjust linewidth accordingly
            const doublingCubeText = two.makeText(doublingCubeTextValue.toString(), cubePosition.x, cubePosition.y);
            doublingCubeText.size = 34; // Checker size
            doublingCubeText.alignment = "center";
            doublingCubeText.baseline = "middle";
            doublingCubeText.translation.set(cubePosition.x, cubePosition.y + 0.05 * doublingCubeSize); // Center the text
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

            const pipCountText1 = `pip: ${pipCount1}`;
            const pipCountText2 = `pip: ${pipCount2}`;

            const pipCount1Xpos = boardOrigXpos - boardWidth / 2 - 1.2 * boardCheckerSize;
            const pipCount1Ypos = boardOrigYpos + boardHeight / 2 + 0.2 * boardCheckerSize; // Align with score height

            const pipCount2Xpos = boardOrigXpos - boardWidth / 2 - 1.2 * boardCheckerSize;
            const pipCount2Ypos = boardOrigYpos - boardHeight / 2 - 0.2 * boardCheckerSize; // Align with score height

            const pipCountText1Element = two.makeText(pipCountText1, pipCount1Xpos, pipCount1Ypos);
            pipCountText1Element.size = 20; // Same size as score
            pipCountText1Element.alignment = "center";
            pipCountText1Element.baseline = "middle";
            pipCountText1Element.weight = "bold";

            const pipCountText2Element = two.makeText(pipCountText2, pipCount2Xpos, pipCount2Ypos);
            pipCountText2Element.size = 20; // Same size as score
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

            let bearoff1Xpos, bearoff1Ypos, bearoff2Xpos, bearoff2Ypos;

            if (boardCfg.orientation === "right") {
                bearoff1Xpos = boardOrigXpos + boardWidth / 2 + gap;
                bearoff1Ypos = boardOrigYpos + boardHeight / 2 - 3.7 * boardCheckerSize;

                bearoff2Xpos = boardOrigXpos + boardWidth / 2 + gap;
                bearoff2Ypos = boardOrigYpos - boardHeight / 2 + 3.7 * boardCheckerSize;
            } else if (boardCfg.orientation === "left") {
                bearoff1Xpos = boardOrigXpos - boardWidth / 2 - gap;
                bearoff1Ypos = boardOrigYpos + boardHeight / 2 - 3.7 * boardCheckerSize;

                bearoff2Xpos = boardOrigXpos - boardWidth / 2 - gap;
                bearoff2Ypos = boardOrigYpos - boardHeight / 2 + 3.7 * boardCheckerSize;
            }

            const bearoffText1Element = two.makeText(bearoffText1, bearoff1Xpos, bearoff1Ypos);
            bearoffText1Element.size = 20;
            bearoffText1Element.alignment = "center";
            bearoffText1Element.baseline = "middle";

            const bearoffText2Element = two.makeText(bearoffText2, bearoff2Xpos, bearoff2Ypos);
            bearoffText2Element.size = 20;
            bearoffText2Element.alignment = "center";
            bearoffText2Element.baseline = "middle";

            // Define score positions
            const score1Ypos = boardOrigYpos + boardHeight / 2 + 0.2 * boardCheckerSize;
            const score2Ypos = boardOrigYpos - boardHeight / 2 - 0.2 * boardCheckerSize;

            // Add transparent rectangles with red borders
            const rectangle1 = two.makeRectangle(bearoff1Xpos, (bearoff1Ypos + score1Ypos) / 2, 1.5 * boardCheckerSize, Math.abs(bearoff1Ypos - score1Ypos));
            rectangle1.fill = "transparent";
            rectangle1.stroke = "red"; // Make border visible
            rectangle1.linewidth = 0;

            const rectangle2 = two.makeRectangle(bearoff2Xpos, (bearoff2Ypos + score2Ypos) / 2, 1.5 * boardCheckerSize, Math.abs(bearoff2Ypos - score2Ypos));
            rectangle2.fill = "transparent";
            rectangle2.stroke = "red"; // Make border visible
            rectangle2.linewidth = 0;
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
            const gap = 0.325 * boardCheckerSize; // Move the dice closer to the board
            const diceSize = 0.7 * boardCheckerSize; // Reduce the size of the dice

            const diceXpos = boardOrigXpos + boardWidth / 2 + 2 * gap;
            const diceYpos = playerOnRoll === 0 ? boardOrigYpos + 0.5 * boardHeight - 1.5 * boardCheckerSize : boardOrigYpos - 0.5 * boardHeight + 1.5 * boardCheckerSize;

            dice.forEach((die, index) => {
                const dieXpos = diceXpos + index * (diceSize + gap);
                const dieElement = two.makeRectangle(dieXpos, diceYpos, diceSize, diceSize);
                dieElement.fill = "#ffffff"; // White dice
                dieElement.stroke = "#333333"; // Dark grey border
                dieElement.linewidth = 2.5; // Adjust linewidth accordingly

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
            const score1Ypos = boardOrigYpos + boardHeight / 2 + 0.2 * boardCheckerSize; // Move closer to the middle

            const score2Xpos = boardOrigXpos + boardWidth / 2 + 1.2 * boardCheckerSize;
            const score2Ypos = boardOrigYpos - boardHeight / 2 - 0.2 * boardCheckerSize; // Move closer to the middle

            // Add visible red rectangles behind the score text
            const redRectangle1 = two.makeRectangle(score1Xpos, score1Ypos, 1.5 * boardCheckerSize, 0.5 * boardCheckerSize);
            redRectangle1.fill = "transparent";
            redRectangle1.stroke = "red"; // Make border visible
            redRectangle1.linewidth = 0;

            const redRectangle2 = two.makeRectangle(score2Xpos, score2Ypos, 1.5 * boardCheckerSize, 0.5 * boardCheckerSize);
            redRectangle2.fill = "transparent";
            redRectangle2.stroke = "red"; // Make border visible
            redRectangle2.linewidth = 0;

            // Add score text
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

            // Add transparent green rectangles on top of the score text
            const greenRectangle1 = two.makeRectangle(score1Xpos, score1Ypos, 1.5 * boardCheckerSize, 0.5 * boardCheckerSize);
            greenRectangle1.fill = "transparent";
            greenRectangle1.stroke = "transparent"; // Make border invisible
            greenRectangle1.linewidth = 2;

            const greenRectangle2 = two.makeRectangle(score2Xpos, score2Ypos, 1.5 * boardCheckerSize, 0.5 * boardCheckerSize);
            greenRectangle2.fill = "transparent";
            greenRectangle2.stroke = "transparent"; // Make border invisible
            greenRectangle2.linewidth = 2;
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

        drawDoublingCube();
        drawCheckers();
        drawBearoff();        
        drawPipCounts();
        drawDice();
        drawScores();

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
        user-select: none; /* Prevent text or element selection */
    }
</style>
