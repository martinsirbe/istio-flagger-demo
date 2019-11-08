# Simple service to service demo
.PHONY: bin
bin:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o questions/app questions/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o surveys/app surveys/main.go

.PHONY: docker-build
docker-build: bin
	@eval $$(minikube docker-env) ; \
	docker build -t questions:latest -f questions/Dockerfile ./questions ; \
	docker build -t surveys:latest -f surveys/Dockerfile ./surveys
	@rm surveys/app
	@rm questions/app

.PHONY: start-demo
start-demo: docker-build
	@kubectl apply -f questions/questions.yaml --context=minikube
	@kubectl apply -f surveys/surveys.yaml --context=minikube

.PHONY: stop-demo
stop-demo:
	@kubectl delete -f questions/questions.yaml --context=minikube
	@kubectl delete -f surveys/surveys.yaml --context=minikube

# Canary deployment demo
.PHONY: canary-bin
canary-bin:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o questions/canary/v1/app questions/canary/v1/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o questions/canary/v2/app questions/canary/v2/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o questions/canary/v3/app questions/canary/v3/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o surveys/app surveys/main.go

.PHONY: canary-docker-build
canary-docker-build: canary-bin
	@eval $$(minikube docker-env) ; \
	docker build -t questions:v1 -f questions/canary/v1/Dockerfile ./questions/canary/v1 ; \
	docker build -t questions:v2 -f questions/canary/v2/Dockerfile ./questions/canary/v2 ; \
	docker build -t questions:v3 -f questions/canary/v3/Dockerfile ./questions/canary/v3 ; \
	docker build -t surveys:latest -f surveys/Dockerfile ./surveys; \
	rm questions/canary/v1/app; \
	rm questions/canary/v2/app; \
	rm questions/canary/v3/app

.PHONY: start-canary-demo
start-canary-demo: canary-docker-build
	@kubectl apply -f questions/canary/questions-service.yaml --context=minikube
	@kubectl apply -f questions/canary/v1/questions.yaml --context=minikube
	@kubectl apply -f questions/canary/v2/questions.yaml --context=minikube
	@kubectl apply -f questions/canary/v3/questions.yaml --context=minikube
	@kubectl apply -f surveys/surveys.yaml --context=minikube
	@kubectl apply -f questions/canary/questions-canary.yaml --context=minikube
	@kubectl autoscale deployment questions-v1 --cpu-percent=50 --min=1 --max=10
	@kubectl autoscale deployment questions-v2 --cpu-percent=50 --min=1 --max=10
	@kubectl autoscale deployment questions-v3 --cpu-percent=50 --min=1 --max=10

.PHONY: stop-canary-demo
stop-canary-demo:
	@kubectl delete -f questions/canary/questions-service.yaml --context=minikube
	@kubectl delete -f questions/canary/v1/questions.yaml --context=minikube
	@kubectl delete -f questions/canary/v2/questions.yaml --context=minikube
	@kubectl delete -f questions/canary/v3/questions.yaml --context=minikube
	@kubectl delete -f surveys/surveys.yaml --context=minikube
	@kubectl delete -f questions/canary/questions-canary.yaml --context=minikube
	@kubectl delete hpa questions-v1 questions-v2 questions-v3

.PHONY: test-canary
test-canary:
	@echo "GET http://localhost:8080/survey" | vegeta attack -duration=1m -rate=10/1s | tee results.bin | vegeta report

# Generate traffic to deployed demo services (after port-forwarding to the service).
.PHONY: traffic-surveys
traffic-surveys:
	@while true; do \
		curl -s http://localhost:8080/survey > /dev/null \
		sleep 0.2; \
	done

.PHONY: traffic-surveys-user
traffic-surveys-user:
	@while true; do \
		curl -s http://localhost:8080/survey?user=martins > /dev/null \
		sleep 5; \
	done

.PHONY: testty
testty:
	echo $(TEST_ENV)

.PHONY: minikube
minikube:
	minikube start --memory=16384 --cpus=4 --kubernetes-version=v1.13.10

# Install Istio targets
istio: get-istio apply-istio-crds label-default-ns apply-istio-demo-auth is-istio-ready
.PHONY: istio

get-istio:
	@curl -L https://git.io/getLatestIstio | ISTIO_VERSION=1.3.3 sh -
	@./istio-1.3.3/bin/istioctl verify-install

apply-istio-crds:
	@kubectl create namespace istio-system
	@for i in istio-1.3.3/install/kubernetes/helm/istio-init/files/crd*yaml; do kubectl apply -f $$i --context=minikube; done

apply-istio-demo-auth:
	@kubectl apply -f istio-1.3.3/install/kubernetes/istio-demo.yaml --context=minikube

label-default-ns:
	@kubectl label namespace default istio-injection=enabled --context=minikube

retries ?= 10
is-istio-ready:
	@retries=$(retries); \
	for ((i=1; i <= ${retries}; ++i)) ; do \
		if [ "`kubectl get pods --field-selector=status.phase!=Running -nistio-system --context=minikube | grep -v 'Completed' | wc -l | sed -e 's/^[ \t]*//'`" != "1" ]; then \
			printf "\033[G⏳ Istio is not ready yet..."; \
		else \
			printf "\033[G🚀 Istio is ready..."; \
			exit 0; \
		fi; \
		sleep 25; \
	done; \
	echo "Istio is not ready after several checks.."; \
	kubectl get pods --field-selector=status.phase!=Running -nistio-system --context=minikube; \
	exit 1;

PHONY: jaeger
jaeger:
	kubectl port-forward -n istio-system $$(kubectl get pod -n istio-system -l app=jaeger -o jsonpath='{.items[0].metadata.name}' --context=minikube) --context=minikube 16686:16686

PHONY: kiali
kiali:
	kubectl port-forward -n istio-system $$(kubectl get pod -n istio-system -l app=kiali -o jsonpath='{.items[0].metadata.name}' --context=minikube) --context=minikube 20001:20001

PHONY: grafana
grafana:
	kubectl port-forward -n istio-system $$(kubectl get pod -n istio-system -l app=grafana -o jsonpath='{.items[0].metadata.name}' --context=minikube) --context=minikube 3000:3000

# Flagger automated canary deployment demo
.PHONY: flagger
flagger:
	helm init; sleep 30
	helm repo add flagger https://flagger.app
	kubectl apply -f https://raw.githubusercontent.com/weaveworks/flagger/master/artifacts/flagger/crd.yaml --context=minikube
	helm upgrade -i flagger flagger/flagger \
        --namespace=istio-system \
        --set crd.create=false \
        --set meshProvider=istio \
        --set metricsServer=http://prometheus:9090

.PHONY: flagger-demo-bin
flagger-demo-bin:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o questions/canary/v1/app questions/canary/v1/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o questions/canary/v2/app questions/canary/v2/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o surveys/app surveys/main.go

.PHONY: flagger-demo-docker-build
flagger-demo-docker-build: flagger-demo-bin
	@eval $$(minikube docker-env) ; \
	docker build -t questions:latest -f questions/canary/v1/Dockerfile ./questions/canary/v1 ; \
	docker build -t questions:broken -f questions/canary/v2/Dockerfile ./questions/canary/v2 ; \
	docker build -t surveys:latest -f surveys/Dockerfile ./surveys
	@rm questions/canary/v1/app
	@rm questions/canary/v2/app
	@rm surveys/app

.PHONY: start-flagger-demo
start-flagger-demo: flagger-demo-docker-build
	@kubectl apply -f flagger/questions.yaml --context=minikube
	@kubectl apply -f flagger/canary.yaml --context=minikube
	@kubectl apply -f surveys/surveys.yaml --context=minikube

.PHONY: deploy-bad-service-flagger-demo
deploy-bad-service-flagger-demo:
	@kubectl set image deployment/questions questions=questions:broken --record --context=minikube

.PHONY: stop-flagger-demo
stop-flagger-demo:
	@kubectl delete -f surveys/surveys.yaml --context=minikube
	@kubectl delete -f flagger/questions.yaml --context=minikube
	@kubectl delete -f flagger/canary.yaml --context=minikube
