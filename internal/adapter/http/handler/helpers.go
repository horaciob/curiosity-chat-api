package handler

import (
	"encoding/json"
	"net/http"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
)

// decodeRequestBody decodes JSON from request body with size limit checking.
// Returns true if successful, false if an error response was written.
func decodeRequestBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, response.MaxRequestBodySize)
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if err.Error() == "http: request body too large" {
			response.Error(w, http.StatusRequestEntityTooLarge, "Request Entity Too Large", "request body exceeds 1MB limit")
			return false
		}
		response.Error(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return false
	}
	return true
}

// decodeAttributes reads a JSON:API document and decodes data.attributes into dst.
func decodeAttributes(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	var doc struct {
		Data struct {
			Attributes json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if !decodeRequestBody(w, r, &doc) {
		return false
	}
	if len(doc.Data.Attributes) == 0 || json.Unmarshal(doc.Data.Attributes, dst) != nil {
		response.Error(w, http.StatusBadRequest, "Bad Request", "body must be a JSON:API document with data.attributes")
		return false
	}
	return true
}
