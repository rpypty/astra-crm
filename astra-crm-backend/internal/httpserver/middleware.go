package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type requestLogMetadata struct {
	userID  int64
	teamID  int64
	hasUser bool
}

type requestLogMetadataKey struct{}

func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w}
			metadata := &requestLogMetadata{}
			r = r.WithContext(context.WithValue(r.Context(), requestLogMetadataKey{}, metadata))

			next.ServeHTTP(recorder, r)

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []slog.Attr{
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int("bytes", recorder.bytes),
				slog.Duration("latency", time.Since(startedAt)),
			}
			if metadata.hasUser {
				attrs = append(attrs, slog.Int64("user_id", metadata.userID), slog.Int64("team_id", metadata.teamID))
			}

			log.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
		})
	}
}

func SetRequestLogUser(ctx context.Context, userID int64, teamID int64) {
	metadata, ok := ctx.Value(requestLogMetadataKey{}).(*requestLogMetadata)
	if !ok || metadata == nil {
		return
	}
	metadata.userID = userID
	metadata.teamID = teamID
	metadata.hasUser = true
}

func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Error(
						"http handler panic",
						slog.String("request_id", middleware.GetReqID(r.Context())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("panic", "[REDACTED]"),
						slog.String("panic_type", fmt.Sprintf("%T", recovered)),
					)
					RespondError(w, InternalError())
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func CSRFOriginGuard(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, origin := range allowedOrigins {
		if normalized := normalizeOrigin(origin); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isUnsafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			origin := normalizeOrigin(r.Header.Get("Origin"))
			if origin == "" || sameOrigin(r, origin) {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := allowed[origin]; ok {
				next.ServeHTTP(w, r)
				return
			}

			RespondError(w, ForbiddenError())
		})
	}
}

type LoginRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	attempts map[string][]time.Time
	now      func() time.Time
}

func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	return &LoginRateLimiter{
		limit:    limit,
		window:   window,
		attempts: map[string][]time.Time{},
		now:      time.Now,
	}
}

func (l *LoginRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(loginRateLimitKey(r)) {
			RespondError(w, RateLimitedError())
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *LoginRateLimiter) allow(key string) bool {
	now := l.now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	values := l.attempts[key]
	kept := values[:0]
	for _, value := range values {
		if value.After(cutoff) {
			kept = append(kept, value)
		}
	}
	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}
	kept = append(kept, now)
	l.attempts[key] = kept
	return true
}

func loginRateLimitKey(r *http.Request) string {
	if addr := requestIP(r); addr != nil {
		return addr.String()
	}
	return r.RemoteAddr
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func normalizeOrigin(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
}

func sameOrigin(r *http.Request, origin string) bool {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = strings.ToLower(forwardedProto)
	}
	return origin == strings.ToLower(scheme+"://"+r.Host)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}

	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}

	written, err := r.ResponseWriter.Write(data)
	r.bytes += written
	return written, err
}
