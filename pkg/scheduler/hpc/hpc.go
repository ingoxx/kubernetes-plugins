package hpc

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// GangPreFilterName 插件的名称
const GangPreFilterName = "GangPreFilter"
const GangIDLabel = "scheduling.sigs.k8s.io/gang-id"

type GangPreFilter struct {
	handle framework.Handle
}

func (g *GangPreFilter) Name() string {
	return GangPreFilterName
}

// PreFilter 检查 Pods 组是否有足够的整体资源
func (g *GangPreFilter) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	// 1. 检查 Pod 是否属于一个 Gang
	_, ok := pod.Labels[GangIDLabel]
	if !ok {
		return framework.NewStatus(framework.Success, "") // 不是 Gang Pod，通过
	}

	// 2. 查找集群中所有属于这个 Gang 的 Pods
	// 实际操作会查询 SharedLister 或缓存
	// 简化：假设我们需要 5 个 Pods 才能开始调度
	requiredPods := 5

	// 简化：假设我们已经计算出整个 Gang 所需的总资源
	// totalCPU, totalMem := calculateTotalResources(gangID)

	// 3. 检查是否有足够的 Node 资源来容纳所有 Pods (非常简化，仅检查计数)
	// 实际需要复杂的模拟调度或资源估算

	// 假设我们已经知道集群中 Ready 节点数量
	readyNodes, err := g.handle.SnapshotSharedLister().NodeInfos().List()
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, "Not enough ready nodes for the entire Gang")
	}

	if len(readyNodes) < requiredPods {
		return framework.NewStatus(framework.Unschedulable, "Not enough ready nodes for the entire Gang")
	}

	return framework.NewStatus(framework.Success, "")
}

// NewGangPreFilter 的 New 函数 (框架要求)
func NewGangPreFilter(runtime.Object, framework.Handle) (framework.Plugin, error) {
	return &GangPreFilter{}, nil
}
