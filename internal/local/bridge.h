// The engine's functions, bound at run time: from the downloaded
// library (bridge_dynamic.c) or, on iOS, linked in (bridge_static.c).
#include <stdint.h>
#include "pocketllama.h"

typedef struct {
	int32_t (*abi_version)(void);
	const char *(*engine_version)(void);
	pa_model *(*load)(const char *, const pa_load_params *, volatile int32_t *, char *, int32_t);
	void (*free)(pa_model *);
	int32_t (*n_ctx)(pa_model *);
	int32_t (*apply_template)(pa_model *, const pa_message *, int32_t, int32_t, char *, int32_t);
	int32_t (*count_tokens)(pa_model *, const char *, int32_t);
	int32_t (*generate)(pa_model *, const char *, int32_t, const pa_params *, pa_piece_fn, void *, pa_stats *, char *, int32_t);
} pa_api_t;

extern pa_api_t pa_api;

// pa_bind fills pa_api, loading the library at path when the engine
// isn't linked in. It returns 0, or -1 with a message in err.
int pa_bind(const char *path, char *err, int32_t errlen);
