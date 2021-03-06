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
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
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
	http.HandleFunc("/healthz", handlers.GetHealth)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logrus.WithError(err).Panicf("%s server failed to listen and serve", app)
	}

}

type Handlers struct {
	tracer opentracing.Tracer
	closer io.Closer
}

func (h *Handlers) GetQuestions(w http.ResponseWriter, r *http.Request) {
	spanCtx, _ := h.tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	span := h.tracer.StartSpan("get_questions", ext.RPCServerOption(spanCtx))
	span.SetTag("endpoint", "/questions")
	span.SetTag("version", "v1")
	defer span.Finish()

	span.LogFields(
		otlog.String("description", "a healthy questions service which returns only 200s"),
	)

	questions := h.createQuestions(span)
	if err := json.NewEncoder(w).Encode(&questions); err != nil {
		logrus.WithError(err).Error("failed to encode response body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) createQuestions(rootSpan opentracing.Span) []Question {
	span := h.tracer.StartSpan(
		"create_questions",
		opentracing.ChildOf(rootSpan.Context()),
	)
	defer span.Finish()

	var qs []Question
	for i := 0; i < rand.Intn(25); i++ {
		qs = append(qs, Question{ID: uuid.Must(uuid.NewV4()), Text: "What's your name?"})
	}

	return qs
}

func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	spanCtx, _ := h.tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	span := h.tracer.StartSpan("get_healthz", ext.RPCServerOption(spanCtx))
	span.SetTag("endpoint", "/healthz")
	defer span.Finish()

	w.WriteHeader(http.StatusOK)
}
