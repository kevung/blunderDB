<script>
    // Plays a match video next to a panel: a file through the loopback media server and a
    // <video>, a YouTube source through the hosted player page and postMessage. The caller
    // drives it by bind:this — currentTimeMs(), seek(ms), togglePlay(), stepRate(direction) —
    // and never touches the element, so the Transcription panel and the match review share one
    // player. It fills the box its caller gives it; the media keeps its own ratio inside.
    import { untrack } from 'svelte';
    import { CheckVideoMux, MediaURL, ReleaseMedia, PickTranscriptionVideo, VideoSourceKind, YouTubeEmbedURL } from '../../wailsjs/go/gui/App.js';
    import { ClipboardSetText } from '../../wailsjs/runtime/runtime.js';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { VIDEO_RATES, stepRate as nextRate } from '../utils/videoRate.js';

    // `rotation`: 0, 90, 180 or 270 degrees clockwise. The media turns inside the box (the
    // player's own controls with it); at 90 and 270 it takes the box's height as its width and
    // the reverse, so the picture stays whole.
    /** @type {{ source: string, startMs?: number, rotation?: number, onrelocate?: (path: string) => void, onended?: () => void }} */
    let { source, startMs = 0, rotation = 0, onrelocate = undefined, onended = undefined } = $props();

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
    // A seek posted to the player, in seconds: the reports it sent before the seek reached it
    // would pull the instant back, and the Cursor that follows it with it. They are set aside
    // until one lands near the seek, or for a few seconds at most.
    /** @type {number | null} */
    let ytSeekingTo = null;
    let ytSeekingUntil = 0;
    const YT_SEEK_SETTLE_MS = 3000;
    /** @type {HTMLDivElement | null} */
    let root = $state(null);
    // The speed applied, as the player reports it, and the speeds the source accepts.
    let rate = $state(1);
    // WebKitGTK unmutes the element when the speed changes (and fires volumechange for it). The
    // viewer's mute is remembered, and put back when it is undone inside the window that a speed
    // change opens: an unmute outside it is the viewer's own.
    let userMuted = false;
    let rateWindowEnd = 0;
    function keepMuted() {
        if (!video) return;
        // A reload changes the speed too, through load() and then the speed it puts back.
        if (performance.now() < rateWindowEnd || reloading) {
            if (userMuted && !video.muted) video.muted = true;
        } else userMuted = video.muted;
    }
    /** @type {readonly number[]} */
    let rates = $state(VIDEO_RATES);
    // The speed shows a moment after each change, and for as long as it is not 1×.
    let rateFlash = $state(false);
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let rateFlashTimer;
    // The instant asked before the media is ready; read once, later seeks go through seek().
    let pendingMs = untrack(() => startMs);

    // Container by extension, with the MIME type canPlayType is asked about.
    /** @type {Record<string, [string, string]>} */
    const CONTAINERS = {
        mp4: ['MP4', 'video/mp4'],
        m4v: ['MP4', 'video/mp4'],
        mov: ['QuickTime', 'video/quicktime'],
        webm: ['WebM', 'video/webm'],
        mkv: ['Matroska', 'video/x-matroska'],
        ogv: ['Ogg', 'video/ogg']
    };

    /** @param {string} path */
    function containerOf(path) {
        const ext = (String(path).split('.').pop() ?? '').toLowerCase();
        return CONTAINERS[ext] || [ext.toUpperCase(), ''];
    }

    // A .webm/.mkv whose cue-referenced clusters open on a delta frame freezes on a seek there
    // (WebKitGTK decodes from the cluster's start). The check runs when the source is set;
    // its command, the ffmpeg remux that repairs the file, is also what the pane offers when
    // it gives up reloading.
    let mux = $state(/** @type {{ checked: boolean, faulty: number[] | null, command: string } | null} */ (null));
    let muxDismissed = $state(false);
    let copied = $state(false);
    const muxFaulty = $derived(mux?.faulty?.length ?? 0);

    async function copyCommand() {
        const text = mux?.command ?? '';
        if (!text) return;
        try {
            if (!(await ClipboardSetText(text))) throw new Error('backend clipboard write returned false');
        } catch (_e) {
            try {
                await navigator.clipboard.writeText(text);
            } catch (error) {
                logger.error('copying the remux command:', error);
                return;
            }
        }
        copied = true;
    }

    /** @param {string} path */
    function checkMux(path) {
        const ext = (String(path).split('.').pop() ?? '').toLowerCase();
        if (ext !== 'webm' && ext !== 'mkv') return Promise.resolve(null);
        return Promise.resolve(CheckVideoMux(path)).catch((/** @type {unknown} */ error) => {
            logger.warn('checking the video multiplexing:', error);
            return null;
        });
    }

    /** @param {MessageEvent} event */
    function onYouTubeMessage(event) {
        if (!ytOrigin || event.origin !== ytOrigin || event.source !== frame?.contentWindow) return;
        const m = event.data;
        if (!m || m.source !== 'blunderdb-yt') return;
        if (typeof m.time === 'number' && m.type !== 'ready' && reportLanded(m.time)) ytTime = m.time;
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

    /** @param {number} t - the player's reported instant, in seconds */
    function reportLanded(t) {
        if (ytSeekingTo === null) return true;
        if ((t >= ytSeekingTo - 0.5 && t <= ytSeekingTo + 2) || Date.now() > ytSeekingUntil) {
            ytSeekingTo = null;
            return true;
        }
        return false;
    }

    /** @param {Record<string, unknown>} message */
    function postToPlayer(message) {
        if (!frame?.contentWindow || !ytOrigin) return;
        frame.contentWindow.postMessage(message, ytOrigin);
    }

    /** @param {string} url */
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
        ytSeekingTo = null;
        reloads = 0;
        reloading = null;
        stalledOut = null;
        mux = null;
        muxDismissed = false;
        copied = false;
        (async () => {
            try {
                const k = await VideoSourceKind(current);
                if (cancelled) return;
                kind = /** @type {typeof kind} */ (k);
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
                    if (k === 'file')
                        checkMux(current).then((r) => {
                            if (!cancelled && r?.command) mux = r;
                        });
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
                status = /** @type {string} */ (kind) === 'file' ? 'missing' : 'error';
                detail = String((error instanceof Error && error.message) || error);
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
        // A late answer from a source given up on leaves the pane as it said.
        if (stalledOut) return;
        status = 'ready';
        queuedMs = null;
        const resume = reloading;
        reloading = null;
        if (pendingMs > 0 && video) askTime(pendingMs);
        if (!video) return;
        if (resume) {
            quietRateUntil = Date.now() + RATE_QUIET_MS;
            rateWindowEnd = performance.now() + RATE_QUIET_MS;
        }
        video.playbackRate = resume ? resume.rate : rate;
        if (resume?.playing) Promise.resolve(video.play()).catch((error) => logger.error('play:', error));
    }

    // What the webview cannot read is named by its container and by the packages that give a
    // Linux webview its decoders; a black frame says nothing.
    function onVideoError() {
        const [container, mime] = containerOf(source);
        const code = video?.error?.code;
        stalledOut = null;
        status = 'codec';
        const verdict = mime ? video?.canPlayType(mime) || '' : '';
        detail = $t('video.codecDetail', { container, support: verdict === '' ? $t('video.unsupported') : verdict, code: code ?? '?' });
    }

    // A pipeline missing a demuxer or decoder can load a file without ever raising an error
    // event; the pane then names the missing plugins instead of waiting on a black frame.
    const METADATA_MS = 10000;
    $effect(() => {
        if (status !== 'loading' || kind !== 'file' || !src) return;
        const timer = setTimeout(() => {
            const [container] = containerOf(source);
            stalledOut = null;
            status = 'codec';
            detail = $t('video.noMetadata', { seconds: METADATA_MS / 1000, container });
        }, METADATA_MS);
        return () => clearTimeout(timer);
    });

    async function relocate() {
        try {
            const path = await PickTranscriptionVideo();
            if (path) onrelocate?.(path);
        } catch (error) {
            logger.error('relocating the video:', error);
        }
    }

    // A seek asked while the <video> is still seeking waits for it, and only the last one
    // asked is kept: WebKitGTK's pipeline flushes and re-reads the file on every seek, and a
    // burst of them (Shift+Arrow held down) can leave the playback stalled.
    /** @type {number | null} */
    let queuedMs = null;

    function onSeeked() {
        // Landing is progress only when paused: playing, a reload that lands then freezes again
        // must still count against the limit.
        if (video?.paused) reloads = 0;
        if (queuedMs === null || !video) return;
        const at = queuedMs;
        queuedMs = null;
        askTime(at);
    }

    /** @param {number} ms */
    function askTime(ms) {
        if (!video) return;
        seekSince = Date.now();
        video.currentTime = ms / 1000;
    }

    // WebKitGTK can leave a file's pipeline dead without an error event: a decoder that fails
    // after a seek leaves the element seeking for good, and playback can freeze on one frame.
    // Without progress the source is reloaded, then put back at the instant last asked, at its
    // speed, playing or paused as it was. Each reload in a row waits longer; past MAX_RELOADS
    // the pane says so and offers to try again, rather than reloading a file that always fails
    // at the same instant.
    const STALL_MS = 3000;
    // Playback the element itself says is waiting for data is given longer before it counts.
    const BUFFERING_MS = 15000;
    const MAX_RELOADS = 3;
    const WATCH_MS = 500;
    // The speed a reload restores is not a gesture: its ratechange shows no indicator.
    const RATE_QUIET_MS = 500;
    const HAVE_FUTURE_DATA = 3;
    /** When the seek in flight was asked or first seen, null when none is. */
    /** @type {number | null} */
    let seekSince = null;
    /** Whether a seek that does not land while paused has been asked again. */
    let pausedAsked = false;
    /** The instant playback was last seen at, and since when. */
    let lastTime = NaN;
    let lastTimeSince = 0;
    /** Reloads in a row with no progress between them. */
    let reloads = 0;
    /** The instant, in seconds, the last reload went back to. */
    let progressFrom = NaN;
    const PROGRESS_S = 1;
    let quietRateUntil = 0;
    /** What a reload under way puts back once the metadata is in, null when none is. */
    /** @type {{ playing: boolean, rate: number, since: number } | null} */
    let reloading = null;
    /** What a retry after giving up resumes. */
    let stalledOut = $state(/** @type {{ playing: boolean, rate: number } | null} */ (null));

    function stallLimit() {
        return STALL_MS * (reloads + 1);
    }

    function watchdog() {
        if (!video || kind !== 'file' || status !== 'ready') return;
        const now = Date.now();
        if (reloading) {
            if (now - reloading.since > stallLimit()) reload(pendingMs, reloading.playing, reloading.rate);
            return;
        }
        if (video.seeking) {
            seekSince ??= now;
            if (now - seekSince > stallLimit()) {
                // A paused video is never reloaded: the source vanishes and nothing replaces it.
                // The instant is asked once more; a playing one may have a dead decoder.
                if (!video.paused) reload(queuedMs ?? pendingMs, true, rate);
                else if (!pausedAsked) {
                    pausedAsked = true;
                    askTime(queuedMs ?? pendingMs);
                }
            }
            return;
        }
        seekSince = null;
        pausedAsked = false;
        if (video.paused || video.ended) {
            lastTime = NaN;
            return;
        }
        // Progress is playback past the reload point, not one frame drawn there.
        if (reloads > 0 && video.currentTime > progressFrom + PROGRESS_S) reloads = 0;
        if (video.currentTime !== lastTime) {
            lastTime = video.currentTime;
            lastTimeSince = now;
            return;
        }
        const limit = video.readyState < HAVE_FUTURE_DATA ? Math.max(BUFFERING_MS, stallLimit()) : stallLimit();
        if (now - lastTimeSince > limit) reload(Math.round(video.currentTime * 1000), true, rate);
    }

    /**
     * @param {number} atMs
     * @param {boolean} playing
     * @param {number} speed
     */
    function reload(atMs, playing, speed) {
        if (!video) return;
        pendingMs = atMs;
        queuedMs = null;
        seekSince = null;
        pausedAsked = false;
        lastTime = NaN;
        if (reloads >= MAX_RELOADS) {
            logger.warn(`video still stalled after ${reloads} reloads, at ${atMs} ms`);
            reloading = null;
            stalledOut = { playing, rate: speed };
            status = 'error';
            // Only a check that found faulty clusters names the cause; otherwise the remux is a guess.
            detail = !mux?.command ? $t('video.stalled') : muxFaulty ? $t('video.stalledRemux') : $t('video.stalledRemuxMaybe');
            return;
        }
        reloads++;
        progressFrom = atMs / 1000;
        logger.warn(`video stalled, reload ${reloads} at ${atMs} ms`);
        reloading = { playing, rate: speed, since: Date.now() };
        // load() resets the speed to the default one.
        video.defaultPlaybackRate = speed;
        video.removeAttribute('src');
        video.load();
        video.src = src;
    }

    function retryStalled() {
        const resume = stalledOut;
        if (!resume) return;
        stalledOut = null;
        reloads = 0;
        status = 'ready';
        detail = '';
        reload(pendingMs, resume.playing, resume.rate);
    }

    $effect(() => {
        if (!video) return;
        const timer = setInterval(watchdog, WATCH_MS);
        return () => clearInterval(timer);
    });

    /** @returns {number | null} the player's current instant in ms, null when unknown. */
    export function currentTimeMs() {
        if (status !== 'ready') return null;
        if (kind === 'youtube') return Math.round(ytTime * 1000);
        if (queuedMs !== null) return queuedMs;
        if (reloading) return pendingMs;
        return video ? Math.round(video.currentTime * 1000) : null;
    }

    /** @param {number} ms */
    export function seek(ms) {
        if (!Number.isFinite(ms)) return;
        // Past its end a file has nothing to show: the seek stops at the last instant.
        const duration = kind === 'file' && video ? video.duration * 1000 : NaN;
        const at = Math.max(0, Number.isFinite(duration) ? Math.min(ms, duration) : ms);
        pendingMs = at;
        if (status !== 'ready') return;
        if (kind === 'youtube') {
            ytTime = at / 1000;
            ytSeekingTo = ytTime;
            ytSeekingUntil = Date.now() + YT_SEEK_SETTLE_MS;
            postToPlayer({ type: 'seek', time: at / 1000 });
        } else if (video && !reloading) {
            // A paused pipeline already blocked has no landing to wait for: the new instant goes now.
            const blocked = video.seeking && video.paused && pausedAsked;
            pausedAsked = false;
            if (video.seeking && !blocked) queuedMs = at;
            else {
                queuedMs = null;
                askTime(at);
            }
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
        if (reloading && video) {
            // Applied once the source is back; load() would reset it now.
            reloading.rate = next;
            video.defaultPlaybackRate = next;
            showRate(next);
        } else if (kind === 'youtube') {
            // The shown speed is the one the player answers with, not the one asked.
            postToPlayer({ type: 'rate', rate: next });
        } else if (video) {
            rateWindowEnd = performance.now() + 500;
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
        } else if (reloading) {
            reloading.playing = !reloading.playing;
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
            controlslist="nofullscreen"
            preload="metadata"
            onloadedmetadata={onLoadedMetadata}
            onseeked={onSeeked}
            onerror={onVideoError}
            onended={() => onended?.()}
            onfocus={reclaimFocus}
            class:turned={rotation !== 0}
            class:sideways={rotation === 90 || rotation === 270}
            style:--turn="{rotation}deg"
            data-rotation={rotation}
            onvolumechange={keepMuted}
            onratechange={() => {
                if (!video) return;
                keepMuted();
                if (reloading || Date.now() < quietRateUntil) return;
                showRate(video.playbackRate);
            }}
        ></video>
    {:else if kind === 'youtube' && src}
        <iframe
            bind:this={frame}
            {src}
            title={$t('video.player')}
            allow="autoplay; encrypted-media"
            class:turned={rotation !== 0}
            class:sideways={rotation === 90 || rotation === 270}
            style:--turn="{rotation}deg"
            data-rotation={rotation}
        ></iframe>
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
            {#if stalledOut && mux?.command}
                <code class="video-command" data-testid="video-remux-command">{mux.command}</code>
                <button class="video-btn" onclick={copyCommand}>{copied ? $t('video.copied') : $t('video.copyCommand')}</button>
            {/if}
            {#if stalledOut}<button class="video-btn" data-testid="video-retry" onclick={retryStalled}>{$t('video.stalledRetry')}</button>{/if}
        </div>
    {/if}
    {#if muxFaulty && !muxDismissed && status !== 'error' && status !== 'missing' && mux}
        <div class="video-mux" data-testid="video-mux-warning" role="status">
            <p>{$t('video.muxWarning', { count: muxFaulty })}</p>
            <code class="video-command">{mux.command}</code>
            <div class="video-mux-actions">
                <button class="video-btn" onclick={copyCommand}>{copied ? $t('video.copied') : $t('video.copyCommand')}</button>
                <button class="video-btn" data-testid="video-mux-dismiss" onclick={() => (muxDismissed = true)}>{$t('video.muxDismiss')}</button>
            </div>
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
    /* Turned about the box's centre; sideways, the element is as wide as the box is high. */
    video.turned,
    iframe.turned {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%) rotate(var(--turn));
    }
    video.sideways,
    iframe.sideways {
        width: 100cqh;
        height: 100cqw;
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
    .video-command {
        max-width: 100%;
        overflow-wrap: anywhere;
        user-select: text;
        font-size: var(--font-size-small);
    }
    .video-mux {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        display: flex;
        flex-direction: column;
        gap: 4px;
        padding: 6px 10px;
        background: rgb(0 0 0 / 0.75);
        color: white;
        font-size: var(--font-size-small);
    }
    .video-mux p {
        margin: 0;
    }
    .video-mux-actions {
        display: flex;
        gap: 6px;
    }
</style>
