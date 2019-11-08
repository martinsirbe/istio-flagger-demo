package tracer

import (
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

func InitTracer(app, agentHost string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Disabled:    false,
		ServiceName: app,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: agentHost,
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialise Jaeger tracer")
	}

	return tracer, closer
}
