#!/usr/bin/env node
// Rejoue un journal de descripteurs (bouton « Enregistrer le journal (banc) » du volet
// transcription) dans les automates, sans la vidéo, et écrit les événements au format de
// « Enregistrer les événements (banc) » : { events: [{ type, mediaTime, state }] }, prêt pour
// mesure.py.
//
//   node tools/vision/replay.mjs journal.json [--session S] [--overrides '{"seuil":1}'] > events.json
//
// Chaque session est rejouée dans son propre détecteur (les relevés de deux sessions ne
// se suivent pas) ; les événements de plusieurs sessions sont fusionnés par instant. Sans
// --session, toutes les sessions du fichier sont rejouées. Accepte aussi une fixture
// `{ records: [...] }` de frontend/src/__tests__/fixtures/vision.
import { readFileSync } from "node:fs";
import {
  createDetector,
  replay,
} from "../../frontend/src/vision/descriptors.js";

/**
 * @param {unknown} journal le fichier lu
 * @param {{ session?: string, overrides?: object }} [options]
 * @returns {{ type: string, mediaTime: number, state: string }[]}
 */
export function replayJournal(journal, { session, overrides } = {}) {
  const j = /** @type {any} */ (journal);
  const sessions = j && j.sessions ? j.sessions : { "": j && j.records };
  const names = session === undefined ? Object.keys(sessions) : [session];
  const events = [];
  for (const name of names) {
    if (!Array.isArray(sessions[name]))
      throw new Error(
        `session absente du journal : ${name} (présentes : ${Object.keys(sessions).join(", ")})`,
      );
    events.push(...replay(sessions[name], createDetector(overrides)));
  }
  // One session keeps the order the automata gave; several are merged by instant (Array#sort is stable).
  if (names.length > 1) events.sort((a, b) => a.mediaTime - b.mediaTime);
  return events.map((e) => ({
    type: e.type,
    mediaTime: e.mediaTime,
    state: e.state,
  }));
}

function main(argv) {
  const args = argv.slice(2);
  let file;
  let session;
  let overrides;
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--session") session = args[++i];
    else if (args[i] === "--overrides") overrides = JSON.parse(args[++i]);
    else if (file === undefined) file = args[i];
    else throw new Error(`argument inattendu : ${args[i]}`);
  }
  if (!file)
    throw new Error(
      "usage : node tools/vision/replay.mjs journal.json [--session S] [--overrides '{...}']",
    );
  const events = replayJournal(JSON.parse(readFileSync(file, "utf8")), {
    session,
    overrides,
  });
  process.stdout.write(JSON.stringify({ events }, null, 2) + "\n");
}

if (
  process.argv[1] &&
  import.meta.url === new URL(process.argv[1], "file://").href
) {
  try {
    main(process.argv);
  } catch (err) {
    process.stderr.write(`${err.message}\n`);
    process.exit(1);
  }
}
