package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/hibiken/asynq"
	"upanime/api/jobs"
	"upanime/api/service"
)

func ClassifyAllHandler(classifier *service.GenreClassifier, enq jobs.Enqueuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !classifier.Enabled() {
			http.Error(w, `{"error":"classificador desativado: configure OPENROUTER_API_KEY"}`, http.StatusServiceUnavailable)
			return
		}

		if err := enq.EnqueueClassifyAll(r.Context()); err != nil {
			http.Error(w, `{"error":"enfileirar classificação falhou"}`, http.StatusInternalServerError)
			return
		}

		writeAccepted(w)
	}
}

func ClassifyAllTask(classifier *service.GenreClassifier) asynq.HandlerFunc {
	return func(ctx context.Context, _ *asynq.Task) error {
		result, err := classifier.ClassifyAll(ctx)
		if err != nil {
			return err
		}
		log.Printf("classificação em massa concluída: %+v", result)
		return nil
	}
}

func writeAccepted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"queued"}`))
}
