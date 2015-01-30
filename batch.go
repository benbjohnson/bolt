package bolt

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Batch calls fn as part of a batch. It behaves similar to Update,
// except:
//
// 1. multiple Batch function calls can be combined into a single
// Bolt transaction.
//
// 2. the function passed to Batch may be called multiple times,
// regardless of whether it returns error or not.
//
// This means that Batch function side effects must be idempotent and
// take permanent effect only after a successful return is seen in
// caller.
func (db *DB) Batch(fn func(*Tx) error) error {
	db.batchlock.Lock()
	if db.batch == nil {
		db.batch = newBatch(db)
	}
	db.batch.fns = append(db.batch.fns, batchCall{fn: fn, err: make(chan error, 1)})
	db.batchlock.Unlock()
}

type batchCall struct {
	fn  func(*Tx) error
	err chan<- error
}

type batch struct {
	db      *DB
	calls   []call
	start   chan struct{}
	started bool
}

// newBatch creates a batch attached to a DB.
func newBatch(db *DB) *batch {
	b := &batch{
		start: make(chan struct{}, 1),
	}
}

// add adds a call to the batch.
// If the batch is full then then force start the existing batch and create a new one.
func (b *batch) add(fn func(*Tx) error) error {
}

// caller has locked batch.mu for us
func (b *batch) master(db *DB, fn func(*Tx) error) error {
	b.full = make(chan struct{}, 1)
	ch := make(chan error, 1)
	b.calls = append(b.calls, call{fn: fn, err: ch})
	b.mu.Unlock()

	t := time.NewTimer(b.db.batchMaxDelay)
	select {
	case <-t.C:
	case <-b.full:
		t.Stop()
	}

	b.mu.Lock()
	b.started = true
	b.mu.Unlock()

retry:
	for len(b.calls) > 0 {
		var failIdx = -1
		err := db.Update(func(tx *Tx) error {
			for i, c := range b.calls {
				if err := safelyCall(c.fn, tx); err != nil {
					failIdx = i
					return err
				}
			}
			return nil
		})

		if failIdx >= 0 {
			// take the failing transaction out of the batch. it's safe to
			// shorten b.calls here because b.started has been set.
			c := b.calls[failIdx]
			b.calls[failIdx], b.calls = b.calls[len(b.calls)-1], b.calls[:len(b.calls)-1]
			// run it solo, report result, continue with the rest of the batch
			c.err <- db.Update(func(tx *Tx) error {
				return safelyCall(fn, tx)
			})
			continue retry
		}

		// pass success, or bolt internal errors, to all callers
		for _, c := range b.calls {
			if c.err != nil {
				c.err <- err
			}
		}
		break retry
	}

	err := <-ch
	if p, ok := err.(panicked); ok {
		panic(p.reason)
	}
	return err
}

type panicked struct {
	reason interface{}
}

func (p panicked) Error() string {
	if err, ok := p.reason.(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("panic: %v", p.reason)
}

func safelyCall(fn func(*Tx) error, tx *Tx) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = panicked{p}
		}
	}()
	return fn(tx)
}

func (b *batch) merge(fn func(*Tx) error) chan error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return nil
	}

	var ch chan error
	if len(b.calls) < b.db.batchMaxSize {
		ch = make(chan error, 1)
		c := call{
			fn:  fn,
			err: ch,
		}
		b.calls = append(b.calls, c)
	}

	if len(b.calls) >= b.db.batchMaxSize {
		// wake up batch, it's ready to run
		select {
		case b.full <- struct{}{}:
		default:
		}
	}

	return ch
}
