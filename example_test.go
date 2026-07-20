package querygo

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
)

// Example shows a full round trip: a Handler serving QUERY, and a Client
// sending a JSON query and decoding the JSON response.
func Example() {
	server := httptest.NewServer(Handler{
		AcceptQuery: []string{MediaTypeJSON},
		Query: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(HeaderContentType, MediaTypeJSON)
			_, _ = w.Write([]byte(`{"hits":2}`))
		},
	})
	defer server.Close()

	client := NewClient(server.Client())

	var result struct {
		Hits int `json:"hits"`
	}
	if err := client.QueryJSONInto(context.Background(), server.URL, map[string]string{"q": "golang"}, &result, nil); err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Hits)
	// Output: 2
}

// ExampleClient_Options shows discovering QUERY support via an OPTIONS request.
func ExampleClient_Options() {
	server := httptest.NewServer(Handler{AcceptQuery: []string{MediaTypeJSON}})
	defer server.Close()

	client := NewClient(server.Client())

	discovery, err := client.Options(context.Background(), server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(discovery.SupportsQuery, discovery.AcceptQuery[0])
	// Output: true application/json
}

// ExampleCacheKey shows that the cache key depends on the request body, so two
// different queries to the same URL produce different keys.
func ExampleCacheKey() {
	a := CacheKey("https://example.org/search", MediaTypeJSON, []byte(`{"q":"go"}`))
	b := CacheKey("https://example.org/search", MediaTypeJSON, []byte(`{"q":"rust"}`))

	fmt.Println(a == b)
	// Output: false
}
