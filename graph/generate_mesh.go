package graph

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"text/template"
)

type service struct {
	idx      int
	edges    []int
	replicas int
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
        {{- if eq .reachableServices "none" }}
        kuma.io/transparent-proxying-reachable-services: ""
        {{- else }}
        kuma.io/transparent-proxying-reachable-services: "{{.reachableServices }}"
        {{- end}}
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

func toUri(idx int, namespace string) string {
	return fmt.Sprintf("http://%s.mesh:80", toKumaService(idx, namespace))
}

func toKumaService(idx int, namespace string) string {
	return fmt.Sprintf("%s_%s_svc_80", toName(idx), namespace)
}

func toName(idx int) string {
	return fmt.Sprintf("srv-%03d", idx)
}

func (s service) ToYaml(writer io.Writer, namespace, mesh, image string, withReachableServices bool) error {

	opt := map[string]interface{}{
		"name":              toName(s.idx),
		"namespace":         namespace,
		"mesh":              mesh,
		"uris":              strings.Join(s.mapEdges(func(i int) string { return toUri(i, namespace) }), ","),
		"image":             image,
		"replicas":          s.replicas,
		"reachableServices": "",
	}
	if withReachableServices {
		if len(s.edges) == 0 {
			opt["reachableServices"] = "none"
		} else {
			opt["reachableServices"] = strings.Join(s.mapEdges(func(i int) string { return toKumaService(i, namespace) }), ",")
		}
	}
	return srvTemplate.Execute(writer, opt)
}
func (s service) mapEdges(fn func(int) string) []string {
	var all []string
	for _, edge := range s.edges {
		all = append(all, fn(edge))
	}
	return all
}

type Services []service

func (s Services) ToDot() string {
	var allEdges []string
	for _, srv := range s {
		for _, other := range srv.edges {
			allEdges = append(allEdges, fmt.Sprintf("%d -> %d;", srv.idx, other))
		}
	}
	return fmt.Sprintf("digraph{\n%s\n}\n", strings.Join(allEdges, "\n"))
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
		params := map[string]string{"namespace": conf.Namespace, "mesh": conf.Mesh, "uri": toUri(0, conf.Namespace), "reachableServices": ""}
		if conf.WithReachableServices {
			params["reachableServices"] = toKumaService(0, conf.Namespace)
		}
		if err := clientTemplate.Execute(writer, params); err != nil {
			return err
		}
	}
	for _, srv := range s {
		if _, err := writer.Write([]byte("---")); err != nil {
			return err
		}
		if err := srv.ToYaml(writer, conf.Namespace, conf.Mesh, conf.Image, conf.WithReachableServices); err != nil {
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
		srvs = append(srvs, service{idx: i, replicas: numInstances})
	}
	// That's the whole story of DAG and topological sort with triangular matrix.
	for i := 0; i < numServices; i++ {
		for j := i + 1; j < numServices; j++ {
			if r.Int()%(j-i) == 0 && r.Int()%100 < percentEdges {
				srvs[i].edges = append(srvs[i].edges, j)
			}
		}
	}
	return srvs
}
