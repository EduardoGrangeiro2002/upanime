package jobs

import (
	"log"

	"github.com/hibiken/asynq"
)

const (
	upscaleConcurrency = 3
	catalogConcurrency = 1
)

type Handlers struct {
	Download        asynq.HandlerFunc
	UpscaleDispatch asynq.HandlerFunc
	ClassifyAll     asynq.HandlerFunc
	Organize        asynq.HandlerFunc
}

func StartWorkers(redisAddr string, maxDownloads int, h Handlers) func() {
	opt := asynq.RedisClientOpt{Addr: redisAddr}

	servers := []*asynq.Server{
		startServer(opt, QueueDownloads, maxDownloads, map[string]asynq.HandlerFunc{TypeDownload: h.Download}),
		startServer(opt, QueueUpscale, upscaleConcurrency, map[string]asynq.HandlerFunc{TypeUpscaleDispatch: h.UpscaleDispatch}),
		startServer(opt, QueueCatalog, catalogConcurrency, map[string]asynq.HandlerFunc{
			TypeClassifyAll: h.ClassifyAll,
			TypeOrganize:    h.Organize,
		}),
	}

	return func() {
		for _, srv := range servers {
			srv.Shutdown()
		}
	}
}

func startServer(opt asynq.RedisClientOpt, queue string, concurrency int, handlers map[string]asynq.HandlerFunc) *asynq.Server {
	mux := asynq.NewServeMux()
	for taskType, handler := range handlers {
		mux.HandleFunc(taskType, handler)
	}
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
		Queues:      map[string]int{queue: 1},
	})
	if err := srv.Start(mux); err != nil {
		log.Fatal(err)
	}
	return srv
}
