package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/jsonapi"
)

// Success writes a 200 JSON:API response with a single resource.
func Success(w http.ResponseWriter, model interface{}) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(http.StatusOK)
	if err := jsonapi.MarshalPayload(w, model); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Created writes a 201 JSON:API response with a single resource.
func Created(w http.ResponseWriter, model interface{}) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(http.StatusCreated)
	if err := jsonapi.MarshalPayload(w, model); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Collection writes a 200 JSON:API paginated collection response.
func Collection(w http.ResponseWriter, models interface{}, total, limit, offset int, baseURL string) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(http.StatusOK)

	payload, err := jsonapi.Marshal(models)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	many, ok := payload.(*jsonapi.ManyPayload)
	if !ok {
		http.Error(w, "unexpected payload type", http.StatusInternalServerError)
		return
	}

	many.Meta = &jsonapi.Meta{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	self := fmt.Sprintf("%s?page[limit]=%d&page[offset]=%d", baseURL, limit, offset)
	first := fmt.Sprintf("%s?page[limit]=%d&page[offset]=0", baseURL, limit)
	many.Links = &jsonapi.Links{
		"self":  self,
		"first": first,
	}
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		prev := fmt.Sprintf("%s?page[limit]=%d&page[offset]=%d", baseURL, limit, prevOffset)
		(*many.Links)["prev"] = prev
	}
	if offset+limit < total {
		next := fmt.Sprintf("%s?page[limit]=%d&page[offset]=%d", baseURL, limit, offset+limit)
		(*many.Links)["next"] = next
	}

	if err := json.NewEncoder(w).Encode(many); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Error writes a JSON:API error response.
func Error(w http.ResponseWriter, statusCode int, title, detail string) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(statusCode)
	payload := &jsonapi.ErrorsPayload{
		Errors: []*jsonapi.ErrorObject{
			{
				Status: fmt.Sprintf("%d", statusCode),
				Title:  title,
				Detail: detail,
			},
		},
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// JSON writes a plain JSON response.
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
