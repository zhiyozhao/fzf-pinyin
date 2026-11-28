#ifndef FZFP_H
#define FZFP_H
#include <stddef.h>
#include <stdint.h>
#ifdef __cplusplus
extern "C" {
#endif
/* ABI 1: JSON UTF-8 requests/responses, explicit byte lengths (NUL allowed).
 * Returns C-owned output on success AND errors; release exactly once with
 * fzfp_free. status: 0 success, 1 invalid request/options, 2 recovered panic.
 * NULL result means invalid out params or allocation failure. Thread safe;
 * requests are serialized. No Go pointers or retained caller buffers. */
uint32_t fzfp_abi_version(void);
#ifndef FZFP_CGO
char *fzfp_call(const char *request, size_t length, size_t *output_length, int32_t *status);
#endif
void fzfp_free(void *buffer);
/* ABI 2 is additive: ABI 1 functions/version remain available unchanged.
 * All integers are fixed width; natural C alignment (never packed).
 * Each input struct must have abi_version=2, struct_size=sizeof(struct),
 * reserved=0. Scheme: 0 default, 1 path, 2 history. workers: 0 auto, 1..64.
 * Inputs are borrowed for the call only; create copies the UTF-8 corpus.
 * offsets has count+1 entries, starts at 0 and ends at byte_length.
 * Limits: corpus/query <=256 MiB, count <=10M, valid UTF-8 including NUL.
 * Status: 0 success, 1 invalid argument/UTF-8, 2 internal panic, 3 invalid
 * handle (including double destroy), 4 allocation failure.
 * Thread safe; in-flight queries retain the index across destroy. Same index
 * scratch is serialized. Different schemes/requests share the scheme mutex.
 * No Go pointers escape. Caller must supply valid pointer ranges.
 */
typedef struct {
    uint32_t abi_version, struct_size, scheme, case_sensitive;
    uint32_t normalize, workers, reserved0, reserved1;
} fzfp_options;
typedef struct {
    uint32_t abi_version, struct_size;
    int64_t limit; /* -1 unlimited; 0 returns empty; less than -1 invalid */
    uint32_t flags, reserved; /* bit 0 positions, bit 1 exact bounds, bit 2 contiguous matching */
} fzfp_find_options;
typedef struct {
    uint64_t index;
    int64_t score, start, end; /* rune offsets; end exclusive */
    uint64_t position_offset, position_count;
} fzfp_hit;
typedef struct {
    uint32_t abi_version, struct_size;
    uint64_t count, position_count;
    fzfp_hit *hits;
    uint64_t *positions; /* canonical rune positions, sorted ascending */
    double prepare_ms, scan_ms, select_ms, trace_ms, copy_ms, native_ms;
} fzfp_results;
uint32_t fzfp_typed_abi_version(void);
int32_t fzfp_index_create(char *bytes, uint64_t byte_length, uint64_t *offsets,
                          uint64_t count, fzfp_options *options, uint64_t *handle);
int32_t fzfp_index_find(uint64_t handle, char *query, uint64_t query_length,
                        fzfp_find_options *options, fzfp_results **results);
int32_t fzfp_index_destroy(uint64_t handle);
/* Successful find always returns a C-owned result, including empty results.
 * Free exactly once; NULL is accepted. Rank-only start/end are unspecified
 * (-1 except path ranking); use flags to request exact bounds/alignment.
 * positions off does not change scores or ranking. */
void fzfp_results_free(fzfp_results *results);
#ifdef __cplusplus
}
#endif
#endif
