package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofrs/uuid"
	trcr "github.com/martinsirbe/istio-demo/tracer"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const app = "surveys"

type Survey struct {
	ID        uuid.UUID  `json:"id"`
	Questions []Question `json:"questions"`
}

type Question struct {
	ID   uuid.UUID `json:"id"`
	Text string    `json:"text"`
}

func main() {
	fmt.Println(app, "server started")

	tracer, closer := trcr.InitTracer(app, "jaeger-agent.istio-system:6831")
	handlers := Handlers{tracer: tracer, closer: closer, client: &http.Client{}}

	http.HandleFunc("/survey", handlers.GetSurvey)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logrus.WithError(err).Panicf("%s server failed to listen and serve", app)
	}
}

type Handlers struct {
	tracer opentracing.Tracer
	closer io.Closer
	client *http.Client
}

func (h *Handlers) GetSurvey(w http.ResponseWriter, r *http.Request) {
	span := h.tracer.StartSpan("get_survey")
	span.SetTag("endpoint", "/survey")
	defer span.Finish()

	//ctx := opentracing.ContextWithSpan(context.Background(), span)
	//logrus.Info("handling get survey request")

	qs, err := h.getQuestions(span, r)
	if err != nil {
		logrus.WithError(err).Error("failed to get questions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.createResponse(span, qs, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) getQuestions(rootSpan opentracing.Span, r *http.Request) ([]Question, error) {
	span := h.tracer.StartSpan(
		"call_questions_service",
		opentracing.ChildOf(rootSpan.Context()),
	)
	span.SetTag("endpoint", "/survey")
	defer span.Finish()

	url := "http://questions:8080/questions"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new http get request to obtain questions from questions service")
	}

	keys, ok := r.URL.Query()["user"]
	if ok && len(keys[0]) > 0 {
		logrus.Infof("user query parameter - %s", keys[0])
		span.SetBaggageItem("user", keys[0])
		req.Header.Set("user", keys[0])
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, http.MethodGet)
	if err := span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	); err != nil {
		logrus.WithError(err).Error("failed to inject opentracing span context")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make http get request for questions")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("non 200 response received from questions service")
	}

	var qs []Question
	if err := json.NewDecoder(resp.Body).Decode(&qs); err != nil {
		return nil, errors.Wrap(err, "failed to decode questions service response body")
	}

	return qs, nil
}

func (h *Handlers) createResponse(rootSpan opentracing.Span, qs []Question, w http.ResponseWriter) error {
	span := h.tracer.StartSpan(
		"create_response",
		opentracing.ChildOf(rootSpan.Context()),
	)
	defer span.Finish()

	survey := &Survey{ID: uuid.Must(uuid.NewV4()), Questions: qs}
	if err := json.NewEncoder(w).Encode(survey); err != nil {
		return errors.Wrap(err, "failed to encode survey response")
	}

	return nil
}
