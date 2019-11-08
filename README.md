# Istio Demo (+ Flagger)

Deploying [Istio][istio] on a local [minikube][minikube] Kubernetes cluster.
1. Create a new cluster by running `make minikube`
2. Once cluster is ready, run `make istio` to apply Istio Custom Resource Definitions, 
label default namespace with `istio-injection=enabled` so that Istio can inject [Envoy][envoy] 
proxy sidecar containers to every pod deployed in default namespace.
3. Once you see `🚀 Istio is ready...` you are ready to proceed.
4. Either run `make start-demo` or `make start-canary-demo` to start either a simple service to 
service demo or to start canary deployment demo.
5. Generate traffic by running (first you need to port forward to appropriate pod:
  * `make traffic-surveys` - to generate traffic to surveys (for canary demo will generate 
traffic to `questions-v1` and `questions-v2`).
  * `traffic-surveys-user` - to generate traffic to surveys with specific user name in HTTP 
request headers (used for canary deployment demo to generate traffic to `questions-v3`).


### Visualising service mesh
Port-forward to Kiali UI by running `make kiali` and open http://localhost:20001.
![Kiali UI](images/canary-deployment.gif?raw=true "Kiali UI")  

### Distributed tracing
Port-forward to Jaeger UI by running `make jaeger` and open http://localhost:16686.
![Jaeger UI](images/jaeger.png?raw=true "Jaeger UI")  

### Istio Grafana dashboard
Port-forward to Grafana by running `make grafana` and open http://localhost:3000.

# Automated canary deployments with Flagger
Once Istio is ready, you will need to install [Flagger][flagger] by running `make flagger`, note that you will require 
[helm][helm] for this to work.  

Start Flagger demo by running `make start-flagger-demo`, you will want to port-forward to the `surveys` pod and generate 
some traffic by running `make traffic-surveys`. You can observe the automated canary deployment for `questions` service 
via Kiali UI. To start automated canary deploy run `make deploy-bad-service-flagger-demo`, this demonstrates automatic 
rollback when the new deployment results in elevated 500s.
![Flagger](images/flagger.gif?raw=true "Flagger")  


**Requirements**:  
[kubectl][kubectl] **(v1.13.10)**  
[minikube][minikube]  
[helm][helm]  

[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube]: https://kubernetes.io/docs/tasks/tools/install-minikube/
[istio]: https://istio.io/
[envoy]: https://www.envoyproxy.io/
[helm]: https://helm.sh/docs/using_helm/
[flagger]: https://docs.flagger.app/
