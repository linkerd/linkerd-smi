package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linkerd/linkerd-smi/pkg/adaptor"
	spclientset "github.com/linkerd/linkerd2/controller/gen/client/clientset/versioned"
	k8sAPI "github.com/linkerd/linkerd2/controller/k8s"
	"github.com/linkerd/linkerd2/pkg/admin"
	"github.com/linkerd/linkerd2/pkg/k8s"
	tsclientset "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	tsinformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	log "github.com/sirupsen/logrus"
)

func main() {
	cmd := flag.NewFlagSet("smi-adaptor", flag.ExitOnError)

	kubeConfigPath := cmd.String("kubeconfig", "", "path to kube config")
	metricsAddr := cmd.String("metrics-addr", ":9995", "address to serve scrapable metrics on")
	clusterDomain := cmd.String("cluster-domain", "cluster.local", "kubernetes cluster domain")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{}, 1)

	go func() {
		// Close done when stop signal is received
		<-stop
		close(done)
	}()

	log.Info("Using cluster domain: ", *clusterDomain)
	ctx := context.Background()
	config, err := k8s.GetConfig(*kubeConfigPath, "")
	if err != nil {
		log.Fatalf("error configuring Kubernetes API client: %v", err)
	}

	k8sAPI, err := k8sAPI.InitializeAPI(
		ctx,
		*kubeConfigPath,
		false,
		k8sAPI.SP, k8sAPI.TS,
	)
	if err != nil {
		log.Fatalf("Failed to initialize K8s API: %s", err)
	}

	// Create SP and TS clientsets
	spClient, err := spclientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building serviceprofile clientset: %s", err.Error())
	}

	tsClient, err := tsclientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building trafficsplit clientset: %s", err.Error())
	}

	tsInformerFactory := tsinformers.NewSharedInformerFactory(tsClient, 10*time.Minute)

	controller := adaptor.NewController(
		k8sAPI.Client,
		*clusterDomain,
		tsClient,
		spClient,
		tsInformerFactory.Split().V1alpha1().TrafficSplits(),
	)

	// Start the Admin Server
	go admin.StartServer(*metricsAddr)

	tsInformerFactory.Start(done)

	// Run the controller until a shutdown signal is received
	if err = controller.Run(done); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}

	log.Info("Shutting down")
}
