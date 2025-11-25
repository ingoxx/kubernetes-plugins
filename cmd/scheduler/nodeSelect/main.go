package main

import (
	"github.com/ingoxx/kubernetes-plugins/pkg/scheduler/nodeSelect"
	"os"

	"k8s.io/klog/v2"

	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin(nodeSelect.PluginName, nodeSelect.New),
	)

	if err := command.Execute(); err != nil {
		klog.Fatalf("scheduler failed: %v", err)
		os.Exit(1)
	}
}
