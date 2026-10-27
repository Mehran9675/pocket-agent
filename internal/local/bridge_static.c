//go:build cgo && ios && llama

// iOS apps can't load downloaded code, so the engine is compiled in.
#include "bridge.h"
#include "../../engine/pocketllama.c"

pa_api_t pa_api;

int pa_bind(const char *path, char *err, int32_t errlen) {
	(void)path;
	(void)err;
	(void)errlen;
	pa_api.abi_version = pa_abi_version;
	pa_api.engine_version = pa_engine_version;
	pa_api.load = pa_load;
	pa_api.free = pa_free;
	pa_api.n_ctx = pa_n_ctx;
	pa_api.apply_template = pa_apply_template;
	pa_api.count_tokens = pa_count_tokens;
	pa_api.generate = pa_generate;
	return 0;
}
