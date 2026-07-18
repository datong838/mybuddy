package autoreply

import (
	"sync"
)

type dispatchItem struct {
	kind    ReplyDispatchKind
	payload ReplyPayload
}

// Dispatcher serializes and delivers reply payloads.
type Dispatcher struct {
	opts    DispatcherOptions
	queue   chan dispatchItem
	done    chan struct{}
	mu      sync.Mutex
	pending int
}

// NewDispatcher creates a new reply dispatcher.
func NewDispatcher(opts DispatcherOptions) *Dispatcher {
	if opts.Deliver == nil {
		opts.Deliver = func(ReplyPayload, ReplyDispatchKind) error { return nil }
	}
	d := &Dispatcher{
		opts:  opts,
		queue: make(chan dispatchItem, 64),
		done:  make(chan struct{}),
	}
	go d.run()
	return d
}

func (d *Dispatcher) run() {
	for item := range d.queue {
		d.mu.Lock()
		d.pending--
		p := d.pending
		d.mu.Unlock()
		if err := d.opts.Deliver(item.payload, item.kind); err != nil && d.opts.OnError != nil {
			d.opts.OnError(err, item.kind)
		}
		if p == 0 && d.opts.OnIdle != nil {
			d.opts.OnIdle()
		}
	}
	close(d.done)
}

// SendToolResult enqueues a tool result.
func (d *Dispatcher) SendToolResult(payload ReplyPayload) bool {
	return d.enqueue(ReplyKindTool, payload)
}

// SendBlockReply enqueues a block reply.
func (d *Dispatcher) SendBlockReply(payload ReplyPayload) bool {
	return d.enqueue(ReplyKindBlock, payload)
}

// SendFinalReply enqueues a final reply.
func (d *Dispatcher) SendFinalReply(payload ReplyPayload) bool {
	return d.enqueue(ReplyKindFinal, payload)
}

func (d *Dispatcher) enqueue(kind ReplyDispatchKind, payload ReplyPayload) bool {
	if payload.Text == "" && payload.MediaURL == "" && len(payload.MediaURLs) == 0 {
		return false
	}
	select {
	case d.queue <- dispatchItem{kind: kind, payload: payload}:
		d.mu.Lock()
		d.pending++
		d.mu.Unlock()
		return true
	default:
		return false
	}
}

// Close stops the dispatcher. Call after all sends are done.
func (d *Dispatcher) Close() {
	close(d.queue)
	<-d.done
}

// Pending returns the number of queued or in-flight deliveries.
func (d *Dispatcher) Pending() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pending
}
