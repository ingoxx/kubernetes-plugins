package main

import (
	"os"

	"k8s.io/klog/v2"

	"github.com/ingoxx/kubernetes-plugins/pkg/scheduler"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin(scheduler.PluginName, scheduler.New),
	)

	if err := command.Execute(); err != nil {
		klog.Fatalf("scheduler failed: %v", err)
		os.Exit(1)
	}
}
