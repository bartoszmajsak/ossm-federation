package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jewertow/federation/internal/pkg/config"
	"github.com/jewertow/federation/internal/pkg/mcp"
	server "github.com/jewertow/federation/internal/pkg/xds"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Global variable to store the parsed arguments and "flag" arguments
var (
	meshPeer = flag.String("meshPeers", "",
		"Mesh peers that include address ip/hostname to remote Peer, and the ports for dataplane and discovery")
	exportedServiceSet = flag.String("exportedServiceSet", "",
		"ExportedServiceSet that include selectors to match the services that will be exported")
	importedServiceSet = flag.String("importedServiceSet", "",
		"ImportedServiceSet that include selectors to match the services that will be imported")
)

// unmarshalJSON is a utility function to unmarshal a YAML string into a struct
// and return an error if the unmarshalling fails.
func unmarshalJSON(input string, out interface{}) error {
	if err := json.Unmarshal([]byte(input), out); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// Parse the command line arguments by using the flag package
// Export the parsed arguments to the AppConfig variable
func parse() (*config.Federation, error) {
	var (
		peers    config.MeshPeers
		exported config.ExportedServices
		imported config.ImportedServices
	)

	if err := unmarshalJSON(*meshPeer, &peers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mesh peers: %w", err)
	}
	if err := unmarshalJSON(*exportedServiceSet, &exported); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exported services: %w", err)
	}
	if err := unmarshalJSON(*importedServiceSet, &imported); err != nil {
		return nil, fmt.Errorf("failed to unmarshal imported services: %w", err)
	}

	return &config.Federation{
		MeshPeers:          peers,
		ExportedServiceSet: exported,
		ImportedServiceSet: imported,
	}, nil
}

func main() {
	flag.Parse()
	cfg, err := parse()
	if err != nil {
		os.Exit(1)
	}
	fmt.Println("Configuration: ", cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal("failed to create in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Fatalf("failed to create Kubernetes clientset: %s", err.Error())
	}

	pushMCP := make(chan []*anypb.Any)

	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	serviceInformer := informerFactory.Core().V1().Services().Informer()

	stopCh := make(chan struct{})
	defer close(stopCh)

	w := mcp.NewExportedServiceSetWatcher(*cfg, serviceInformer, pushMCP)
	if err := w.AddHandlers(serviceInformer); err != nil {
		klog.Fatal("failed to init watcher for exported service set: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(ctx, pushMCP); err != nil {
			log.Fatal("Error starting server: ", err)
		}
	}()

	// TODO: informers should be somehow notified by the server that it already started and receives connection
	time.Sleep(5 * time.Second)
	informerFactory.Start(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, serviceInformer.HasSynced); !ok {
		klog.Fatalf("Failed to wait for caches to sync")
	}

	wg.Wait()
	os.Exit(0)
}
