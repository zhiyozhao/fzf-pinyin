'use strict';
const fs = require('node:fs');
const { Index, filter, _json } = require('./');
const [corpus, query, scheme, count, mode, workers, pos] = process.argv.slice(2);
const candidates = fs.readFileSync(corpus, 'utf8').split('\n');
if (candidates.at(-1) === '') candidates.pop();
const positions = pos === 'true';
const options = { scheme, positions, workers: +workers };
let start = performance.now();
const index = ['json', 'convenience'].includes(mode) ? null : new Index(candidates, options);
const create_ms = performance.now() - start;
let hits;
function run() {
  start = performance.now();
  if (mode === 'json') hits = _json({ operation: 'filter', candidates, query, options: { scheme, positions } }).results || [];
  else if (mode === 'convenience') hits = filter(candidates, query, options);
  else hits = index.find(query, { limit: mode === 'top50' ? 50 : null, positions });
  return performance.now() - start;
}
const cold_ms = run();
const milliseconds = [], phases = [];
for (let i = 0; i < +count; i++) { milliseconds.push(run()); if (index) phases.push(index.timings); }
index?.close();
process.stdout.write(JSON.stringify({ milliseconds, phases, create_ms, cold_ms, indexes: hits.map(h => h.index), scores: hits.map(h => h.score) }));
