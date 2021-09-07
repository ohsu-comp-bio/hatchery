/*
 * Hatchery API
 *
 * Workspace service for launching and interacting with containers.
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	//"encoding/json"
	"net/http"
	"strings"

	//"github.com/gorilla/mux"
)

// WorkspaceApiController binds http requests to an api service and writes the service results to the http response
type WorkspaceApiController struct {
	service WorkspaceApiServicer
	errorHandler ErrorHandler
}

// WorkspaceApiOption for how the controller is set up.
type WorkspaceApiOption func(*WorkspaceApiController)

// WithWorkspaceApiErrorHandler inject ErrorHandler into controller
func WithWorkspaceApiErrorHandler(h ErrorHandler) WorkspaceApiOption {
	return func(c *WorkspaceApiController) {
		c.errorHandler = h
	}
}

// NewWorkspaceApiController creates a default api controller
func NewWorkspaceApiController(s WorkspaceApiServicer, opts ...WorkspaceApiOption) Router {
	controller := &WorkspaceApiController{
		service:      s,
		errorHandler: DefaultErrorHandler,
	}

	for _, opt := range opts {
		opt(controller)
	}

	return controller
}

// Routes returns all of the api route for the WorkspaceApiController
func (c *WorkspaceApiController) Routes() Routes {
	return Routes{
		{
			"Launch",
			strings.ToUpper("Post"),
			"/lw-workspace/launch",
			c.Launch,
		},
		{
			"Options",
			strings.ToUpper("Get"),
			"/lw-workspace/options",
			c.Options,
		},
		{
			"Paymodels",
			strings.ToUpper("Get"),
			"/lw-workspace/paymodels",
			c.Paymodels,
		},
		{
			"Status",
			strings.ToUpper("Get"),
			"/lw-workspace/status",
			c.Status,
		},
		{
			"Terminate",
			strings.ToUpper("Post"),
			"/lw-workspace/terminate",
			c.Terminate,
		},
	}
}

// Launch - LaunchAWorkspace
func (c *WorkspaceApiController) Launch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("id")
	rEMOTEUSER := r.Header.Get("REMOTE_USER")
	authorization := r.Header.Get("Authorization")
	result, err := c.service.Launch(r.Context(), id, rEMOTEUSER, authorization)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}

// Options - Get the available workspace options that can be launched
func (c *WorkspaceApiController) Options(w http.ResponseWriter, r *http.Request) {
	rEMOTEUSER := r.Header.Get("REMOTE_USER")
	authorization := r.Header.Get("Authorization")
	result, err := c.service.Options(r.Context(), rEMOTEUSER, authorization)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}

// Paymodels -
func (c *WorkspaceApiController) Paymodels(w http.ResponseWriter, r *http.Request) {
	rEMOTEUSER := r.Header.Get("REMOTE_USER")
	result, err := c.service.Paymodels(r.Context(), rEMOTEUSER)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}

// Status - Get the current status of the workspace
func (c *WorkspaceApiController) Status(w http.ResponseWriter, r *http.Request) {
	rEMOTEUSER := r.Header.Get("REMOTE_USER")
	authorization := r.Header.Get("Authorization")
	result, err := c.service.Status(r.Context(), rEMOTEUSER, authorization)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}

// Terminate - Terminate the actively running workspace
func (c *WorkspaceApiController) Terminate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rEMOTEUSER := r.Header.Get("REMOTE_USER")
	authorization := r.Header.Get("Authorization")
	id := query.Get("id")
	result, err := c.service.Terminate(r.Context(), rEMOTEUSER, authorization, id)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}
