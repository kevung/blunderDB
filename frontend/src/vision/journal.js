/**
 * Le journal local des descripteurs : les relevés de `descriptors.js`, session par session,
 * dans IndexedDB.
 *
 * C'est une commodité du poste (ADR-0108 règle 6) : hors base, hors
 * `issuance.CarriedMetadataKeys`, jamais exporté, effaçable depuis le volet. Il ne contient
 * que des descripteurs (quelques octets par ROI et par image), jamais d'image. Il sert à
 * rejouer les automates sans la vidéo (`replay`).
 *
 * Une session est une chaîne choisie par l'appelant (le `draftId` et le nom de la
 * calibration, par exemple). Les écritures sont groupées par lots ; `flush` vide le lot.
 */

export const JOURNAL_DB_NAME = 'blunderdb-vision-journal';
const STORE = 'descriptors';
const BY_SESSION = 'session';

/** @param {IDBRequest} req */
function done(req) {
    return new Promise((resolve, reject) => {
        req.onsuccess = () => resolve(req.result);
        req.onerror = () => reject(req.error);
    });
}

/** @param {IDBTransaction} tx */
function committed(tx) {
    return new Promise((resolve, reject) => {
        tx.oncomplete = () => resolve(undefined);
        tx.onerror = () => reject(tx.error);
        tx.onabort = () => reject(tx.error ?? new Error('vision: journal transaction aborted'));
    });
}

/** @param {IDBFactory} idb @returns {Promise<IDBDatabase>} */
function openDb(idb) {
    const req = idb.open(JOURNAL_DB_NAME, 1);
    req.onupgradeneeded = () => {
        const store = req.result.createObjectStore(STORE, { autoIncrement: true });
        store.createIndex(BY_SESSION, BY_SESSION);
    };
    return done(req);
}

/**
 * @param {object} options
 * @param {string} options.session
 * @param {IDBFactory} [options.indexedDB]
 * @param {number} [options.batchSize] relevés gardés en mémoire avant une écriture
 */
export async function openJournal({ session, indexedDB = globalThis.indexedDB, batchSize = 50 }) {
    if (typeof session !== 'string' || session === '') throw new TypeError('vision: journal session required');
    if (!indexedDB) return createMemoryJournal(session);
    const db = await openDb(indexedDB);
    /** @type {import('./descriptors.js').DescriptorRecord[]} */
    let pending = [];
    /** @type {Promise<unknown>} */
    let writing = Promise.resolve();

    function flush() {
        if (pending.length === 0) return writing;
        const batch = pending;
        pending = [];
        // Writes are chained so the journal keeps the order the frames were analysed in.
        writing = writing.then(() => {
            const tx = db.transaction(STORE, 'readwrite');
            const store = tx.objectStore(STORE);
            for (const record of batch) store.add({ session, record });
            return committed(tx);
        });
        return writing;
    }

    return {
        session,
        /** @param {import('./descriptors.js').DescriptorRecord} record */
        append(record) {
            pending.push(record);
            if (pending.length >= batchSize) void flush().catch(() => {});
        },
        flush,
        /** @returns {Promise<import('./descriptors.js').DescriptorRecord[]>} */
        async read() {
            await flush();
            const tx = db.transaction(STORE, 'readonly');
            const rows = await done(tx.objectStore(STORE).index(BY_SESSION).getAll(IDBKeyRange.only(session)));
            return rows.map((/** @type {{ record: import('./descriptors.js').DescriptorRecord }} */ row) => row.record);
        },
        async clear() {
            pending = [];
            await writing.catch(() => {});
            await deleteSession(db, session);
        },
        async close() {
            await flush();
            db.close();
        }
    };
}

/** @param {IDBDatabase} db @param {string} session */
async function deleteSession(db, session) {
    const tx = db.transaction(STORE, 'readwrite');
    const index = tx.objectStore(STORE).index(BY_SESSION);
    const keys = await done(index.getAllKeys(IDBKeyRange.only(session)));
    const store = tx.objectStore(STORE);
    for (const key of keys) store.delete(key);
    await committed(tx);
}

/**
 * Efface le journal d'une session, ou tout le journal sans session.
 * @param {{ session?: string, indexedDB?: IDBFactory }} [options]
 */
export async function clearJournal({ session, indexedDB = globalThis.indexedDB } = {}) {
    if (!indexedDB) return;
    // The store is emptied rather than the database deleted: a deletion waits, blocked, for
    // every open connection — the Worker's journal among them — to close.
    const db = await openDb(indexedDB);
    try {
        if (session === undefined) {
            const tx = db.transaction(STORE, 'readwrite');
            tx.objectStore(STORE).clear();
            await committed(tx);
        } else {
            await deleteSession(db, session);
        }
    } finally {
        db.close();
    }
}

/**
 * Journal en mémoire, même interface : là où IndexedDB manque, rien n'est gardé au-delà de la
 * page.
 * @param {string} session
 */
export function createMemoryJournal(session) {
    /** @type {import('./descriptors.js').DescriptorRecord[]} */
    let records = [];
    return {
        session,
        /** @param {import('./descriptors.js').DescriptorRecord} record */
        append(record) {
            records.push(record);
        },
        flush: () => Promise.resolve(),
        read: () => Promise.resolve(records.slice()),
        clear() {
            records = [];
            return Promise.resolve();
        },
        close: () => Promise.resolve()
    };
}
