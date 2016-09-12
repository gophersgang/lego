package journey_test

import (
	"strings"
	"testing"

	"github.com/stairlin/lego/ctx/journey"
	lt "github.com/stairlin/lego/testing"
)

// TestUniqueness tests whether the context ID is unique
func TestUniqueness(t *testing.T) {
	tt := lt.New(t)
	j := journey.New(tt.NewAppCtx("journey-test"))
	other := journey.New(tt.NewAppCtx("journey-test"))

	if j.UUID() == other.UUID() {
		tt.Error("expect context to have different UUIDs")
	}
}

// TestShortID tests whether ShortID is a substring of ID
func TestShortID(t *testing.T) {
	tt := lt.New(t)
	j := journey.New(tt.NewAppCtx("journey-test"))

	if !strings.HasPrefix(j.UUID(), j.ShortID()) {
		tt.Error("expect ShortID to be a substring of UUID", j.UUID(), j.ShortID())
	}
}

// TestAppConfig tests whether it returns the given app config
func TestAppConfig(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	j := journey.New(app)

	if j.AppConfig() != app.Config() {
		tt.Errorf("expect AppConfig to return the given app config (%v - %v)", j.AppConfig(), app.Config())
	}
}

func TestLogger(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	j := journey.New(app)

	// Send a few log lines
	j.Debug("j.test.debug", "A debug line")
	j.Debugf("j.test.debug", "A %s debug line", "formatted")
	j.Info("j.test.info", "A info line")
	j.Infof("j.test.info", "A %s info line", "formatted")
	j.Warning("A warning line")
	j.Warning("Another warning line")
	j.Warningf("A %s warning line", "formatted")
	j.Error("A error line")
	j.Errorf("A %s error line", "formatted")

	logger := tt.Logger().(*lt.Logger)
	tests := []struct {
		severity string
		expected int
	}{
		{severity: lt.I, expected: 4},
		{severity: lt.W, expected: 3},
		{severity: lt.E, expected: 2},
	}

	// Ensure they have been sent to the logger
	for _, test := range tests {
		res := logger.Lines(test.severity)
		if res != test.expected {
			tt.Errorf(
				"expect logger to receive %d log lines for severity <%s>, but got %d",
				test.expected,
				test.severity,
				res,
			)
		}
	}
}
