package util

import (
	"fmt"
	"net/http"
)

func SerializeHeader(header http.Header) []string {
	headers := make([]string, 0)
	for header, values := range header {
		if header == "" {
			continue
		}
		for _, val := range values {
			headers = append(headers, fmt.Sprintf("%s: %s", header, val))
		}
	}
	return headers
}
