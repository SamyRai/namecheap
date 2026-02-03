package namecheap

import (
	"context"
	"testing"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	dnsprovider "zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
)

// MockNamecheapClient is a mock implementation of NamecheapClient
type MockNamecheapClient struct {
	mock.Mock
}

func (m *MockNamecheapClient) DomainsGetList(args *namecheap.DomainsGetListArgs) (*namecheap.DomainsGetListCommandResponse, error) {
	ret := m.Called(args)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*namecheap.DomainsGetListCommandResponse), ret.Error(1)
}

func (m *MockNamecheapClient) DomainsGetInfo(domainName string) (*namecheap.DomainsGetInfoCommandResponse, error) {
	ret := m.Called(domainName)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*namecheap.DomainsGetInfoCommandResponse), ret.Error(1)
}

func (m *MockNamecheapClient) DomainsDNSGetHosts(domainName string) (*namecheap.DomainsDNSGetHostsCommandResponse, error) {
	ret := m.Called(domainName)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*namecheap.DomainsDNSGetHostsCommandResponse), ret.Error(1)
}

func (m *MockNamecheapClient) DomainsDNSSetHosts(args *namecheap.DomainsDNSSetHostsArgs) (*namecheap.DomainsDNSSetHostsCommandResponse, error) {
	ret := m.Called(args)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*namecheap.DomainsDNSSetHostsCommandResponse), ret.Error(1)
}

func TestListZones(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	domainName := "example.com"
	mockResp := &namecheap.DomainsGetListCommandResponse{
		Domains: &[]namecheap.Domain{
			{Name: &domainName},
		},
	}

	mockClient.On("DomainsGetList", mock.Anything).Return(mockResp, nil)

	zones, err := p.ListZones(context.Background())
	require.NoError(t, err)
	require.Len(t, zones, 1)
	assert.Equal(t, "example.com", zones[0].Name)
}

func TestGetZone(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	domainName := "example.com"
	mockResp := &namecheap.DomainsGetInfoCommandResponse{
		DomainDNSGetListResult: &namecheap.DomainsGetInfoResult{
			DomainName: &domainName,
			IsPremium:  namecheap.Bool(false),
			DnsDetails: &namecheap.DnsDetails{IsUsingOurDNS: namecheap.Bool(true)},
		},
	}

	mockClient.On("DomainsGetInfo", "example.com").Return(mockResp, nil)

	zone, err := p.GetZone(context.Background(), "example.com")
	require.NoError(t, err)
	assert.Equal(t, "example.com", zone.Name)
}

func TestListRecords(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	hostName := "www"
	recordType := "A"
	address := "1.2.3.4"
	ttl := 1800

	mockResp := &namecheap.DomainsDNSGetHostsCommandResponse{
		DomainDNSGetHostsResult: &namecheap.DomainDNSGetHostsResult{
			Hosts: &[]namecheap.DomainsDNSHostRecordDetailed{
				{
					Name: &hostName,
					Type: &recordType,
					Address: &address,
					TTL: &ttl,
				},
			},
		},
	}

	mockClient.On("DomainsDNSGetHosts", "example.com").Return(mockResp, nil)

	records, err := p.ListRecords(context.Background(), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "www", records[0].HostName)
	assert.Equal(t, "A", records[0].RecordType)
	assert.Equal(t, "1.2.3.4", records[0].Address)
}

func TestCreateRecord(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	// Mock ListRecords (initially empty)
	mockGetResp := &namecheap.DomainsDNSGetHostsCommandResponse{
		DomainDNSGetHostsResult: &namecheap.DomainDNSGetHostsResult{
			Hosts: &[]namecheap.DomainsDNSHostRecordDetailed{},
		},
	}
	mockClient.On("DomainsDNSGetHosts", "example.com").Return(mockGetResp, nil)

	// Mock SetHosts
	mockSetResp := &namecheap.DomainsDNSSetHostsCommandResponse{}
	mockClient.On("DomainsDNSSetHosts", mock.MatchedBy(func(args *namecheap.DomainsDNSSetHostsArgs) bool {
		return *args.Domain == "example.com" && len(*args.Records) == 1
	})).Return(mockSetResp, nil)

	newRecord := dnsrecord.Record{
		HostName:   "test",
		RecordType: "A",
		Address:    "1.2.3.4",
		TTL:        300,
	}

	created, err := p.CreateRecord(context.Background(), "example.com", newRecord)
	require.NoError(t, err)
	assert.Equal(t, "test", created.HostName)
}

func TestUpdateRecord(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	hostName := "www"
	recordType := "A"
	address := "1.2.3.4"

	// Mock ListRecords (1 existing)
	mockGetResp := &namecheap.DomainsDNSGetHostsCommandResponse{
		DomainDNSGetHostsResult: &namecheap.DomainDNSGetHostsResult{
			Hosts: &[]namecheap.DomainsDNSHostRecordDetailed{
				{
					Name: &hostName,
					Type: &recordType,
					Address: &address,
				},
			},
		},
	}
	mockClient.On("DomainsDNSGetHosts", "example.com").Return(mockGetResp, nil)

	// Mock SetHosts
	mockClient.On("DomainsDNSSetHosts", mock.MatchedBy(func(args *namecheap.DomainsDNSSetHostsArgs) bool {
		recs := *args.Records
		if len(recs) != 1 {
			return false
		}
		// Should have updated address
		return *recs[0].Address == "5.6.7.8"
	})).Return(&namecheap.DomainsDNSSetHostsCommandResponse{}, nil)

	updateRecord := dnsrecord.Record{
		HostName:   "www",
		RecordType: "A",
		Address:    "5.6.7.8", // New address
	}

	// recordID empty as Namecheap doesn't use it
	updated, err := p.UpdateRecord(context.Background(), "example.com", "", updateRecord)
	require.NoError(t, err)
	assert.Equal(t, "5.6.7.8", updated.Address)
}

// Conformance check
var _ dnsprovider.Provider = (*NamecheapProvider)(nil)
