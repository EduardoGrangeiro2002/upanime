package service

import (
	"context"
	"log"
	"time"

	"upanime/api/model"
	"upanime/api/store"
)

type RunPodPoller struct {
	jobs     store.UpscaleJobStore
	episodes store.EpisodeStore
	worker   UpscaleWorkerClient
	interval time.Duration
	stop     chan struct{}
}

func NewRunPodPoller(
	jobs store.UpscaleJobStore,
	episodes store.EpisodeStore,
	worker UpscaleWorkerClient,
	interval time.Duration,
) *RunPodPoller {
	return &RunPodPoller{
		jobs:     jobs,
		episodes: episodes,
		worker:   worker,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (p *RunPodPoller) Start() {
	go p.run()
}

func (p *RunPodPoller) Stop() {
	close(p.stop)
}

func (p *RunPodPoller) run() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.poll()
		}
	}
}

func (p *RunPodPoller) poll() {
	ctx := context.Background()

	jobs, err := p.jobs.ListProcessing(ctx)
	if err != nil {
		log.Printf("poller: list processing jobs: %v", err)
		return
	}

	for _, job := range jobs {
		p.checkJob(ctx, job)
	}
}

func (p *RunPodPoller) checkJob(ctx context.Context, job model.UpscaleJob) {
	status, err := p.worker.Status(ctx, job.RunPodJobID)
	if err != nil {
		log.Printf("poller: check job %d (runpod %s): %v", job.ID.Int64(), job.RunPodJobID, err)
		return
	}

	if status.Status == "COMPLETED" {
		p.handleCompleted(ctx, job, status)
		return
	}

	if status.Status == "FAILED" {
		errMsg := status.Error
		if errMsg == "" {
			errMsg = "RunPod job failed"
		}
		_ = p.jobs.UpdateStatus(ctx, job.ID.Int64(), "failed", errMsg)
		return
	}
}

func (p *RunPodPoller) handleCompleted(ctx context.Context, job model.UpscaleJob, status *RunPodJobStatus) {
	resultKey := status.Output["resultStorageKey"]
	if resultKey != "" {
		_ = p.jobs.UpdateResult(ctx, job.ID.Int64(), resultKey)
		_ = p.episodes.UpdateUpscaledStorageKey(ctx, job.EpisodeID.Int64(), resultKey)
	}

	_ = p.jobs.UpdateStatus(ctx, job.ID.Int64(), "completed", "")
}
