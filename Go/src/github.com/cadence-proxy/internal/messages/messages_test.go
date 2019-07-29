//-----------------------------------------------------------------------------
// FILE:		messages_test.go
// CONTRIBUTOR: John C Burns
// COPYRIGHT:	Copyright (c) 2016-2019 by neonFORGE, LLC.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package messages_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	cadenceshared "go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	goleak "go.uber.org/goleak"
	"go.uber.org/zap"

	"github.com/a3linux/amazon-ssm-agent/agent/times"

	"github.com/stretchr/testify/suite"

	globals "github.com/cadence-proxy/internal"
	"github.com/cadence-proxy/internal/cadence/cadenceclient"
	"github.com/cadence-proxy/internal/cadence/cadenceerrors"
	"github.com/cadence-proxy/internal/endpoints"
	"github.com/cadence-proxy/internal/logger"
	"github.com/cadence-proxy/internal/messages"
	messagetypes "github.com/cadence-proxy/internal/messages/types"
	"github.com/cadence-proxy/internal/server"
)

type (
	UnitTestSuite struct {
		suite.Suite
		instance *server.Instance
	}
)

const (
	_listenAddress = "127.0.0.1:5000"
)

// --------------------------------------------------------------------------
// Test suite methods.  Set up the test suite and entrypoint for test suite

func TestUnitTestSuite(t *testing.T) {

	// setup the suite
	s := new(UnitTestSuite)
	s.setupTestSuiteServer()

	// start the server as a go routine
	go s.instance.Start()

	// run the tests
	suite.Run(t, s)

	// send the server shutdown signal and wait
	// to exit until the server shuts down gracefully
	s.instance.ShutdownChannel <- true
	time.Sleep(time.Second * 1)

	// check for goroutine leaks
	goleak.VerifyNoLeaks(t)
}

func (s *UnitTestSuite) setupTestSuiteServer() {

	// set the logger
	logger.SetLogger("debug", true)

	// create the new server instance,
	// set the routes, and start the server listening
	// on host:port 127.0.0.1:5000
	s.instance = server.NewInstance(_listenAddress)
	s.instance.Logger = zap.L()
	endpoints.Instance = s.instance
	endpoints.SetupRoutes(s.instance.Router)
}

// --------------------------------------------------------------------------
// Test all implemented message types

func (s *UnitTestSuite) echoToConnection(message messages.IProxyMessage) (messages.IProxyMessage, error) {
	proxyMessage := message.GetProxyMessage()
	content, err := proxyMessage.Serialize(false)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(content)
	address := fmt.Sprintf("http://%s/echo", _listenAddress)
	req, err := http.NewRequest(http.MethodPut, address, buf)
	if err != nil {
		return nil, err
	}

	// set the request header to specified content type
	req.Header.Set("Content-Type", globals.ContentType)

	// initialize the http.Client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// create an empty []byte and read the
	// request body into it if not nil
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return messages.Deserialize(bytes.NewBuffer(payload), false)
}

func (s *UnitTestSuite) TestInitializeRequest() {

	var message messages.IProxyMessage = messages.NewInitializeRequest()
	if v, ok := message.(*messages.InitializeRequest); ok {
		s.Equal(messagetypes.InitializeReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.InitializeRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetLibraryAddress())
		s.Equal(int32(0), v.GetLibraryPort())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		str := "1.2.3.4"
		v.SetLibraryAddress(&str)
		s.Equal("1.2.3.4", *v.GetLibraryAddress())

		v.SetLibraryPort(int32(666))
		s.Equal(int32(666), v.GetLibraryPort())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.InitializeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("1.2.3.4", *v.GetLibraryAddress())
		s.Equal(int32(666), v.GetLibraryPort())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.InitializeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("1.2.3.4", *v.GetLibraryAddress())
		s.Equal(int32(666), v.GetLibraryPort())
	}
}

func (s *UnitTestSuite) TestInitializeReply() {
	var message messages.IProxyMessage = messages.NewInitializeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.InitializeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.InitializeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.InitializeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}
func (s *UnitTestSuite) TestConnectRequest() {

	var message messages.IProxyMessage = messages.NewConnectRequest()
	if v, ok := message.(*messages.ConnectRequest); ok {
		s.Equal(messagetypes.ConnectReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ConnectRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetEndpoints())
		s.Nil(v.GetIdentity())
		s.Equal(time.Duration(0), v.GetClientTimeout())
		s.False(v.GetCreateDomain())
		s.Nil(v.GetDomain())
		s.Equal(time.Duration(0), v.GetRetryDelay())
		s.Equal(int32(0), v.GetRetries())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		endpointsStr := "1.1.1.1:555,2.2.2.2:5555"
		v.SetEndpoints(&endpointsStr)
		s.Equal("1.1.1.1:555,2.2.2.2:5555", *v.GetEndpoints())

		identityStr := "my-identity"
		v.SetIdentity(&identityStr)
		s.Equal("my-identity", *v.GetIdentity())

		v.SetClientTimeout(time.Second * 30)
		s.Equal(time.Second*30, v.GetClientTimeout())

		domain := "my-domain"
		v.SetDomain(&domain)
		s.Equal("my-domain", *v.GetDomain())

		v.SetCreateDomain(true)
		s.True(v.GetCreateDomain())

		v.SetRetries(int32(3))
		s.Equal(int32(3), v.GetRetries())

		v.SetRetryDelay(time.Second * 30)
		s.Equal(time.Second*30, v.GetRetryDelay())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ConnectRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("1.1.1.1:555,2.2.2.2:5555", *v.GetEndpoints())
		s.Equal("my-identity", *v.GetIdentity())
		s.Equal(time.Second*30, v.GetClientTimeout())
		s.Equal("my-domain", *v.GetDomain())
		s.True(v.GetCreateDomain())
		s.Equal(int32(3), v.GetRetries())
		s.Equal(time.Second*30, v.GetRetryDelay())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ConnectRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("1.1.1.1:555,2.2.2.2:5555", *v.GetEndpoints())
		s.Equal("my-identity", *v.GetIdentity())
		s.Equal(time.Second*30, v.GetClientTimeout())
		s.Equal("my-domain", *v.GetDomain())
		s.True(v.GetCreateDomain())
		s.Equal(int32(3), v.GetRetries())
		s.Equal(time.Second*30, v.GetRetryDelay())
	}
}

func (s *UnitTestSuite) TestConnectReply() {
	var message messages.IProxyMessage = messages.NewConnectReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ConnectReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ConnectReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ConnectReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestDomainDescribeRequest() {

	var message messages.IProxyMessage = messages.NewDomainDescribeRequest()
	if v, ok := message.(*messages.DomainDescribeRequest); ok {
		s.Equal(messagetypes.DomainDescribeReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDescribeRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetName())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		nameStr := "my-domain"
		v.SetName(&nameStr)
		s.Equal("my-domain", *v.GetName())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDescribeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDescribeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
	}
}

func (s *UnitTestSuite) TestDomainDescribeReply() {
	var message messages.IProxyMessage = messages.NewDomainDescribeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDescribeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())
		s.False(v.GetConfigurationEmitMetrics())
		s.Equal(int32(0), v.GetConfigurationRetentionDays())
		s.Nil(v.GetDomainInfoName())
		s.Nil(v.GetDomainInfoDescription())
		s.Nil(v.GetDomainInfoOwnerEmail())
		s.Nil(v.GetDomainInfoStatus())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())

		v.SetConfigurationEmitMetrics(true)
		s.True(v.GetConfigurationEmitMetrics())

		v.SetConfigurationRetentionDays(int32(7))
		s.Equal(int32(7), v.GetConfigurationRetentionDays())

		domainInfoNameStr := "my-name"
		v.SetDomainInfoName(&domainInfoNameStr)
		s.Equal("my-name", *v.GetDomainInfoName())

		domainInfoDescriptionStr := "my-description"
		v.SetDomainInfoDescription(&domainInfoDescriptionStr)
		s.Equal("my-description", *v.GetDomainInfoDescription())

		domainStatus := cadenceclient.Deprecated
		v.SetDomainInfoStatus(&domainStatus)
		s.Equal(cadenceclient.Deprecated, *v.GetDomainInfoStatus())

		domainInfoOwnerEmailStr := "joe@bloe.com"
		v.SetDomainInfoOwnerEmail(&domainInfoOwnerEmailStr)
		s.Equal("joe@bloe.com", *v.GetDomainInfoOwnerEmail())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDescribeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
		s.Equal("my-name", *v.GetDomainInfoName())
		s.Equal("my-description", *v.GetDomainInfoDescription())
		s.Equal(cadenceclient.Deprecated, *v.GetDomainInfoStatus())
		s.Equal("joe@bloe.com", *v.GetDomainInfoOwnerEmail())
		s.Equal(int32(7), v.GetConfigurationRetentionDays())
		s.True(v.GetConfigurationEmitMetrics())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDescribeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
		s.Equal("my-name", *v.GetDomainInfoName())
		s.Equal("my-description", *v.GetDomainInfoDescription())
		s.Equal(cadenceclient.Deprecated, *v.GetDomainInfoStatus())
		s.Equal("joe@bloe.com", *v.GetDomainInfoOwnerEmail())
		s.Equal(int32(7), v.GetConfigurationRetentionDays())
		s.True(v.GetConfigurationEmitMetrics())
	}
}

func (s *UnitTestSuite) TestDomainRegisterRequest() {

	var message messages.IProxyMessage = messages.NewDomainRegisterRequest()
	if v, ok := message.(*messages.DomainRegisterRequest); ok {
		s.Equal(messagetypes.DomainRegisterReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainRegisterRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetName())
		s.Nil(v.GetDescription())
		s.Nil(v.GetOwnerEmail())
		s.False(v.GetEmitMetrics())
		s.Equal(int32(0), v.GetRetentionDays())
		s.Nil(v.GetSecurityToken())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		nameStr := "my-domain"
		v.SetName(&nameStr)
		s.Equal("my-domain", *v.GetName())

		descriptionStr := "my-description"
		v.SetDescription(&descriptionStr)
		s.Equal("my-description", *v.GetDescription())

		ownerEmailStr := "my-email"
		v.SetOwnerEmail(&ownerEmailStr)
		s.Equal("my-email", *v.GetOwnerEmail())

		v.SetEmitMetrics(true)
		s.True(v.GetEmitMetrics())

		v.SetRetentionDays(int32(14))
		s.Equal(int32(14), v.GetRetentionDays())

		securityToken := "security-token"
		v.SetSecurityToken(&securityToken)
		s.Equal("security-token", *v.GetSecurityToken())

	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainRegisterRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
		s.Equal("my-description", *v.GetDescription())
		s.Equal("my-email", *v.GetOwnerEmail())
		s.True(v.GetEmitMetrics())
		s.Equal(int32(14), v.GetRetentionDays())
		s.Equal("security-token", *v.GetSecurityToken())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainRegisterRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
		s.Equal("my-description", *v.GetDescription())
		s.Equal("my-email", *v.GetOwnerEmail())
		s.True(v.GetEmitMetrics())
		s.Equal(int32(14), v.GetRetentionDays())
		s.Equal("security-token", *v.GetSecurityToken())
	}
}

func (s *UnitTestSuite) TestDomainRegisterReply() {
	var message messages.IProxyMessage = messages.NewDomainRegisterReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainRegisterReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainRegisterReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainRegisterReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestDomainDeprecateRequest() {

	var message messages.IProxyMessage = messages.NewDomainDeprecateRequest()
	if v, ok := message.(*messages.DomainDeprecateRequest); ok {
		s.Equal(messagetypes.DomainDeprecateReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDeprecateRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetName())
		s.Nil(v.GetSecurityToken())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		nameStr := "my-domain"
		v.SetName(&nameStr)
		s.Equal("my-domain", *v.GetName())

		securityToken := "security-token"
		v.SetSecurityToken(&securityToken)
		s.Equal("security-token", *v.GetSecurityToken())

	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDeprecateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
		s.Equal("security-token", *v.GetSecurityToken())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDeprecateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
		s.Equal("security-token", *v.GetSecurityToken())
	}
}

func (s *UnitTestSuite) TestDomainDeprecateReply() {
	var message messages.IProxyMessage = messages.NewDomainDeprecateReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDeprecateReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDeprecateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainDeprecateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestDomainUpdateRequest() {

	var message messages.IProxyMessage = messages.NewDomainUpdateRequest()
	if v, ok := message.(*messages.DomainUpdateRequest); ok {
		s.Equal(messagetypes.DomainUpdateReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainUpdateRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetName())
		s.Nil(v.GetUpdatedInfoDescription())
		s.Nil(v.GetUpdatedInfoOwnerEmail())
		s.False(v.GetConfigurationEmitMetrics())
		s.Equal(int32(0), v.GetConfigurationRetentionDays())
		s.Nil(v.GetSecurityToken())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		nameStr := "my-domain"
		v.SetName(&nameStr)
		s.Equal("my-domain", *v.GetName())

		descriptionStr := "my-description"
		v.SetUpdatedInfoDescription(&descriptionStr)
		s.Equal("my-description", *v.GetUpdatedInfoDescription())

		ownerEmailStr := "my-email"
		v.SetUpdatedInfoOwnerEmail(&ownerEmailStr)
		s.Equal("my-email", *v.GetUpdatedInfoOwnerEmail())

		v.SetConfigurationEmitMetrics(true)
		s.True(v.GetConfigurationEmitMetrics())

		v.SetConfigurationRetentionDays(int32(14))
		s.Equal(int32(14), v.GetConfigurationRetentionDays())

		securityToken := "security-token"
		v.SetSecurityToken(&securityToken)
		s.Equal("security-token", *v.GetSecurityToken())

	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainUpdateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
		s.Equal("my-description", *v.GetUpdatedInfoDescription())
		s.Equal("my-email", *v.GetUpdatedInfoOwnerEmail())
		s.True(v.GetConfigurationEmitMetrics())
		s.Equal(int32(14), v.GetConfigurationRetentionDays())
		s.Equal("security-token", *v.GetSecurityToken())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainUpdateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetName())
		s.Equal("my-description", *v.GetUpdatedInfoDescription())
		s.Equal("my-email", *v.GetUpdatedInfoOwnerEmail())
		s.True(v.GetConfigurationEmitMetrics())
		s.Equal(int32(14), v.GetConfigurationRetentionDays())
		s.Equal("security-token", *v.GetSecurityToken())
	}
}

func (s *UnitTestSuite) TestDomainUpdateReply() {
	var message messages.IProxyMessage = messages.NewDomainUpdateReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainUpdateReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainUpdateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.DomainUpdateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestTerminateRequest() {

	var message messages.IProxyMessage = messages.NewTerminateRequest()
	if v, ok := message.(*messages.TerminateRequest); ok {
		s.Equal(messagetypes.TerminateReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.TerminateRequest); ok {
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.TerminateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.TerminateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestTerminateReply() {
	var message messages.IProxyMessage = messages.NewTerminateReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.TerminateReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.TerminateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.TerminateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestHeartbeatRequest() {

	var message messages.IProxyMessage = messages.NewHeartbeatRequest()
	if v, ok := message.(*messages.HeartbeatRequest); ok {
		s.Equal(messagetypes.HeartbeatReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.HeartbeatRequest); ok {
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.HeartbeatRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.HeartbeatRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestHeartbeatReply() {
	var message messages.IProxyMessage = messages.NewHeartbeatReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.HeartbeatReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.HeartbeatReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.HeartbeatReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestCancelRequest() {

	var message messages.IProxyMessage = messages.NewCancelRequest()
	if v, ok := message.(*messages.CancelRequest); ok {
		s.Equal(messagetypes.CancelReply, v.GetReplyType())
	}

	proxyMessage := message.GetProxyMessage()
	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.CancelRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetTargetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetTargetRequestID(int64(666))
		s.Equal(int64(666), v.GetTargetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.CancelRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetTargetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.CancelRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetTargetRequestID())
	}
}

func (s *UnitTestSuite) TestCancelReply() {
	var message messages.IProxyMessage = messages.NewCancelReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.CancelReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())
		s.False(v.GetWasCancelled())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())

		v.SetWasCancelled(true)
		s.True(v.GetWasCancelled())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.CancelReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
		s.True(v.GetWasCancelled())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.CancelReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
		s.True(v.GetWasCancelled())
	}
}

func (s *UnitTestSuite) TestNewWorkerReply() {
	var message messages.IProxyMessage = messages.NewNewWorkerReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.NewWorkerReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetWorkerID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetWorkerID(int64(666))
		s.Equal(int64(666), v.GetWorkerID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.NewWorkerReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetWorkerID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.NewWorkerReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetWorkerID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestNewWorkerRequest() {
	var message messages.IProxyMessage = messages.NewNewWorkerRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.NewWorkerRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetDomain())
		s.Nil(v.GetTaskList())
		s.Nil(v.GetOptions())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		domain := "my-domain"
		v.SetDomain(&domain)
		s.Equal("my-domain", *v.GetDomain())

		tasks := "my-tasks"
		v.SetTaskList(&tasks)
		s.Equal("my-tasks", *v.GetTaskList())

		opts := worker.Options{Identity: "my-identity", MaxConcurrentActivityExecutionSize: 1234}
		v.SetOptions(&opts)
		s.Equal(1234, v.GetOptions().MaxConcurrentActivityExecutionSize)
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.NewWorkerRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-tasks", *v.GetTaskList())
		s.Equal(1234, v.GetOptions().MaxConcurrentActivityExecutionSize)
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.NewWorkerRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-tasks", *v.GetTaskList())
		s.Equal(1234, v.GetOptions().MaxConcurrentActivityExecutionSize)
	}
}

func (s *UnitTestSuite) TestStopWorkerRequest() {
	var message messages.IProxyMessage = messages.NewStopWorkerRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.StopWorkerRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetWorkerID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetWorkerID(int64(666))
		s.Equal(int64(666), v.GetWorkerID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.StopWorkerRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetWorkerID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.StopWorkerRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetWorkerID())
	}
}

func (s *UnitTestSuite) TestStopWorkerReply() {
	var message messages.IProxyMessage = messages.NewStopWorkerReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.StopWorkerReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.StopWorkerReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.StopWorkerReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestPingRequest() {
	var message messages.IProxyMessage = messages.NewPingRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.PingRequest); ok {
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.PingRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.PingRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestPingReply() {
	var message messages.IProxyMessage = messages.NewPingReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.PingReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.PingReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.PingReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSetCacheSizeRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSetCacheSizeRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetCacheSizeRequest); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(0, v.GetSize())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetSize(20000)
		s.Equal(20000, v.GetSize())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetCacheSizeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(20000, v.GetSize())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetCacheSizeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(20000, v.GetSize())
	}
}

func (s *UnitTestSuite) TestWorkflowSetCacheSizeReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSetCacheSizeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetCacheSizeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetCacheSizeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetCacheSizeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowRegisterRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowRegisterRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRegisterRequest); ok {
		s.Equal(messagetypes.WorkflowRegisterReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetName())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		name := "Foo"
		v.SetName(&name)
		s.Equal("Foo", *v.GetName())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRegisterRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("Foo", *v.GetName())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRegisterRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("Foo", *v.GetName())
	}
}

func (s *UnitTestSuite) TestWorkflowRegisterReply() {
	var message messages.IProxyMessage = messages.NewWorkflowRegisterReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRegisterReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRegisterReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRegisterReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowExecuteRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowExecuteRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteRequest); ok {
		s.Equal(messagetypes.WorkflowExecuteReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetDomain())
		s.Nil(v.GetWorkflow())
		s.Nil(v.GetArgs())
		s.Nil(v.GetOptions())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		domain := "my-domain"
		v.SetDomain(&domain)
		s.Equal("my-domain", *v.GetDomain())

		workflow := "Foo"
		v.SetWorkflow(&workflow)
		s.Equal("Foo", *v.GetWorkflow())

		args := []byte{0, 1, 2, 3, 4}
		v.SetArgs(args)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())

		opts := client.StartWorkflowOptions{TaskList: "my-list", ExecutionStartToCloseTimeout: time.Second * 100}
		v.SetOptions(&opts)
		s.Equal(time.Second*100, v.GetOptions().ExecutionStartToCloseTimeout)
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("Foo", *v.GetWorkflow())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal(time.Second*100, v.GetOptions().ExecutionStartToCloseTimeout)
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("Foo", *v.GetWorkflow())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal(time.Second*100, v.GetOptions().ExecutionStartToCloseTimeout)
	}
}

func (s *UnitTestSuite) TestWorkflowExecuteReply() {
	var message messages.IProxyMessage = messages.NewWorkflowExecuteReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetExecution())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		exe := workflow.Execution{ID: "foo", RunID: "bar"}
		v.SetExecution(&exe)
		s.Equal("foo", v.GetExecution().ID)
		s.Equal("bar", v.GetExecution().RunID)

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("foo", v.GetExecution().ID)
		s.Equal("bar", v.GetExecution().RunID)
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("foo", v.GetExecution().ID)
		s.Equal("bar", v.GetExecution().RunID)
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowInvokeRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowInvokeRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowInvokeRequest); ok {
		s.Equal(messagetypes.WorkflowInvokeReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Nil(v.GetArgs())
		s.Nil(v.GetDomain())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())
		s.Nil(v.GetWorkflowType())
		s.Nil(v.GetTaskList())
		s.Equal(time.Duration(0), v.GetExecutionStartToCloseTimeout())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		name := "Foo"
		v.SetName(&name)
		s.Equal("Foo", *v.GetName())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		args := []byte{0, 1, 2, 3, 4}
		v.SetArgs(args)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())

		domain := "my-domain"
		v.SetDomain(&domain)
		s.Equal("my-domain", *v.GetDomain())

		workflowID := "my-workflowid"
		v.SetWorkflowID(&workflowID)
		s.Equal("my-workflowid", *v.GetWorkflowID())

		taskList := "my-tasklist"
		v.SetTaskList(&taskList)
		s.Equal("my-tasklist", *v.GetTaskList())

		runID := "my-runid"
		v.SetRunID(&runID)
		s.Equal("my-runid", *v.GetRunID())

		workflowType := "my-workflowtype"
		v.SetWorkflowType(&workflowType)
		s.Equal("my-workflowtype", *v.GetWorkflowType())

		v.SetExecutionStartToCloseTimeout(time.Hour * 24)
		s.Equal(time.Hour*24, v.GetExecutionStartToCloseTimeout())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("Foo", *v.GetName())
		s.Equal(int64(666), v.GetContextID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflowid", *v.GetWorkflowID())
		s.Equal("my-tasklist", *v.GetTaskList())
		s.Equal("my-runid", *v.GetRunID())
		s.Equal("my-workflowtype", *v.GetWorkflowType())
		s.Equal(time.Hour*24, v.GetExecutionStartToCloseTimeout())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("Foo", *v.GetName())
		s.Equal(int64(666), v.GetContextID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflowid", *v.GetWorkflowID())
		s.Equal("my-tasklist", *v.GetTaskList())
		s.Equal("my-runid", *v.GetRunID())
		s.Equal("my-workflowtype", *v.GetWorkflowType())
		s.Equal(time.Hour*24, v.GetExecutionStartToCloseTimeout())
	}
}

func (s *UnitTestSuite) TestWorkflowInvokeReply() {
	var message messages.IProxyMessage = messages.NewWorkflowInvokeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowInvokeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())
		s.Equal(int64(0), v.GetContextID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		result := []byte{0, 1, 2, 3, 4}
		v.SetResult(result)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowCancelRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowCancelRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelRequest); ok {
		s.Equal(messagetypes.WorkflowCancelReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())
		s.Nil(v.GetDomain())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		workflowID := "777"
		v.SetWorkflowID(&workflowID)
		s.Equal("777", *v.GetWorkflowID())

		runID := "666"
		v.SetRunID(&runID)
		s.Equal("666", *v.GetRunID())

		domain := "my-domain"
		v.SetDomain(&domain)
		s.Equal("my-domain", *v.GetDomain())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("666", *v.GetRunID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("666", *v.GetRunID())
	}
}

func (s *UnitTestSuite) TestWorkflowCancelReply() {
	var message messages.IProxyMessage = messages.NewWorkflowCancelReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowTerminateRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowTerminateRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowTerminateRequest); ok {
		s.Equal(messagetypes.WorkflowTerminateReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())
		s.Nil(v.GetReason())
		s.Nil(v.GetDetails())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		workflowID := "777"
		v.SetWorkflowID(&workflowID)
		s.Equal("777", *v.GetWorkflowID())

		runID := "666"
		v.SetRunID(&runID)
		s.Equal("666", *v.GetRunID())

		reason := "my-reason"
		v.SetReason(&reason)
		s.Equal("my-reason", *v.GetReason())

		details := []byte{0, 1, 2, 3, 4}
		v.SetDetails(details)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowTerminateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-reason", *v.GetReason())
		s.Equal("666", *v.GetRunID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowTerminateRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-reason", *v.GetReason())
		s.Equal("666", *v.GetRunID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())
	}
}

func (s *UnitTestSuite) TestWorkflowTerminateReply() {
	var message messages.IProxyMessage = messages.NewWorkflowTerminateReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowTerminateReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowTerminateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowTerminateReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalRequest); ok {
		s.Equal(messagetypes.WorkflowSignalReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())
		s.Nil(v.GetSignalName())
		s.Nil(v.GetSignalArgs())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		workflowID := "777"
		v.SetWorkflowID(&workflowID)
		s.Equal("777", *v.GetWorkflowID())

		runID := "666"
		v.SetRunID(&runID)
		s.Equal("666", *v.GetRunID())

		signalName := "my-signal"
		v.SetSignalName(&signalName)
		s.Equal("my-signal", *v.GetSignalName())

		signalArgs := []byte{0, 1, 2, 3, 4}
		v.SetSignalArgs(signalArgs)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-signal", *v.GetSignalName())
		s.Equal("666", *v.GetRunID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-signal", *v.GetSignalName())
		s.Equal("666", *v.GetRunID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalWithStartRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalWithStartRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalWithStartRequest); ok {
		s.Equal(messagetypes.WorkflowSignalWithStartReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflow())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetSignalName())
		s.Nil(v.GetSignalArgs())
		s.Nil(v.GetOptions())
		s.Nil(v.GetWorkflowArgs())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		workflow := "my-workflow"
		v.SetWorkflow(&workflow)
		s.Equal("my-workflow", *v.GetWorkflow())

		workflowID := "777"
		v.SetWorkflowID(&workflowID)
		s.Equal("777", *v.GetWorkflowID())

		signalName := "my-signal"
		v.SetSignalName(&signalName)
		s.Equal("my-signal", *v.GetSignalName())

		signalArgs := []byte{0, 1, 2, 3, 4}
		v.SetSignalArgs(signalArgs)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())

		opts := client.StartWorkflowOptions{TaskList: "my-tasklist", WorkflowIDReusePolicy: client.WorkflowIDReusePolicyAllowDuplicate}
		v.SetOptions(&opts)
		s.Equal("my-tasklist", v.GetOptions().TaskList)
		s.Equal(client.WorkflowIDReusePolicyAllowDuplicate, v.GetOptions().WorkflowIDReusePolicy)

		workflowArgs := []byte{5, 6, 7, 8, 9}
		v.SetWorkflowArgs(workflowArgs)
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetWorkflowArgs())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalWithStartRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-workflow", *v.GetWorkflow())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-signal", *v.GetSignalName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
		s.Equal("my-tasklist", v.GetOptions().TaskList)
		s.Equal(client.WorkflowIDReusePolicyAllowDuplicate, v.GetOptions().WorkflowIDReusePolicy)
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetWorkflowArgs())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalWithStartRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-workflow", *v.GetWorkflow())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-signal", *v.GetSignalName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
		s.Equal("my-tasklist", v.GetOptions().TaskList)
		s.Equal(client.WorkflowIDReusePolicyAllowDuplicate, v.GetOptions().WorkflowIDReusePolicy)
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetWorkflowArgs())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalWithStartReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalWithStartReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalWithStartReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())
		s.Nil(v.GetExecution())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		exe := workflow.Execution{ID: "666", RunID: "777"}
		v.SetExecution(&exe)
		s.Equal("666", v.GetExecution().ID)
		s.Equal("777", v.GetExecution().RunID)

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalWithStartReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("666", v.GetExecution().ID)
		s.Equal("777", v.GetExecution().RunID)
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalWithStartReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("666", v.GetExecution().ID)
		s.Equal("777", v.GetExecution().RunID)
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowCancelChildReply() {
	var message messages.IProxyMessage = messages.NewWorkflowCancelChildReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelChildReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowCancelChildRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowCancelChildRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelChildRequest); ok {
		s.Equal(messagetypes.WorkflowCancelChildReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetChildID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetChildID(int64(666))
		s.Equal(int64(666), v.GetChildID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetChildID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowCancelChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetChildID())
	}
}

func (s *UnitTestSuite) TestWorkflowQueryRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowQueryRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryRequest); ok {
		s.Equal(messagetypes.WorkflowQueryReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())
		s.Nil(v.GetQueryName())
		s.Nil(v.GetQueryArgs())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		workflowID := "777"
		v.SetWorkflowID(&workflowID)
		s.Equal("777", *v.GetWorkflowID())

		runID := "666"
		v.SetRunID(&runID)
		s.Equal("666", *v.GetRunID())

		queryName := "my-query"
		v.SetQueryName(&queryName)
		s.Equal("my-query", *v.GetQueryName())

		queryArgs := []byte{0, 1, 2, 3, 4}
		v.SetQueryArgs(queryArgs)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetQueryArgs())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("666", *v.GetRunID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-query", *v.GetQueryName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetQueryArgs())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("666", *v.GetRunID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("my-query", *v.GetQueryName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetQueryArgs())
	}
}

func (s *UnitTestSuite) TestWorkflowQueryReply() {
	var message messages.IProxyMessage = messages.NewWorkflowQueryReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())
		s.Nil(v.GetResult())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		result := []byte{0, 1, 2, 3, 4}
		v.SetResult(result)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowMutableRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowMutableRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowMutableRequest); ok {
		s.Equal(messagetypes.WorkflowMutableReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetMutableID())
		s.Nil(v.GetResult())
		s.Equal(int64(0), v.GetContextID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		mutableID := "777"
		v.SetMutableID(&mutableID)
		s.Equal("777", *v.GetMutableID())

		v.SetContextID(int64(888))
		s.Equal(int64(888), v.GetContextID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowMutableRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetMutableID())
		s.Equal(int64(888), v.GetContextID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowMutableRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetMutableID())
		s.Equal(int64(888), v.GetContextID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
	}
}

func (s *UnitTestSuite) TestWorkflowMutableReply() {
	var message messages.IProxyMessage = messages.NewWorkflowMutableReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowMutableReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())
		s.Nil(v.GetResult())
		s.Equal(int64(0), v.GetContextID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(888))
		s.Equal(int64(888), v.GetContextID())

		result := []byte{0, 1, 2, 3, 4}
		v.SetResult(result)
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowMutableReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(int64(888), v.GetContextID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowMutableReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(888), v.GetContextID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowDescribeExecutionRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowDescribeExecutionRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDescribeExecutionRequest); ok {
		s.Equal(messagetypes.WorkflowDescribeExecutionReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		workflowID := "777"
		v.SetWorkflowID(&workflowID)
		s.Equal("777", *v.GetWorkflowID())

		runID := "666"
		v.SetRunID(&runID)
		s.Equal("666", *v.GetRunID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDescribeExecutionRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("666", *v.GetRunID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDescribeExecutionRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("777", *v.GetWorkflowID())
		s.Equal("666", *v.GetRunID())
	}
}

func (s *UnitTestSuite) TestWorkflowDescribeExecutionReply() {
	var message messages.IProxyMessage = messages.NewWorkflowDescribeExecutionReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDescribeExecutionReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetDetails())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		details := cadenceshared.DescribeWorkflowExecutionResponse{}
		v.SetDetails(&details)
		s.Equal(cadenceshared.DescribeWorkflowExecutionResponse{}, *v.GetDetails())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDescribeExecutionReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceshared.DescribeWorkflowExecutionResponse{}, *v.GetDetails())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDescribeExecutionReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceshared.DescribeWorkflowExecutionResponse{}, *v.GetDetails())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowDisconnectContextReply() {
	var message messages.IProxyMessage = messages.NewWorkflowDisconnectContextReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDisconnectContextReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDisconnectContextReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDisconnectContextReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowDisconnectContextRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowDisconnectContextRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDisconnectContextRequest); ok {
		s.Equal(messagetypes.WorkflowDisconnectContextReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDisconnectContextRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowDisconnectContextRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestWorkflowExecuteChildReply() {
	var message messages.IProxyMessage = messages.NewWorkflowExecuteChildReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteChildReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetChildID())
		s.Nil(v.GetExecution())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetChildID(int64(666))
		s.Equal(int64(666), v.GetChildID())

		we := workflow.Execution{
			ID:    "my-workflow",
			RunID: "my-run",
		}
		v.SetExecution(&we)
		s.Equal("my-workflow", v.GetExecution().ID)
		s.Equal("my-run", v.GetExecution().RunID)

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetChildID())
		s.Equal("my-workflow", v.GetExecution().ID)
		s.Equal("my-run", v.GetExecution().RunID)
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetChildID())
		s.Equal("my-workflow", v.GetExecution().ID)
		s.Equal("my-run", v.GetExecution().RunID)
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowExecuteChildRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowExecuteChildRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteChildRequest); ok {
		s.Equal(messagetypes.WorkflowExecuteChildReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflow())
		s.Nil(v.GetArgs())
		s.Nil(v.GetOptions())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		wf := "my-workflow"
		v.SetWorkflow(&wf)
		s.Equal("my-workflow", *v.GetWorkflow())

		v.SetArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())

		opts := workflow.ChildWorkflowOptions{
			TaskList:                     "my-tasklist",
			Domain:                       "my-domain",
			ChildPolicy:                  workflow.ChildWorkflowPolicyRequestCancel,
			WorkflowID:                   "my-workflow",
			ExecutionStartToCloseTimeout: time.Second * 20,
		}
		v.SetOptions(&opts)
		s.Equal(workflow.ChildWorkflowOptions{TaskList: "my-tasklist", Domain: "my-domain", ChildPolicy: workflow.ChildWorkflowPolicyRequestCancel, WorkflowID: "my-workflow", ExecutionStartToCloseTimeout: time.Second * 20}, *v.GetOptions())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-workflow", *v.GetWorkflow())
		s.Equal(workflow.ChildWorkflowOptions{TaskList: "my-tasklist", Domain: "my-domain", ChildPolicy: workflow.ChildWorkflowPolicyRequestCancel, WorkflowID: "my-workflow", ExecutionStartToCloseTimeout: time.Second * 20}, *v.GetOptions())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowExecuteChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-workflow", *v.GetWorkflow())
		s.Equal(workflow.ChildWorkflowOptions{TaskList: "my-tasklist", Domain: "my-domain", ChildPolicy: workflow.ChildWorkflowPolicyRequestCancel, WorkflowID: "my-workflow", ExecutionStartToCloseTimeout: time.Second * 20}, *v.GetOptions())
	}
}

func (s *UnitTestSuite) TestWorkflowGetLastResultReply() {
	var message messages.IProxyMessage = messages.NewWorkflowGetLastResultReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetLastResultReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetLastResultReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetLastResultReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowGetLastResultRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowGetLastResultRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetLastResultRequest); ok {
		s.Equal(messagetypes.WorkflowGetLastResultReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetLastResultRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetLastResultRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestWorkflowGetResultReply() {
	var message messages.IProxyMessage = messages.NewWorkflowGetResultReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetResultReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetResultReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetResultReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowGetResultRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowGetResultRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetResultRequest); ok {
		s.Equal(messagetypes.WorkflowGetResultReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		workflowID := "my-workflow"
		v.SetWorkflowID(&workflowID)
		s.Equal("my-workflow", *v.GetWorkflowID())

		runID := "my-run"
		v.SetRunID(&runID)
		s.Equal("my-run", *v.GetRunID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetResultRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-run", *v.GetRunID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetResultRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-run", *v.GetRunID())
	}
}

func (s *UnitTestSuite) TestWorkflowGetTimeReply() {
	var message messages.IProxyMessage = messages.NewWorkflowGetTimeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetTimeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(time.Time{}, v.GetTime())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetTime(time.Date(2019, time.May, 27, 0, 0, 0, 0, time.UTC))
		s.Equal(time.Date(2019, time.May, 27, 0, 0, 0, 0, time.UTC), v.GetTime())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetTimeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(time.Date(2019, time.May, 27, 0, 0, 0, 0, time.UTC), v.GetTime())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetTimeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(time.Date(2019, time.May, 27, 0, 0, 0, 0, time.UTC), v.GetTime())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowGetTimeRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowGetTimeRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetTimeRequest); ok {
		s.Equal(messagetypes.WorkflowGetTimeReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetTimeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetTimeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestWorkflowHasLastResultReply() {
	var message messages.IProxyMessage = messages.NewWorkflowHasLastResultReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowHasLastResultReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.False(v.GetHasResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetHasResult(true)
		s.True(v.GetHasResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowHasLastResultReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.True(v.GetHasResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowHasLastResultReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.True(v.GetHasResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowHasLastResultRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowHasLastResultRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowHasLastResultRequest); ok {
		s.Equal(messagetypes.WorkflowHasLastResultReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowHasLastResultRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowHasLastResultRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestWorkflowQueryInvokeReply() {
	var message messages.IProxyMessage = messages.NewWorkflowQueryInvokeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryInvokeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowQueryInvokeRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowQueryInvokeRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryInvokeRequest); ok {
		s.Equal(messagetypes.WorkflowQueryInvokeReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Nil(v.GetQueryName())
		s.Nil(v.GetQueryArgs())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		queryName := "query"
		v.SetQueryName(&queryName)
		s.Equal("query", *v.GetQueryName())

		v.SetQueryArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetQueryArgs())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("query", *v.GetQueryName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetQueryArgs())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowQueryInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("query", *v.GetQueryName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetQueryArgs())
	}
}

func (s *UnitTestSuite) TestWorkflowSetQueryHandlerReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSetQueryHandlerReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetQueryHandlerReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetQueryHandlerReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetQueryHandlerReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSetQueryHandlerRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSetQueryHandlerRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetQueryHandlerRequest); ok {
		s.Equal(messagetypes.WorkflowSetQueryHandlerReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Nil(v.GetQueryName())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		queryName := "query"
		v.SetQueryName(&queryName)
		s.Equal("query", *v.GetQueryName())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetQueryHandlerRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("query", *v.GetQueryName())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSetQueryHandlerRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("query", *v.GetQueryName())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalChildReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalChildReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalChildReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalChildRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalChildRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalChildRequest); ok {
		s.Equal(messagetypes.WorkflowSignalChildReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Equal(int64(0), v.GetChildID())
		s.Nil(v.GetSignalName())
		s.Nil(v.GetSignalArgs())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		v.SetChildID(int64(777))
		s.Equal(int64(777), v.GetChildID())

		signalName := "my-signal"
		v.SetSignalName(&signalName)
		s.Equal("my-signal", *v.GetSignalName())

		v.SetSignalArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(int64(777), v.GetChildID())
		s.Equal("my-signal", *v.GetSignalName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(int64(777), v.GetChildID())
		s.Equal("my-signal", *v.GetSignalName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalInvokeReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalInvokeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalInvokeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalInvokeRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalInvokeRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalInvokeRequest); ok {
		s.Equal(messagetypes.WorkflowSignalInvokeReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Nil(v.GetSignalName())
		s.Nil(v.GetSignalArgs())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		signalName := "signal"
		v.SetSignalName(&signalName)
		s.Equal("signal", *v.GetSignalName())

		v.SetSignalArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("signal", *v.GetSignalName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("signal", *v.GetSignalName())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetSignalArgs())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalSubscribeReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalSubscribeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalSubscribeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalSubscribeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalSubscribeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSignalSubscribeRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSignalSubscribeRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalSubscribeRequest); ok {
		s.Equal(messagetypes.WorkflowSignalSubscribeReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Nil(v.GetSignalName())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		signalName := "signal"
		v.SetSignalName(&signalName)
		s.Equal("signal", *v.GetSignalName())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalSubscribeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("signal", *v.GetSignalName())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSignalSubscribeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal("signal", *v.GetSignalName())
	}
}

func (s *UnitTestSuite) TestWorkflowSleepReply() {
	var message messages.IProxyMessage = messages.NewWorkflowSleepReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSleepReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSleepReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSleepReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowSleepRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowSleepRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSleepRequest); ok {
		s.Equal(messagetypes.WorkflowSleepReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Equal(int64(0), v.GetDuration().Nanoseconds())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		v.SetDuration(time.Second * 30)
		s.Equal(time.Second*30, v.GetDuration())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSleepRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(time.Second*30, v.GetDuration())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowSleepRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(time.Second*30, v.GetDuration())
	}
}

func (s *UnitTestSuite) TestWorkflowWaitForChildReply() {
	var message messages.IProxyMessage = messages.NewWorkflowWaitForChildReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowWaitForChildReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowWaitForChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowWaitForChildReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowWaitForChildRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowWaitForChildRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowWaitForChildRequest); ok {
		s.Equal(messagetypes.WorkflowWaitForChildReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Equal(int64(0), v.GetChildID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		v.SetChildID(int64(777))
		s.Equal(int64(777), v.GetChildID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowWaitForChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(int64(777), v.GetChildID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowWaitForChildRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(int64(777), v.GetChildID())
	}
}

func (s *UnitTestSuite) TestWorkflowGetVersionReply() {
	var message messages.IProxyMessage = messages.NewWorkflowGetVersionReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetVersionReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int32(0), v.GetVersion())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetVersion(int32(20))
		s.Equal(int32(20), v.GetVersion())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetVersionReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int32(20), v.GetVersion())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetVersionReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int32(20), v.GetVersion())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestWorkflowGetVersionRequest() {
	var message messages.IProxyMessage = messages.NewWorkflowGetVersionRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetVersionRequest); ok {
		s.Equal(messagetypes.WorkflowGetVersionReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())
		s.Equal(int32(0), v.GetMaxSupported())
		s.Equal(int32(0), v.GetMinSupported())
		s.Nil(v.GetChangeID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(666))
		s.Equal(int64(666), v.GetContextID())

		v.SetMinSupported(int32(777))
		s.Equal(int32(777), v.GetMinSupported())

		v.SetMaxSupported(int32(888))
		s.Equal(int32(888), v.GetMaxSupported())

		changeID := "my-change"
		v.SetChangeID(&changeID)
		s.Equal("my-change", *v.GetChangeID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetVersionRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(int32(777), v.GetMinSupported())
		s.Equal(int32(888), v.GetMaxSupported())
		s.Equal("my-change", *v.GetChangeID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowGetVersionRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(666), v.GetContextID())
		s.Equal(int32(777), v.GetMinSupported())
		s.Equal(int32(888), v.GetMaxSupported())
		s.Equal("my-change", *v.GetChangeID())
	}
}

func (s *UnitTestSuite) TestActivityCompleteRequest() {
	var message messages.IProxyMessage = messages.NewActivityCompleteRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityCompleteRequest); ok {
		s.Equal(messagetypes.ActivityCompleteReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetTaskToken())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())
		s.Nil(v.GetDomain())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())
		s.Nil(v.GetActivityID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetTaskToken([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetTaskToken())

		v.SetResult([]byte{5, 6, 7, 8, 9})
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Generic))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Generic), v.GetError())

		domain := "my-domain"
		workflowID := "my-workflow"
		runID := "my-workflowrun"
		activityID := "my-activity"

		v.SetDomain(&domain)
		v.SetWorkflowID(&workflowID)
		v.SetRunID(&runID)
		v.SetActivityID(&activityID)

		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-workflowrun", *v.GetRunID())
		s.Equal("my-activity", *v.GetActivityID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityCompleteRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetTaskToken())
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Generic), v.GetError())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-workflowrun", *v.GetRunID())
		s.Equal("my-activity", *v.GetActivityID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityCompleteRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetTaskToken())
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Generic), v.GetError())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-workflowrun", *v.GetRunID())
		s.Equal("my-activity", *v.GetActivityID())
	}
}

func (s *UnitTestSuite) TestActivityCompleteReply() {
	var message messages.IProxyMessage = messages.NewActivityCompleteReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityCompleteReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityCompleteReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityCompleteReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityExecuteLocalReply() {
	var message messages.IProxyMessage = messages.NewActivityExecuteLocalReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteLocalReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteLocalReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteLocalReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityExecuteLocalRequest() {
	var message messages.IProxyMessage = messages.NewActivityExecuteLocalRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteLocalRequest); ok {
		s.Equal(messagetypes.ActivityExecuteLocalReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetArgs())
		s.Nil(v.GetOptions())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())

		opts := workflow.LocalActivityOptions{
			ScheduleToCloseTimeout: time.Second * 30,
		}
		v.SetOptions(&opts)
		s.Equal(workflow.LocalActivityOptions{ScheduleToCloseTimeout: time.Second * 30}, *v.GetOptions())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteLocalRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal(workflow.LocalActivityOptions{ScheduleToCloseTimeout: time.Second * 30}, *v.GetOptions())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteLocalRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal(workflow.LocalActivityOptions{ScheduleToCloseTimeout: time.Second * 30}, *v.GetOptions())
	}
}

func (s *UnitTestSuite) TestActivityExecuteReply() {
	var message messages.IProxyMessage = messages.NewActivityExecuteReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityExecuteRequest() {
	var message messages.IProxyMessage = messages.NewActivityExecuteRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteRequest); ok {
		s.Equal(messagetypes.ActivityExecuteReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetArgs())
		s.Nil(v.GetOptions())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())

		opts := workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Second * 30,
			WaitForCancellation:    false,
			TaskList:               "my-tasklist",
		}
		v.SetOptions(&opts)
		s.Equal(workflow.ActivityOptions{ScheduleToCloseTimeout: time.Second * 30, WaitForCancellation: false, TaskList: "my-tasklist"}, *v.GetOptions())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal(workflow.ActivityOptions{ScheduleToCloseTimeout: time.Second * 30, WaitForCancellation: false, TaskList: "my-tasklist"}, *v.GetOptions())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityExecuteRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal(workflow.ActivityOptions{ScheduleToCloseTimeout: time.Second * 30, WaitForCancellation: false, TaskList: "my-tasklist"}, *v.GetOptions())
	}
}

func (s *UnitTestSuite) TestActivityGetHeartbeatDetailsReply() {
	var message messages.IProxyMessage = messages.NewActivityGetHeartbeatDetailsReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetHeartbeatDetailsReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetDetails())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetDetails([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetHeartbeatDetailsReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetHeartbeatDetailsReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityGetHeartbeatDetailsRequest() {
	var message messages.IProxyMessage = messages.NewActivityGetHeartbeatDetailsRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetHeartbeatDetailsRequest); ok {
		s.Equal(messagetypes.ActivityGetHeartbeatDetailsReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetHeartbeatDetailsRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetHeartbeatDetailsRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestActivityGetInfoReply() {
	var message messages.IProxyMessage = messages.NewActivityGetInfoReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetInfoReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetInfo())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		info := activity.Info{
			TaskList:       "my-tasklist",
			Attempt:        4,
			ActivityID:     "my-activity",
			WorkflowDomain: "my-domain",
			ActivityType:   activity.Type{Name: "activity"},
		}
		v.SetInfo(&info)
		s.Equal(activity.Info{TaskList: "my-tasklist", Attempt: 4, ActivityID: "my-activity", WorkflowDomain: "my-domain", ActivityType: activity.Type{Name: "activity"}}, *v.GetInfo())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetInfoReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(activity.Info{TaskList: "my-tasklist", Attempt: 4, ActivityID: "my-activity", WorkflowDomain: "my-domain", ActivityType: activity.Type{Name: "activity"}}, *v.GetInfo())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetInfoReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(activity.Info{TaskList: "my-tasklist", Attempt: 4, ActivityID: "my-activity", WorkflowDomain: "my-domain", ActivityType: activity.Type{Name: "activity"}}, *v.GetInfo())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityGetInfoRequest() {
	var message messages.IProxyMessage = messages.NewActivityGetInfoRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetInfoRequest); ok {
		s.Equal(messagetypes.ActivityGetInfoReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetInfoRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityGetInfoRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestActivityHasHeartbeatDetailsReply() {
	var message messages.IProxyMessage = messages.NewActivityHasHeartbeatDetailsReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityHasHeartbeatDetailsReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.False(v.GetHasDetails())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetHasDetails(true)
		s.Equal(true, v.GetHasDetails())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityHasHeartbeatDetailsReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(true, v.GetHasDetails())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityHasHeartbeatDetailsReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(true, v.GetHasDetails())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityHasHeartbeatDetailsRequest() {
	var message messages.IProxyMessage = messages.NewActivityHasHeartbeatDetailsRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityHasHeartbeatDetailsRequest); ok {
		s.Equal(messagetypes.ActivityHasHeartbeatDetailsReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityHasHeartbeatDetailsRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityHasHeartbeatDetailsRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
	}
}

func (s *UnitTestSuite) TestActivityInvokeLocalReply() {
	var message messages.IProxyMessage = messages.NewActivityInvokeLocalReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeLocalReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeLocalReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeLocalReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityInvokeLocalRequest() {
	var message messages.IProxyMessage = messages.NewActivityInvokeLocalRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeLocalRequest); ok {
		s.Equal(messagetypes.ActivityInvokeLocalReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetArgs())
		s.Equal(int64(0), v.GetActivityTypeID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())

		v.SetActivityTypeID(int64(666))
		s.Equal(int64(666), v.GetActivityTypeID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeLocalRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		v.SetActivityTypeID(int64(666))
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeLocalRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		v.SetActivityTypeID(int64(666))
	}
}

func (s *UnitTestSuite) TestActivityInvokeReply() {
	var message messages.IProxyMessage = messages.NewActivityInvokeReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetResult())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetResult([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetResult())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityInvokeRequest() {
	var message messages.IProxyMessage = messages.NewActivityInvokeRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeRequest); ok {
		s.Equal(messagetypes.ActivityInvokeReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetArgs())
		s.Nil(v.GetActivity())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetArgs([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())

		activity := "my-activity"
		v.SetActivity(&activity)
		s.Equal("my-activity", *v.GetActivity())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal("my-activity", *v.GetActivity())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityInvokeRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetArgs())
		s.Equal("my-activity", *v.GetActivity())
	}
}

func (s *UnitTestSuite) TestActivityRecordHeartbeatReply() {
	var message messages.IProxyMessage = messages.NewActivityRecordHeartbeatReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRecordHeartbeatReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRecordHeartbeatReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRecordHeartbeatReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityRecordHeartbeatRequest() {
	var message messages.IProxyMessage = messages.NewActivityRecordHeartbeatRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRecordHeartbeatRequest); ok {
		s.Equal(messagetypes.ActivityRecordHeartbeatReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetDetails())
		s.Nil(v.GetTaskToken())
		s.Nil(v.GetDomain())
		s.Nil(v.GetWorkflowID())
		s.Nil(v.GetRunID())
		s.Nil(v.GetActivityID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetDetails([]byte{0, 1, 2, 3, 4})
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())

		v.SetTaskToken([]byte{5, 6, 7, 8, 9})
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetTaskToken())

		domain := "my-domain"
		workflowID := "my-workflow"
		runID := "my-workflowrun"
		activityID := "my-activity"

		v.SetDomain(&domain)
		v.SetWorkflowID(&workflowID)
		v.SetRunID(&runID)
		v.SetActivityID(&activityID)

		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-workflowrun", *v.GetRunID())
		s.Equal("my-activity", *v.GetActivityID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRecordHeartbeatRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetTaskToken())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-workflowrun", *v.GetRunID())
		s.Equal("my-activity", *v.GetActivityID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRecordHeartbeatRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal([]byte{0, 1, 2, 3, 4}, v.GetDetails())
		s.Equal([]byte{5, 6, 7, 8, 9}, v.GetTaskToken())
		s.Equal("my-domain", *v.GetDomain())
		s.Equal("my-workflow", *v.GetWorkflowID())
		s.Equal("my-workflowrun", *v.GetRunID())
		s.Equal("my-activity", *v.GetActivityID())
	}
}

func (s *UnitTestSuite) TestActivityRegisterReply() {
	var message messages.IProxyMessage = messages.NewActivityRegisterReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRegisterReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRegisterReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRegisterReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityRegisterRequest() {
	var message messages.IProxyMessage = messages.NewActivityRegisterRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRegisterRequest); ok {
		s.Equal(messagetypes.ActivityRegisterReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetName())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		name := "my-activity"
		v.SetName(&name)
		s.Equal("my-activity", *v.GetName())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRegisterRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-activity", *v.GetName())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRegisterRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-activity", *v.GetName())
	}
}

func (s *UnitTestSuite) TestActivityStoppingReply() {
	var message messages.IProxyMessage = messages.NewActivityStoppingReply()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityStoppingReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetError(cadenceerrors.NewCadenceError(errors.New("foo")))
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityStoppingReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityStoppingReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.NewCadenceError(errors.New("foo"), cadenceerrors.Custom), v.GetError())
	}
}

func (s *UnitTestSuite) TestActivityStoppingRequest() {
	var message messages.IProxyMessage = messages.NewActivityStoppingRequest()
	proxyMessage := message.GetProxyMessage()

	serializedMessage, err := proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityStoppingRequest); ok {
		s.Equal(messagetypes.ActivityStoppingReply, v.ReplyType)
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetActivityID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		name := "my-activity"
		v.SetActivityID(&name)
		s.Equal("my-activity", *v.GetActivityID())
	}

	proxyMessage = message.GetProxyMessage()
	serializedMessage, err = proxyMessage.Serialize(false)
	s.NoError(err)

	message, err = messages.Deserialize(bytes.NewBuffer(serializedMessage), false)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityStoppingRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-activity", *v.GetActivityID())
	}

	message, err = s.echoToConnection(message)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityStoppingRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal("my-activity", *v.GetActivityID())
	}
}

// --------------------------------------------------------------------------
// Test the base messages (messages.ProxyMessage, messages.ProxyRequest, messages.ProxyReply,
// messages.WorkflowRequest, messages.WorkflowReply)

// TestProxyMessage ensures that we can
// serializate and deserialize a base messages.ProxyMessage
func (s *UnitTestSuite) TestProxyMessage() {

	// empty buffer to create empty proxy message
	buf := bytes.NewBuffer(make([]byte, 0))
	message, err := messages.Deserialize(buf, true)
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ProxyMessage); ok {
		s.Equal(messagetypes.Unspecified, v.Type)
		s.Empty(v.Properties)
		s.Empty(v.Attachments)
	}

	// new proxy message to fill
	message = messages.NewProxyMessage()

	if v, ok := message.(*messages.ProxyMessage); ok {

		// fill the properties map
		p1 := "1"
		p2 := "2"
		p3 := ""
		v.Properties["One"] = &p1
		v.Properties["Two"] = &p2
		v.Properties["Empty"] = &p3
		v.Properties["Nil"] = nil

		v.SetJSONProperty("Error", cadenceerrors.NewCadenceError(errors.New("foo")))

		b, err := base64.StdEncoding.DecodeString("c29tZSBkYXRhIHdpdGggACBhbmQg77u/")
		s.NoError(err)
		v.SetBytesProperty("Bytes", b)

		// fill the attachments map
		v.Attachments = append(v.Attachments, []byte{0, 1, 2, 3, 4})
		v.Attachments = append(v.Attachments, make([]byte, 0))
		v.Attachments = append(v.Attachments, nil)

		// serialize the new message
		serializedMessage, err := v.Serialize(true)
		s.NoError(err)

		// byte buffer to deserialize
		buf = bytes.NewBuffer(serializedMessage)
	}

	// deserialize
	message, err = messages.Deserialize(buf, true)
	s.NoError(err)

	// check that the values are the same
	if v, ok := message.(*messages.ProxyMessage); ok {

		// type and property values
		s.Equal(messagetypes.Unspecified, v.Type)
		s.Equal(6, len(v.Properties))
		s.Equal("1", *v.Properties["One"])
		s.Equal("2", *v.Properties["Two"])
		s.Empty(v.Properties["Empty"])
		s.Nil(v.Properties["Nil"])
		s.Equal("c29tZSBkYXRhIHdpdGggACBhbmQg77u/", *v.Properties["Bytes"])

		cadenceError := cadenceerrors.NewCadenceErrorEmpty()
		v.GetJSONProperty("Error", cadenceError)
		s.Equal("foo", *cadenceError.String)
		s.Equal(cadenceerrors.Custom, cadenceError.GetType())

		// attachment values
		s.Equal(3, len(v.Attachments))
		s.Equal([]byte{0, 1, 2, 3, 4}, v.Attachments[0])
		s.Empty(v.Attachments[1])
		s.Nil(v.Attachments[2])
	}
}
func (s *UnitTestSuite) TestProxyRequest() {

	// Ensure that we can serialize and deserialize request messages

	buf := bytes.NewBuffer(make([]byte, 0))
	message, err := messages.Deserialize(buf, true, "ProxyRequest")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ProxyRequest); ok {
		s.Equal(messagetypes.Unspecified, v.GetType())
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(messagetypes.Unspecified, v.GetReplyType())
		s.False(v.GetIsCancellable())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetIsCancellable(true)
		s.True(v.GetIsCancellable())

		// serialize the new message
		serializedMessage, err := v.Serialize(true)
		s.NoError(err)

		// byte buffer to deserialize
		buf = bytes.NewBuffer(serializedMessage)
	}

	message, err = messages.Deserialize(buf, true, "ProxyRequest")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ProxyRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(messagetypes.Unspecified, v.GetReplyType())
		s.True(v.GetIsCancellable())
	}
}

func (s *UnitTestSuite) TestProxyReply() {

	// Ensure that we can serialize and deserialize reply messages

	buf := bytes.NewBuffer(make([]byte, 0))
	message, err := messages.Deserialize(buf, true, "ProxyReply")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ProxyReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip
		v.SetRequestID(int64(555))
		v.SetError(cadenceerrors.NewCadenceError(errors.New("MyError")))

		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.Custom, v.GetError().GetType())
		s.Equal("MyError", *v.GetError().String)

		// serialize the new message
		serializedMessage, err := v.Serialize(true)
		s.NoError(err)

		// byte buffer to deserialize
		buf = bytes.NewBuffer(serializedMessage)
	}

	message, err = messages.Deserialize(buf, true, "ProxyReply")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ProxyReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.Custom, v.GetError().GetType())
		s.Equal("MyError", *v.GetError().String)
	}
}

func (s *UnitTestSuite) TestWorkflowRequest() {

	// Ensure that we can serialize and deserialize request messages

	buf := bytes.NewBuffer(make([]byte, 0))
	message, err := messages.Deserialize(buf, true, "WorkflowRequest")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRequest); ok {
		s.Equal(v.ReplyType, messagetypes.Unspecified)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(555))
		s.Equal(int64(555), v.GetContextID())

		// serialize the new message
		serializedMessage, err := v.Serialize(true)
		s.NoError(err)

		// byte buffer to deserialize
		buf = bytes.NewBuffer(serializedMessage)
	}

	message, err = messages.Deserialize(buf, true, "WorkflowRequest")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(555), v.GetContextID())
	}
}

func (s *UnitTestSuite) TestWorkflowReply() {

	// Ensure that we can serialize and deserialize reply messages

	buf := bytes.NewBuffer(make([]byte, 0))
	message, err := messages.Deserialize(buf, true, "WorkflowReply")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())

		// Round-trip
		v.SetRequestID(int64(555))
		v.SetError(cadenceerrors.NewCadenceError(errors.New("MyError")))

		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.Custom, v.GetError().GetType())
		s.Equal("MyError", *v.GetError().String)

		// serialize the new message
		serializedMessage, err := v.Serialize(true)
		s.NoError(err)

		// byte buffer to deserialize
		buf = bytes.NewBuffer(serializedMessage)
	}

	message, err = messages.Deserialize(buf, true, "WorkflowReply")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.WorkflowReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.Custom, v.GetError().GetType())
		s.Equal("MyError", *v.GetError().String)
	}
}

func (s *UnitTestSuite) TestActivityRequest() {

	// Ensure that we can serialize and deserialize request messages

	buf := bytes.NewBuffer(make([]byte, 0))
	message, err := messages.Deserialize(buf, true, "ActivityRequest")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRequest); ok {
		s.Equal(v.ReplyType, messagetypes.Unspecified)
		s.Equal(int64(0), v.GetRequestID())
		s.Equal(int64(0), v.GetContextID())

		// Round-trip

		v.SetRequestID(int64(555))
		s.Equal(int64(555), v.GetRequestID())

		v.SetContextID(int64(555))
		s.Equal(int64(555), v.GetContextID())

		// serialize the new message
		serializedMessage, err := v.Serialize(true)
		s.NoError(err)

		// byte buffer to deserialize
		buf = bytes.NewBuffer(serializedMessage)
	}

	message, err = messages.Deserialize(buf, true, "ActivityRequest")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityRequest); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(int64(555), v.GetContextID())
	}
}

func (s *UnitTestSuite) TestActivityReply() {

	// Ensure that we can serialize and deserialize reply messages

	buf := bytes.NewBuffer(make([]byte, 0))
	message, err := messages.Deserialize(buf, true, "ActivityReply")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityReply); ok {
		s.Equal(int64(0), v.GetRequestID())
		s.Nil(v.GetError())
		s.Equal(int64(0), v.GetActivityContextID())

		// Round-trip
		v.SetRequestID(int64(555))
		v.SetError(cadenceerrors.NewCadenceError(errors.New("MyError")))

		v.SetActivityContextID(int64(555))
		s.Equal(int64(555), v.GetActivityContextID())

		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.Custom, v.GetError().GetType())
		s.Equal("MyError", *v.GetError().String)

		// serialize the new message
		serializedMessage, err := v.Serialize(true)
		s.NoError(err)

		// byte buffer to deserialize
		buf = bytes.NewBuffer(serializedMessage)
	}

	message, err = messages.Deserialize(buf, true, "ActivityReply")
	s.NoError(err)
	s.NotNil(message)

	if v, ok := message.(*messages.ActivityReply); ok {
		s.Equal(int64(555), v.GetRequestID())
		s.Equal(cadenceerrors.Custom, v.GetError().GetType())
		s.Equal(int64(555), v.GetActivityContextID())
		s.Equal("MyError", *v.GetError().String)
	}
}

// --------------------------------------------------------------------------
// Test the messages.ProxyMessage helper methods

func (s *UnitTestSuite) TestPropertyHelpers() {

	// verify that the property helper methods work as expected
	message := messages.NewProxyMessage()

	// verify that non-existant property values return the default for the requested type
	s.Nil(message.GetStringProperty("foo"))
	s.Equal(int32(0), message.GetIntProperty("foo"))
	s.Equal(int64(0), message.GetLongProperty("foo"))
	s.False(message.GetBoolProperty("foo"))
	s.Equal(0.0, message.GetDoubleProperty("foo"))
	s.Equal(times.ParseIso8601UTC(times.ToIso8601UTC(time.Time{})), message.GetDateTimeProperty("foo"))
	s.Equal(time.Duration(0)*time.Nanosecond, message.GetTimeSpanProperty("foo"))

	// Verify that we can override default values for non-existant properties.

	s.Equal(int32(123), message.GetIntProperty("foo", int32(123)))
	s.Equal(int64(456), message.GetLongProperty("foo", int64(456)))
	s.True(message.GetBoolProperty("foo", true))
	s.Equal(float64(123.456), message.GetDoubleProperty("foo", float64(123.456)))
	s.Equal(time.Date(2019, time.April, 14, 0, 0, 0, 0, time.UTC), message.GetDateTimeProperty("foo", time.Date(2019, time.April, 14, 0, 0, 0, 0, time.UTC)))
	s.Equal(time.Second*123, message.GetTimeSpanProperty("foo", time.Second*123))

	// verify that we can write and then read properties
	str := "bar"
	message.SetStringProperty("foo", &str)
	s.Equal("bar", *message.GetStringProperty("foo"))

	message.SetIntProperty("foo", int32(123))
	s.Equal(int32(123), message.GetIntProperty("foo"))

	message.SetLongProperty("foo", int64(456))
	s.Equal(int64(456), message.GetLongProperty("foo"))

	message.SetBoolProperty("foo", true)
	s.True(message.GetBoolProperty("foo"))

	message.SetDoubleProperty("foo", 123.456)
	s.Equal(123.456, message.GetDoubleProperty("foo"))

	date := time.Date(2019, time.April, 14, 0, 0, 0, 0, time.UTC)
	message.SetDateTimeProperty("foo", date)
	s.Equal(date, message.GetDateTimeProperty("foo"))

	message.SetTimeSpanProperty("foo", time.Second*123)
	s.Equal(time.Second*123, message.GetTimeSpanProperty("foo"))

	jsonStr := "{\"String\":\"john\",\"Details\":\"22\",\"Type\":\"mca\"}"
	cadenceError := cadenceerrors.NewCadenceErrorEmpty()
	cadenceErrorCheck := cadenceerrors.NewCadenceErrorEmpty()
	err := json.Unmarshal([]byte(jsonStr), cadenceError)
	if err != nil {
		panic(err)
	}

	message.SetJSONProperty("foo", cadenceError)
	message.GetJSONProperty("foo", cadenceErrorCheck)
	s.Equal(cadenceError, cadenceErrorCheck)

	b, err := base64.StdEncoding.DecodeString("c29tZSBkYXRhIHdpdGggACBhbmQg77u/")
	s.NoError(err)
	message.SetBytesProperty("foo", b)
	s.Equal(b, message.GetBytesProperty("foo"))
}
