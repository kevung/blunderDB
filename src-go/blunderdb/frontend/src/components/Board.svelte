<script>
    import { onMount, onDestroy } from 'svelte';
    import Two from 'two.js'

    let canvasCfg = {
        aspectFactor: 0.63,
    }

    let boardCfg = {
        widthFactor: 0.63,
        orientation: 'right',
        fill: 'white',
        stroke: 'black',
        linewidth: 3,
        triangle: {
            fill1: 'white',
            fill2: 'rgb(208, 208, 208)',
            stroke: 'black',
            linewidth: 2,
        },
        label: {
            size: 25,
            distanceToBoard: 0.4
        }
    };

    let two;
    let width = window.innerWidth;
    let height = canvasCfg.aspectFactor * width;


    function resizeBoard() {
        width = window.innerWidth * 1.0;
        height = canvasCfg.aspectFactor * width;
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
        two.clear();

        const boardAspectFactor = 11./13.;
        const boardWidth = boardCfg.widthFactor * width;
        const boardHeight = boardAspectFactor * boardWidth;
        const boardCheckerSize = boardHeight/11.;
        const boardTriangleHeight = 5. * boardCheckerSize;
        const boardTriangleWidth = 1.0 * boardCheckerSize;
        const boardOrigXpos = width/2.;
        const boardOrigYpos = height/2.;

        // draw board
        const board = two.makeRectangle(boardOrigXpos, boardOrigYpos,
            boardWidth, boardHeight);
        board.fill = boardCfg.fill; // Light brown color
        board.stroke = boardCfg.stroke;
        board.linewidth = boardCfg.linewidth;

        // draw bar
        const bar = two.makeRectangle(boardOrigXpos, boardOrigYpos,
            boardCheckerSize, boardHeight);
        bar.fill = boardCfg.fill;
        bar.stoke = boardCfg.stoke;
        bar.linewidth = boardCfg.linewidth;

        function createTriangle(x, y, flip){
            if(flip==false) { 
                const triangle = two.makePath(
                    x, y,
                    x+boardCheckerSize, y,
                    x+0.5*boardCheckerSize, y+5*boardCheckerSize);
                triangle.stroke = boardCfg.triangle.stroke;
                triangle.linewidth = boardCfg.triangle.linewidth;
                return triangle;

            } else {
                const triangle = two.makePath(
                    x, y+boardTriangleHeight,
                    x+boardCheckerSize, y+boardTriangleHeight,
                    x+0.5*boardCheckerSize, y+boardTriangleHeight
                    -5*boardCheckerSize);

                triangle.stroke = boardCfg.triangle.stroke;
                triangle.linewidth = boardCfg.triangle.linewidth;
                return triangle;
            }
        }

        function createQuadrant(x, y, flip){
            let quadrant = two.makeGroup();
            for(let i=0; i<6; i++){
                const offsetX = x +i*boardCheckerSize;
                const offsetY = y;
                const t = createTriangle(offsetX, offsetY, flip);
                if(i % 2 == 1){
                    t.fill = boardCfg.triangle.fill1;
                } else {
                    t.fill = boardCfg.triangle.fill2;
                }

                //invert color
                if (flip){
                    if(i % 2 == 1){
                        t.fill = boardCfg.triangle.fill2;
                    } else {
                        t.fill = boardCfg.triangle.fill1;
                    }
                }

                quadrant.add(t);
            }
            return quadrant;
        }

        function createLabels(){
            let labels = two.makeGroup();
            for(let i=0; i<6; i++){
                const x = boardOrigXpos+(6-i)*boardCheckerSize;
                const y = boardOrigYpos+0.5*boardHeight
                +boardCfg.label.distanceToBoard*boardCheckerSize;
                const t = two.makeText((i+1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = 'center';
                t.baseline = 'top';
                labels.add(t);
            }
            for(let i=6; i<12; i++){
                const x = boardOrigXpos-(i-5)*boardCheckerSize;
                const y = boardOrigYpos+0.5*boardHeight
                    +boardCfg.label.distanceToBoard*boardCheckerSize;
                const t = two.makeText((i+1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = 'center';
                t.baseline = 'top';
                labels.add(t);
            }
            for(let i=12; i<18; i++){
                const x = boardOrigXpos+(i-18)*boardCheckerSize;
                const y = boardOrigYpos-0.5*boardHeight
                    -boardCfg.label.distanceToBoard*boardCheckerSize;
                const t = two.makeText((i+1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = 'center';
                t.baseline = 'middle';
                labels.add(t);
            }
            for(let i=18; i<24; i++){
                const x = boardOrigXpos+(i-17)*boardCheckerSize;
                const y = boardOrigYpos-0.5*boardHeight
                    -boardCfg.label.distanceToBoard*boardCheckerSize;
                const t = two.makeText((i+1).toString(), x, y);
                t.size = boardCfg.label.size;
                t.alignment = 'center';
                t.baseline = 'middle';
                labels.add(t);
            }
            return labels;
        }

        const labels = createLabels();

        const quadrant4 = createQuadrant(
            boardOrigXpos+0.5*boardCheckerSize,
            boardOrigYpos-boardTriangleHeight-0.5*boardCheckerSize,
            false
        );

        const quadrant3 = createQuadrant(
            boardOrigXpos-0.5*boardWidth,
            boardOrigYpos-boardTriangleHeight-0.5*boardCheckerSize,
            false
        );

        const quadrant2 = createQuadrant(
            boardOrigXpos-0.5*boardWidth,
            boardOrigYpos+0.5*boardCheckerSize,
            true
        );

        const quadrant1 = createQuadrant(
            boardOrigXpos+0.5*boardCheckerSize,
            boardOrigYpos+0.5*boardCheckerSize,
            true
        );

        two.update();
    }

</script>

<div class="canvas-container">
<div id="backgammon-board">
</div>
</div>
<p>Vars</p>
<ul>
    <li>width: {width}</li>
    <li>height: {height}</li>
    {#each Object.entries(canvasCfg) as [key, value]}
        <li>canvasCfg.{key}: {value}</li>
  {/each}
    {#each Object.entries(boardCfg) as [key, value]}
        <li>boardCfg.{key}: {value}</li>
  {/each}
</ul>

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
