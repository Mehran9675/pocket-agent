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

pa_model *pa_load(const char *path, int32_t n_ctx, int32_t n_threads, int32_t n_gpu_layers,
	volatile int32_t *abort, char *err, int32_t errlen) {
	static bool started;
	if (!started) {
		llama_log_set(pa_quiet, NULL);
		llama_backend_init();
		started = true;
	}

	struct llama_model_params mp = llama_model_default_params();
	mp.n_gpu_layers = n_gpu_layers;
	struct llama_model *model = llama_model_load_from_file(path, mp);
	if (!model) {
		pa_err(err, errlen, "couldn't load the model - the file may be damaged or use an architecture this engine doesn't support");
		return NULL;
	}

	int32_t train = llama_model_n_ctx_train(model);
	if (n_ctx <= 0) {
		n_ctx = 4096;
	}
	if (train > 0 && n_ctx > train) {
		n_ctx = train;
	}
	struct llama_context_params cp = llama_context_default_params();
	cp.n_ctx = (uint32_t)n_ctx;
	cp.n_batch = 512;
	cp.n_ubatch = 512;
	cp.n_threads = n_threads;
	cp.n_threads_batch = n_threads;
	cp.no_perf = true;
	cp.abort_callback = pa_abort_cb;
	cp.abort_callback_data = (void *)abort;
	struct llama_context *ctx = llama_init_from_model(model, cp);
	if (!ctx) {
		llama_model_free(model);
		pa_err(err, errlen, "not enough memory for this model's context - pick a smaller model or context size");
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

static struct llama_sampler *sampler(pa_model *m, const pa_params *p) {
	struct llama_sampler *chain = llama_sampler_chain_init(llama_sampler_chain_default_params());
	if (p->repeat_penalty > 0 && p->repeat_penalty != 1) {
		llama_sampler_chain_add(chain, llama_sampler_init_penalties(llama_vocab_n_tokens(m->vocab), 64, p->repeat_penalty, 0, 0));
	}
	if (p->temperature <= 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_greedy());
		return chain;
	}
	if (p->top_k > 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_top_k(p->top_k));
	}
	if (p->top_p > 0 && p->top_p < 1) {
		llama_sampler_chain_add(chain, llama_sampler_init_top_p(p->top_p, 1));
	}
	if (p->min_p > 0) {
		llama_sampler_chain_add(chain, llama_sampler_init_min_p(p->min_p, 1));
	}
	llama_sampler_chain_add(chain, llama_sampler_init_temp(p->temperature));
	llama_sampler_chain_add(chain, llama_sampler_init_dist(LLAMA_DEFAULT_SEED));
	return chain;
}

int32_t pa_generate(pa_model *m, const char *prompt, int32_t len, const pa_params *p,
	pa_piece_fn cb, void *user, pa_stats *st, char *err, int32_t errlen) {
	memset(st, 0, sizeof *st);
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
