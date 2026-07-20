package querygo

import (
	"mime"
	"net/http"
	"strings"
)

// Handler serves the HTTP QUERY method for a resource. It dispatches QUERY
// requests to Query, answers OPTIONS with the advertised QUERY support, and
// rejects other methods with 405 Method Not Allowed. The zero value is not
// useful; set Query (see NewHandler).
//
// Handler implements http.Handler and can be mounted on any router, including
// the standard http.ServeMux (works on Go 1.21+, which lacks method-based
// route patterns).
type Handler struct {
	// Query handles QUERY requests. Required. The request body carries the
	// query to evaluate.
	Query http.HandlerFunc

	// AcceptQuery lists the query media types advertised via the Accept-Query
	// header field (on OPTIONS and 405 responses). When non-empty, QUERY
	// requests whose Content-Type is not in the list are rejected with 415
	// Unsupported Media Type.
	AcceptQuery []string

	// AllowedMethods lists additional methods (besides QUERY and OPTIONS) that
	// the resource supports, advertised via the Allow header field.
	AllowedMethods []string

	// OnOptions, when set, handles OPTIONS requests instead of the built-in
	// response.
	OnOptions http.HandlerFunc
}

// NewHandler returns a Handler for the given QUERY handler function,
// advertising the provided query media types via Accept-Query.
func NewHandler(query http.HandlerFunc, acceptQuery ...string) Handler {
	return Handler{Query: query, AcceptQuery: acceptQuery}
}

// ServeHTTP implements http.Handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case MethodQuery:
		if h.Query == nil {
			h.methodNotAllowed(w)
			return
		}

		if len(h.AcceptQuery) > 0 && !h.acceptsContentType(r) {
			h.advertise(w.Header())
			http.Error(w, "querygo: unsupported query media type", http.StatusUnsupportedMediaType)
			return
		}

		h.Query(w, r)

	case http.MethodOptions:
		if h.OnOptions != nil {
			h.OnOptions(w, r)
			return
		}

		h.advertise(w.Header())
		w.Header().Set(HeaderAllow, strings.Join(h.allowedMethods(), ", "))
		w.WriteHeader(http.StatusNoContent)

	default:
		h.methodNotAllowed(w)
	}
}

func (h Handler) acceptsContentType(r *http.Request) bool {
	value := r.Header.Get(HeaderContentType)
	if value == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}

	for _, allowed := range h.AcceptQuery {
		if strings.EqualFold(allowed, mediaType) {
			return true
		}
	}

	return false
}

func (h Handler) allowedMethods() []string {
	methods := []string{MethodQuery, http.MethodOptions}
	seen := map[string]bool{MethodQuery: true, http.MethodOptions: true}

	for _, method := range h.AllowedMethods {
		if !seen[method] {
			seen[method] = true
			methods = append(methods, method)
		}
	}

	return methods
}

func (h Handler) advertise(header http.Header) {
	if len(h.AcceptQuery) > 0 {
		header.Set(HeaderAcceptQuery, strings.Join(h.AcceptQuery, ", "))
	}
}

func (h Handler) methodNotAllowed(w http.ResponseWriter) {
	h.advertise(w.Header())
	w.Header().Set(HeaderAllow, strings.Join(h.allowedMethods(), ", "))
	http.Error(w, "querygo: method not allowed", http.StatusMethodNotAllowed)
}

// SetResultLocation sets the Content-Location header field on a response,
// identifying the GET-retrievable resource that holds the query result. This
// is the server-side counterpart to Client.FetchResult.
func SetResultLocation(header http.Header, uri string) {
	header.Set(HeaderContentLocation, uri)
}

// AdvertiseQuery sets the Accept-Query header field on a response, advertising
// the supported query media types. It can be called from any handler (for
// example a GET handler) so clients discover QUERY support inline.
func AdvertiseQuery(header http.Header, mediaTypes ...string) {
	if len(mediaTypes) > 0 {
		header.Set(HeaderAcceptQuery, strings.Join(mediaTypes, ", "))
	}
}
