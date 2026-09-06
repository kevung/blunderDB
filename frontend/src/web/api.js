// Le client HTTP du front web (#295, fiche J.5).
//
// Aucune route n'a été ajoutée pour lui : il appelle `/v1/…` comme le client
// Python et comme n'importe quel client, ce que G.8 (#289) a rendu possible.
// C'est aussi ce qui garantit qu'il ne peut rien faire que le contrat n'expose
// déjà — le périmètre de l'ADR-0039 tient parce qu'il n'y a rien d'autre à
// appeler.
//
// # Le tenant n'est PAS envoyé par la page
//
// Le démon n'authentifie personne et fait confiance à `X-Tenant-ID`
// (ADR-0005) : il tourne derrière un mandataire qui pose cet en-tête. Une page
// qui poserait le sien laisserait n'importe quel visiteur choisir la
// bibliothèque qu'il lit. Le seul cas où la page en pose un est le
// développement local, sur un tenant nommé explicitement dans l'URL
// (`?tenant=1`) — c'est écrit dans la documentation comme un usage de
// développement, et cela ne change rien à la sécurité d'un démon qui accepte
// déjà l'en-tête de quiconque.

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
 * Les routes qui DIFFUSENT (rpcStream) répondent en NDJSON : un objet JSON par
 * ligne. Les décoder en une passe est ce qui permet à la liste de positions
 * d'arriver entière sans que la page ait à connaître la pagination.
 * @param {string} text
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
