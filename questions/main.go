package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/gofrs/uuid"
	trcr "github.com/martinsirbe/istio-demo/tracer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const app = "questions"

// Question a dummy question :]
type Question struct {
	ID   uuid.UUID `json:"id"`
	Text string    `json:"text"`
}

func main() {
	fmt.Println(app, "server started")

	tracer, closer := trcr.InitTracer(app, "jaeger-agent.istio-system:6831")
	handlers := Handlers{tracer: tracer, closer: closer}

	http.HandleFunc("/questions", handlers.GetQuestions)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logrus.WithError(err).Panicf("%s server failed to listen and serve", app)
	}

}

type Handlers struct {
	tracer opentracing.Tracer
	closer io.Closer
}

func (h *Handlers) GetQuestions(w http.ResponseWriter, r *http.Request) {
	logrus.Info("handling get question request")
	span := h.tracer.StartSpan("get_questions")
	span.SetTag("endpoint", "/questions")
	defer span.Finish()

	if f := rand.Intn(5); f == 1 {
		logrus.Error("no questions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var qs []Question
	for i := 0; i < rand.Intn(25); i++ {
		qs = append(qs, Question{ID: uuid.Must(uuid.NewV4()), Text: "What's your name?"})
	}

	if err := json.NewEncoder(w).Encode(&qs); err != nil {
		logrus.WithError(err).Error("failed to encode response body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
