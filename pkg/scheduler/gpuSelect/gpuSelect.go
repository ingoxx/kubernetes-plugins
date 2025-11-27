package gpuSelect

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const GpuFilterName = "GpuTopologyFilter"
const RequiredGpuModel = "nvidia.com/gpu.model" // 假设节点上有这种标签

type GpuTopologyFilter struct{}

func (g *GpuTopologyFilter) Name() string {
	return GpuFilterName
}

// Filter 检查节点是否满足 GPU 及其拓扑要求
func (g *GpuTopologyFilter) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	node := nodeInfo.Node()

	// 1. 检查 Pod 是否请求了特定的 GPU 型号 (A100)
	requiredModel, ok := pod.ObjectMeta.Annotations["requires-gpu-model"]

	// 如果 Pod 没有 Annotation 或请求的不是 A100，直接通过（不干预）
	// 您可以根据实际需求修改这里的逻辑，但我们假设只有请求A100的Pod才需要拓扑检查
	if !ok || requiredModel != "A100" {
		return framework.NewStatus(framework.Success, "")
	}

	// --- 只有请求 A100 的 Pod 才会执行下面的检查 ---

	// 2. 检查 Node 是否有 A100 标签
	if model, ok := node.Labels[RequiredGpuModel]; !ok || model != "A100" {
		// 调度器会拒绝此 Pod，并给出原因
		return framework.NewStatus(framework.Unschedulable, "Node does not have A100 GPU label as required by pod")
	}

	// 3. 检查 NUMA 拓扑
	// 修正：检查标签是否存在且值为 "true"
	if value, ok := node.Labels["topology.aware/numa-ready"]; !ok || value != "true" {
		return framework.NewStatus(framework.Unschedulable, "Node is not NUMA topology ready or label is false")
	}

	// 所有检查通过
	return framework.NewStatus(framework.Success, "")
}

// NewGpuTopologyFilter 的 New 函数 (框架要求)
func NewGpuTopologyFilter(runtime.Object, framework.Handle) (framework.Plugin, error) {
	return &GpuTopologyFilter{}, nil
}
