<script>
    import { onMount, onDestroy } from 'svelte';
    import Two from 'two.js'

    let two;
    let width = window.innerWidth * 0.9;
    let height = window.innerHeight * 0.8;

    function resizeBoard() {
        width = window.innerWidth * 0.9;
        height = window.innerHeight * 0.8;
        two.width = width;
        two.height = height;
        two.renderer.setSize(width, height);
        drawBoard();
        two.update();
    }

    onMount(() => {
        const elem = document.getElementById('backgammon-board');
        const params = { width, height };
        two = new Two(params).appendTo(elem);

        drawBoard();
        window.addEventListener('resize', resizeBoard);
    });

    onDestroy(() => {
        window.removeEventListener('resize', resizeBoard);
    });

    function drawBoard() {
        two.clear(); // Clear the board before re-drawing

        const boardWidth = width;
        const boardHeight = height;

        // Draw background rectangle
        const board = two.makeRectangle(boardWidth / 2, boardHeight / 2, boardWidth, boardHeight);
        board.fill = '#D2B48C'; // Light brown color
        board.stroke = 'black';

        // Draw triangles for points
        for (let i = 0; i < 24; i++) {
            const isWhite = i % 2 === 0;
            const triangleHeight = boardHeight / 2;
            const triangleWidth = boardWidth / 12;

            const triangle = two.makePolygon(
                (i % 12) * triangleWidth + triangleWidth / 2,
                i < 12 ? boardHeight - triangleHeight / 2 : triangleHeight / 2,
                triangleWidth / 2,
                triangleHeight,
                3
            );

            triangle.fill = isWhite ? '#FFF' : '#000';
        }

        two.update();
    }

</script>

<div class="canvas-container">
<div id="backgammon-board">
</div>
</div>

<style>

    body, html {
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
