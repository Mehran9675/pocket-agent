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

#define PA_ABI_VERSION 2

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

/* KV cache element types. Quantized V caches need flash attention. */
enum {
	PA_KV_F16 = 0,
	PA_KV_Q8_0 = 1,
	PA_KV_Q4_0 = 2,
	PA_KV_Q5_0 = 3,
	PA_KV_BF16 = 4,
};

/* How model weights are read. */
enum {
	PA_LOAD_AUTO = -1,
	PA_LOAD_READ = 0,  /* read into memory */
	PA_LOAD_MMAP = 1,  /* map the file; the OS pages it in */
	PA_LOAD_MLOCK = 2, /* read and keep resident */
	PA_LOAD_MMAP_MLOCK = 3,
};

/* Both structs start with their size, so an engine can tell which
 * fields a caller built against an older header knows. */
typedef struct {
	int32_t size;
	/* n_ctx 0 picks a default and a negative one the context the
	 * model was trained for. */
	int32_t n_ctx;
	int32_t n_threads;
	int32_t n_threads_batch; /* 0: n_threads */
	int32_t n_gpu_layers;    /* -1: all */
	int32_t n_batch;         /* 0: 512 */
	int32_t n_ubatch;        /* 0: 512 */
	int32_t flash_attn;      /* -1 auto, 0 off, 1 on */
	int32_t type_k;          /* PA_KV_* */
	int32_t type_v;
	int32_t load_mode;       /* PA_LOAD_* */
	int32_t offload_kqv;     /* keep the KV cache on the GPU */
} pa_load_params;

typedef struct {
	int32_t size;
	float temperature;       /* <= 0: greedy */
	int32_t top_k;           /* <= 0: off */
	float top_p;             /* >= 1: off */
	float min_p;             /* <= 0: off */
	float typical_p;         /* >= 1: off */
	float top_n_sigma;       /* <= 0: off */
	float repeat_penalty;    /* 1: off */
	float frequency_penalty; /* 0: off */
	float presence_penalty;  /* 0: off */
	int32_t repeat_last_n;   /* tokens the penalties look back over; -1: the context */
	float dry_multiplier;    /* 0: DRY off */
	float dry_base;
	int32_t dry_allowed_length;
	int32_t dry_last_n;      /* -1: the context */
	float xtc_probability;   /* 0: XTC off */
	float xtc_threshold;
	int32_t mirostat;        /* 0 off, 2 Mirostat 2.0 (replaces top-k/p and the rest) */
	float mirostat_tau;
	float mirostat_eta;
	uint32_t seed;           /* 0: random */
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

/* pa_load loads a GGUF model. abort is polled during decoding and
 * cancels it when non-zero. Returns NULL and writes a message to err on
 * failure. */
PA_API pa_model *pa_load(const char *path, const pa_load_params *p, volatile int32_t *abort, char *err, int32_t errlen);
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
