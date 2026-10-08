<script>
    // Plays a match video next to a panel: a file through the loopback media server and a
    // <video>, a YouTube source through the hosted player page and postMessage. The caller
    // drives it by bind:this — currentTimeMs(), seek(ms), togglePlay() — and never touches the
    // element, so the Transcription panel and the match review share one player.
    import { onDestroy, untrack } from 'svelte';
    import { MediaURL, ReleaseMedia, PickTranscriptionVideo, VideoSourceKind, YouTubeEmbedURL } from '../../wailsjs/go/gui/App.js';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';

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
        if (typeof m.time === 'number') ytTime = m.time;
        if (m.type === 'ready') {
            status = 'ready';
            if (pendingMs > 0) seek(pendingMs);
        } else if (m.type === 'state') {
            ytPlaying = m.state === 1;
            if (m.state === 0) onended?.();
        }
    }

    function postToPlayer(message) {
        if (!frame?.contentWindow || !ytOrigin) return;
        frame.contentWindow.postMessage(message, ytOrigin);
    }

    // A new source replaces the served media; the previous one is released first.
    $effect(() => {
        const current = source;
        let cancelled = false;
        status = 'loading';
        detail = '';
        src = '';
        kind = '';
        (async () => {
            try {
                const k = await VideoSourceKind(current);
                if (cancelled) return;
                kind = k;
                if (k === 'file') {
                    src = await MediaURL(current);
                } else if (k === 'youtube') {
                    src = await YouTubeEmbedURL(current);
                    ytOrigin = new URL(src).origin;
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
        };
    });

    $effect(() => {
        window.addEventListener('message', onYouTubeMessage);
        return () => window.removeEventListener('message', onYouTubeMessage);
    });

    onDestroy(() => {
        try {
            ReleaseMedia();
        } catch (_e) {
            /* no host: nothing was served */
        }
    });

    function onLoadedMetadata() {
        status = 'ready';
        if (pendingMs > 0 && video) video.currentTime = pendingMs / 1000;
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

<div class="video-pane" data-testid="video-pane" data-status={status}>
    {#if kind === 'file' && src && status !== 'missing' && status !== 'codec'}
        <!-- svelte-ignore a11y_media_has_caption -->
        <video bind:this={video} {src} controls preload="metadata" onloadedmetadata={onLoadedMetadata} onerror={onVideoError} onended={() => onended?.()}></video>
    {:else if kind === 'youtube' && src}
        <iframe bind:this={frame} {src} title={$t('video.player')} allow="autoplay; encrypted-media; fullscreen"></iframe>
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
        background: var(--color-text);
        aspect-ratio: 16 / 9;
        max-height: 40vh;
    }
    video,
    iframe {
        width: 100%;
        height: 100%;
        border: 0;
        display: block;
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
