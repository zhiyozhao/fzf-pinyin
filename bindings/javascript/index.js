'use strict';
const fs = require('node:fs');
const path = require('node:path');
const local = path.join(__dirname, '.cache', 'fzfp.node');
const native = fs.existsSync(local) ? require(local) : require(`@fzfp/native-${process.platform}-${process.arch}`);
const finalizer = new FinalizationRegistry(handle => native.destroy(handle));
const schemes = ['default', 'path', 'history'];

function validString(s) {
  if (typeof s !== 'string' || !s.isWellFormed()) throw new TypeError('expected a well-formed Unicode string');
  return s;
}
function alignment(text, row) {
  const [, score, start, end, positions] = row;
  const utf8Positions = [], utf16Positions = [];
  let i = 0, b = 0, u = 0, p = 0;
  if (positions.length) for (const c of text) {
    if (positions[p] === i) { utf8Positions.push(b); utf16Positions.push(u); p++; }
    b += Buffer.byteLength(c); u += c.length; i++;
  }
  return { matched: true, score, start, end, positions, utf8Positions, utf16Positions };
}
class Index {
  #handle;
  #candidates;
  #timings = {};
  constructor(candidates, options = {}) {
    const { scheme = 'default', caseSensitive = false, normalize = true, workers = 0 } = options;
    if (!schemes.includes(scheme) || !Number.isInteger(workers) || workers < 0 || workers > 64 ||
        typeof caseSensitive !== 'boolean' || typeof normalize !== 'boolean') throw new RangeError('invalid options');
    this.#candidates = Object.freeze(Array.from(candidates, validString));
    const chunks = this.#candidates.map(s => Buffer.from(s));
    const offsets = new BigUint64Array(chunks.length + 1);
    for (let i = 0; i < chunks.length; i++) offsets[i + 1] = offsets[i] + BigInt(chunks[i].length);
    this.#handle = native.create(Buffer.concat(chunks), Buffer.from(offsets.buffer), schemes.indexOf(scheme), +caseSensitive, +normalize, workers);
    finalizer.register(this, this.#handle, this);
  }
  get candidates() { return this.#candidates; }
  get timings() { return { ...this.#timings }; }
  _find(query, limit, positions, exactBounds = false, exact = false) {
    validString(query);
    if (limit != null && (!Number.isSafeInteger(limit) || limit < 0)) throw new RangeError('limit must be null or a nonnegative safe integer');
    if (typeof positions !== 'boolean' || typeof exact !== 'boolean') throw new TypeError('positions and exact must be boolean');
    const output = native.find(this.#handle, Buffer.from(query), limit ?? -1, +positions | (exactBounds ? 2 : 0) | (exact ? 4 : 0));
    this.#timings = output.timings;
    return output.rows;
  }
  find(query, { limit = null, positions = false, exact = false } = {}) {
    return this._find(query, limit, positions, false, exact).map(row => ({
      index: row[0], score: row[1], alignment: positions ? alignment(this.#candidates[row[0]], row) : null,
      positions: positions ? row[4] : []
    }));
  }
  close() {
    if (this.#handle) { native.destroy(this.#handle); this.#handle = 0n; finalizer.unregister(this); }
  }
}
function filter(candidates, query, options = {}) {
  const index = new Index(candidates, options);
  try { return index._find(query, null, options.positions ?? true, true, options.exact ?? false).map(row => ({
    index: row[0], text: index.candidates[row[0]], ...alignment(index.candidates[row[0]], row)
  })); } finally { index.close(); }
}
function match(text, query, options = {}) {
  const result = filter([text], query, options)[0];
  if (!result) return { matched: false, score: 0, start: -1, end: -1, positions: [], utf8Positions: [], utf16Positions: [] };
  const { index, text: _, ...m } = result;
  return m;
}
function jsonCall(request) {
  const response = JSON.parse(native.json(Buffer.from(JSON.stringify(request))));
  if (response.error) throw new Error(response.error);
  return response;
}
module.exports = { Index, match, filter, version: () => ({ ...jsonCall({ operation: 'version' }), typed_abi_version: 2 }), _json: jsonCall };
