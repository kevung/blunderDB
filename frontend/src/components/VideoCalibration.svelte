<script>
    // Calibration of the board in the picture: four clicks on the corners of the playing field
    // (1-point, 12-point, 13-point, 24-point of the bottom player), a loupe to place them, ghost
    // checkers drawn from the resulting homography to judge the fit, calibrations named and kept
    // per draft. It lays itself over the <video> element it is given; it reads no pixel except
    // to magnify the picture under the pointer. `viewLost` is the vision core's verdict.
    import { onMount, untrack } from 'svelte';
    import { t } from '../i18n';
    import { calibrateBoard, applyHomography, boardZones, checkerCenter } from '../vision/geometry/index.js';
    import { loadCalibrations, saveCalibrations, freeName, pictureRect, pointerToPicture } from '../services/videoCalibration.js';

    /**
     * @type {{
     *   calibrationKey: string | number | null,
     *   video: HTMLVideoElement | null,
     *   rotation?: number,
     *   viewLost?: boolean,
     *   oncalibration?: (c: { name: string, imageToModel: number[], modelToImage: number[], clicks: [number, number][] } | null) => void
     * }}
     */
    let { calibrationKey, video, rotation = 0, viewLost = false, oncalibration = undefined } = $props();

    const CORNER_LABELS = ['1', '12', '13', '24'];
    const LOUPE = 120;
    const LOUPE_SPAN = 32;

    let set = $state(untrack(() => loadCalibrations(calibrationKey)));
    /** @type {[number, number][] | null} null when not capturing */
    let capture = $state(null);
    let failed = $state(false);
    let ghosts = $state(true);
    /** @type {HTMLDivElement | null} */
    let box = $state(null);
    /** @type {HTMLCanvasElement | null} */
    let canvas = $state(null);
    let size = $state({ w: 0, h: 0 });
    /** @type {{ x: number, y: number, fx: number, fy: number } | null} */
    let pointer = $state(null);

    const active = $derived(set.items.find((c) => c.name === set.active) ?? null);
    const pic = $derived(pictureRect(size.w, size.h, video?.videoWidth ?? 0, video?.videoHeight ?? 0));
    const solved = $derived.by(() => {
        if (!active || pic.w <= 0) return null;
        const r = calibrateBoard(active.clicks.map(([x, y]) => [x * pic.w, y * pic.h]));
        return r.ok ? r : null;
    });

    // A draft opened later reads its own calibrations; the first read is the initial state.
    let lastKey = untrack(() => calibrationKey);
    $effect(() => {
        const key = calibrationKey;
        if (key === lastKey) return;
        lastKey = key;
        set = loadCalibrations(key);
        capture = null;
    });

    // The caller learns the calibration in force, in fractions of the picture's width and height.
    $effect(() => {
        const a = active;
        const r = a ? calibrateBoard(a.clicks) : null;
        untrack(() => oncalibration?.(a && r && r.ok ? { name: a.name, clicks: a.clicks, imageToModel: r.imageToModel, modelToImage: r.modelToImage } : null));
    });

    function measure() {
        if (box) size = { w: box.offsetWidth, h: box.offsetHeight };
    }
    onMount(() => {
        measure();
        if (typeof ResizeObserver === 'undefined' || !box) return;
        const ro = new ResizeObserver(measure);
        ro.observe(box);
        return () => ro.disconnect();
    });

    /** @param {Partial<typeof set>} patch */
    function commit(patch) {
        set = { ...set, ...patch };
        saveCalibrations(calibrationKey, set);
    }

    function startCapture() {
        measure();
        capture = [];
        failed = false;
        pointer = null;
    }
    function cancelCapture() {
        capture = null;
        failed = false;
        pointer = null;
    }

    /** @param {PointerEvent} e */
    function locate(e) {
        if (!box) return null;
        measure();
        const r = box.getBoundingClientRect();
        const dx = e.clientX - (r.left + r.width / 2);
        const dy = e.clientY - (r.top + r.height / 2);
        const p = pointerToPicture(dx, dy, rotation, pictureRect(box.offsetWidth, box.offsetHeight, video?.videoWidth ?? 0, video?.videoHeight ?? 0), box.offsetWidth, box.offsetHeight);
        return p ? { x: e.clientX - r.left, y: e.clientY - r.top, fx: p[0], fy: p[1] } : null;
    }

    /** @param {PointerEvent} e */
    function onMove(e) {
        pointer = locate(e);
        drawLoupe();
    }

    /** @param {PointerEvent} e */
    function onClick(e) {
        const p = locate(e);
        if (!p || !capture) return;
        failed = false;
        const clicks = /** @type {[number, number][]} */ ([...capture, [p.fx, p.fy]]);
        if (clicks.length < 4) {
            capture = clicks;
            return;
        }
        const r = calibrateBoard(clicks);
        if (!r.ok) {
            failed = true;
            capture = [];
            return;
        }
        const name = freeName(set.items, $t('video.calibDefaultName'));
        capture = null;
        pointer = null;
        commit({ items: [...set.items, { name, clicks }], active: name });
    }

    function drawLoupe() {
        const ctx = canvas?.getContext?.('2d');
        if (!ctx || !video || !pointer || !video.videoWidth) return;
        const half = LOUPE_SPAN / 2;
        const sx = pointer.fx * video.videoWidth;
        const sy = pointer.fy * video.videoHeight;
        ctx.clearRect(0, 0, LOUPE, LOUPE);
        try {
            ctx.drawImage(video, sx - half, sy - half, LOUPE_SPAN, LOUPE_SPAN, 0, 0, LOUPE, LOUPE);
        } catch (_e) {
            /* the frame is not drawable yet: the loupe stays empty */
        }
    }

    /** @param {string} name */
    function select(name) {
        commit({ active: name });
    }
    function remove() {
        if (!active) return;
        const items = set.items.filter((c) => c !== active);
        commit({ items, active: items[0]?.name ?? null });
    }
    /** @param {string} value */
    function rename(value) {
        const name = value.trim();
        if (!active || !name || name === active.name || set.items.some((c) => c.name === name)) return;
        const old = active;
        commit({ items: set.items.map((c) => (c === old ? { ...c, name } : c)), active: name });
    }

    /** @param {KeyboardEvent} e */
    function onKey(e) {
        if (capture && e.key === 'Escape') {
            e.preventDefault();
            e.stopImmediatePropagation();
            cancelCapture();
        }
    }
    onMount(() => {
        window.addEventListener('keydown', onKey, true);
        return () => window.removeEventListener('keydown', onKey, true);
    });

    // The ghost checkers: two per point (first and fifth), one per bar half and tray, each a
    // circle of one diameter in the board's frame carried to the picture.
    const ghostShapes = $derived.by(() => {
        if (!solved) return [];
        const H = solved.modelToImage;
        const shapes = [];
        for (const z of boardZones()) {
            const ks = z.kind === 'point' ? [0, 4] : [0];
            for (const k of ks) {
                const [cx, cy] = checkerCenter(z, k);
                const pts = [];
                for (let i = 0; i < 16; i++) {
                    const a = (i / 16) * 2 * Math.PI;
                    pts.push(
                        applyHomography(H, [cx + 0.5 * Math.cos(a), cy + 0.5 * Math.sin(a)])
                            .map((n) => n.toFixed(1))
                            .join(',')
                    );
                }
                shapes.push({ id: `${z.id}:${k}`, d: pts.join(' ') });
            }
        }
        return shapes;
    });

    // The box turns as the video does, so what happens in it is in the unturned picture's frame.
    const sideways = $derived(rotation === 90 || rotation === 270);
    const loupeLeft = $derived.by(() => {
        const p = /** @type {{ x: number, y: number } | null} */ (pointer);
        return !p || p.x > 160 || p.y > 160;
    });
</script>

<div class="calib" data-testid="video-calibration" data-state={capture ? 'capturing' : active ? 'ready' : 'none'}>
    <div class="calib-box" class:turned={rotation !== 0} class:sideways style:--turn="{rotation}deg" bind:this={box}>
        <div class="calib-picture" style:left="{pic.x}px" style:top="{pic.y}px" style:width="{pic.w}px" style:height="{pic.h}px">
            {#if pic.w > 0}
                <svg viewBox="0 0 {pic.w} {pic.h}" width={pic.w} height={pic.h} aria-hidden="true">
                    {#if ghosts && !capture}
                        {#each ghostShapes as g (g.id)}
                            <polygon class="ghost" data-testid="calibration-ghost" points={g.d} />
                        {/each}
                    {/if}
                    {#each capture ?? (active && !solved ? active.clicks : []) as c, i (i)}
                        <circle class="dot" data-testid="calibration-click" cx={c[0] * pic.w} cy={c[1] * pic.h} r="4" />
                    {/each}
                </svg>
            {/if}
        </div>
        {#if capture}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="calib-capture" data-testid="calibration-capture" onpointermove={onMove} onpointerdown={onClick}></div>
        {/if}
    </div>

    {#if capture && pointer}
        <div class="loupe" class:right={!loupeLeft} data-testid="calibration-loupe">
            <canvas bind:this={canvas} width={LOUPE} height={LOUPE}></canvas>
            <span class="cross" aria-hidden="true"></span>
        </div>
    {/if}

    <div class="calib-bar" role="toolbar" aria-label={$t('video.calibration')}>
        {#if failed}<span class="calib-lost" data-testid="calibration-failed" role="status">{$t('video.calibFailed')}</span>{/if}
        {#if capture}
            <span class="calib-step" data-testid="calibration-step">{capture.length + 1}/4 · {CORNER_LABELS[capture.length]}</span>
            <button class="calib-btn" data-testid="calibration-undo" disabled={capture.length === 0} onmousedown={(e) => e.preventDefault()} onclick={() => (capture = capture?.slice(0, -1) ?? null)}
                >↶</button
            >
            <button class="calib-btn" data-testid="calibration-cancel" onmousedown={(e) => e.preventDefault()} onclick={cancelCapture}>✕</button>
        {:else}
            {#if viewLost && active}
                <span class="calib-lost" data-testid="calibration-lost" role="status">{$t('video.viewLost')}</span>
            {/if}
            {#if set.items.length > 0}
                <select class="calib-select" data-testid="calibration-select" aria-label={$t('video.calibration')} value={set.active} onchange={(e) => select(e.currentTarget.value)}>
                    {#each set.items as c (c.name)}<option value={c.name}>{c.name}</option>{/each}
                </select>
                {#if active}
                    <input class="calib-name" data-testid="calibration-name" aria-label={$t('video.calibName')} value={active.name} onchange={(e) => rename(e.currentTarget.value)} />
                    <button
                        class="calib-btn"
                        data-testid="calibration-ghosts"
                        aria-pressed={ghosts}
                        title={$t('video.calibGhosts')}
                        onmousedown={(e) => e.preventDefault()}
                        onclick={() => (ghosts = !ghosts)}>◌</button
                    >
                    <button class="calib-btn" data-testid="calibration-delete" title={$t('video.calibDelete')} onmousedown={(e) => e.preventDefault()} onclick={remove}>🗑</button>
                {/if}
            {/if}
            <button
                class="calib-btn"
                data-testid="calibration-start"
                class:alert={viewLost && !!active}
                title={$t('video.calibrate')}
                aria-label={$t('video.calibrate')}
                onmousedown={(e) => e.preventDefault()}
                onclick={startCapture}>⌖</button
            >
        {/if}
    </div>
</div>

<style>
    .calib {
        position: absolute;
        inset: 0;
        pointer-events: none;
    }
    .calib-box {
        position: absolute;
        inset: 0;
    }
    .calib-box.turned {
        inset: auto;
        top: 50%;
        left: 50%;
        width: 100%;
        height: 100%;
        transform: translate(-50%, -50%) rotate(var(--turn));
    }
    .calib-box.sideways {
        width: 100cqh;
        height: 100cqw;
    }
    .calib-picture {
        position: absolute;
    }
    .ghost {
        fill: rgb(255 255 255 / 0.12);
        stroke: rgb(255 214 10 / 0.9);
        stroke-width: 1.25;
    }
    .dot {
        fill: rgb(255 214 10 / 0.9);
        stroke: black;
    }
    .calib-capture {
        position: absolute;
        inset: 0;
        pointer-events: auto;
        cursor: crosshair;
        touch-action: none;
    }
    .loupe {
        position: absolute;
        top: 8px;
        left: 8px;
        width: 120px;
        height: 120px;
        border: 2px solid white;
        border-radius: 50%;
        overflow: hidden;
        background: black;
    }
    .loupe.right {
        left: auto;
        right: 8px;
    }
    .loupe canvas {
        width: 100%;
        height: 100%;
    }
    .cross::before,
    .cross::after {
        content: '';
        position: absolute;
        background: rgb(255 214 10);
    }
    .cross::before {
        left: 50%;
        top: 0;
        bottom: 0;
        width: 1px;
    }
    .cross::after {
        top: 50%;
        left: 0;
        right: 0;
        height: 1px;
    }
    .calib-bar {
        position: absolute;
        bottom: 6px;
        right: 6px;
        display: flex;
        gap: 4px;
        align-items: center;
        pointer-events: auto;
    }
    .calib-btn,
    .calib-select,
    .calib-name {
        font-size: var(--font-size-small);
        padding: 1px 6px;
        border-radius: 3px;
        border: 0;
        background: rgb(0 0 0 / 0.6);
        color: white;
    }
    .calib-name {
        width: 7em;
    }
    .calib-btn[aria-pressed='false'] {
        opacity: 0.5;
    }
    .calib-btn.alert {
        background: rgb(200 60 0 / 0.9);
    }
    .calib-step,
    .calib-lost {
        font-size: var(--font-size-small);
        padding: 1px 6px;
        border-radius: 3px;
        background: rgb(0 0 0 / 0.6);
        color: white;
    }
    .calib-lost {
        background: rgb(200 60 0 / 0.9);
    }
</style>
