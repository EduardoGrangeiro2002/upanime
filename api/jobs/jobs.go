package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"upanime/api/service"
)

const (
	TypeDownload        = "download:episode"
	TypeUpscaleDispatch = "upscale:dispatch"
	TypeClassifyAll     = "catalog:classify_all"
	TypeOrganize        = "catalog:organize"
	QueueDownloads      = "downloads"
	QueueUpscale        = "upscale"
	QueueCatalog        = "catalog"
)

type DownloadPayload struct {
	DownloadID int64 `json:"downloadId"`
}

func DownloadTaskID(downloadID int64) string {
	return fmt.Sprintf("download:%d", downloadID)
}

func FinalAttempt(ctx context.Context) bool {
	retried, ok := asynq.GetRetryCount(ctx)
	if !ok {
		return true
	}
	maxRetry, ok := asynq.GetMaxRetry(ctx)
	if !ok {
		return true
	}
	return retried >= maxRetry
}

func NewDownloadTask(downloadID int64) *asynq.Task {
	payload, _ := json.Marshal(DownloadPayload{DownloadID: downloadID})
	return asynq.NewTask(TypeDownload, payload,
		asynq.Queue(QueueDownloads),
		asynq.TaskID(DownloadTaskID(downloadID)),
		asynq.MaxRetry(2),
		asynq.Timeout(3*time.Hour),
	)
}

func UpscaleTaskID(jobID int64) string {
	return fmt.Sprintf("upscale:dispatch:%d", jobID)
}

func NewUpscaleDispatchTask(wj service.UpscaleWorkerJob) (*asynq.Task, error) {
	payload, err := json.Marshal(wj)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeUpscaleDispatch, payload,
		asynq.Queue(QueueUpscale),
		asynq.TaskID(UpscaleTaskID(wj.JobID)),
		asynq.MaxRetry(3),
		asynq.Timeout(2*time.Minute),
	), nil
}

type OrganizePayload struct {
	AnimeID int64 `json:"animeId"`
}

func NewClassifyAllTask() *asynq.Task {
	return asynq.NewTask(TypeClassifyAll, nil,
		asynq.Queue(QueueCatalog),
		asynq.MaxRetry(1),
		asynq.Timeout(30*time.Minute),
	)
}

func NewOrganizeTask(animeID int64) *asynq.Task {
	payload, _ := json.Marshal(OrganizePayload{AnimeID: animeID})
	return asynq.NewTask(TypeOrganize, payload,
		asynq.Queue(QueueCatalog),
		asynq.MaxRetry(2),
		asynq.Timeout(10*time.Minute),
	)
}
