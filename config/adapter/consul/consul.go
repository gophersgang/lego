// Package consul reads configuration from the Consul KV store
//
// Consul is a highly available and distributed service discovery and key-value store designed
// with support for the modern data center to make distributed systems and configuration easy.
//
// e.g.
// CONFIG_URI=consul://prod.consul.cloud.com:8301/my/key?dc=frankfurt1&token=123
package consul

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	a "github.com/stairlin/lego/config/adapter"
)

// Name contains the adapter registered name
const Name = "consul"

// ErrMissingStoreKey means the given URL does not contain any key (path)
var ErrMissingStoreKey = errors.New("cannot initialise config without store key")

// ErrStoreKeyNotFound means the configuration does not exist on Consul
var ErrStoreKeyNotFound = errors.New("store config does not exist")

// ErrStoreConfigEmpty means the configuration exists, but it is empty
var ErrStoreConfigEmpty = errors.New("store config is empty")

// New returns a new file config store
func New(uri *url.URL) (a.Store, error) {
	// Configure client
	config := api.DefaultConfig()
	config.Address = uri.Host
	config.Datacenter = uri.Query().Get("dc")
	config.Token = uri.Query().Get("token")

	// Build Consul client
	client, err := api.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "cannot initialise Consul client")
	}

	// Sanitisation/Validation
	path := strings.TrimLeft(uri.Path, "/")
	if path == "" {
		return nil, ErrMissingStoreKey
	}

	return &Store{
		Client: client,
		Config: config,
		Key:    path,
	}, nil
}

// Store reads config from Consul K/V
type Store struct {
	Client *api.Client
	Config *api.Config
	Key    string
}

// Load config for the given environment
func (s *Store) Load(config interface{}) error {
	// Get a handle to the KV API
	kv := s.Client.KV()

	// Lookup the pair
	pair, _, err := kv.Get(s.Key, nil)
	if err != nil {
		if err == io.EOF {
			return ErrStoreKeyNotFound
		}
		return errors.Wrap(err, "cannot get config from Consul")
	}
	if pair == nil {
		return fmt.Errorf("store pair is nil. the key `%s` is probably missing on Consul", s.Key)
	}
	if len(pair.Value) == 0 {
		return ErrStoreConfigEmpty
	}

	// Unmarshal
	return json.Unmarshal(pair.Value, config)
}
