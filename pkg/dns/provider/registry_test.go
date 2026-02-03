package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"zonekit/pkg/dnsrecord"
)

// mockProviderForRegistry is a mock implementation of the Provider interface for registry testing
type mockProviderForRegistry struct {
	name            string
	records         map[string][]dnsrecord.Record
	getRecordsError error
	setRecordsError error
	validateError   error
}

func newMockProviderForRegistry(name string) *mockProviderForRegistry {
	return &mockProviderForRegistry{
		name:    name,
		records: make(map[string][]dnsrecord.Record),
	}
}

func (m *mockProviderForRegistry) Name() string {
	return m.name
}

func (m *mockProviderForRegistry) ListZones(ctx context.Context) ([]Zone, error) {
	return nil, nil
}

func (m *mockProviderForRegistry) GetZone(ctx context.Context, zoneID string) (Zone, error) {
	return Zone{}, nil
}

func (m *mockProviderForRegistry) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	return nil, nil
}

func (m *mockProviderForRegistry) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	return dnsrecord.Record{}, nil
}

func (m *mockProviderForRegistry) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	return dnsrecord.Record{}, nil
}

func (m *mockProviderForRegistry) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	return nil
}

func (m *mockProviderForRegistry) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	return nil
}

func (m *mockProviderForRegistry) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{}
}

func (m *mockProviderForRegistry) Validate() error {
	return m.validateError
}

// RegistryTestSuite is a test suite for provider registry
type RegistryTestSuite struct {
	suite.Suite
}

// TestRegistrySuite runs the provider registry test suite
func TestRegistrySuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (s *RegistryTestSuite) SetupTest() {
	// Clear registry before each test
	Clear()
}

func (s *RegistryTestSuite) TearDownTest() {
	// Clear registry after each test
	Clear()
}

func (s *RegistryTestSuite) TestRegister_Success() {
	provider := newMockProviderForRegistry("test-provider")
	err := Register(provider)
	s.Require().NoError(err)

	// Verify provider was registered
	retrieved, err := Get("test-provider")
	s.Require().NoError(err)
	s.Require().Equal(provider, retrieved)
}

func (s *RegistryTestSuite) TestRegister_Duplicate() {
	provider1 := newMockProviderForRegistry("test-provider")
	provider2 := newMockProviderForRegistry("test-provider")

	err := Register(provider1)
	s.Require().NoError(err)

	err = Register(provider2)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "already registered")
}

func (s *RegistryTestSuite) TestRegister_EmptyName() {
	provider := newMockProviderForRegistry("")
	err := Register(provider)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "name cannot be empty")
}

func (s *RegistryTestSuite) TestGet_Success() {
	provider := newMockProviderForRegistry("test-provider")
	err := Register(provider)
	s.Require().NoError(err)

	retrieved, err := Get("test-provider")
	s.Require().NoError(err)
	s.Require().Equal(provider, retrieved)
}

func (s *RegistryTestSuite) TestGet_NotFound() {
	_, err := Get("non-existent-provider")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not found")
}

func (s *RegistryTestSuite) TestList_Empty() {
	providers := List()
	s.Require().Empty(providers)
}

func (s *RegistryTestSuite) TestList_MultipleProviders() {
	provider1 := newMockProviderForRegistry("provider1")
	provider2 := newMockProviderForRegistry("provider2")
	provider3 := newMockProviderForRegistry("provider3")

	err := Register(provider1)
	s.Require().NoError(err)
	err = Register(provider2)
	s.Require().NoError(err)
	err = Register(provider3)
	s.Require().NoError(err)

	providers := List()
	s.Require().Len(providers, 3)

	// Verify all providers are in the list
	providerMap := make(map[string]Provider)
	for _, p := range providers {
		providerMap[p.Name()] = p
	}

	s.Require().Contains(providerMap, "provider1")
	s.Require().Contains(providerMap, "provider2")
	s.Require().Contains(providerMap, "provider3")
}

func (s *RegistryTestSuite) TestNames_Empty() {
	names := Names()
	s.Require().Empty(names)
}

func (s *RegistryTestSuite) TestNames_MultipleProviders() {
	provider1 := newMockProviderForRegistry("provider1")
	provider2 := newMockProviderForRegistry("provider2")
	provider3 := newMockProviderForRegistry("provider3")

	err := Register(provider1)
	s.Require().NoError(err)
	err = Register(provider2)
	s.Require().NoError(err)
	err = Register(provider3)
	s.Require().NoError(err)

	names := Names()
	s.Require().Len(names, 3)
	s.Require().Contains(names, "provider1")
	s.Require().Contains(names, "provider2")
	s.Require().Contains(names, "provider3")
}

func (s *RegistryTestSuite) TestUnregister_Success() {
	provider := newMockProviderForRegistry("test-provider")
	err := Register(provider)
	s.Require().NoError(err)

	Unregister("test-provider")

	// Verify provider was removed
	_, err = Get("test-provider")
	s.Require().Error(err)
}

func (s *RegistryTestSuite) TestUnregister_NonExistent() {
	// Should not panic when unregistering non-existent provider
	Unregister("non-existent-provider")
}

func (s *RegistryTestSuite) TestClear() {
	provider1 := newMockProviderForRegistry("provider1")
	provider2 := newMockProviderForRegistry("provider2")

	err := Register(provider1)
	s.Require().NoError(err)
	err = Register(provider2)
	s.Require().NoError(err)

	Clear()

	// Verify all providers were removed
	providers := List()
	s.Require().Empty(providers)

	names := Names()
	s.Require().Empty(names)
}

func (s *RegistryTestSuite) TestConcurrentAccess() {
	// Test that registry is thread-safe
	provider1 := newMockProviderForRegistry("provider1")
	provider2 := newMockProviderForRegistry("provider2")

	// Register providers concurrently
	done := make(chan bool, 2)
	go func() {
		_ = Register(provider1)
		done <- true
	}()
	go func() {
		_ = Register(provider2)
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// Verify both were registered
	providers := List()
	s.Require().Len(providers, 2)
}
