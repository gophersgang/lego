// Package journey defines a context type, which carries information about
// a specific inbound request. It is created when it hits the first service
// and it is propagated across all services.
//
// It has been named journey instead of request, because a journey can result
// of multiple sub-requests. And also because it sounds nice, isn't it?
package journey

import (
	"fmt"
	"strings"
	"time"

	"github.com/satori/go.uuid"
	netCtx "golang.org/x/net/context"

	"github.com/stairlin/lego/bg"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

// Type represents a context type (Foreground or Background)
type Type int

const (
	Child Type = iota
	Root
)

// Ctx is the journey context interface
type Ctx interface {
	ctx.Ctx

	UUID() string
	ShortID() string
	AppConfig() *config.Config
	BG(f func(c Ctx)) error
	BranchOff(t Type) Ctx
	Cancel()
	End()

	// Net context functions
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}

// context holds the context of a request (journey) during its whole lifecycle
type context struct {
	Type       Type
	ID         string // (hopefully) globally unique identifier
	C          netCtx.Context
	Stepper    Stepper
	app        app.Ctx
	logger     log.Logger
	cancelFunc func()
}

// New creates a new context and returns it
func New(ctx app.Ctx) Ctx {
	id := uuid.NewV4().String()

	// Log to correlate this journey with the current app environment
	ctx.Trace("ctx.journey.new", "Start journey",
		log.String("id", id),
	)

	j := &context{
		Type:    Root,
		ID:      id,
		Stepper: *NewStepper(),
		app:     ctx,
		logger:  ctx.L(),
	}

	// Build net context from its parent context
	reqConfig := ctx.Config().Request
	if reqConfig.Timeout() != 0 {
		j.Trace("ctx.journey.deadline", "Set deadline", log.Time("deadline", time.Now().Add(reqConfig.Timeout())))
		j.C, j.cancelFunc = netCtx.WithTimeout(ctx.RootContext(), ctx.Config().Request.Timeout())
	} else {
		j.C, j.cancelFunc = netCtx.WithCancel(ctx.RootContext())
	}

	return j
}

// AppConfig returns the application configuration on which this context currently runs
func (c *context) AppConfig() *config.Config {
	return c.app.Config()
}

func (c *context) Stats() stats.Stats {
	return c.app.Stats()
}

// BG executes the given function in background
func (c *context) BG(f func(Ctx)) error {
	childCtx := c.BranchOff(Root)

	return c.app.BG().Dispatch(bg.NewTask(func() {
		f(childCtx)

		// End the context if it has not already been done
		select {
		case <-childCtx.Done():
		default:
			childCtx.End()
		}
	}))
}

// Cancel tells an operation to abandon its work.
// Cancel does not wait for the work to stop.
// After the first call, subsequent calls to Cancel do nothing.
func (c *context) Cancel() {
	c.Trace("ctx.journey.cancel", "Cancelling the operation")
	c.cancelFunc()
}

// End marks the end of a journey. It does the same thing as Cancel, but just reveals better the intention
func (c *context) End() {
	c.Trace("ctx.journey.end", "End of this context")
	c.cancelFunc()
}

// UUID returns the universally unique identifier assigned to this context
func (c *context) UUID() string {
	return c.ID
}

// ShortID returns a partial representation of a request ID for the sake of readability
// However its uniqueness is not guarantee
func (c *context) ShortID() string {
	return strings.Split(c.ID, "-")[0]
}

func (c *context) Trace(tag, msg string, fields ...log.Field) {
	c.incTag(tag)
	c.log().Trace(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelTrace, tag)
}

func (c *context) Warning(tag, msg string, fields ...log.Field) {
	c.incTag(tag)
	c.log().Warning(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelWarning, tag)
}

func (c *context) Error(tag, msg string, fields ...log.Field) {
	c.incTag(tag)
	c.log().Error(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelError, tag)
}

// # Net Context functions
// These are implemented in order to use a journey context as a net context

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *context) Deadline() (deadline time.Time, ok bool) { return c.C.Deadline() }

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
func (c *context) Done() <-chan struct{} { return c.C.Done() }

// Err returns a non-nil error value after Done is closed. Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed. No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (c *context) Err() error { return c.C.Err() }

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
//
// Use context values only for request-scoped data that transits
// processes and API boundaries, not for passing optional parameters to
// functions.
func (c *context) Value(key interface{}) interface{} {
	c.Trace("ctx.journey.value", "Add net context value", log.Object("value", key))
	return c.C.Value(key)
}

func (c *context) logFields(fields []log.Field) []log.Field {
	f := []log.Field{
		log.String("log_type", "J"),
		log.String("id", c.ShortID()),
		log.String("step", c.Stepper.String()),
	}

	return append(f, fields...)
}

func (c *context) log() log.Logger {
	c.Stepper.Inc()
	return c.logger
}

func (c *context) incTag(tag string) {
	tags := map[string]string{
		"tag": tag,
	}

	c.stats().Histogram("log", 1, tags)
}

func (c *context) incLogLevelCount(lvl log.Level, tag string) {
	tags := map[string]string{
		"level":   lvl.String(),
		"tag":     tag,
		"service": c.AppConfig().Service,
		"node":    c.AppConfig().Node,
		"version": c.AppConfig().Version,
	}

	c.app.Stats().Histogram("log.level", 1, tags)
}

func (c *context) stats() stats.Stats {
	return c.app.Stats()
}

// BranchOff returns a new child context that branches off from the original context
func (c *context) BranchOff(t Type) Ctx {
	c.Trace("ctx.journey.branch_off", "New sub context", log.String("id", c.ID))
	ctx := c.createSubCtx()

	// If we have a root context, we break the context cancellation propagation
	if t == Root {
		ctx.C = netCtx.Background()
		return ctx
	}

	// Otherwise, create a new net context from its parent
	if deadline, ok := c.C.Deadline(); ok {
		ctx.C, ctx.cancelFunc = netCtx.WithDeadline(c.C, deadline)
	} else {
		ctx.C, ctx.cancelFunc = netCtx.WithCancel(c.C)
	}
	return ctx
}

func (c *context) createSubCtx() *context {
	return &context{
		ID:         c.ID,
		Stepper:    *c.Stepper.BranchOff(),
		C:          nil,
		app:        c.app,
		logger:     c.logger,
		cancelFunc: func() {},
	}
}

// spaceOut joins the given args and separate them with spaces
func spaceOut(args ...interface{}) string {
	l := make([]string, len(args))
	for i, a := range args {
		l[i] = fmt.Sprint(a)
	}
	return strings.Join(l, " ")
}
