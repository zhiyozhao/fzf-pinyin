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
char *fzfp_call(const char *request, size_t length, size_t *output_length, int32_t *status);
void fzfp_free(void *buffer);
#ifdef __cplusplus
}
#endif
#endif
