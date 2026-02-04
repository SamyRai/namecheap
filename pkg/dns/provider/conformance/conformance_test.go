package conformance

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"zonekit/pkg/dns/provider"
)

// TestConformanceWithMock runs the conformance suite against the MockProvider
func TestConformanceWithMock(t *testing.T) {
	factory := func() (provider.Provider, func(), error) {
		p := NewMockProvider()
		return p, func() {}, nil
	}

	s := &Suite{
		ProviderFactory: factory,
	}
	suite.Run(t, s)
}
