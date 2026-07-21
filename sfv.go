package querygo

import "strings"

// This file implements the small subset of RFC 9651 (Structured Field Values
// for HTTP) needed to parse and serialize the Accept-Query header field, which
// RFC 10008 Section 3 defines as a List of media ranges represented as Tokens
// or Strings, each optionally carrying parameters.
//
// Only the media-range values are surfaced (parameters are recognized and
// stripped), which is what QUERY content negotiation needs.

// parseAcceptQuery parses one or more Accept-Query header field values into the
// list of advertised media ranges (unquoted, without parameters).
func parseAcceptQuery(values []string) []string {
	items := make([]string, 0)
	for _, value := range values {
		for _, member := range splitSFList(value) {
			if mr := sfMemberValue(member); mr != "" {
				items = append(items, mr)
			}
		}
	}

	return items
}

// formatAcceptQuery serializes media ranges into a valid Structured Fields List
// for the Accept-Query header field, quoting values that are not valid Tokens.
func formatAcceptQuery(mediaTypes []string) string {
	parts := make([]string, 0, len(mediaTypes))
	for _, mediaType := range mediaTypes {
		trimmed := strings.TrimSpace(mediaType)
		if trimmed == "" {
			continue
		}

		if isSFToken(trimmed) {
			parts = append(parts, trimmed)
		} else {
			parts = append(parts, encodeSFString(trimmed))
		}
	}

	return strings.Join(parts, ", ")
}

// matchMediaRange reports whether mediaType is covered by the advertised media
// range, honoring the "*/*" and "type/*" wildcards. Comparison is
// case-insensitive, as media types are.
func matchMediaRange(mediaRange, mediaType string) bool {
	mediaRange = strings.ToLower(strings.TrimSpace(mediaRange))
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	switch {
	case mediaRange == "":
		return false
	case mediaRange == "*/*":
		return true
	case strings.HasSuffix(mediaRange, "/*"):
		return strings.HasPrefix(mediaType, strings.TrimSuffix(mediaRange, "*"))
	default:
		return mediaRange == mediaType
	}
}

// splitSFList splits a Structured Fields List on top-level commas, ignoring
// commas that appear inside quoted strings.
func splitSFList(value string) []string {
	var (
		members  []string
		buf      strings.Builder
		inString bool
		escaped  bool
	)

	for i := 0; i < len(value); i++ {
		c := value[i]

		if inString {
			buf.WriteByte(c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}

		switch c {
		case '"':
			inString = true
			buf.WriteByte(c)
		case ',':
			members = append(members, buf.String())
			buf.Reset()
		default:
			buf.WriteByte(c)
		}
	}

	members = append(members, buf.String())

	return members
}

// sfMemberValue extracts the bare value of a Structured Fields List member,
// stripping any parameters and unquoting a String.
func sfMemberValue(member string) string {
	member = strings.TrimSpace(member)
	if member == "" {
		return ""
	}

	// Strip parameters (everything from the first top-level ';').
	if idx := indexTopLevel(member, ';'); idx >= 0 {
		member = strings.TrimSpace(member[:idx])
	}

	if len(member) >= 2 && member[0] == '"' && member[len(member)-1] == '"' {
		return decodeSFString(member)
	}

	return member
}

// indexTopLevel returns the index of the first occurrence of sep that is not
// inside a quoted string, or -1.
func indexTopLevel(value string, sep byte) int {
	inString := false
	escaped := false

	for i := 0; i < len(value); i++ {
		c := value[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			continue
		}

		if c == sep {
			return i
		}
	}

	return -1
}

// decodeSFString unescapes a Structured Fields String (including the
// surrounding quotes), resolving the "\\" and "\"" escape sequences.
func decodeSFString(quoted string) string {
	inner := quoted[1 : len(quoted)-1]

	var buf strings.Builder
	escaped := false
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if escaped {
			buf.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		buf.WriteByte(c)
	}

	return buf.String()
}

// encodeSFString serializes s as a Structured Fields String, escaping "\" and
// '"'.
func encodeSFString(s string) string {
	var buf strings.Builder
	buf.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' || c == '"' {
			buf.WriteByte('\\')
		}
		buf.WriteByte(c)
	}
	buf.WriteByte('"')

	return buf.String()
}

// isSFToken reports whether s is a valid Structured Fields Token: it begins
// with an ALPHA or "*" and contains only tchar, ":" or "/".
func isSFToken(s string) bool {
	if s == "" {
		return false
	}

	if !isALPHA(s[0]) && s[0] != '*' {
		return false
	}

	for i := 1; i < len(s); i++ {
		c := s[i]
		if !isTchar(c) && c != ':' && c != '/' {
			return false
		}
	}

	return true
}

func isALPHA(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isDIGIT(c byte) bool {
	return c >= '0' && c <= '9'
}

// isTchar reports whether c is a valid token character per RFC 9110.
func isTchar(c byte) bool {
	if isALPHA(c) || isDIGIT(c) {
		return true
	}

	switch c {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}
