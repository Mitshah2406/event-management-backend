package response

type StandardApiResponse struct {
	Status     string      `json:"status"`           // "success" or "error"
	StatusCode int         `json:"status_code"`      // HTTP status code
	Message    string      `json:"message"`          // Human-readable message
	Data       interface{} `json:"data,omitempty"`   // Payload for success
	Errors     interface{} `json:"errors,omitempty"` // Validation or error details
}
