package ocpp16_test

import (
	"fmt"
	"github.com/lorenzodonini/go-ocpp/ocpp1.6"
	"github.com/lorenzodonini/go-ocpp/ocppj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

// Utility functions
func getBootNotificationRequest(t *testing.T, request ocppj.Request) *ocpp16.BootNotificationRequest {
	assert.NotNil(t, request)
	result := request.(*ocpp16.BootNotificationRequest)
	assert.NotNil(t, result)
	assert.IsType(t, &ocpp16.BootNotificationRequest{}, result)
	return result
}

func getBootNotificationConfirmation(t *testing.T, confirmation ocppj.Confirmation) *ocpp16.BootNotificationConfirmation {
	assert.NotNil(t, confirmation)
	result := confirmation.(*ocpp16.BootNotificationConfirmation)
	assert.NotNil(t, result)
	assert.IsType(t, &ocpp16.BootNotificationConfirmation{}, result)
	return result
}

// Tests
func (suite *OcppV16TestSuite) TestBootNotificationRequestValidation() {
	t := suite.T()
	var requestTable = []RequestTestEntry{
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointVendor: "test"}, true},
		{ocpp16.BootNotificationRequest{ChargeBoxSerialNumber: "test", ChargePointModel: "test", ChargePointSerialNumber: "number", ChargePointVendor: "test", FirmwareVersion: "version", Iccid: "test", Imsi: "test"}, true},
		{ocpp16.BootNotificationRequest{ChargeBoxSerialNumber: "test", ChargePointSerialNumber: "number", ChargePointVendor: "test", FirmwareVersion: "version", Iccid: "test", Imsi: "test"}, false},
		{ocpp16.BootNotificationRequest{ChargeBoxSerialNumber: "test", ChargePointModel: "test", ChargePointSerialNumber: "number", FirmwareVersion: "version", Iccid: "test", Imsi: "test"}, false},
		{ocpp16.BootNotificationRequest{ChargeBoxSerialNumber: ">25.......................", ChargePointModel: "test", ChargePointVendor: "test"}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: ">20..................", ChargePointVendor: "test"}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointSerialNumber: ">25.......................", ChargePointVendor: "test"}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointVendor: ">20.................."}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointVendor: "test", FirmwareVersion: ">50................................................"}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointVendor: "test", Iccid: ">20.................."}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointVendor: "test", Imsi: ">20.................."}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointVendor: "test", MeterSerialNumber: ">25......................."}, false},
		{ocpp16.BootNotificationRequest{ChargePointModel: "test", ChargePointVendor: "test", MeterType: ">25......................."}, false},
	}
	ExecuteRequestTestTable(t, requestTable)
}

func (suite *OcppV16TestSuite) TestBootNotificationConfirmationValidation() {
	t := suite.T()
	var confirmationTable = []ConfirmationTestEntry{
		{ocpp16.BootNotificationConfirmation{CurrentTime: ocpp16.DateTime{Time: time.Now()}, Interval: 60, Status: ocpp16.RegistrationStatusAccepted}, true},
		{ocpp16.BootNotificationConfirmation{CurrentTime: ocpp16.DateTime{Time: time.Now()}, Interval: 60, Status: ocpp16.RegistrationStatusPending}, true},
		{ocpp16.BootNotificationConfirmation{CurrentTime: ocpp16.DateTime{Time: time.Now()}, Interval: 60, Status: ocpp16.RegistrationStatusRejected}, true},
		{ocpp16.BootNotificationConfirmation{CurrentTime: ocpp16.DateTime{Time: time.Now()}, Interval: 60}, false},
		{ocpp16.BootNotificationConfirmation{CurrentTime: ocpp16.DateTime{Time: time.Now()}, Status: ocpp16.RegistrationStatusAccepted}, false},
		{ocpp16.BootNotificationConfirmation{Interval: 60, Status: ocpp16.RegistrationStatusAccepted}, false},
		{ocpp16.BootNotificationConfirmation{CurrentTime: ocpp16.DateTime{Time: time.Now()}, Interval: -1, Status: ocpp16.RegistrationStatusAccepted}, false},
		//TODO: incomplete list, see core.go
	}
	ExecuteConfirmationTestTable(t, confirmationTable)
}

func (suite *OcppV16TestSuite) TestBootNotificationE2EMocked() {
	t := suite.T()
	wsId := "test_id"
	messageId := "1234"
	wsUrl := "someUrl"
	interval := 60
	chargePointModel := "model1"
	chargePointVendor := "ABL"
	registrationStatus := ocpp16.RegistrationStatusAccepted
	currentTime := ocpp16.DateTime{Time: time.Now()}
	requestJson := fmt.Sprintf(`[2,"%v","%v",{"chargePointModel":"%v","chargePointVendor":"%v"}]`, messageId, ocpp16.BootNotificationFeatureName, chargePointModel, chargePointVendor)
	responseJson := fmt.Sprintf(`[3,"%v",{"currentTime":"%v","interval":%v,"status":"%v"}]`, messageId, currentTime.Time.Format(ocpp16.ISO8601), interval, registrationStatus)
	bootNotificationConfirmation := ocpp16.NewBootNotificationConfirmation(currentTime, interval, registrationStatus)
	channel := NewMockWebSocket(wsId)

	coreListener := MockCentralSystemCoreListener{}
	coreListener.On("OnBootNotification", mock.AnythingOfType("string"), mock.Anything).Return(bootNotificationConfirmation, nil)
	setupDefaultCentralSystemHandlers(suite, coreListener, expectedCentralSystemOptions{clientId: wsId, rawWrittenMessage: []byte(responseJson), forwardWrittenMessage: true})
	setupDefaultChargePointHandlers(suite, nil, expectedChargePointOptions{serverUrl: wsUrl, clientId: wsId, createChannelOnStart: true, channel: channel, rawWrittenMessage: []byte(requestJson), forwardWrittenMessage: true})
	// Run test
	suite.centralSystem.Start(8887, "somePath")
	err := suite.chargePoint.Start(wsUrl)
	assert.Nil(t, err)
	confirmation, protoErr, err := suite.chargePoint.BootNotification(chargePointModel, chargePointVendor)
	assert.Nil(t, err)
	assert.Nil(t, protoErr)
	assert.NotNil(t, confirmation)
	assert.Equal(t, registrationStatus, confirmation.Status)
	assert.Equal(t, interval, confirmation.Interval)
	assertDateTimeEquality(t, currentTime, confirmation.CurrentTime)
}

func (suite *OcppV16TestSuite) TestBootNotificationInvalidEndpoint() {
	t := suite.T()
	wsId := "test_id"
	messageId := "1234"
	chargePointModel := "model1"
	chargePointVendor := "ABL"
	expectedError := fmt.Sprintf("unsupported action %v on central system, cannot send request", ocpp16.BootNotificationFeatureName)
	requestJson := fmt.Sprintf(`[2,"%v","%v",{"chargePointModel":"%v","chargePointVendor":"%v"}]`, messageId, ocpp16.BootNotificationFeatureName, chargePointModel, chargePointVendor)

	setupDefaultCentralSystemHandlers(suite, nil, expectedCentralSystemOptions{clientId: wsId, rawWrittenMessage: []byte(requestJson), forwardWrittenMessage: false})
	// Run test
	bootNotificationRequest := ocpp16.NewBootNotificationRequest(chargePointModel, chargePointVendor)
	suite.centralSystem.Start(8887, "somePath")
	err := suite.centralSystem.SendRequestAsync(wsId, bootNotificationRequest, func(confirmation ocppj.Confirmation, callError *ocppj.ProtoError) {
		t.Fail()
	})
	assert.Error(t, err)
	assert.Equal(t, expectedError, err.Error())
}

func (suite *OcppV16TestSuite) TestBootNotificationInvalidEndpointResponse() {
	t := suite.T()
	wsId := "test_id"
	messageId := defaultMessageId
	wsUrl := "someUrl"
	chargePointModel := "model1"
	chargePointVendor := "ABL"
	errorDescription := fmt.Sprintf("unsupported action %v on charge point", ocpp16.BootNotificationFeatureName)
	requestJson := fmt.Sprintf(`[2,"%v","%v",{"chargePointModel":"%v","chargePointVendor":"%v"}]`, messageId, ocpp16.BootNotificationFeatureName, chargePointModel, chargePointVendor)
	errorJson := fmt.Sprintf(`[4,"%v","%v","%v",null]`, messageId, ocppj.NotSupported, errorDescription)
	channel := NewMockWebSocket(wsId)

	coreListener := MockChargePointCoreListener{}
	setupDefaultCentralSystemHandlers(suite, nil, expectedCentralSystemOptions{clientId: wsId, rawWrittenMessage: []byte(requestJson), forwardWrittenMessage: false})
	setupDefaultChargePointHandlers(suite, coreListener, expectedChargePointOptions{serverUrl: wsUrl, clientId: wsId, createChannelOnStart: true, channel: channel, rawWrittenMessage: []byte(errorJson), forwardWrittenMessage: true})
	suite.ocppjCentralSystem.SetErrorHandler(func(chargePointId string, errorCode ocppj.ErrorCode, description string, details interface{}, requestId string) {
		assert.Equal(t, messageId, requestId)
		assert.Equal(t, wsId, chargePointId)
		assert.Equal(t, ocppj.NotSupported, errorCode)
		assert.Equal(t, errorDescription, description)
		assert.Nil(t, details)
	})
	// Mock pending request
	pendingRequest := ocpp16.NewBootNotificationRequest(chargePointModel, chargePointVendor)
	suite.ocppjCentralSystem.AddPendingRequest(messageId, pendingRequest)
	// Run test
	suite.centralSystem.Start(8887, "somePath")
	err := suite.chargePoint.Start(wsUrl)
	assert.Nil(t, err)
	err = suite.mockWsClient.MessageHandler([]byte(requestJson))
	assert.Nil(t, err)
}
