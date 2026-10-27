//go:build cgo && !(ios && llama)

#include <dlfcn.h>
#include <stdio.h>
#include <string.h>

#include "bridge.h"

pa_api_t pa_api;

int pa_bind(const char *path, char *err, int32_t errlen) {
	void *h = dlopen(path, RTLD_NOW | RTLD_LOCAL);
	if (!h) {
		snprintf(err, (size_t)errlen, "%s", dlerror());
		return -1;
	}
	pa_api_t api;
#define BIND(field, name)                                                  \
	*(void **)(&api.field) = dlsym(h, name);                           \
	if (!api.field) {                                                  \
		snprintf(err, (size_t)errlen, "the engine library lacks %s", name); \
		dlclose(h);                                                \
		return -1;                                                 \
	}
	BIND(abi_version, "pa_abi_version")
	BIND(engine_version, "pa_engine_version")
	BIND(load, "pa_load")
	BIND(free, "pa_free")
	BIND(n_ctx, "pa_n_ctx")
	BIND(apply_template, "pa_apply_template")
	BIND(count_tokens, "pa_count_tokens")
	BIND(generate, "pa_generate")
#undef BIND
	pa_api = api;
	return 0;
}
