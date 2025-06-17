package taskmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTaskExecutor struct {
	mock.Mock
}

func (m *MockTaskExecutor) Configure(envFunc func(string) string) error {
	args := m.Called(envFunc)
	return args.Error(0)
}

func (m *MockTaskExecutor) Execute(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNew(t *testing.T) {
	preTasks := []TaskExecutor{&MockTaskExecutor{}}
	postTasks := []TaskExecutor{&MockTaskExecutor{}}

	tm := New(preTasks, postTasks)

	assert.NotNil(t, tm)
	assert.Equal(t, preTasks, tm.preDeployTasks)
	assert.Equal(t, postTasks, tm.postDeployTasks)
}

func TestExecute_PreDeploy(t *testing.T) {
	// Setup
	mockTask := &MockTaskExecutor{}
	tm := New([]TaskExecutor{mockTask}, nil)
	ctx := context.Background()

	// Configure expectations
	mockTask.On("Configure", mock.Anything).Return(nil)
	mockTask.On("Execute", ctx).Return(nil)

	// Execute
	err := tm.Execute(ctx, false)

	// Assert
	assert.NoError(t, err)
	mockTask.AssertExpectations(t)
}

func TestExecute_PostDeploy(t *testing.T) {
	// Setup
	mockTask := &MockTaskExecutor{}
	tm := New(nil, []TaskExecutor{mockTask})
	ctx := context.Background()

	// Configure expectations
	mockTask.On("Configure", mock.Anything).Return(nil)
	mockTask.On("Execute", ctx).Return(nil)

	// Execute
	err := tm.Execute(ctx, true)

	// Assert
	assert.NoError(t, err)
	mockTask.AssertExpectations(t)
}

func TestExecute_ConfigureError(t *testing.T) {
	// Setup
	mockTask := &MockTaskExecutor{}
	tm := New([]TaskExecutor{mockTask}, nil)
	ctx := context.Background()
	expectedErr := errors.New("configuration error")

	// Configure expectations
	mockTask.On("Configure", mock.Anything).Return(expectedErr)

	// Execute
	err := tm.Execute(ctx, false)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockTask.AssertExpectations(t)
}

func TestExecute_ExecuteError(t *testing.T) {
	// Setup
	mockTask := &MockTaskExecutor{}
	tm := New([]TaskExecutor{mockTask}, nil)
	ctx := context.Background()
	expectedErr := errors.New("execution error")

	// Configure expectations
	mockTask.On("Configure", mock.Anything).Return(nil)
	mockTask.On("Execute", ctx).Return(expectedErr)

	// Execute
	err := tm.Execute(ctx, false)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockTask.AssertExpectations(t)
}

func TestExecute_MultipleTasks(t *testing.T) {
	// Setup
	mockTask1 := &MockTaskExecutor{}
	mockTask2 := &MockTaskExecutor{}
	tm := New([]TaskExecutor{mockTask1, mockTask2}, nil)
	ctx := context.Background()

	// Configure expectations
	mockTask1.On("Configure", mock.Anything).Return(nil)
	mockTask1.On("Execute", ctx).Return(nil)
	mockTask2.On("Configure", mock.Anything).Return(nil)
	mockTask2.On("Execute", ctx).Return(nil)

	// Execute
	err := tm.Execute(ctx, false)

	// Assert
	assert.NoError(t, err)
	mockTask1.AssertExpectations(t)
	mockTask2.AssertExpectations(t)
}

func TestExecute_EmptyTasks(t *testing.T) {
	// Setup
	tm := New(nil, nil)
	ctx := context.Background()

	// Execute pre-deploy
	err := tm.Execute(ctx, false)
	assert.NoError(t, err)

	// Execute post-deploy
	err = tm.Execute(ctx, true)
	assert.NoError(t, err)
}
