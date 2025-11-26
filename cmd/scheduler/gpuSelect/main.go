package main

import (
	"github.com/ingoxx/kubernetes-plugins/pkg/scheduler/gpuSelect"
	"os"

	"k8s.io/klog/v2"

	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin(gpuSelect.GpuFilterName, gpuSelect.NewGpuTopologyFilter),
	)

	if err := command.Execute(); err != nil {
		klog.Fatalf("scheduler failed: %v", err)
		os.Exit(1)
	}
}
