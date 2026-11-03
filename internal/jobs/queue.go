package jobs

import (
	"context"
	"sync"
	"time"
)

// MailQueue 为进程内的邮件发送队列：多个工作协程消费，失败按次数重试，并记录失败任务。
// 对应 Laravel 队列（SendEmailJob 失败时 release 重试）与 Horizon 的统计口径。
type MailQueue struct {
	env     *Env
	ch      chan MailTask
	workers int
	maxTry  int

	mu        sync.Mutex
	processed int64
	failed    int64
	perMinute []time.Time
	failedLog []FailedJob
	running   bool
}

// MailTask 为一封待发送的模板邮件。
type MailTask struct {
	To       string
	Subject  string
	Template string
	Values   map[string]any
}

// FailedJob 记录最终失败的任务。
type FailedJob struct {
	ID       int64     `json:"id"`
	Email    string    `json:"email"`
	Subject  string    `json:"subject"`
	Template string    `json:"template"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
	Attempts int       `json:"attempts"`
}

// QueueStats 为队列统计快照。
type QueueStats struct {
	Processed     int64 `json:"processed"`
	Failed        int64 `json:"failed"`
	JobsPerMinute int   `json:"jobs_per_minute"`
	Pending       int   `json:"pending"`
	Workers       int   `json:"workers"`
	Running       bool  `json:"running"`
}

// NewMailQueue 创建队列；workers 为并发消费协程数，maxTry 为最大尝试次数。
func NewMailQueue(env *Env, workers, maxTry int) *MailQueue {
	if workers < 1 {
		workers = 2
	}
	if maxTry < 1 {
		maxTry = 3
	}
	return &MailQueue{env: env, ch: make(chan MailTask, 1000), workers: workers, maxTry: maxTry}
}

// Start 启动工作协程，ctx 结束时退出。
func (q *MailQueue) Start(ctx context.Context) {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.mu.Unlock()
	for i := 0; i < q.workers; i++ {
		go q.work(ctx)
	}
}

// Enqueue 入队；队列满时返回 false。
func (q *MailQueue) Enqueue(t MailTask) bool {
	select {
	case q.ch <- t:
		return true
	default:
		return false
	}
}

func (q *MailQueue) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-q.ch:
			q.run(ctx, t)
		}
	}
}

func (q *MailQueue) run(ctx context.Context, t MailTask) {
	var err error
	attempts := 0
	for attempts < q.maxTry {
		attempts++
		err = q.env.SendTemplateMail(t.To, t.Subject, t.Template, t.Values)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(attempts) * 2 * time.Second):
		}
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	q.perMinute = append(q.perMinute, now)
	if err == nil {
		q.processed++
		return
	}
	q.failed++
	q.failedLog = append(q.failedLog, FailedJob{ID: q.failed, Email: t.To, Subject: t.Subject,
		Template: t.Template, Error: err.Error(), FailedAt: now, Attempts: attempts})
	if len(q.failedLog) > 100 {
		q.failedLog = q.failedLog[len(q.failedLog)-100:]
	}
}

// Stats 返回统计快照（每分钟处理数按最近 60 秒计算）。
func (q *MailQueue) Stats() QueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()
	cut := time.Now().Add(-time.Minute)
	kept := q.perMinute[:0]
	for _, ts := range q.perMinute {
		if ts.After(cut) {
			kept = append(kept, ts)
		}
	}
	q.perMinute = kept
	return QueueStats{Processed: q.processed, Failed: q.failed, JobsPerMinute: len(kept),
		Pending: len(q.ch), Workers: q.workers, Running: q.running}
}

// Failed 返回最近的失败任务（倒序）。
func (q *MailQueue) Failed() []FailedJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]FailedJob, 0, len(q.failedLog))
	for i := len(q.failedLog) - 1; i >= 0; i-- {
		out = append(out, q.failedLog[i])
	}
	return out
}
