package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pierreblais/gitaihub/api/internal/identity"
	"github.com/pierreblais/gitaihub/api/internal/review"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

// errorBody follows the Google API error model (AIP-193):
// {"error": {"code": 404, "message": "...", "status": "NOT_FOUND"}}.
type errorBody struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	code, httpStatus := classify(err)
	msg := err.Error()
	if st, ok := status.FromError(err); ok {
		msg = st.Message()
	}
	if code == codes.Internal || code == codes.Unknown {
		slog.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "error", err)
		msg = "internal error"
	}
	if code == codes.Unavailable {
		msg = "the gitai server is unavailable"
	}
	var body errorBody
	body.Error.Code = httpStatus
	body.Error.Message = msg
	body.Error.Status = statusName(code)
	writeJSON(w, httpStatus, body)
}

func classify(err error) (codes.Code, int) {
	switch {
	case errors.Is(err, source.ErrNotFound), errors.Is(err, identity.ErrUserNotFound):
		return codes.NotFound, http.StatusNotFound
	case errors.Is(err, source.ErrInvalidArgument):
		return codes.InvalidArgument, http.StatusBadRequest
	case errors.Is(err, identity.ErrInvalidCredentials), errors.Is(err, identity.ErrUnauthenticated):
		return codes.Unauthenticated, http.StatusUnauthorized
	case errors.Is(err, review.ErrNoGitaiIdentity):
		return codes.PermissionDenied, http.StatusForbidden
	}
	if st, ok := status.FromError(err); ok {
		return st.Code(), httpFromCode(st.Code())
	}
	return codes.Internal, http.StatusInternalServerError
}

func httpFromCode(c codes.Code) int {
	switch c {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Canceled:
		return 499
	}
	return http.StatusInternalServerError
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// statusName returns the canonical name of a code, e.g. NOT_FOUND.
func statusName(c codes.Code) string {
	var b strings.Builder
	for i, r := range c.String() {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}
