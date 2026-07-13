package worker

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/metadata/model"
)

// Queue is a deduplicated in-memory metadata job queue.
type Queue struct {
	mu      sync.Mutex
	pending map[string]model.Job
	ch      chan model.Job
}

// NewQueue creates a job queue.
func NewQueue(buffer int) *Queue {
	if buffer < 1 {
		buffer = 64
	}
	return &Queue{
		pending: make(map[string]model.Job),
		ch:      make(chan model.Job, buffer),
	}
}

// Enqueue adds a job if it is not already queued.
func (q *Queue) Enqueue(job model.Job) {
	if job.AlbumID == "" {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.pending[job.AlbumID]; exists {
		return
	}
	q.pending[job.AlbumID] = job

	select {
	case q.ch <- job:
	default:
		go func() { q.ch <- job }()
	}
}

// Jobs returns the receive-only jobs channel.
func (q *Queue) Jobs() <-chan model.Job {
	return q.ch
}

// Done marks a job as completed and allows re-queue later.
func (q *Queue) Done(albumID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.pending, albumID)
}

// JobHandler persists metadata fetch results.
type JobHandler interface {
	ProcessJob(ctx context.Context, job model.Job) error
}

// Worker processes metadata jobs.
type Worker struct {
	queue   *Queue
	handler JobHandler
	logger  zerolog.Logger
}

// New creates a metadata worker.
func New(queue *Queue, handler JobHandler, logger zerolog.Logger) *Worker {
	return &Worker{
		queue:   queue,
		handler: handler,
		logger:  logger.With().Str("component", "metadata-worker").Logger(),
	}
}

// Run consumes jobs until the context is cancelled.
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("metadata worker stopped")
			return
		case job := <-w.queue.Jobs():
			if err := w.handler.ProcessJob(ctx, job); err != nil {
				w.logger.Warn().Err(err).Str("album_id", job.AlbumID).Msg("metadata job failed")
			}
			w.queue.Done(job.AlbumID)
		}
	}
}
