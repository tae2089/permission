package apikey

import "time"

type issueRequest struct {
	Kind Kind `json:"kind"`
}

type issuedKeyResponse struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Kind       Kind      `json:"kind"`
	Status     Status    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Secret     string    `json:"secret"`
	SecretHash string    `json:"-"`
}

type keyResponse struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Kind      Kind       `json:"kind"`
	Status    Status     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type keysResponse struct {
	Keys []keyResponse `json:"keys"`
}
