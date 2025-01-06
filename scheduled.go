package workers

import (
	"context"
	"strings"
	"time"
)

type scheduledWorker struct {
	opts Options
}

func (s *scheduledWorker) run(ctx context.Context) {
	ticker := time.NewTicker(s.opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *scheduledWorker) poll(ctx context.Context) {
	now := nowToSecondsWithNanoPrecision()

	for {
		rawMessage, err := s.opts.store.DequeueScheduledMessage(ctx, now)

		if err != nil {
			break
		}

		message, _ := NewMsg(rawMessage)
		queue, _ := message.Get("queue").String()
		queue = strings.TrimPrefix(queue, s.opts.Namespace)
		message.Set("enqueued_at", nowToSecondsWithNanoPrecision())

		s.opts.store.EnqueueMessageNow(ctx, queue, message.ToJson())
	}

	for {
		rawMessage, err := s.opts.store.DequeueRetriedMessage(ctx, now)

		if err != nil {
			break
		}

		message, _ := NewMsg(rawMessage)
		queue, _ := message.Get("queue").String()
		queue = strings.TrimPrefix(queue, s.opts.Namespace)
		message.Set("enqueued_at", nowToSecondsWithNanoPrecision())

		s.opts.store.EnqueueMessageNow(ctx, queue, message.ToJson())
	}
}

func newScheduledWorker(opts Options) *scheduledWorker {
	return &scheduledWorker{
		opts: opts,
	}
}
