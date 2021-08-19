package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"text/template"
	"time"
)

type service struct {
	idx   int
	edges []int
}

var srvTemplate, _ = template.New("").Parse(`
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
  replicas: 1
  selector:
    matchLabels:
      app: {{.name}}
  serviceName: {{.name}}
  template:
    metadata:
      labels:
        app: {{.name}}
    spec:
      containers:
        - name: service
          image: nicholasjackson/fake-service:v0.21.1
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
              cpu: "100m"
---
apiVersion: v1
kind: Service
metadata:
  name: {{.name}}
  {{- if .namespace}}
  namespace: {{.namespace}}
  {{- end}}
  annotations:
    80.service.kuma.io/protocol: http
spec:
  selector:
    app: {{.name}}
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9090
`)

var clientTemplate, _ = template.New("").Parse(`
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
    spec:
      containers:
        - name: client
          image: buoyantio/slow_cooker:1.3.0
          args: ["-qps", "1", "-concurrency", "10", "{{.uri}}"]
          resources:
            limits:
              memory: "32Mi"
              cpu: "200m"
`)

func toUri(idx int) string {
	return fmt.Sprintf("http://%s_kuma-test_svc_80.mesh:80", toName(idx))
}

func toName(idx int) string {
	return fmt.Sprintf("srv-%03d", idx)
}

func (s service) ToYaml(writer io.Writer, namespace string) error {
	return srvTemplate.Execute(writer, map[string]string{
		"name":      toName(s.idx),
		"namespace": namespace,
		"uris":      s.toUris(),
	})
}

func (s service) toUris() string {
	all := []string{}
	for _, edge := range s.edges {
		all = append(all, toUri(edge))
	}
	return strings.Join(all, ",")
}

type services []service

func (s services) ToDot() string {
	allEdges := []string{}
	for _, srv := range s {
		for _, other := range srv.edges {
			allEdges = append(allEdges, fmt.Sprintf("%d -> %d;", srv.idx, other))
		}
	}
	return fmt.Sprintf("digraph{\n%s\n}\n", strings.Join(allEdges, "\n"))
}

func (s services) ToYaml(writer io.Writer, conf serviceConf) error {
	if conf.namespace != "" {
		_, err := writer.Write([]byte(fmt.Sprintf(`
apiVersion: kuma.io/v1alpha1
kind: Mesh
metadata:
  name: default
spec:
  metrics:
    backends:
    - conf:
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
---
apiVersion: v1
kind: Namespace
metadata:
  name: %s
  annotations:
   kuma.io/sidecar-injection: enabled
`, conf.namespace)))
		if err != nil {
			return err
		}
	}
	if conf.withGenerator {
		if err := clientTemplate.Execute(writer, map[string]string{"namespace": conf.namespace, "uri": toUri(0)}); err != nil {
			return err
		}
	}
	for _, srv := range s {
		if _, err := writer.Write([]byte("---")); err != nil {
			return err
		}
		if err := srv.ToYaml(writer, conf.namespace); err != nil {
			return err
		}
	}
	return nil
}

type serviceConf struct {
	withFailure   bool
	withGenerator bool
	namespace     string
}

func GenerateRandomServiceMesh(seed int64, numServices int, percentEdges int) services {
	r := rand.New(rand.NewSource(seed))
	srvs := services{}
	for i := 0; i < numServices; i++ {
		srvs = append(srvs, service{idx: i})
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

func main() {
	conf := serviceConf{}
	flag.BoolVar(&conf.withGenerator, "withGenerator", false, "Whether we should start a job that generates synthetic load to the first service")
	flag.StringVar(&conf.namespace, "namespace", "", "The name of the namespace to deploy to")
	numServices := flag.Int("numServices", 20, "The number of services to use")
	percentEdge := flag.Int("percentEdge", 50, "The for an edge between 2 nodes to exist (100 == sure)")
	seed := flag.Int64("seed", time.Now().Unix(), "the seed for the random generate (set to now by default)")
	flag.Parse()

	fmt.Printf("# Using seed: %d\n", *seed)
	srvs := GenerateRandomServiceMesh(*seed, *numServices, *percentEdge)
	srvs.ToYaml(os.Stdout, conf)
}
