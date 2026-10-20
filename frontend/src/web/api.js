// Le client HTTP du front web : il appelle `/v1/…` comme tout client, donc ne peut rien que le
// contrat n'expose (ADR-0039).
// Le tenant n'est PAS envoyé par la page : le démon fait confiance à `X-Tenant-ID` posé par le
// mandataire (ADR-0005), et une page qui le poserait laisserait le visiteur choisir sa
// bibliothèque. Seule exception, le développement local (`?tenant=1` dans l'URL).

const devTenant = new URLSearchParams(window.location.search).get('tenant');

/**
 * L'erreur d'un appel refusé : le message du serveur, avec son code HTTP en `status`. Le refus
 * « pas de bibliothèque » dit ce qu'il faut faire, parce qu'ouvrir la page sans mandataire est
 * l'essai le plus courant et qu'un message technique n'y mène nulle part.
 * @param {Response} response
 * @param {string} text
 */
function failure(response, text) {
    const raw = extractError(text);
    const noTenant = response.status === 400 && /X-Tenant-ID/i.test(raw);
    const message = noTenant
        ? "Cette page se sert derrière un mandataire qui désigne la bibliothèque (en-tête X-Tenant-ID) ; sans mandataire, essayez en local avec ?tenant=1 dans l'adresse."
        : raw || `${response.status} ${response.statusText}`;
    return Object.assign(new Error(message), { status: response.status, readable: noTenant });
}

/**
 * Appelle une route du contrat. Rend le corps décodé, ou lève une erreur
 * portant le message du serveur — jamais un « quelque chose a échoué ».
 * @param {string} route par exemple 'search.query'
 * @param {object} body
 */
export async function call(route, body = {}) {
    /** @type {Record<string, string>} */
    const headers = { 'Content-Type': 'application/json' };
    if (devTenant) headers['X-Tenant-ID'] = devTenant;
    const response = await fetch(`/v1/${route}`, {
        method: 'POST',
        headers,
        body: JSON.stringify(body)
    });
    const text = await response.text();
    if (!response.ok) {
        throw failure(response, text);
    }
    return text ? parseBody(text) : null;
}

/**
 * Lecture conditionnelle (ADR-0057 règle 4) : l'étiquette d'une réponse précédente renvoyée dans
 * `If-None-Match` vaut un 304 sans corps tant que rien de ce que la route lit n'a changé — c'est
 * ce qui rend le sondage d'un affichage de tournoi presque gratuit.
 * @param {string} route
 * @param {object} body
 * @param {string} etag étiquette de la réponse précédente, '' la première fois
 * @returns {Promise<{ changed: boolean, etag: string, body: any }>}
 */
export async function callConditional(route, body, etag) {
    /** @type {Record<string, string>} */
    const headers = { 'Content-Type': 'application/json' };
    if (devTenant) headers['X-Tenant-ID'] = devTenant;
    if (etag) headers['If-None-Match'] = etag;
    const response = await fetch(`/v1/${route}`, {
        method: 'POST',
        headers,
        body: JSON.stringify(body)
    });
    if (response.status === 304) return { changed: false, etag, body: null };
    const text = await response.text();
    if (!response.ok) {
        throw failure(response, text);
    }
    return { changed: true, etag: response.headers.get('ETag') || '', body: text ? parseBody(text) : null };
}

/**
 * Les routes qui diffusent (rpcStream) répondent en NDJSON, un objet par ligne ; décodées en une
 * passe, sans que la page connaisse la pagination.
 */
/** @param {string} text */
function parseBody(text) {
    const trimmed = text.trim();
    if (!trimmed) return null;
    if (trimmed.startsWith('{') && !trimmed.includes('\n{')) {
        return JSON.parse(trimmed);
    }
    if (trimmed.startsWith('[')) return JSON.parse(trimmed);
    return trimmed
        .split('\n')
        .filter(Boolean)
        .map((/** @type {string} */ line) => JSON.parse(line));
}

/** @param {string} text */
function extractError(text) {
    try {
        const parsed = JSON.parse(text);
        return parsed?.error?.message || parsed?.message || '';
    } catch {
        return text.slice(0, 200);
    }
}
