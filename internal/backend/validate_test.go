package backend

import (
	"errors"
	"testing"
)

func validRequest() Request {
	return Request{
		ID:           "req-1",
		Input:        "turn on the light",
		DeviceID:     "device-1",
		Alias:        "kitchen",
		PersonID:     "person-1",
		ProfileID:    "profile-1",
		ModelID:      "model-1",
		BackendID:    "backend-1",
		ModelName:    "hermes-test-model",
		SystemPrompt: "be concise",
		Metadata: map[string]string{
			"source": "test",
		},
	}
}

func TestRequestValidateAcceptsValidRequest(t *testing.T) {
	if err := validRequest().Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestRequestValidateRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{name: "input", mutate: func(req *Request) { req.Input = "" }},
		{name: "input whitespace", mutate: func(req *Request) { req.Input = " \t\n " }},
		{name: "person ID", mutate: func(req *Request) { req.PersonID = "" }},
		{name: "person ID whitespace", mutate: func(req *Request) { req.PersonID = "   " }},
		{name: "profile ID", mutate: func(req *Request) { req.ProfileID = "" }},
		{name: "profile ID whitespace", mutate: func(req *Request) { req.ProfileID = "   " }},
		{name: "model ID", mutate: func(req *Request) { req.ModelID = "" }},
		{name: "model ID whitespace", mutate: func(req *Request) { req.ModelID = "   " }},
		{name: "backend ID", mutate: func(req *Request) { req.BackendID = "" }},
		{name: "backend ID whitespace", mutate: func(req *Request) { req.BackendID = "   " }},
		{name: "model name", mutate: func(req *Request) { req.ModelName = "" }},
		{name: "model name whitespace", mutate: func(req *Request) { req.ModelName = "   " }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			tt.mutate(&req)

			err := req.Validate()
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("Validate() error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}
