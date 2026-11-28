import { Index, match, filter, version, _json, Hit, Match, Options } from './index';
const options: Options = { scheme: 'path', workers: 4, positions: true };
const m: Match = match('中国', 'zg', options);
const results = filter(['北京'], 'bj', options);
const index = new Index(['北京'], options);
const hits: Hit[] = index.find('bj', { limit: 50, positions: true, exact: true });
match('北京', 'bj', { exact: true });
filter(['北京'], 'bj', { exact: true });
// @ts-expect-error exact is a boolean
index.find('bj', { exact: 1 });
hits[0]?.alignment?.utf16Positions.map(n => n + 1);
results[0]?.utf8Positions.map(n => n + 1);
const abi: number = version().typed_abi_version;
const response: Record<string, unknown> = _json({ operation: 'version' });
// @ts-expect-error invalid scheme
new Index([], { scheme: 'bad' });
// @ts-expect-error boolean is not a limit
index.find('bj', { limit: true });
// @ts-expect-error immutable candidate list
index.candidates.push('x');
index.close();
void [m, abi, response];
