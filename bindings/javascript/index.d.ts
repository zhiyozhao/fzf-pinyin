export type Scheme = 'default' | 'path' | 'history';
export interface Options { scheme?: Scheme; caseSensitive?: boolean; normalize?: boolean; positions?: boolean; workers?: number }
export interface Match {
  matched: boolean; score: number;
  /** Unicode scalar indexes, end exclusive; -1 for no match. */
  start: number; end: number; positions: number[]; utf8Positions: number[]; utf16Positions: number[];
}
export interface Result extends Match { index: number; text: string }
export interface Hit { index: number; score: number; alignment: Match | null; positions: number[] }
export class Index {
  constructor(candidates: readonly string[], options?: Options);
  readonly candidates: readonly string[];
  /** Native phase wall times in milliseconds; scan includes worker-local top-K. */
  readonly timings: Readonly<Record<string, number>>;
  find(query: string, options?: { limit?: number | null; positions?: boolean }): Hit[];
  close(): void;
}
export function match(text: string, query: string, options?: Options): Match;
export function filter(candidates: readonly string[], query: string, options?: Options): Result[];
export function version(): { version: string; core_revision: string; abi_version: number; typed_abi_version: number; dictionary_version: string };
