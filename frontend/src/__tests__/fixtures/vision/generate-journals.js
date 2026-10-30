// Writes the automata journals of this directory from the scripted scenes:
//   node src/__tests__/fixtures/vision/generate-journals.js && npx prettier --write src/__tests__/fixtures/vision/
// A journal is the live pipeline's output on a scene; `expected` is what the automata made of
// it when the thresholds were set. Regenerate after changing the describer, never to make a
// threshold change pass: `vision.automata.replay.test.js` checks the files match the scenes.

import { writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { simulate } from './board.js';
import { SCENARIOS } from './scenarios.js';

const dir = dirname(fileURLToPath(import.meta.url));

/** @param {import('../../../vision/contract.js').VisionEvent} e */
const plain = (e) => ({ type: e.type, mediaTime: e.mediaTime, state: e.state, payload: { ...e.payload } });

for (const [name, sc] of Object.entries(SCENARIOS)) {
    const { records, events, calibration } = simulate(sc.script, sc.duration);
    const file = join(dir, `${name}.journal.json`);
    const body = { description: sc.description, calibration, truth: sc.truth, expected: events.map(plain), records };
    writeFileSync(file, JSON.stringify(body) + '\n');
    process.stdout.write(`${file}: ${records.length} records, ${events.length} events\n`);
}
