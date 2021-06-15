package poller

import (
	"net/http"
)

// Client handles HTTP requests, including transient state, when polling
// publication websites
type Client struct {
	http.Client
}
