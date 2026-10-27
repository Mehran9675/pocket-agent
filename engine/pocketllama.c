#include "pocketllama.h"

#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "llama.h"

#ifndef PA_ENGINE_VERSION
#define PA_ENGINE_VERSION "dev"
#endif

struct pa_model {
	struct llama_model *model;
	struct llama_context *ctx;
	const struct llama_vocab *vocab;
	int32_t n_ctx;
	volatile int32_t *abort;
	/* cached are the tokens in the KV cache, so a follow-up turn only
	 * decodes what's new; it holds up to n_ctx tokens. */
	llama_token *cached;
	int32_t n_cached;
};

static bool pa_abort_cb(void *data) { return *(volatile int32_t *)data != 0; }

static void pa_quiet(enum ggml_log_level level, const char *text, void *data) {
	(void)level;
	(void)text;
	(void)data;
}

static void pa_err(char *err, int32_t errlen, const char *msg) {
	if (err && errlen > 0) {
		snprintf(err, (size_t)errlen, "%s", msg);
	}
}

int32_t pa_abi_version(void) { return PA_ABI_VERSION; }

const char *pa_engine_version(void) { return PA_ENGINE_VERSION; }

/* load_params fills the fields a caller's older header doesn't know
 * with defaults. */
static pa_load_params load_params(const pa_load_params *in) {
	pa_load_params p = {
		.size = sizeof p, .n_threads = 4, .n_gpu_layers = -1, .n_batch = 512, .n_ubatch = 512,
		.flash_attn = -1, .type_k = PA_KV_F16, .type_v = PA_KV_F16, .load_mode = PA_LOAD_AUTO, .offload_kqv = 1,
	};
	if (in && in->size > 0) {
		memcpy(&p, in, (size_t)in->size < sizeof p ? (size_t)in->size : sizeof p);
	}
	p.size = sizeof p;
	return p;
}

static enum ggml_type kv_type(int32_t t) {
	switch (t) {
	case PA_KV_Q8_0:
		return GGML_TYPE_Q8_0;
	case PA_KV_Q4_0:
		return GGML_TYPE_Q4_0;
	case PA_KV_Q5_0:
		return GGML_TYPE_Q5_0;
	case PA_KV_BF16:
		return GGML_TYPE_BF16;
	}
	return GGML_TYPE_F16;
}

pa_model *pa_load(const char *path, const pa_load_params *in, volatile int32_t *abort, char *err, int32_t errlen) {
	static bool started;
	if (!started) {
		llama_log_set(pa_quiet, NULL);
		llama_backend_init();
		started = true;
	}
	pa_load_params p = load_params(in);
	bool quant_v = p.type_v != PA_KV_F16 && p.type_v != PA_KV_BF16;
	if (quant_v && p.flash_attn == 0) {
		pa_err(err, errlen, "a quantized V cache needs flash attention - set flash attention to Auto or On, or the V cache to F16");
		return NULL;
	}

	struct llama_model_params mp = llama_model_default_params();
	mp.n_gpu_layers = p.n_gpu_layers;
	mp.load_mode = (enum llama_load_mode)p.load_mode;
	struct llama_model *model = llama_model_load_from_file(path, mp);
	if (!model) {
		pa_err(err, errlen, "couldn't load the model - the file may be damaged, use an architecture this engine doesn't support, or not fit in memory");
		return NULL;
	}

	int32_t n_ctx = p.n_ctx;
	int32_t train = llama_model_n_ctx_train(model);
	if (n_ctx == 0) {
		n_ctx = 4096;
	}
	if (n_ctx < 0 || (train > 0 && n_ctx > train)) {
		n_ctx = train > 0 ? train : 4096;
	}
	struct llama_context_params cp = llama_context_default_params();
	cp.n_ctx = (uint32_t)n_ctx;
	cp.n_batch = (uint32_t)(p.n_batch > 0 ? p.n_batch : 512);
	cp.n_ubatch = (uint32_t)(p.n_ubatch > 0 ? p.n_ubatch : 512);
	if (cp.n_ubatch > cp.n_batch) {
		cp.n_ubatch = cp.n_batch;
	}
	cp.n_threads = p.n_threads > 0 ? p.n_threads : 4;
	cp.n_threads_batch = p.n_threads_batch > 0 ? p.n_threads_batch : cp.n_threads;
	cp.flash_attn_type = p.flash_attn < 0 ? LLAMA_FLASH_ATTN_TYPE_AUTO : p.flash_attn ? LLAMA_FLASH_ATTN_TYPE_ENABLED : LLAMA_FLASH_ATTN_TYPE_DISABLED;
	cp.type_k = kv_type(p.type_k);
	cp.type_v = kv_type(p.type_v);
	cp.offload_kqv = p.offload_kqv != 0;
	cp.no_perf = true;
	cp.abort_callback = pa_abort_cb;
	cp.abort_callback_data = (void *)abort;
	struct llama_context *ctx = llama_init_from_model(model, cp);
	if (!ctx) {
		llama_model_free(model);
		if (quant_v || p.type_k != PA_KV_F16) {
			pa_err(err, errlen, "couldn't create the context - the KV cache type may not work with this model or device; try F16, or a smaller context");
		} else {
			pa_err(err, errlen, "not enough memory for this model's context - pick a smaller model or context size");
		}
		return NULL;
	}

	pa_model *m = calloc(1, sizeof *m);
	m->model = model;
	m->ctx = ctx;
	m->vocab = llama_model_get_vocab(model);
	m->n_ctx = (int32_t)llama_n_ctx(ctx);
	m->abort = abort;
	m->cached = malloc(sizeof(llama_token) * (size_t)m->n_ctx);
	return m;
}

void pa_free(pa_model *m) {
	if (!m) {
		return;
	}
	llama_free(m->ctx);
	llama_model_free(m->model);
	free(m->cached);
	free(m);
}

int32_t pa_n_ctx(pa_model *m) { return m ? m->n_ctx : 0; }

int32_t pa_apply_template(pa_model *m, const pa_message *msgs, int32_t n, int32_t add_assistant,
	char *buf, int32_t len) {
	if (!m || n <= 0) {
		return -1;
	}
	const char *tmpl = llama_model_chat_template(m->model, NULL);
	struct llama_chat_message *chat = malloc(sizeof *chat * (size_t)n);
	for (int32_t i = 0; i < n; i++) {
		chat[i].role = msgs[i].role;
		chat[i].content = msgs[i].content;
	}
	int32_t got = llama_chat_apply_template(tmpl, chat, (size_t)n, add_assistant != 0, buf, len);
	free(chat);
	return got;
}

/* tokenize returns a malloc'd token array, NULL on failure. */
static llama_token *tokenize(pa_model *m, const char *s, int32_t len, bool add_special, int32_t *n_out) {
	int32_t cap = len + 16;
	llama_token *t = malloc(sizeof *t * (size_t)cap);
	int32_t n = llama_tokenize(m->vocab, s, len, t, cap, add_special, true);
	if (n < 0) {
		cap = -n;
		t = realloc(t, sizeof *t * (size_t)cap);
		n = llama_tokenize(m->vocab, s, len, t, cap, add_special, true);
	}
	if (n < 0) {
		free(t);
		return NULL;
	}
	*n_out = n;
	return t;
}

int32_t pa_count_tokens(pa_model *m, const char *text, int32_t len) {
	if (!m) {
		return len / 4;
	}
	int32_t n = 0;
	llama_token *t = tokenize(m, text, len, false, &n);
	if (!t) {
		return len / 4;
	}
	free(t);
	return n;
}

static int32_t decode(pa_model *m, llama_token *toks, int32_t n, char *err, int32_t errlen) {
	for (int32_t i = 0; i < n; i += 512) {
		int32_t k = n - i < 512 ? n - i : 512;
		int32_t rc = llama_decode(m->ctx, llama_batch_get_one(toks + i, k));
		if (rc == 2) {
			return PA_CANCELED;
		}
		if (rc != 0) {
			char msg[64];
			snprintf(msg, sizeof msg, "decode failed (%d)", rc);
			pa_err(err, errlen, msg);
			return PA_ERROR;
		}
	}
	return PA_OK;
}

/* gen_params fills the fields a caller's older header doesn't know
 * with defaults. */
static pa_params gen_params(const pa_params *in) {
	pa_params p = {
		.size = sizeof p, .temperature = 0.7f, .top_k = 40, .top_p = 0.95f, .min_p = 0.05f, .typical_p = 1,
		.repeat_penalty = 1.1f, .repeat_last_n = 64, .dry_base = 1.75f, .dry_allowed_length = 2, .dry_last_n = -1,
		.xtc_threshold = 0.1f, .mirostat_tau = 5, .mirostat_eta = 0.1f, .max_tokens = 2048,
	};
	if (in && in->size > 0) {
		memcpy(&p, in, (size_t)in->size < sizeof p ? (size_t)in->size : sizeof p);
	}
	p.size = sizeof p;
	return p;
}

/* sampler builds llama.cpp's usual chain: penalties, DRY, top-n-sigma,
 * top-k, typical, top-p, min-p, XTC, temperature, then a draw - or
 * Mirostat in place of the truncation steps. */
static struct llama_sampler *sampler(pa_model *m, const pa_params *p) {
	uint32_t seed = p->seed ? p->seed : LLAMA_DEFAULT_SEED;
	struct llama_sampler *chain = llama_sampler_chain_init(llama_sampler_chain_default_params());
	int32_t last_n = p->repeat_last_n < 0 ? m->n_ctx : p->repeat_last_n;
	if (last_n > 0 && (p->repeat_penalty != 1 || p->frequency_penalty != 0 || p->presence_penalty != 0)) {
		llama_sampler_chain_add(chain, llama_sampler_init_penalties(llama_vocab_n_tokens(m->vocab), last_n,
			p->repeat_penalty > 0 ? p->repeat_penalty : 1, p->frequency_penalty, p->presence_penalty));
	}
	if (p->dry_multiplier > 0) {
		static const char *breakers[] = {"\n", ":", "\"", "*"};
		int32_t dry_n = p->dry_last_n < 0 ? m->n_ctx : p->dry_last_n;
		llama_sampler_chain_add(chain, llama_sampler_init_dry(m->vocab, p->dry_multiplier, p->dry_base,
			p->dry_allowed_length, dry_n, breakers, sizeof breakers / sizeof breakers[0]));
	}
	if (p->temperature <= 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_greedy());
		return chain;
	}
	if (p->mirostat == 2) {
		llama_sampler_chain_add(chain, llama_sampler_init_temp(p->temperature));
		llama_sampler_chain_add(chain, llama_sampler_init_mirostat_v2(seed, p->mirostat_tau, p->mirostat_eta));
		return chain;
	}
	if (p->top_n_sigma > 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_top_n_sigma(p->top_n_sigma));
	}
	if (p->top_k > 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_top_k(p->top_k));
	}
	if (p->typical_p > 0 && p->typical_p < 1) {
		llama_sampler_chain_add(chain, llama_sampler_init_typical(p->typical_p, 1));
	}
	if (p->top_p > 0 && p->top_p < 1) {
		llama_sampler_chain_add(chain, llama_sampler_init_top_p(p->top_p, 1));
	}
	if (p->min_p > 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_min_p(p->min_p, 1));
	}
	if (p->xtc_probability > 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_xtc(p->xtc_probability, p->xtc_threshold, 1, seed));
	}
	llama_sampler_chain_add(chain, llama_sampler_init_temp(p->temperature));
	llama_sampler_chain_add(chain, llama_sampler_init_dist(seed));
	return chain;
}

int32_t pa_generate(pa_model *m, const char *prompt, int32_t len, const pa_params *in,
	pa_piece_fn cb, void *user, pa_stats *st, char *err, int32_t errlen) {
	memset(st, 0, sizeof *st);
	pa_params params = gen_params(in);
	const pa_params *p = &params;
	if (!m) {
		pa_err(err, errlen, "no model loaded");
		return PA_ERROR;
	}
	int32_t n = 0;
	llama_token *toks = tokenize(m, prompt, len, true, &n);
	if (!toks || n == 0) {
		free(toks);
		pa_err(err, errlen, "empty prompt");
		return PA_ERROR;
	}
	st->prompt_tokens = n;
	if (n >= m->n_ctx - 8) {
		free(toks);
		return PA_TOO_LONG;
	}

	/* Reuse the longest cached prefix; at least one token must be
	 * decoded to get logits. */
	int32_t keep = 0;
	while (keep < m->n_cached && keep < n && m->cached[keep] == toks[keep]) {
		keep++;
	}
	if (keep == n) {
		keep--;
	}
	llama_memory_t mem = llama_get_memory(m->ctx);
	if (!llama_memory_seq_rm(mem, 0, keep, -1)) {
		llama_memory_clear(mem, true);
		keep = 0;
	}
	m->n_cached = keep;
	st->reused_tokens = keep;
	int32_t rc = decode(m, toks + keep, n - keep, err, errlen);
	if (rc != PA_OK) {
		free(toks);
		m->n_cached = 0;
		llama_memory_clear(mem, true);
		return rc;
	}
	memcpy(m->cached + keep, toks + keep, sizeof *toks * (size_t)(n - keep));
	m->n_cached = n;
	free(toks);

	struct llama_sampler *smpl = sampler(m, p);
	int32_t max_tokens = p->max_tokens > 0 ? p->max_tokens : 2048;
	int64_t start = ggml_time_us();
	int32_t result = PA_OK;
	while (st->generated_tokens < max_tokens) {
		if (*m->abort) {
			result = PA_CANCELED;
			break;
		}
		if (m->n_cached >= m->n_ctx - 1) {
			break;
		}
		llama_token tok = llama_sampler_sample(smpl, m->ctx, -1);
		if (llama_vocab_is_eog(m->vocab, tok)) {
			break;
		}
		st->generated_tokens++;
		char small[256];
		char *piece = small, *big = NULL;
		int32_t k = llama_token_to_piece(m->vocab, tok, small, sizeof small, 0, false);
		if (k < 0) {
			big = malloc((size_t)-k);
			k = llama_token_to_piece(m->vocab, tok, big, -k, 0, false);
			piece = big;
		}
		int32_t more = k > 0 ? cb(user, piece, k) : 1;
		free(big);
		if (!more) {
			break;
		}
		rc = llama_decode(m->ctx, llama_batch_get_one(&tok, 1));
		if (rc == 2) {
			result = PA_CANCELED;
			break;
		}
		if (rc != 0) {
			char msg[64];
			snprintf(msg, sizeof msg, "decode failed (%d)", rc);
			pa_err(err, errlen, msg);
			result = PA_ERROR;
			break;
		}
		m->cached[m->n_cached++] = tok;
	}
	st->seconds = (double)(ggml_time_us() - start) / 1e6;
	llama_sampler_free(smpl);
	return result;
}
