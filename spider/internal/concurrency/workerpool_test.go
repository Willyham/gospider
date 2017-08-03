package concurrency

import (
	"testing"
	"time"

	"github.com/Willyham/gospider/spider/internal/concurrency/mocks"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewIngesterPool(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ingester := NewWorkerPool(logger, &mocks.Worker{}, 1)
	assert.Implements(t, (*AsyncRunnable)(nil), ingester)
}

func TestStart(t *testing.T) {
	// Create a worker which doesn't error
	worker := &mocks.Worker{}
	worker.On("Work").Return(nil)

	logger, _ := zap.NewDevelopment()
	ingester := NewWorkerPool(logger, worker, 1)
	go ingester.Start()
	// Advance some time to allow the workers to process a job before stopping
	select {
	case <-time.After(1 * time.Millisecond):
		ingester.StopWait()
	}
	mock.AssertExpectationsForObjects(t, worker)
}

func TestStartError(t *testing.T) {
	// Create a worker which always errors
	worker := &mocks.Worker{}
	worker.On("Work").Return(assert.AnError)

	logger, _ := zap.NewDevelopment()
	ingester := NewWorkerPool(logger, worker, 1)

	err := ingester.Start()
	assert.Error(t, err)
	mock.AssertExpectationsForObjects(t, worker)
}

func TestStartMultipleErrors(t *testing.T) {
	// Create a worker which always errors
	worker := &mocks.Worker{}
	worker.On("Work").Return(assert.AnError).Twice()

	// Use 2 workers to cause 2 errors
	logger, _ := zap.NewDevelopment()
	ingester := NewWorkerPool(logger, worker, 2)

	err := ingester.Start()
	assert.Error(t, err)
	mock.AssertExpectationsForObjects(t, worker)
}

func TestStartRetryableError(t *testing.T) {
	// Create a worker which returns a retryable error
	worker := &mocks.Worker{}
	worker.On("Work").Return(NewRetryableError(assert.AnError))

	logger, _ := zap.NewDevelopment()
	ingester := NewWorkerPool(logger, worker, 1)

	go ingester.Start()

	// Advance some time to allow the workers to process a job before stopping
	select {
	case <-time.After(1 * time.Microsecond):
		ingester.StopWait()
	}
	mock.AssertExpectationsForObjects(t, worker)
}
