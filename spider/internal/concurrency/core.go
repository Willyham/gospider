// Package concurrency provides common concurrency patterns and utilities.
package concurrency

import "github.com/pkg/errors"

// Runnable describes something which can start and stop.
type Runnable interface {
	Start() error
	Stop()
}

// AsyncRunnable is a runnable which is can run asynchrounously
type AsyncRunnable interface {
	Runnable
	StopWait()
}

// Worker is anything that can work
type Worker interface {
	Work() error
}

//go:generate mockery -name Worker -case underscore

// Retryable is an interface which describes whether something is retryable
type Retryable interface {
	Retryable() bool
}

// Stopped is a special error value is signals that the runnable is stopped
var Stopped = errors.New("stopped")
