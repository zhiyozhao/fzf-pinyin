#include "fzfp.h"
#include <stdio.h>
#include <string.h>

int main(void) {
    const char *request = "{\"operation\":\"match\",\"text\":\"中国\",\"query\":\"zg\"}";
    size_t count = 0;
    int32_t status = 0;
    if (fzfp_abi_version() != 1) return 3;
    char *output = fzfp_call(request, strlen(request), &count, &status);
    if (!output) return 2;
    fwrite(output, 1, count, stdout);
    fzfp_free(output);
    return status;
}
