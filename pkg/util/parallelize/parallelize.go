package parallelize

import (
	"context"

	"k8s.io/client-go/util/workqueue"
)

const maxParallelism = 8

// ErrorChannel supports non-blocking send and receive operation to capture error.
// A maximum of one error is kept in the channel and the rest of the errors sent
// are ignored, unless the existing error is received and the channel becomes empty
// again.
type ErrorChannel struct {
	ch chan error
}

func NewErrorChannel() *ErrorChannel {
	return &ErrorChannel{
		ch: make(chan error, 1),
	}
}

func (e *ErrorChannel) SendError(err error) {
	if err == nil {
		return
	}
	select {
	case e.ch <- err:
	default:
	}
}

func (e *ErrorChannel) Receive() error {
	select {
	case err := <-e.ch:
		return err
	default:
		return nil
	}
}

func Until(ctx context.Context, pieces int, doWorkPiece func(i int) error) error {
	errCh := NewErrorChannel()
	workers := min(pieces, maxParallelism)
	workqueue.ParallelizeUntil(ctx, workers, pieces, func(i int) {
		errCh.SendError(doWorkPiece(i))
	})
	return errCh.Receive()
}
