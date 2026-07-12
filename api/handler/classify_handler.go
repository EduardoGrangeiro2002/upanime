package handler

import (
	"net/http"

	"upanime/api/service"
)

func ClassifyAllHandler(classifier *service.GenreClassifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !classifier.Enabled() {
			http.Error(w, `{"error":"classificador desativado: configure OPENROUTER_API_KEY"}`, http.StatusServiceUnavailable)
			return
		}

		result, err := classifier.ClassifyAll(r.Context())
		if err != nil {
			http.Error(w, `{"error":"classificação em massa falhou: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		writeJSON(w, result)
	}
}
