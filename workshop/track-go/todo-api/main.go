package main

import (
	"net/http"
	"os"
)

func main() {
	if err := waitForDependencies(); err != nil {
		log.Error("startup failed: " + err.Error())
		os.Exit(1)
	}

	port := envString("PORT", "8085")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	routes(mux)

	log.Info("todo-api listening on " + port)
	if err := http.ListenAndServe(":"+port, cors(mux)); err != nil {
		log.Error("server stopped: " + err.Error())
		os.Exit(1)
	}
}

// The frontend is served from a different origin, so allow any origin, method
// and header.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("access-control-allow-origin", "*")
		h.Set("access-control-allow-headers", "*")
		h.Set("access-control-allow-methods", "*")
		h.Set("access-control-max-age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handler is an http.HandlerFunc that can fail. A returned error is logged and
// answered with a 500, like an error handler in a web framework.
type handler func(w http.ResponseWriter, r *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		log.Error("request failed: " + err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
}

