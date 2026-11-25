'use strict';
// Stages distributable packages locally; npm pack never publishes or installs.
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const root = path.resolve(__dirname, '..');
const out = path.join(root, '.cache/bindings/npm');
const binding = path.join(root, 'bindings/javascript');
for (const kind of ['main', 'native']) {
  const dir = path.join(out, kind); fs.mkdirSync(dir, { recursive: true });
  for (const name of ['LICENSE', 'NOTICE', 'THIRD_PARTY_NOTICES.md']) fs.copyFileSync(path.join(root, name), path.join(dir, name));
  if (kind === 'main') for (const name of ['package.json', 'index.js', 'index.d.ts', 'README.md']) fs.copyFileSync(path.join(binding, name), path.join(dir, name));
  else {
    const manifest = { name: `@fzfp/native-${process.platform}-${process.arch}`, version: '0.1.0', main: 'fzfp.node',
      os: [process.platform], cpu: [process.arch], license: 'MIT', files: ['fzfp.node', '*LICENSE', 'NOTICE', 'THIRD_PARTY_NOTICES.md'] };
    if (process.platform === 'linux') manifest.libc = ['glibc'];
    fs.writeFileSync(path.join(dir, 'package.json'), JSON.stringify(manifest, null, 2) + '\n');
    fs.copyFileSync(path.join(binding, '.cache/fzfp.node'), path.join(dir, 'fzfp.node'));
    const goRoot = execFileSync('go', ['env', 'GOROOT'], { encoding: 'utf8' }).trim();
    const license = [path.join(goRoot, 'LICENSE'), path.join(goRoot, '../LICENSE')].find(p => fs.existsSync(p));
    fs.copyFileSync(license, path.join(dir, 'Go-LICENSE'));
  }
  execFileSync('npm', ['pack', '--ignore-scripts', '--pack-destination', out], { cwd: dir, stdio: 'inherit' });
}
