package jobs

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"
	"upanime/api/service"
)

type Enqueuer interface {
	EnqueueDownload(ctx context.Context, downloadID int64) error
	CancelDownload(ctx context.Context, downloadID int64) error
	EnqueueUpscaleDispatch(ctx context.Context, wj service.UpscaleWorkerJob) error
	CancelUpscale(ctx context.Context, jobID int64) error
	EnqueueClassifyAll(ctx context.Context) error
	EnqueueOrganize(ctx context.Context, animeID int64) error
}

type AsynqEnqueuer struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

func NewAsynqEnqueuer(redisAddr string) *AsynqEnqueuer {
	opt := asynq.RedisClientOpt{Addr: redisAddr}
	return &AsynqEnqueuer{client: asynq.NewClient(opt), inspector: asynq.NewInspector(opt)}
}

func (e *AsynqEnqueuer) Close() error {
	e.inspector.Close()
	return e.client.Close()
}

func (e *AsynqEnqueuer) Inspector() *asynq.Inspector {
	return e.inspector
}

func (e *AsynqEnqueuer) EnqueueDownload(ctx context.Context, downloadID int64) error {
	_, err := e.client.EnqueueContext(ctx, NewDownloadTask(downloadID))
	return err
}

func (e *AsynqEnqueuer) CancelDownload(ctx context.Context, downloadID int64) error {
	return e.cancel(QueueDownloads, DownloadTaskID(downloadID))
}

func (e *AsynqEnqueuer) EnqueueUpscaleDispatch(ctx context.Context, wj service.UpscaleWorkerJob) error {
	task, err := NewUpscaleDispatchTask(wj)
	if err != nil {
		return err
	}
	_, err = e.client.EnqueueContext(ctx, task)
	return err
}

func (e *AsynqEnqueuer) CancelUpscale(ctx context.Context, jobID int64) error {
	return e.cancel(QueueUpscale, UpscaleTaskID(jobID))
}

func (e *AsynqEnqueuer) EnqueueClassifyAll(ctx context.Context) error {
	_, err := e.client.EnqueueContext(ctx, NewClassifyAllTask())
	return err
}

func (e *AsynqEnqueuer) EnqueueOrganize(ctx context.Context, animeID int64) error {
	_, err := e.client.EnqueueContext(ctx, NewOrganizeTask(animeID))
	return err
}

func (e *AsynqEnqueuer) cancel(queue, taskID string) error {
	_ = e.inspector.CancelProcessing(taskID)
	err := e.inspector.DeleteTask(queue, taskID)
	if err == nil || errors.Is(err, asynq.ErrTaskNotFound) || errors.Is(err, asynq.ErrQueueNotFound) {
		return nil
	}
	return err
}
