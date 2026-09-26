// Le client HTTP du front web : il appelle `/v1/…` comme tout client, donc ne peut rien que le
// contrat n'expose (ADR-0039).
// Le tenant n'est PAS envoyé par la page : le démon fait confiance à `X-Tenant-ID` posé par le
// mandataire (ADR-0005), et une page qui le poserait laisserait le visiteur choisir sa
// bibliothèque. Seule exception, le développement local (`?tenant=1` dans l'URL).

const devTenant = new URLSearchParams(window.location.search).get('tenant');

/**
 * Appelle une route du contrat. Rend le corps décodé, ou lève une erreur
 * portant le message du serveur — jamais un « quelque chose a échoué ».
 * @param {string} route par exemple 'search.query'
 * @param {object} body
 */
export async function call(route, body = {}) {
    const headers = { 'Content-Type': 'application/json' };
    if (devTenant) headers['X-Tenant-ID'] = devTenant;
    const response = await fetch(`/v1/${route}`, {
        method: 'POST',
        headers,
        body: JSON.stringify(body)
    });
    const text = await response.text();
    if (!response.ok) {
        throw new Error(extractError(text) || `${response.status} ${response.statusText}`);
    }
    return text ? parseBody(text) : null;
}

/**
 * Les routes qui diffusent (rpcStream) répondent en NDJSON, un objet par ligne ; décodées en une
 * passe, sans que la page connaisse la pagination.
 */
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
        .map((line) => JSON.parse(line));
}

function extractError(text) {
    try {
        const parsed = JSON.parse(text);
        return parsed?.error?.message || parsed?.message || '';
    } catch {
        return text.slice(0, 200);
    }
}
