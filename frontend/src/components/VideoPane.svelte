<script>
    // Plays a match video next to a panel: a file through the loopback media server and a
    // <video>, a YouTube source through the hosted player page and postMessage. The caller
    // drives it by bind:this — currentTimeMs(), seek(ms), togglePlay(), stepRate(direction) —
    // and never touches the element, so the Transcription panel and the match review share one
    // player. It fills the box its caller gives it; the media keeps its own ratio inside.
    import { untrack } from 'svelte';
    import { MediaURL, ReleaseMedia, PickTranscriptionVideo, VideoSourceKind, YouTubeEmbedURL } from '../../wailsjs/go/gui/App.js';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { VIDEO_RATES, stepRate as nextRate } from '../utils/videoRate.js';

    /** @type {{ source: string, startMs?: number, onrelocate?: (path: string) => void, onended?: () => void }} */
    let { source, startMs = 0, onrelocate = undefined, onended = undefined } = $props();

    /** @type {'' | 'file' | 'youtube' | 'url'} */
    let kind = $state('');
    let src = $state('');
    // 'loading' | 'ready' | 'missing' | 'codec' | 'error'
    let status = $state('loading');
    let detail = $state('');
    /** @type {HTMLVideoElement | null} */
    let video = $state(null);
    /** @type {HTMLIFrameElement | null} */
    let frame = $state(null);
    let ytOrigin = '';
    let ytTime = 0;
    let ytPlaying = false;
    /** @type {HTMLDivElement | null} */
    let root = $state(null);
    // The speed applied, as the player reports it, and the speeds the source accepts.
    let rate = $state(1);
    /** @type {readonly number[]} */
    let rates = $state(VIDEO_RATES);
    // The speed shows a moment after each change, and for as long as it is not 1×.
    let rateFlash = $state(false);
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let rateFlashTimer;
    // The instant asked before the media is ready; read once, later seeks go through seek().
    let pendingMs = untrack(() => startMs);

    // Container by extension, with the MIME type canPlayType is asked about.
    const CONTAINERS = {
        mp4: ['MP4', 'video/mp4'],
        m4v: ['MP4', 'video/mp4'],
        mov: ['QuickTime', 'video/quicktime'],
        webm: ['WebM', 'video/webm'],
        mkv: ['Matroska', 'video/x-matroska'],
        ogv: ['Ogg', 'video/ogg']
    };

    function containerOf(path) {
        const ext = String(path).split('.').pop().toLowerCase();
        return CONTAINERS[ext] || [ext.toUpperCase(), ''];
    }

    function onYouTubeMessage(event) {
        if (!ytOrigin || event.origin !== ytOrigin || event.source !== frame?.contentWindow) return;
        const m = event.data;
        if (!m || m.source !== 'blunderdb-yt') return;
        if (typeof m.time === 'number' && m.type !== 'ready') ytTime = m.time;
        if (Array.isArray(m.rates) && m.rates.length) rates = m.rates;
        if (m.type === 'ready') {
            status = 'ready';
            ytPlaying = false;
            // A frame moved in the DOM reloads its page: it resumes where it was, at its speed.
            const at = ytTime > 0 ? ytTime * 1000 : pendingMs;
            if (at > 0) seek(at);
            if (rate !== 1) postToPlayer({ type: 'rate', rate });
        } else if (m.type === 'rate' && typeof m.rate === 'number') {
            showRate(m.rate);
        } else if (m.type === 'state') {
            ytPlaying = m.state === 1;
            if (m.state === 0) onended?.();
        }
    }

    function postToPlayer(message) {
        if (!frame?.contentWindow || !ytOrigin) return;
        frame.contentWindow.postMessage(message, ytOrigin);
    }

    function release(url) {
        if (!url) return;
        try {
            Promise.resolve(ReleaseMedia(url)).catch(() => {});
        } catch (_e) {
            /* no host: nothing was served */
        }
    }

    // The pane releases the URL it was given when it closes or changes source, and only that
    // one: another pane's video keeps playing.
    $effect(() => {
        const current = source;
        let cancelled = false;
        let served = '';
        status = 'loading';
        detail = '';
        src = '';
        kind = '';
        rate = 1;
        rates = VIDEO_RATES;
        ytTime = 0;
        (async () => {
            try {
                const k = await VideoSourceKind(current);
                if (cancelled) return;
                kind = k;
                if (k === 'file' || k === 'youtube') {
                    const url = await (k === 'file' ? MediaURL(current) : YouTubeEmbedURL(current));
                    // Closed or re-sourced while the host answered: nobody will play it.
                    if (cancelled) {
                        release(url);
                        return;
                    }
                    served = url;
                    // The hosted page reads the parent's origin from the referrer; a webview
                    // that sends none leaves it this parameter to post back to.
                    src = k === 'youtube' ? `${url}?origin=${encodeURIComponent(window.location.origin)}` : url;
                    if (k === 'youtube') ytOrigin = new URL(url).origin;
                } else {
                    status = 'error';
                    detail = $t('video.notPlayable');
                    return;
                }
                if (cancelled) return;
                // A file waits for the element's own events; YouTube for the player's 'ready'.
            } catch (error) {
                if (cancelled) return;
                logger.error('video source:', error);
                status = kind === 'file' ? 'missing' : 'error';
                detail = String(error?.message || error);
            }
        })();
        return () => {
            cancelled = true;
            release(served);
        };
    });

    $effect(() => {
        window.addEventListener('message', onYouTubeMessage);
        window.addEventListener('blur', onWindowBlur);
        return () => {
            window.removeEventListener('message', onYouTubeMessage);
            window.removeEventListener('blur', onWindowBlur);
            clearTimeout(rateFlashTimer);
        };
    });

    // A focused <video> or player frame keeps the keyboard: the frame's document swallows every
    // key, the element's native controls read the arrows. The pane takes the focus back, so the
    // caller's video keys and the board's Ctrl+Left/Right keep working; clicks still reach the
    // controls, which do not need the focus.
    function reclaimFocus() {
        root?.focus({ preventScroll: true });
    }

    function onWindowBlur() {
        setTimeout(() => {
            if (frame && document.activeElement === frame) reclaimFocus();
        }, 0);
    }

    /** @param {number} applied */
    function showRate(applied) {
        rate = applied;
        rateFlash = true;
        clearTimeout(rateFlashTimer);
        rateFlashTimer = setTimeout(() => (rateFlash = false), 1200);
    }

    function onLoadedMetadata() {
        status = 'ready';
        if (pendingMs > 0 && video) video.currentTime = pendingMs / 1000;
        if (video) video.playbackRate = rate;
    }

    // What the webview cannot read is named by its container and by the packages that give a
    // Linux webview its decoders; a black frame says nothing.
    function onVideoError() {
        const [container, mime] = containerOf(source);
        const code = video?.error?.code;
        status = 'codec';
        const verdict = mime ? video?.canPlayType(mime) || '' : '';
        detail = $t('video.codecDetail', { container, support: verdict === '' ? $t('video.unsupported') : verdict, code: code ?? '?' });
    }

    async function relocate() {
        try {
            const path = await PickTranscriptionVideo();
            if (path) onrelocate?.(path);
        } catch (error) {
            logger.error('relocating the video:', error);
        }
    }

    /** @returns {number | null} the player's current instant in ms, null when unknown. */
    export function currentTimeMs() {
        if (status !== 'ready') return null;
        if (kind === 'youtube') return Math.round(ytTime * 1000);
        return video ? Math.round(video.currentTime * 1000) : null;
    }

    /** @param {number} ms */
    export function seek(ms) {
        const at = Math.max(0, ms);
        pendingMs = at;
        if (status !== 'ready') return;
        if (kind === 'youtube') {
            ytTime = at / 1000;
            postToPlayer({ type: 'seek', time: at / 1000 });
        } else if (video) {
            video.currentTime = at / 1000;
        }
    }

    /**
     * One step slower (-1) or faster (+1) among the speeds the source accepts; the ends hold.
     *
     * @param {-1 | 1} direction
     */
    export function stepRate(direction) {
        if (status !== 'ready') return;
        const next = nextRate(rate, direction, rates);
        if (next === rate) return;
        if (kind === 'youtube') {
            // The shown speed is the one the player answers with, not the one asked.
            postToPlayer({ type: 'rate', rate: next });
        } else if (video) {
            video.playbackRate = next;
            showRate(video.playbackRate);
        }
    }

    /** @returns {number} the speed applied */
    export function playbackRate() {
        return rate;
    }

    export function togglePlay() {
        if (status !== 'ready') return;
        if (kind === 'youtube') {
            postToPlayer({ type: ytPlaying ? 'pause' : 'play' });
            ytPlaying = !ytPlaying;
        } else if (video) {
            if (video.paused) video.play().catch((error) => logger.error('play:', error));
            else video.pause();
        }
    }
</script>

<!-- A click anywhere in the pane, the native controls included, leaves the keyboard to the
     caller: the focus comes back here once the click is through. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="video-pane" data-testid="video-pane" data-status={status} tabindex="-1" bind:this={root} onpointerdown={() => setTimeout(reclaimFocus, 0)}>
    {#if kind === 'file' && src && status !== 'missing' && status !== 'codec'}
        <!-- svelte-ignore a11y_media_has_caption -->
        <video
            bind:this={video}
            {src}
            controls
            preload="metadata"
            onloadedmetadata={onLoadedMetadata}
            onerror={onVideoError}
            onended={() => onended?.()}
            onfocus={reclaimFocus}
            onratechange={() => video && showRate(video.playbackRate)}
        ></video>
    {:else if kind === 'youtube' && src}
        <iframe bind:this={frame} {src} title={$t('video.player')} allow="autoplay; encrypted-media; fullscreen"></iframe>
    {/if}
    {#if status === 'ready' && rateFlash}
        {#key rate}
            <span class="video-rate-flash" data-testid="video-rate-flash" aria-hidden="true">{rate}×</span>
        {/key}
    {/if}
    {#if status === 'ready' && rate !== 1}
        <span class="video-rate" data-testid="video-rate" title={$t('video.rate')}>{rate}×</span>
    {/if}
    {#if status === 'loading'}
        <p class="video-note">{$t('common.loading')}</p>
    {:else if status === 'missing'}
        <div class="video-note video-problem" data-testid="video-missing">
            <p>{$t('video.missing')}</p>
            {#if onrelocate}<button class="video-btn" onclick={relocate}>{$t('video.relocate')}</button>{/if}
        </div>
    {:else if status === 'codec' || status === 'error'}
        <div class="video-note video-problem" data-testid="video-diagnostic">
            <p>{detail}</p>
            {#if status === 'codec'}<p class="video-hint">{$t('video.codecHint')}</p>{/if}
        </div>
    {/if}
</div>

<style>
    .video-pane {
        position: relative;
        width: 100%;
        height: 100%;
        background: var(--color-text);
        outline: none;
        container-type: size;
    }
    .video-rate-flash {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        padding: 0.15em 0.5em;
        border-radius: 0.25em;
        background: rgb(0 0 0 / 0.6);
        color: white;
        font-size: var(--font-size-video-overlay);
        font-weight: 700;
        line-height: 1.1;
        pointer-events: none;
        animation: video-rate-fade 1.2s ease-in forwards;
    }
    @keyframes video-rate-fade {
        0%,
        60% {
            opacity: 1;
        }
        100% {
            opacity: 0;
        }
    }
    @media (prefers-reduced-motion: reduce) {
        .video-rate-flash {
            animation: none;
        }
    }
    .video-rate {
        position: absolute;
        top: 6px;
        left: 6px;
        padding: 1px 6px;
        border-radius: 3px;
        background: rgb(0 0 0 / 0.6);
        color: white;
        font-size: var(--font-size-small);
        pointer-events: none;
    }
    video,
    iframe {
        width: 100%;
        height: 100%;
        border: 0;
        display: block;
        object-fit: contain;
    }
    .video-note {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 6px;
        padding: 12px;
        text-align: center;
        color: var(--color-surface);
        font-size: var(--font-size-small);
        margin: 0;
    }
    .video-note p {
        margin: 0;
    }
    .video-hint {
        opacity: 0.8;
    }
    .video-btn {
        padding: 4px 10px;
        cursor: pointer;
    }
</style>
