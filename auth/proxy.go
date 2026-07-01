//spellchecker:words auth
package auth

//spellchecker:words bytes encoding json slog http github quickpid pkglib errorsx lazy
import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/tkw1536/quickpid/api"
	"go.tkw01536.de/pkglib/errorsx"
	"go.tkw01536.de/pkglib/lazy"
)

// Proxy performs authorization and conditionally forwards requests to the backend.
type Proxy struct {
	Forward func(http.ResponseWriter, *http.Request) // Forward forwards a request to the backend.
	Auth    Authorization                            // Auth implements the authorization logic.

	Logger *slog.Logger

	mux lazy.Lazy[*http.ServeMux]
}

func (ap *Proxy) initMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("GET /resolver", ap.forward())
	mux.Handle("GET /resolver/namespaces", ap.forward())
	mux.Handle("GET /resolver/resources/count", ap.forward())
	mux.Handle("POST /resolver/namespaces", ap.forwardCreateNamespace())

	mux.Handle("GET /resolver/namespaces/{namespace}", ap.forwardMinLevel(LevelContributor))
	mux.Handle("GET /resolver/namespaces/{namespace}/resources", ap.forwardMinLevel(LevelEditor))
	mux.Handle("POST /resolver/namespaces/{namespace}/resources", ap.forwardMinLevel(LevelContributor))
	mux.Handle("POST /resolver/namespaces/{namespace}/resources:batch", ap.forwardMinLevel(LevelContributor))
	mux.Handle("GET /resolver/namespaces/{namespace}/resources/{pid}", ap.forwardMinLevel(LevelNone))
	mux.Handle("PATCH /resolver/namespaces/{namespace}/resources/{pid}", ap.forwardMinLevel(LevelEditor))

	mux.Handle("/", ap.forward())

	return mux
}

func (ap *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ap.mux.Get(ap.initMux).ServeHTTP(w, r)
}

func (ap *Proxy) forward() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ap.writeForwardResponse(w, r, nil)
	}
}

func (ap *Proxy) forwardMinLevel(min Level) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace := r.PathValue("namespace")
		level, err := ap.level(r, namespace)
		if err != nil {
			if min.Int() > LevelNone.Int() {
				ap.writeUnauthorized(w, r, err)
				return
			}
			level = LevelNone
		}
		if level.Int() < min.Int() {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		ap.writeForwardResponse(w, r, nil)
	}
}

func (ap *Proxy) forwardCreateNamespace() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ap.Auth != nil {
			ok, err := ap.Auth.CanCreate(ap.Logger, r)
			if err != nil {
				ap.writeUnauthorized(w, r, err)
				return
			}
			if !ok {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		ap.writeForwardResponse(w, r, func(code int, body []byte) {
			if code != http.StatusCreated {
				return
			}
			var created api.NamespaceResponse
			if err := json.Unmarshal(body, &created); err != nil {
				ap.logError("failed to parse created namespace response", err)
				return
			}
			if created.ID == "" {
				return
			}
			if err := ap.Auth.OnCreateNamespace(ap.Logger, r, created.ID); err != nil {
				ap.logError("failed to record namespace creation permissions", err)
			}
		})
	}
}

func (ap *Proxy) level(r *http.Request, namespace string) (Level, error) {
	if ap.Auth == nil {
		return LevelEditor, nil
	}
	return ap.Auth.Level(ap.Logger, r, namespace)
}

// writeForwardResponse forwards the request to the backend and calls the capture function with the response code and body.
// If capture is nil, no capture is performed.
func (ap *Proxy) writeForwardResponse(w http.ResponseWriter, r *http.Request, capture func(code int, body []byte)) {
	if ap.Forward == nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	// if we don't have an afterRead function, we can just forward immediately.
	if capture == nil {
		ap.Forward(w, r)
		return
	}

	var buffer bytes.Buffer
	rc := &responseCapture{ResponseWriter: w, statusCode: http.StatusOK, capture: &buffer}
	ap.Forward(rc, r)

	capture(rc.statusCode, buffer.Bytes())
}

func (ap *Proxy) writeUnauthorized(w http.ResponseWriter, r *http.Request, err error) {
	ap.logError(http.StatusText(http.StatusUnauthorized), err)
	if ap.Auth != nil {
		ap.Auth.OnUnauthorized(ap.Logger, w, r)
		return
	}
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func (ap *Proxy) logError(msg string, err error) {
	if ap.Logger == nil {
		return
	}
	ap.Logger.Error(msg, slog.Any("error", err))
}

type responseCapture struct {
	http.ResponseWriter
	statusCode int
	capture    io.Writer
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	_, errCapture := rc.capture.Write(b)
	n, err := rc.ResponseWriter.Write(b) // pass through to real client
	return n, errorsx.Combine(errCapture, err)
}
