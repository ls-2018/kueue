package over_kubeversion

import (
	"context"
	"sync"
	"time"

	versionutil "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
)

const fetchServerVersionInterval = time.Minute * 10

type ServerVersionFetcher struct {
	dc            discovery.DiscoveryInterface
	ticker        *time.Ticker
	serverVersion *versionutil.Version
	rwm           sync.RWMutex
}

type Options struct {
	Interval time.Duration
}

type Option func(*Options)

var defaultOptions = Options{
	Interval: fetchServerVersionInterval,
}

// Start implements the Runnable interface to run ServerVersionFetcher as a controller.
func (s *ServerVersionFetcher) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("serverVersionFetcher")
	ctx = ctrl.LoggerInto(ctx, log)

	if err := s.FetchServerVersion(); err != nil {
		log.Error(err, "Unable to fetch server version")
	}

	defer s.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.V(5).Info("Context cancelled; stop fetching server version")
			return nil
		case <-s.ticker.C:
			if err := s.FetchServerVersion(); err != nil {
				log.Error(err, "Unable to fetch server version")
			} else {
				log.V(5).Info("Fetch server version", "serverVersion", s.serverVersion)
			}
		}
	}
}

// FetchServerVersion gets API server version
func (s *ServerVersionFetcher) FetchServerVersion() error {
	v, err := FetchServerVersion(s.dc)
	if err != nil {
		return err
	}
	s.rwm.Lock()
	defer s.rwm.Unlock()
	s.serverVersion = v
	return nil
}

func (s *ServerVersionFetcher) GetServerVersion() versionutil.Version {
	s.rwm.RLock()
	defer s.rwm.RUnlock()
	return *s.serverVersion
}

func FetchServerVersion(d discovery.DiscoveryInterface) (*versionutil.Version, error) {
	clusterVersionInfo, err := d.ServerVersion()
	if err != nil {
		return nil, err
	}
	return versionutil.ParseSemantic(clusterVersionInfo.String())
}

func NewServerVersionFetcher(dc discovery.DiscoveryInterface, opts ...Option) *ServerVersionFetcher {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &ServerVersionFetcher{
		dc:            dc,
		ticker:        time.NewTicker(options.Interval),
		serverVersion: &versionutil.Version{},
	}
}
