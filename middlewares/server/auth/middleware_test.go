package auth

import (
	"github.com/mysterium/node/openvpn/management"
	"github.com/stretchr/testify/assert"
	"testing"
)

type fakeAuthenticatorStub struct {
	username      string
	password      string
	called        bool
	authenticated bool
}

func (f *fakeAuthenticatorStub) fakeAuthenticator(username, password string) (bool, error) {
	f.called = true
	f.username = username
	f.password = password
	return f.authenticated, nil
}

func newFakeAuthenticatorStub() fakeAuthenticatorStub {
	return fakeAuthenticatorStub{}
}

func Test_Factory(t *testing.T) {

	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
	assert.NotNil(t, middleware)
}

func Test_ConsumeLineSkips(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">SOME_LINE_TO_BE_DELIVERED"},
		{">ANOTHER_LINE_TO_BE_DELIVERED"},
		{">PASSWORD"},
		{">USERNAME"},
	}
	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.False(t, consumed, test.line)
	}
}

func Test_ConsumeLineTakes(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:REAUTH,0,0"},
		{">CLIENT:CONNECT,0,0"},
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,username=username"},
	}

	fas := newFakeAuthenticatorStub()
	middleware := NewMiddleware(fas.fakeAuthenticator)
	mockWriter := &management.MockConnection{}
	middleware.Start(mockWriter)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
}

func Test_ConsumeLineAuthState(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:REAUTH,0,0"},
		{">CLIENT:CONNECT,0,0"},
	}

	for _, test := range tests {
		fas := newFakeAuthenticatorStub()
		middleware := NewMiddleware(fas.fakeAuthenticator)
		mockWritter := &management.MockConnection{}
		middleware.Start(mockWritter)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
}

func Test_ConsumeLineNotAuthState(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,username=username"},
	}

	for _, test := range tests {
		fas := newFakeAuthenticatorStub()
		middleware := NewMiddleware(fas.fakeAuthenticator)
		mockWriter := &management.MockConnection{}
		middleware.Start(mockWriter)

		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
		assert.False(t, fas.called)
	}
}

func Test_ConsumeLineAuthTrueChecker(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:CONNECT,1,2"},
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,username=username1"},
		{">CLIENT:ENV,END"},
	}
	fas := newFakeAuthenticatorStub()
	fas.authenticated = true
	middleware := NewMiddleware(fas.fakeAuthenticator)
	mockWriter := &management.MockConnection{}
	middleware.Start(mockWriter)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
	assert.True(t, fas.called)
	assert.Equal(t, "username1", fas.username)
	assert.Equal(t, "12341234", fas.password)
	assert.Equal(t, "client-auth-nt 1 2", mockWriter.LastLine)
}

func Test_ConsumeLineAuthFalseChecker(t *testing.T) {
	var tests = []struct {
		line string
	}{
		{">CLIENT:CONNECT,3,4"},
		{">CLIENT:ENV,username=bad"},
		{">CLIENT:ENV,password=12341234"},
		{">CLIENT:ENV,END"},
	}
	fas := newFakeAuthenticatorStub()
	fas.authenticated = false
	middleware := NewMiddleware(fas.fakeAuthenticator)
	mockWriter := &management.MockConnection{}
	middleware.Start(mockWriter)

	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		assert.NoError(t, err, test.line)
		assert.True(t, consumed, test.line)
	}
	assert.Equal(t, "client-deny 3 4 wrong username or password", mockWriter.LastLine)
}
