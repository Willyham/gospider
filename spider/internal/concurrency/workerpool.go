package concurrency

import (
	"sync"

	"go.uber.org/zap"
)

type poolState int

const (
	stateRunning poolState = iota
	stateStopping
	stateStopped
)

// WorkerPool is an implementation of a start/stoppable worker pool.
//
// In this implementation, jobs are essentially tokens to perform some work. Jobs are not delivered
// to the pool, but instead 'claimed' by a worker, and 'returned' when finished (technically, a new job
// is posted to the queue). In this case, the worker itself is the one which determines the work it should do.
// This gives a lot of flexibility when combined with implementations of Worker.
//
// It uses a number of channels to control the concurrency:
// - jobs is a buffered channel that signals that a worker should process a job
// - results signals that a result was computed by work()
// - errors collects any errors from work(). An error on the channel will stop the ingester
// - done is used to signal when the ingester has totally stopped (i.e. all workers drained)
type WorkerPool struct {
	logger     *zap.Logger
	worker     Worker
	numWorkers int

	jobs      chan struct{}
	results   chan struct{}
	errors    chan error
	done      chan bool
	waitGroup sync.WaitGroup
	state     poolState
	stateLock sync.Mutex
}

// NewWorkerPool creates a new worker-pool
func NewWorkerPool(logger *zap.Logger, numWorkers int, w Worker) *WorkerPool {
	return &WorkerPool{
		logger:     logger,
		worker:     w,
		numWorkers: numWorkers,

		jobs: make(chan struct{}, numWorkers),
		// results is a buffered channel so we can drain results after signalling to stop
		results:   make(chan struct{}, numWorkers),
		errors:    make(chan error),
		done:      make(chan bool),
		waitGroup: sync.WaitGroup{},
		state:     stateStopped,
	}
}

// Start makes the ingester-pool start to process messages.
//
// It continually loops and looks for either a result, in which case it
// adds another job to the pool to be processed, or an error, in which
// case it stops the ingester-pool, waits for the workers to drain, then signals
// that it is done.
func (s *WorkerPool) Start() error {
	s.stateLock.Lock()
	s.state = stateRunning
	s.stateLock.Unlock()

	// Create workers with initial jobs
	s.waitGroup.Add(s.numWorkers)
	for i := 0; i < s.numWorkers; i++ {
		s.logger.Info("Creating worker")
		s.jobs <- struct{}{}
		go s.runWorker()
	}

	for {
		select {
		case err := <-s.errors:
			s.state = stateStopping

			// If the error is something we don't know about or is not retryable, log it and stop
			if err != Stopped {
				s.logger.Error("got error from workers", zap.Error(err))
			}

			// Close jobs to shut down the workers, then wait for them to finish
			close(s.jobs)

			// Drain off any errors from other workers
			go func() {
				for e := range s.errors {
					s.logger.Error(e.Error())
				}
			}()

			s.waitGroup.Wait()
			close(s.results)
			close(s.errors)
			close(s.done)

			s.state = stateStopped
			return err
		case <-s.results:
			// If we get a result, add another job to the queue
			s.logger.Debug("Got result, adding job")
			s.jobs <- struct{}{}
		}
	}
}

// runWorker defers to the Worker to process jobs.
//
// If there is no error from the worker, it continues.
// If there is an error, but it's retryable, it continues after logging the error.
// If there is an error which isn't retryable, it passes the error onto the error
// channel, which causes the ingester-pool to drain the queue and stop.
func (s *WorkerPool) runWorker() {
	defer s.waitGroup.Done()

	for range s.jobs {
		s.logger.Debug("Processing job")
		err := s.worker.Work()

		if err == nil {
			// Processed Successfully, signal a result
			s.results <- struct{}{}
			continue
		}

		// If the error is retryable, log and signal results
		r, ok := err.(Retryable)
		if ok && r.Retryable() {
			s.logger.Info("Got retryable error, continuing.", zap.Error(err))
			s.results <- struct{}{}
			continue
		}

		// Unrecoverable error, signal an error
		s.errors <- err
	}
}

// Stop signals the ingester-pool to stop processing new messages. Use StopWait
// to wait until all messages are processed
func (s *WorkerPool) Stop() {
	s.logger.Info("Stopping worker-pool")
	s.errors <- Stopped
}

func (s *WorkerPool) UntilPoolExhausted() {
	s.waitGroup.Wait()
	s.StopWait()
	return
}

// StopWait starts the process of stopping, and waits for all workers to
// stop before returning.
func (s *WorkerPool) StopWait() {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	if s.state == stateRunning {
		s.Stop()
	}
	<-s.done
}
