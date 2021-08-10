package request

import "net/http"

type (
	SearchResources   struct{}
	SearchSandbox     struct{}
	SearchHealthCheck struct{}
)

// NewSearchListResources request
func NewSearchListResources() *SearchResources {
	return &SearchResources{}
}

// Auditable returns all auditable/loggable parameters
func (r SearchResources) Auditable() map[string]interface{} {
	return map[string]interface{}{}
}

// Fill processes request and fills internal variables
func (r *SearchResources) Fill(req *http.Request) (err error) {

	return err
}

// NewSearchSandbox request
func NewSearchSandbox() *SearchSandbox {
	return &SearchSandbox{}
}

// Auditable returns all auditable/loggable parameters
func (r SearchSandbox) Auditable() map[string]interface{} {
	return map[string]interface{}{}
}

// Fill processes request and fills internal variables
func (r *SearchSandbox) Fill(req *http.Request) (err error) {

	return err
}

// NewSearchHealthCheck request
func NewSearchHealthCheck() *SearchHealthCheck {
	return &SearchHealthCheck{}
}

// Auditable returns all auditable/loggable parameters
func (r SearchHealthCheck) Auditable() map[string]interface{} {
	return map[string]interface{}{}
}

// Fill processes request and fills internal variables
func (r *SearchHealthCheck) Fill(req *http.Request) (err error) {

	return err
}
