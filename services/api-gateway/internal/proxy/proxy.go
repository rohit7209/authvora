package proxy

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// errorResponse is the JSON structure for error responses.
type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error"`
}

// NewServiceProxy creates a reverse proxy that forwards requests to the target URL.
// It preserves original headers, adds X-Forwarded-For, and returns 502 JSON error if backend is down.
func NewServiceProxy(targetURL string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		slog.Error("invalid target URL for proxy", "url", targetURL, "err", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSONError(w, "PROXY_ERROR", "Invalid proxy configuration", 500)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		if r.RemoteAddr != "" {
			if existing := r.Header.Get("X-Forwarded-For"); existing != "" {
				r.Header.Set("X-Forwarded-For", existing+", "+r.RemoteAddr)
			} else {
				r.Header.Set("X-Forwarded-For", r.RemoteAddr)
			}
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("proxy backend error", "target", targetURL, "path", r.URL.Path, "err", err)
		writeJSONError(w, "BACKEND_UNAVAILABLE", "Backend service is unavailable", 502)
	}

	return proxy
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Status  int    `json:"status"`
		}{
			Code:    code,
			Message: message,
			Status:  status,
		},
	})
}
