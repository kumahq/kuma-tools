package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kumahq/kuma-tools/graph"
)

func main() {
	conf := graph.ServiceConf{
		WithNamespace: true,
		WithMesh:      true,
	}
	flag.BoolVar(&conf.WithGenerator, "withGenerator", false, "Whether we should start a job that generates synthetic load to the first service")
	flag.StringVar(&conf.Namespace, "namespace", "kuma-test", "The name of the namespace to deploy to")
	flag.StringVar(&conf.Mesh, "mesh", "default", "The name of the mesh to deploy to")
	flag.StringVar(&conf.Image, "image", "nicholasjackson/fake-service:v0.25.2", "The fake-service image")
	flag.BoolVar(&conf.WithReachableServices, "withReachableServices", true, "Whether we should use reachable services or not")
	flag.BoolVar(&conf.WithReachableBackends, "withReachableBackends", true, "Whether we should use reachable backends or not")
	flag.BoolVar(&conf.WithExternalPrometheus, "withExternalPrometheus", false, "Whether we should use a prometheus inside or outside the mesh")
	flag.BoolVar(&conf.WithKubeURIs, "withKubeURIs", false, "Whether we should use Kubenetes DNS names")
	flag.StringVar(&conf.MeshServicesMode, "meshServicesMode", "", "MeshServices mode, for example Exclusive")
	numServices := flag.Int("numServices", 20, "The number of services to use")
	minReplicas := flag.Int("minReplicas", 1, "The minimum number of replicas to use (will pick a number between min and max)")
	maxReplicas := flag.Int("maxReplicas", 1, "The max number of replicas to use (will pick a number between min and max)")
	percentEdge := flag.Int("percentEdge", 50, "The for an edge between 2 nodes to exist (100 == sure)")
	seed := flag.Int64("seed", time.Now().Unix(), "the seed for the random generate (set to now by default)")
	output := flag.String("output", "yaml", "output format (yaml,dot,mermaid)")
	flag.Parse()

	fmt.Printf("# Using seed: %d\n", *seed)
	srvs := graph.GenerateRandomServiceMesh(*seed, *numServices, *percentEdge, *minReplicas, *maxReplicas)
	var err error
	switch *output {
	case "yaml":
		err = srvs.ToYaml(os.Stdout, conf)
	case "dot":
		err = srvs.ToDot(os.Stdout)
	case "mermaid":
		err = srvs.ToMermaid(os.Stdout)
	default:
		err = fmt.Errorf("format '%s' not supported accepted format: yaml, dot, mermaid", *output)
	}
	if err != nil {
		panic(any(err))
	}
}
