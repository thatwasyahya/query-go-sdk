package querygo

import (
	"reflect"
	"testing"
)

func TestParseAcceptQueryStructuredFields(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "rfc example with token and quoted string with param",
			in:   []string{`"application/jsonpath", application/sql;charset="UTF-8"`},
			want: []string{"application/jsonpath", "application/sql"},
		},
		{
			name: "comma inside quoted parameter is not a separator",
			in:   []string{`application/foo;p="a,b", text/plain`},
			want: []string{"application/foo", "text/plain"},
		},
		{
			name: "plain comma separated tokens",
			in:   []string{"application/sql, application/graphql"},
			want: []string{"application/sql", "application/graphql"},
		},
		{
			name: "multiple header lines",
			in:   []string{"application/sql", `"application/json"`},
			want: []string{"application/sql", "application/json"},
		},
		{
			name: "empty",
			in:   []string{""},
			want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAcceptQuery(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseAcceptQuery(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatAcceptQueryQuotesNonTokens(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"valid tokens stay bare", []string{"application/json", "application/sql"}, "application/json, application/sql"},
		{"wildcards are tokens", []string{"*/*", "image/*"}, "*/*, image/*"},
		{"leading digit must be quoted", []string{"3d/model"}, `"3d/model"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatAcceptQuery(tc.in); got != tc.want {
				t.Fatalf("formatAcceptQuery(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	in := []string{"application/json", "3d/model", "application/sql"}
	formatted := formatAcceptQuery(in)
	got := parseAcceptQuery([]string{formatted})
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("round trip: %q -> %q -> %q", in, formatted, got)
	}
}

func TestMatchMediaRange(t *testing.T) {
	cases := []struct {
		mediaRange string
		mediaType  string
		want       bool
	}{
		{"application/json", "application/json", true},
		{"application/json", "APPLICATION/JSON", true},
		{"application/json", "application/xml", false},
		{"*/*", "anything/here", true},
		{"image/*", "image/png", true},
		{"image/*", "text/plain", false},
	}

	for _, tc := range cases {
		if got := matchMediaRange(tc.mediaRange, tc.mediaType); got != tc.want {
			t.Fatalf("matchMediaRange(%q, %q) = %v, want %v", tc.mediaRange, tc.mediaType, got, tc.want)
		}
	}
}
