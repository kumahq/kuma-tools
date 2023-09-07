package graph

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"text/template"
)

type Service struct {
	Idx      int
	Edges    []int
	Replicas int
}

var srvTemplate = template.Must(template.New("").Parse(`
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{.name}}
  {{- if .namespace}}
  namespace: {{.namespace}}
  {{- end}}
  labels:
    app: {{.name}}
spec:
  replicas: {{.replicas}}
  selector:
    matchLabels:
      app: {{.name}}
  serviceName: {{.name}}
  template:
    metadata:
      labels:
        app: {{.name}}
      annotations:
        kuma.io/mesh: {{.mesh}}
        {{- if ne .reachableServices "" }}
        kuma.io/transparent-proxying-reachable-services: "{{.reachableServices }}"
        {{- end}}
    spec:
      containers:
        - name: service
          image: {{.image}}
          ports:
            - containerPort: 9090
          env:
            - name: SERVICE
              value: "{{.name}}"
            - name: UPSTREAM_URIS
              value: "{{.uris}}"
          resources:
            limits:
              memory: "32Mi"
              cpu: "50m"
---
apiVersion: v1
kind: Service
metadata:
  name: {{.name}}
  {{- if .namespace}}
  namespace: {{.namespace}}
  {{- end}}
  annotations:
spec:
  selector:
    app: {{.name}}
  ports:
    - protocol: TCP
      appProtocol: http
      port: 80
      targetPort: 9090
`)).Option("missingkey=error")

var clientTemplate = template.Must(template.New("").Parse(`
---
apiVersion: apps/v1
kind: Deployment
metadata:
  {{- if .namespace}}
  namespace: {{.namespace}}
  {{- end}}
  name: "fake-client"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: "fake-client"
  template:
    metadata:
      labels:
        app: "fake-client" 
      annotations:
        kuma.io/mesh: {{.mesh}}
        {{- if ne .reachableServices "" }}
        kuma.io/transparent-proxying-reachable-services: "{{.reachableServices }}"
        {{- end}}
    spec:
      containers:
        - name: client
          image: buoyantio/slow_cooker:1.3.0
          args: ["-qps", "1", "-concurrency", "10", "{{.uri}}"]
          resources:
            limits:
              memory: "32Mi"
              cpu: "200m"
`)).Option("missingkey=error")

var meshTemplate = template.Must(template.New("").Parse(`
---
apiVersion: kuma.io/v1alpha1
kind: Mesh
metadata:
  name: {{.mesh}}
spec:
  metrics:
    backends:
    - conf:
        {{- if .externalPrometheus }}
        skipMTLS: true
        {{- end }}
        path: /metrics
        port: 5670
        tags:
          kuma.io/service: dataplane-metrics
      name: prometheus-1
      type: prometheus
    enabledBackend: prometheus-1
  mtls:
    backends:
    - name: ca-1
      type: builtin
    enabledBackend: ca-1
`)).Option("missingkey=error")

var namespaceTemplate = template.Must(template.New("").Parse(`
---
apiVersion: v1
kind: Namespace
metadata:
  name: {{.namespace}}
  labels:
   kuma.io/sidecar-injection: enabled
`)).Option("missingkey=error")

func ToUri(idx int, namespace string) string {
	return fmt.Sprintf("http://%s.mesh:80", ToKumaService(idx, namespace))
}

func ToKumaService(idx int, namespace string) string {
	return fmt.Sprintf("%s_%s_svc_80", ToName(idx), namespace)
}

func ToName(idx int) string {
	return fmt.Sprintf("srv-%03d", idx)
}

func (s Service) Uris(namespace string) string {
	return strings.Join(s.mapEdges(func(i int) string { return ToUri(i, namespace) }), ",")
}

func (s Service) mapEdges(fn func(int) string) []string {
	var all []string
	for _, edge := range s.Edges {
		all = append(all, fn(edge))
	}
	return all
}

func (s Service) ToReachableServices(namespace string) string {
	if len(s.Edges) == 0 {
		return "non-existing-service"
	}
	return strings.Join(s.mapEdges(func(i int) string { return ToKumaService(i, namespace) }), ",")
}

type Services []Service

func (s Services) ToDot(writer io.Writer) error {
	var allEdges []string
	for _, srv := range s {
		for _, other := range srv.Edges {
			allEdges = append(allEdges, fmt.Sprintf("%d -> %d;", srv.Idx, other))
		}
	}
	_, err := fmt.Fprintf(writer, "digraph{\n%s\n}\n", strings.Join(allEdges, "\n"))
	return err
}

func (s Services) ToMermaid(writer io.Writer) error {
	var allEdges []string
	for _, srv := range s {
		for _, other := range srv.Edges {
			allEdges = append(allEdges, fmt.Sprintf("\t%d --> %d;", srv.Idx, other))
		}
	}
	_, err := fmt.Fprintf(writer, "graph TD;\n%s\n\n", strings.Join(allEdges, "\n"))
	return err
}

func (s Services) ToYaml(writer io.Writer, conf ServiceConf) error {
	if conf.WithNamespace {
		if err := namespaceTemplate.Execute(writer, map[string]interface{}{"namespace": conf.Namespace}); err != nil {
			return err
		}
	}
	if conf.WithMesh {
		if err := meshTemplate.Execute(writer, map[string]interface{}{"mesh": conf.Mesh, "externalPrometheus": conf.WithExternalPrometheus}); err != nil {
			return err
		}
	}
	if conf.WithGenerator {
		params := map[string]string{"namespace": conf.Namespace, "mesh": conf.Mesh, "uri": ToUri(0, conf.Namespace), "reachableServices": ""}
		if conf.WithReachableServices {
			params["reachableServices"] = ToKumaService(0, conf.Namespace)
		}
		if err := clientTemplate.Execute(writer, params); err != nil {
			return err
		}
	}
	for _, srv := range s {
		if _, err := writer.Write([]byte("---")); err != nil {
			return err
		}
		opt := map[string]interface{}{
			"name":             ToName(srv.Idx),
			"namespace":        conf.Namespace,
			"mesh":             conf.Mesh,
			"uris":             srv.Uris(conf.Namespace),
			"image":            conf.Image,
			"replicas":         srv.Replicas,
			"reachableService": "",
		}
		if conf.WithReachableServices {
			opt["reachableServices"] = srv.ToReachableServices(conf.Namespace)
		}
		if err := srvTemplate.Execute(writer, opt); err != nil {
			return err
		}
	}
	return nil
}

type ServiceConf struct {
	WithFailure            bool
	WithGenerator          bool
	WithReachableServices  bool
	WithNamespace          bool
	WithMesh               bool
	Namespace              string
	Mesh                   string
	Image                  string
	WithExternalPrometheus bool
}

func GenerateRandomServiceMesh(seed int64, numServices, percentEdges, minReplicas, maxReplicas int) Services {
	r := rand.New(rand.NewSource(seed))
	srvs := Services{}
	for i := 0; i < numServices; i++ {
		numInstances := 1
		if maxReplicas >= minReplicas {
			numInstances = (r.Int() % (1 + maxReplicas - minReplicas)) + minReplicas
		}
		srvs = append(srvs, Service{Idx: i, Replicas: numInstances})
	}
	// That's the whole story of DAG and topological sort with triangular matrix.
	for i := 0; i < numServices; i++ {
		for j := i + 1; j < numServices; j++ {
			if r.Int()%(j-i) == 0 && r.Int()%100 < percentEdges {
				srvs[i].Edges = append(srvs[i].Edges, j)
			}
		}
	}
	return srvs
}
