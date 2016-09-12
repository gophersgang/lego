package http

import (
	"net/http"
	"time"

	"github.com/stairlin/lego/ctx/journey"
)

// Action is an endpoint that handles incoming HTTP requests for a specific route.
// An action is stateless, self contained and should not share its context with other actions.
type Action interface {
	Call(c *Context) Renderer
}

// CallFunc is the contract required to be callable on the call chain
type CallFunc func(c *Context) Renderer

// Context holds the request context that is injected into an action
type Context struct {
	Ctx     journey.Ctx
	Res     http.ResponseWriter
	Req     *http.Request
	Parser  Parser
	Params  map[string]string
	StartAt time.Time

	isDraining func() bool
	action     Action
}

// JSON encodes the given data to JSON
func (c *Context) JSON(code int, data interface{}) Renderer {
	return &RenderJSON{Code: code, V: data}
}

// Head returns a body-less response
func (c *Context) Head(code int) Renderer {
	return &RenderHead{Code: code}
}

// Redirect returns an HTTP redirection response
func (c *Context) Redirect(url string) Renderer {
	return &RenderRedirect{URL: url}
}
