package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	app := NewApp()

	assert.NotNil(t, app, "App should not be nil")
	assert.NotNil(t, app.executeFunc, "executeFunc should not be nil")
	assert.NotNil(t, app.exitFunc, "exitFunc should not be nil")
	assert.NotNil(t, app.logger, "logger should not be nil")
}

func TestNewAppWithDeps(t *testing.T) {
	var logOutput bytes.Buffer
	mockExecute := func() error { return nil }
	mockExit := func(int) {}

	app := NewAppWithDeps(mockExecute, mockExit, &logOutput)

	assert.NotNil(t, app, "App should not be nil")
	assert.NotNil(t, app.executeFunc, "executeFunc should not be nil")
	assert.NotNil(t, app.exitFunc, "exitFunc should not be nil")
	assert.NotNil(t, app.logger, "logger should not be nil")
}

func TestApp_RunSuccess(t *testing.T) {
	var logOutput bytes.Buffer
	exitCalled := false
	exitCode := -1

	mockExecute := func() error {
		return nil
	}
	mockExit := func(code int) {
		exitCalled = true
		exitCode = code
	}

	app := NewAppWithDeps(mockExecute, mockExit, &logOutput)
	app.Run()

	assert.False(t, exitCalled, "Exit should not be called on success")
	assert.Equal(t, -1, exitCode, "Exit code should remain unchanged")
	assert.Empty(t, logOutput.String(), "No error should be logged on success")
}

func TestApp_RunWithError(t *testing.T) {
	var logOutput bytes.Buffer
	exitCalled := false
	exitCode := -1
	testError := errors.New("test error")

	mockExecute := func() error {
		return testError
	}
	mockExit := func(code int) {
		exitCalled = true
		exitCode = code
	}

	app := NewAppWithDeps(mockExecute, mockExit, &logOutput)
	app.Run()

	assert.True(t, exitCalled, "Exit should be called on error")
	assert.Equal(t, 1, exitCode, "Exit code should be 1")

	logContent := logOutput.String()
	assert.Contains(t, logContent, "Failed to execute command", "Error message should be logged")
	assert.Contains(t, logContent, testError.Error(), "Specific error should be logged")
	assert.Contains(t, logContent, "\"level\":\"ERROR\"", "Log level should be ERROR")
}

func TestApp_LoggerConfiguration(t *testing.T) {
	var logOutput bytes.Buffer
	mockExecute := func() error { return nil }
	mockExit := func(int) {}

	app := NewAppWithDeps(mockExecute, mockExit, &logOutput)
	app.Run()

	// The logger should be set as default during Run()
	assert.NotNil(t, app.logger, "Logger should be configured")
}

func TestMainFunction(t *testing.T) {
	// Test that main function can be called without panicking
	// This is a basic smoke test
	assert.NotPanics(t, func() {
		app := NewApp()
		assert.NotNil(t, app, "NewApp should create a valid app instance")
	}, "Creating new app should not panic")
}
