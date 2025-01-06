package workers

import (
	"context"
	"strings"
	"time"
)

type scheduledWorker struct {
	opts Options
	ctx  context.Context
}

func (s *scheduledWorker) run() {
	ticker := time.NewTicker(s.opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.poll()
		}
	}
}

func (s *scheduledWorker) poll() {
	now := nowToSecondsWithNanoPrecision()

	for {
		rawMessage, err := s.opts.store.DequeueScheduledMessage(s.ctx, now)

		if err != nil {
			break
		}

		message, _ := NewMsg(rawMessage)
		queue, _ := message.Get("queue").String()
		queue = strings.TrimPrefix(queue, s.opts.Namespace)
		message.Set("enqueued_at", nowToSecondsWithNanoPrecision())

		s.opts.store.EnqueueMessageNow(s.ctx, queue, message.ToJson())
	}

	for {
		rawMessage, err := s.opts.store.DequeueRetriedMessage(s.ctx, now)

		if err != nil {
			break
		}

		message, _ := NewMsg(rawMessage)
		queue, _ := message.Get("queue").String()
		queue = strings.TrimPrefix(queue, s.opts.Namespace)
		message.Set("enqueued_at", nowToSecondsWithNanoPrecision())

		s.opts.store.EnqueueMessageNow(s.ctx, queue, message.ToJson())
	}
}

func newScheduledWorker(opts Options, ctx context.Context) *scheduledWorker {
	return &scheduledWorker{
		opts: opts,
		ctx:  ctx,
	}
}
