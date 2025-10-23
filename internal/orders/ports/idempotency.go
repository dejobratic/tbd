package ports

import "context"

// StoredResponse contains the response data to replay for a reused key.
type StoredResponse struct {
	StatusCode int
	Body       []byte
	OrderID    string
}

// IdempotencyStore ensures create operations can be retried safely.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (*StoredResponse, error)
	Save(ctx context.Context, key string, response StoredResponse) error
}
