'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { Worker } = require('node:worker_threads');
const { spawnSync } = require('node:child_process');
const { Index, match, filter, version, _json } = require('./');
const fixture = JSON.parse(fs.readFileSync(process.env.FZFP_GOLDEN || path.join(__dirname, '../testdata/golden.json')));
test('exact matches CLI quote syntax and preserves ascending Unicode alignment', {
  skip: !process.env.FZFP_CLI && 'set FZFP_CLI for exact CLI differential'
}, () => {
  const { candidates: texts, queries } = JSON.parse(fs.readFileSync(path.join(__dirname, '../testdata/exact.json')));
  for (const scheme of ['default', 'path', 'history']) for (const query of queries) {
    const cli = spawnSync(process.env.FZFP_CLI, ['-i', '--read0', '--print0', '--scheme='+scheme, "--filter='"+query], {
      input: Buffer.from(texts.join('\0')+'\0'), env: {...process.env, FZF_DEFAULT_OPTS: '', FZF_DEFAULT_OPTS_FILE: '/dev/null'}
    });
    assert.ok(cli.status === 0 || cli.status === 1, String(cli.stderr));
    const expected = filter(texts, query, { scheme, exact: true });
    assert.deepEqual(Buffer.from(expected.map(h => h.text+'\0').join('')), cli.stdout);
    for (const workers of [1, 4]) {
      const index = new Index(texts, { scheme, workers });
      try {
        for (const limit of [null, 0, 1, 50]) for (const positions of [false, true]) {
          const hits = index.find(query, { limit, positions, exact: true });
          const want = limit === null ? expected : expected.slice(0, limit);
          assert.deepEqual(hits.map(h => [h.index,h.score]), want.map(h => [h.index,h.score]));
          if (positions) assert.deepEqual(hits.map(h => h.alignment), want.map(({index,text,...m}) => m));
        }
      } finally { index.close(); }
    }
    texts.forEach((text, i) => {
      const m = match(text, query, { scheme, exact: true });
      const hit = expected.find(h => h.index === i);
      assert.equal(m.matched, !!hit);
      if (hit) {
        const { index, text: _, ...want } = hit;
        assert.deepEqual(m, want);
        assert.deepEqual(m.positions, [...m.positions].sort((a,b)=>a-b));
      }
    });
  }
  assert.deepEqual(match('😀ab', 'ab').positions, [1,2]);
  assert.deepEqual(match('😀ab', 'ab').utf16Positions, [2,3]);
  const x = new Index(['ab']);
  try { assert.throws(() => x.find('ab', { exact: 1 })); } finally { x.close(); }
});
test('live CLI tie-heavy oracle: byte order, scores, exact alignment, workers and top-K', {
  skip: !process.env.FZFP_CLI_ORACLE_DIR && 'scripts/test-bindings.sh generates the mandatory live CLI oracle'
}, () => {
  let count = 0;
  for (const scheme of ['default', 'path', 'history']) {
    const data = JSON.parse(fs.readFileSync(path.join(process.env.FZFP_CLI_ORACLE_DIR, scheme + '.json')));
    for (const { name, request: r, expected: e } of data.vectors) {
      count++;
      const expected = e.results || [];
      const options = { scheme, positions: true };
      const actual = filter(r.candidates, r.query, options);
      assert.deepEqual(actual.map(({ text, ...m }) => canonical(m)), expected, name);
      assert.deepEqual(Buffer.from(actual.map(h => h.text + '\0').join('')),
        Buffer.from(expected.map(h => r.candidates[h.index] + '\0').join('')), name);
      for (const workers of [1, 4]) {
        const index = new Index(r.candidates, { ...options, workers });
        try {
          for (const [limit, positions] of [[null, true], [50, true], [50, false]]) {
            const hits = index.find(r.query, { limit, positions });
            const want = limit === null ? expected : expected.slice(0, limit);
            assert.deepEqual(hits.map(h => [h.index, h.score]), want.map(h => [h.index, h.score]), name);
            if (positions) assert.deepEqual(hits.map(h => canonical(h.alignment)), want.map(({ index, ...m }) => m), name);
          }
        } finally { index.close(); }
      }
    }
  }
  assert.equal(count, 1260);
  console.log(`Verified ${count} live CLI tie-heavy datasets (JS)`);
});
function canonical(m) {
  const { utf8Positions, utf16Positions, ...rest } = m;
  return { ...rest, utf8_positions: utf8Positions, utf16_positions: utf16Positions };
}
test('shared golden, top-K, workers, positions, Unicode and long fallback', () => {
  assert.equal(version().core_revision, fixture.core_revision);
  for (const { name, request: r, expected: e } of fixture.vectors) {
    const options = { scheme: r.options.scheme, caseSensitive: r.options.case_sensitive,
      normalize: r.options.normalize ?? true, positions: r.options.positions ?? true };
    let want;
    if (r.operation === 'match') {
      assert.deepEqual(canonical(match(r.text, r.query, options)), e.match, name);
      want = e.match.matched ? [{ index: 0, ...e.match }] : [];
    } else {
      const actual = filter(r.candidates, r.query, options).map(({ text, ...hit }) => canonical(hit));
      assert.deepEqual(actual, e.results || [], name);
      want = e.results || [];
    }
    for (const workers of [1, 4]) {
      const index = new Index(r.operation === 'match' ? [r.text] : r.candidates, { ...options, workers });
      try {
        for (const limit of [null, 0, 1, 7, 50]) for (const positions of [false, true]) {
          const hits = index.find(r.query, { limit, positions });
          const expected = limit === null ? want : want.slice(0, limit);
          assert.deepEqual(hits.map(h => [h.index, h.score]), expected.map(h => [h.index, h.score]), name);
          if (positions && options.positions) assert.deepEqual(hits.map(h => canonical(h.alignment)), expected.map(({ index, ...m }) => m), name);
        }
      } finally { index.close(); index.close(); }
      assert.throws(() => index.find(r.query));
    }
  }
  console.log(`Verified ${fixture.vectors.length} shared golden vectors`);
});
test('validation and corpus ownership', () => {
  const texts = ['a\0😀中国'];
  const index = new Index(texts);
  texts[0] = 'changed';
  const hit = index.find('zg', { positions: true })[0];
  assert.deepEqual(hit.positions, [3, 4]);
  assert.deepEqual(hit.alignment.utf16Positions, [4, 5]);
  for (const limit of [-1, 0.5, NaN, Infinity, 2**54]) assert.throws(() => index.find('zg', { limit }));
  assert.throws(() => index.find('\ud800'));
  assert.throws(() => new Index(['\udc00']));
  assert.throws(() => new Index([], { scheme: 'bad' }));
  assert.throws(() => new Index([], { workers: 65 }));
  assert.deepEqual(new Index([]).find('x'), []);
  index.close();
  assert.throws(() => index.find('zg'));
});
test('mixed schemes across JS worker isolates', async () => {
  await Promise.all(['default', 'path', 'history'].map(scheme => new Promise((resolve, reject) => {
    const worker = new Worker(`const {parentPort,workerData}=require('node:worker_threads');
      const {Index}=require(workerData.module); const x=new Index(['/马超'],{scheme:workerData.scheme});
      for(let i=0;i<100;i++){if(x.find('machao')[0].score !== (workerData.scheme==='history'?151:158)) throw Error('scheme');}
      x.close(); parentPort.postMessage('ok');`, { eval: true, workerData: { module: __dirname, scheme } });
    worker.on('error', reject); worker.on('exit', code => code ? reject(Error('worker failed')) : resolve());
  })));
});
