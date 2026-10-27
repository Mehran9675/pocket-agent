/*
 * Pocket Agent's on-device engine: a small, stable C interface over
 * llama.cpp. On Android and desktops it is built as a shared library
 * the app downloads and loads at run time (see internal/local); on iOS,
 * where apps can't load downloaded code, it is linked into the app.
 *
 * PA_ABI_VERSION changes whenever a declaration below does; the app
 * only loads an engine with the ABI it was built for.
 */
#ifndef POCKETLLAMA_H
#define POCKETLLAMA_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define PA_ABI_VERSION 1

#if defined(_WIN32)
#define PA_API __declspec(dllexport)
#else
#define PA_API __attribute__((visibility("default")))
#endif

typedef struct pa_model pa_model;

typedef struct {
	const char *role;
	const char *content;
} pa_message;

typedef struct {
	float temperature;
	float top_p;
	float min_p;
	float repeat_penalty; /* 1 disables it */
	int32_t top_k;
	int32_t max_tokens;
} pa_params;

typedef struct {
	int32_t prompt_tokens;
	int32_t generated_tokens;
	/* reused_tokens were served from the cached prompt prefix. */
	int32_t reused_tokens;
	/* seconds spent generating, excluding prompt processing. */
	double seconds;
} pa_stats;

/* pa_piece_fn receives each decoded piece (possibly an incomplete UTF-8
 * sequence) and returns 0 to stop generating. */
typedef int32_t (*pa_piece_fn)(void *user, const char *piece, int32_t len);

enum {
	PA_OK = 0,
	PA_CANCELED = 1,
	PA_ERROR = -1,
	/* the prompt doesn't fit the context; stats.prompt_tokens says how long it is */
	PA_TOO_LONG = -2,
};

PA_API int32_t pa_abi_version(void);
/* pa_engine_version identifies the build: the llama.cpp commit and this file's revision. */
PA_API const char *pa_engine_version(void);

/* pa_load loads a GGUF model. n_ctx <= 0 picks a default; abort is
 * polled during decoding and cancels it when non-zero. Returns NULL and
 * writes a message to err on failure. */
PA_API pa_model *pa_load(const char *path, int32_t n_ctx, int32_t n_threads, int32_t n_gpu_layers,
	volatile int32_t *abort, char *err, int32_t errlen);
PA_API void pa_free(pa_model *m);
PA_API int32_t pa_n_ctx(pa_model *m);

/* pa_apply_template formats msgs with the model's chat template into
 * buf, returning the formatted length (which may exceed len: call again
 * with a bigger buffer) or a negative value when the model has no
 * usable template. */
PA_API int32_t pa_apply_template(pa_model *m, const pa_message *msgs, int32_t n, int32_t add_assistant,
	char *buf, int32_t len);
PA_API int32_t pa_count_tokens(pa_model *m, const char *text, int32_t len);

/* pa_generate continues prompt, reusing the longest prefix already in
 * the KV cache, and calls cb with every piece until the model stops,
 * max_tokens is reached, cb returns 0 or abort is set. */
PA_API int32_t pa_generate(pa_model *m, const char *prompt, int32_t len, const pa_params *p,
	pa_piece_fn cb, void *user, pa_stats *st, char *err, int32_t errlen);

#ifdef __cplusplus
}
#endif

#endif
