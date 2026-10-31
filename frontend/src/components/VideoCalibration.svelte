<script>
    // Calibration of the board in the picture: four clicks on the corners of the playing field
    // (1-point, 12-point, 13-point, 24-point of the bottom player), a loupe to place them, ghost
    // checkers drawn from the resulting homography to judge the fit, calibrations named and kept
    // per draft. It lays itself over the <video> element it is given; it reads no pixel except
    // to magnify the picture under the pointer. `viewLost` is the vision core's verdict.
    import { onMount, untrack } from 'svelte';
    import { t } from '../i18n';
    import { calibrateBoard, invertHomography } from '../vision/geometry/index.js';
    import { closeOnEscape } from '../services/escapeService.js';
    import { loadCalibrations, saveCalibrations, freeName, pictureRect, pointerToPicture, scaleHomography, ghostCheckers, overlayInFrame } from '../services/videoCalibration.js';

    /**
     * @type {{
     *   calibrationKey: string | number | null,
     *   video: HTMLVideoElement | null,
     *   rotation?: number,
     *   viewLost?: boolean,
     *   oncalibration?: (c: { name: string, imageToModel: number[], modelToImage: number[], clicks: [number, number][], overlay?: { x: number, y: number, w: number, h: number } } | null) => void
     * }}
     */
    let { calibrationKey, video, rotation = 0, viewLost = false, oncalibration = undefined } = $props();

    const CORNER_LABELS = ['1', '12', '13', '24'];
    const LOUPE = 120;
    const LOUPE_SPAN = 32;

    let set = $state(untrack(() => loadCalibrations(calibrationKey)));
    /** @type {[number, number][] | null} null when not capturing */
    let capture = $state(null);
    /** @type {[number, number][] | null} the corners of the overlay being traced, null when not tracing */
    let tracing = $state(null);
    const capturing = $derived(capture !== null || tracing !== null);
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
    const toPixels = $derived(active && pic.w > 0 ? scaleHomography(active.modelToImage, pic.w, pic.h) : null);
    // A draft opened later reads its own calibrations; the first read is the initial state.
    let lastKey = untrack(() => calibrationKey);
    $effect(() => {
        const key = calibrationKey;
        if (key === lastKey) return;
        lastKey = key;
        set = loadCalibrations(key);
        capture = null;
        tracing = null;
    });

    // The caller learns the calibration in force, in fractions of the picture's width and height.
    $effect(() => {
        const a = active;
        const inverse = a ? invertHomography(a.modelToImage) : null;
        untrack(() => oncalibration?.(a && inverse ? { name: a.name, clicks: a.clicks, imageToModel: inverse, modelToImage: a.modelToImage, ...(a.overlay ? { overlay: a.overlay } : {}) } : null));
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
    function startTracing() {
        measure();
        tracing = [];
        capture = null;
        failed = false;
        pointer = null;
    }
    function cancelCapture() {
        capture = null;
        tracing = null;
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
        if (e.button !== 0) return;
        const p = locate(e);
        if (!p) return;
        if (tracing) {
            if (tracing.length === 0) {
                tracing = [[p.fx, p.fy]];
                return;
            }
            const [a] = tracing;
            const x = Math.min(a[0], p.fx);
            const y = Math.min(a[1], p.fy);
            const rect = { x, y, w: Math.max(a[0], p.fx) - x, h: Math.max(a[1], p.fy) - y };
            tracing = null;
            pointer = null;
            if (!active) return;
            // The core's reduced frame is 320×180: a zone under its hash grid there is refused.
            if (!overlayInFrame(rect, 320, 180)) {
                failed = true;
                return;
            }
            const target = active;
            commit({ items: set.items.map((c) => (c === target ? { ...c, overlay: rect } : c)) });
            return;
        }
        if (!capture) return;
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
        commit({ items: [...set.items, { name, clicks, modelToImage: r.modelToImage }], active: name });
    }

    // The loupe shows the picture as the screen shows it, so it is turned with the video.
    function drawLoupe() {
        const ctx = canvas?.getContext?.('2d');
        const p = /** @type {{ fx: number, fy: number } | null} */ (pointer);
        if (!ctx || !video || !p || !video.videoWidth) return;
        const half = LOUPE_SPAN / 2;
        ctx.save();
        ctx.clearRect(0, 0, LOUPE, LOUPE);
        ctx.translate(LOUPE / 2, LOUPE / 2);
        ctx.rotate((rotation * Math.PI) / 180);
        try {
            const k = LOUPE / LOUPE_SPAN;
            ctx.drawImage(video, p.fx * video.videoWidth - half, p.fy * video.videoHeight - half, LOUPE_SPAN, LOUPE_SPAN, -LOUPE / 2, -LOUPE / 2, LOUPE_SPAN * k, LOUPE_SPAN * k);
        } catch (_e) {
            /* the frame is not drawable yet: the loupe stays empty */
        }
        ctx.restore();
    }
    // Drawn as soon as the canvas exists and the pointer moves, and again as the video plays.
    $effect(() => {
        if (!capturing || !pointer || !canvas) return;
        drawLoupe();
        const id = setInterval(drawLoupe, 100);
        return () => clearInterval(id);
    });

    /** @param {string} name */
    function select(name) {
        commit({ active: name });
    }
    function clearOverlay() {
        const target = active;
        if (!target) return;
        commit({ items: set.items.map((c) => (c === target ? { name: c.name, clicks: c.clicks, modelToImage: c.modelToImage } : c)) });
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

    $effect(() => {
        if (capturing) return closeOnEscape(cancelCapture);
    });

    // The opening position's checkers, carried to the picture's pixels.
    const ghostShapes = $derived(toPixels ? ghostCheckers(toPixels).map((g) => ({ id: g.id, side: g.side, d: g.points.map((q) => `${q[0].toFixed(1)},${q[1].toFixed(1)}`).join(' ') })) : []);

    // The box turns as the video does, so what happens in it is in the unturned picture's frame.
    const sideways = $derived(rotation === 90 || rotation === 270);
    const loupeLeft = $derived.by(() => {
        const p = /** @type {{ x: number, y: number } | null} */ (pointer);
        return !p || p.x > 160 || p.y > 160;
    });
</script>

<div class="calib" data-testid="video-calibration" data-state={capturing ? 'capturing' : active ? 'ready' : 'none'}>
    <div class="calib-box" class:turned={rotation !== 0} class:sideways style:--turn="{rotation}deg" bind:this={box}>
        <div class="calib-picture" style:left="{pic.x}px" style:top="{pic.y}px" style:width="{pic.w}px" style:height="{pic.h}px">
            {#if pic.w > 0}
                <svg viewBox="0 0 {pic.w} {pic.h}" width={pic.w} height={pic.h} aria-hidden="true">
                    {#if ghosts && !capture}
                        {#each ghostShapes as g (g.id)}
                            <polygon class="ghost {g.side}" data-testid="calibration-ghost" data-side={g.side} points={g.d} />
                        {/each}
                    {/if}
                    {#each capture ?? [] as c, i (i)}
                        <circle class="dot" data-testid="calibration-click" cx={c[0] * pic.w} cy={c[1] * pic.h} r="4" />
                    {/each}
                    {#each tracing ?? [] as c, i (i)}
                        <circle class="dot" data-testid="calibration-overlay-click" cx={c[0] * pic.w} cy={c[1] * pic.h} r="4" />
                    {/each}
                    {#if active?.overlay && !capturing}
                        <rect
                            class="overlay-zone"
                            data-testid="calibration-overlay-zone"
                            x={active.overlay.x * pic.w}
                            y={active.overlay.y * pic.h}
                            width={active.overlay.w * pic.w}
                            height={active.overlay.h * pic.h}
                        />
                    {/if}
                </svg>
            {/if}
        </div>
        {#if capturing}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="calib-capture" data-testid="calibration-capture" onpointermove={onMove} onpointerdown={onClick}></div>
        {/if}
    </div>

    {#if capturing && pointer}
        <div class="loupe" class:right={!loupeLeft} data-testid="calibration-loupe">
            <canvas bind:this={canvas} width={LOUPE} height={LOUPE}></canvas>
            <span class="cross" aria-hidden="true"></span>
        </div>
    {/if}

    <div class="calib-bar r{rotation}" role="toolbar" aria-label={$t('video.calibration')}>
        {#if failed}<span class="calib-lost" data-testid="calibration-failed" role="status">{$t('video.calibFailed')}</span>{/if}
        {#if tracing}
            <span class="calib-step" data-testid="calibration-overlay-step">▭ {tracing.length + 1}/2</span>
            <button class="calib-btn" data-testid="calibration-cancel" onmousedown={(e) => e.preventDefault()} onclick={cancelCapture}>✕</button>
        {:else if capture}
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
                    <button
                        class="calib-btn"
                        data-testid="calibration-overlay"
                        aria-pressed={!!active.overlay}
                        title={$t('video.calibOverlay')}
                        aria-label={$t('video.calibOverlay')}
                        onmousedown={(e) => e.preventDefault()}
                        onclick={startTracing}>▭</button
                    >
                    {#if active.overlay}
                        <button class="calib-btn" data-testid="calibration-overlay-clear" title={$t('video.calibOverlayClear')} onmousedown={(e) => e.preventDefault()} onclick={clearOverlay}
                            >▭✕</button
                        >
                    {/if}
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
    .ghost.far {
        fill: rgb(0 0 0 / 0.35);
        stroke: rgb(255 140 0 / 0.9);
    }
    .overlay-zone {
        fill: none;
        stroke: rgb(80 200 255 / 0.9);
        stroke-width: 1.5;
        stroke-dasharray: 4 3;
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
        top: 36px;
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
        top: 6px;
        right: 6px;
        display: flex;
        gap: 4px;
        align-items: center;
        pointer-events: auto;
    }
    .calib-bar.r180 {
        top: auto;
        bottom: 6px;
    }
    .calib-bar.r270 {
        right: auto;
        left: 60px;
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
