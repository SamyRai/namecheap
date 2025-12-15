package client

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"zonekit/internal/testutil"
	"zonekit/pkg/config"
)

// ClientTestSuite is a test suite for client package
type ClientTestSuite struct {
	suite.Suite
}

// TestClientSuite runs the client test suite
func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TestNewClient_ValidConfiguration() {
	fixture := testutil.AccountConfigFixture()
	accountConfig := &config.AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}
	client, err := NewClient(accountConfig)

	s.AssertNoError(err)
	s.AssertNotNil(client)
	s.AssertNotNil(client.GetNamecheapClient())
	s.AssertNotNil(client.GetConfig())
}

func (s *ClientTestSuite) TestNewClient_MissingUsername() {
	fixture := testutil.AccountConfigFixtureWithValues("", "testapiuser", "testapikey", "127.0.0.1", true)
	accountConfig := &config.AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	client, err := NewClient(accountConfig)

	s.AssertError(err, true)
	s.AssertNil(client)
}

func (s *ClientTestSuite) TestNewClient_MissingAPIUser() {
	fixture := testutil.AccountConfigFixtureWithValues("testuser", "", "testapikey", "127.0.0.1", true)
	accountConfig := &config.AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	client, err := NewClient(accountConfig)

	s.AssertError(err, true)
	s.AssertNil(client)
}

func (s *ClientTestSuite) TestNewClient_MissingAPIKey() {
	fixture := testutil.AccountConfigFixtureWithValues("testuser", "testapiuser", "", "127.0.0.1", true)
	accountConfig := &config.AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	client, err := NewClient(accountConfig)

	s.AssertError(err, true)
	s.AssertNil(client)
}

func (s *ClientTestSuite) TestNewClient_MissingClientIP() {
	fixture := testutil.AccountConfigFixtureWithValues("testuser", "testapiuser", "testapikey", "", true)
	accountConfig := &config.AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	client, err := NewClient(accountConfig)

	s.AssertError(err, true)
	s.AssertNil(client)
}

func (s *ClientTestSuite) TestNewClient_AllFieldsMissing() {
	accountConfig := &config.AccountConfig{}
	client, err := NewClient(accountConfig)

	s.AssertError(err, true)
	s.AssertNil(client)
}

func (s *ClientTestSuite) TestClient_GetAccountName() {
	fixture := testutil.AccountConfigFixture()
	accountConfig := &config.AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}
	client, err := NewClient(accountConfig)
	s.Require().NoError(err)

	accountName := client.GetAccountName()
	s.AssertEqual(fixture.Username, accountName)
}

func (s *ClientTestSuite) TestClient_GetConfig() {
	fixture := testutil.AccountConfigFixture()
	accountConfig := &config.AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}
	client, err := NewClient(accountConfig)
	s.Require().NoError(err)

	cfg := client.GetConfig()
	s.AssertNotNil(cfg)
	s.AssertEqual(fixture.Username, cfg.Username)
	s.AssertEqual(fixture.APIUser, cfg.APIUser)
	s.AssertEqual(fixture.APIKey, cfg.APIKey)
	s.AssertEqual(fixture.ClientIP, cfg.ClientIP)
	s.AssertEqual(fixture.UseSandbox, cfg.UseSandbox)
}

// Helper methods for cleaner assertions
func (s *ClientTestSuite) AssertNoError(err error) {
	s.Require().NoError(err)
}

func (s *ClientTestSuite) AssertError(err error, wantErr bool) {
	if wantErr {
		s.Require().Error(err)
	} else {
		s.Require().NoError(err)
	}
}

func (s *ClientTestSuite) AssertNil(value interface{}) {
	s.Require().Nil(value)
}

func (s *ClientTestSuite) AssertNotNil(value interface{}) {
	s.Require().NotNil(value)
}

func (s *ClientTestSuite) AssertEqual(expected, actual interface{}) {
	s.Require().Equal(expected, actual)
}
