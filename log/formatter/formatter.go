package formatter

import (
	"fmt"
	"sort"
	"sync"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/log/formatter/json"
	"github.com/stairlin/lego/log/formatter/logf"
)

func init() {
	Register(json.Name, json.New)
	Register(logf.Name, logf.New)
}

// Adapter returns a new logger initialised with the given config
type Adapter func(config map[string]string) (log.Formatter, error)

func New(config *config.Log) (log.Formatter, error) {
	return newFormatter(config.Formatter.Adapter, config.Formatter.Config)
}

var (
	adaptersMu sync.RWMutex
	adapters   = make(map[string]Adapter)
)

// Adapters returns the list of registered adapters
func Adapters() []string {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	var l []string
	for a := range adapters {
		l = append(l, a)
	}

	sort.Strings(l)

	return l
}

// Register makes a logger adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	adaptersMu.Lock()
	defer adaptersMu.Unlock()

	if adapter == nil {
		panic("logs: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("logs: Duplicated adapter")
	}

	adapters[name] = adapter
}

// newLogger returns a new logger instance
func newFormatter(adapter string, config map[string]string) (log.Formatter, error) {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	if f, ok := adapters[adapter]; ok {
		return f(config)
	}

	return nil, fmt.Errorf("log formatter not found <%s>", adapter)
}
