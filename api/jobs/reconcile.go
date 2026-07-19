package jobs

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"
	"upanime/api/store"
)

const orphanError = "interrompido por reinício do servidor"

func Reconcile(ctx context.Context, insp *asynq.Inspector, downloads store.DownloadStore, upscales store.UpscaleJobStore) error {
	if err := reconcileDownloads(ctx, insp, downloads); err != nil {
		return err
	}
	return reconcileUpscales(ctx, insp, upscales)
}

func reconcileDownloads(ctx context.Context, insp *asynq.Inspector, downloads store.DownloadStore) error {
	active, err := downloads.ListActive(ctx)
	if err != nil {
		return err
	}
	for _, d := range active {
		if !downloadPending(d.Status) {
			continue
		}
		if !taskDead(insp, QueueDownloads, DownloadTaskID(d.ID.Int64())) {
			continue
		}
		_ = downloads.UpdateStatus(ctx, d.ID.Int64(), "failed", orphanError)
	}
	return nil
}

func reconcileUpscales(ctx context.Context, insp *asynq.Inspector, upscales store.UpscaleJobStore) error {
	active, err := upscales.ListActive(ctx)
	if err != nil {
		return err
	}
	for _, job := range active {
		if job.Status != "queued" || job.RunPodJobID != "" {
			continue
		}
		if !taskDead(insp, QueueUpscale, UpscaleTaskID(job.ID.Int64())) {
			continue
		}
		_ = upscales.UpdateStatus(ctx, job.ID.Int64(), "failed", orphanError)
	}
	return nil
}

func downloadPending(status string) bool {
	return status == "queued" || status == "resolving" || status == "downloading"
}

func taskDead(insp *asynq.Inspector, queue, taskID string) bool {
	info, err := insp.GetTaskInfo(queue, taskID)
	if err != nil {
		return errors.Is(err, asynq.ErrTaskNotFound) || errors.Is(err, asynq.ErrQueueNotFound)
	}
	return info.State == asynq.TaskStateArchived
}
