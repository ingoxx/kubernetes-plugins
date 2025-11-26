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

	// 1. 检查 Pod 是否请求了特定的 GPU 型号 (简化判断)
	if pod.ObjectMeta.Annotations["requires-gpu-model"] != "A100" {
		// 如果 Pod 没有特定要求，通过
		return framework.NewStatus(framework.Success, "")
	}

	// 2. 检查 Node 是否有 A100 标签
	if model, ok := node.Labels[RequiredGpuModel]; !ok || model != "A100" {
		return framework.NewStatus(framework.Unschedulable, "Node does not have A100 GPU label")
	}

	// 3. 检查 NUMA 拓扑（这里是概念性占位，实际需要更复杂的库和 Node 资源拓扑信息）
	// 假设 Node 的标签表明它支持 NUMA 感知调度
	if _, ok := node.Labels["topology.aware/numa-ready"]; !ok {
		return framework.NewStatus(framework.Unschedulable, "Node is not NUMA topology ready")
	}

	return framework.NewStatus(framework.Success, "")
}

// NewGpuTopologyFilter 的 New 函数 (框架要求)
func NewGpuTopologyFilter(runtime.Object, framework.Handle) (framework.Plugin, error) {
	return &GpuTopologyFilter{}, nil
}
