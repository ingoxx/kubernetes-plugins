package resourceSpread

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// ResourceSpreadName 插件名
const ResourceSpreadName = "ResourceSpreadScore"

const cycleStateKey = "resource-spread/pod-requests"

type podRequests struct {
	milliCPU int64
	memory   int64
}

func (p *podRequests) Clone() framework.StateData {
	if p == nil {
		return &podRequests{}
	}
	return &podRequests{
		milliCPU: p.milliCPU,
		memory:   p.memory,
	}
}

type ResourceSpreadScore struct {
	handle framework.Handle
}

func (r *ResourceSpreadScore) Name() string {
	return ResourceSpreadName
}

// PreFilter 计算并缓存 Pod 的 CPU/内存请求量，避免后续重复计算
func (r *ResourceSpreadScore) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	req := getPodRequests(pod)
	state.Write(cycleStateKey, req)
	return framework.NewStatus(framework.Success, "")
}

func (r *ResourceSpreadScore) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// Filter 保证节点可用资源足够容纳该 Pod
func (r *ResourceSpreadScore) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	req := readPodRequests(state, pod)

	alloc := nodeInfo.Allocatable
	requested := nodeInfo.Requested

	availableCPU := alloc.MilliCPU - requested.MilliCPU
	availableMem := alloc.Memory - requested.Memory

	if req.milliCPU > availableCPU {
		return framework.NewStatus(framework.Unschedulable, "insufficient cpu")
	}
	if req.memory > availableMem {
		return framework.NewStatus(framework.Unschedulable, "insufficient memory")
	}

	return framework.NewStatus(framework.Success, "")
}

// Score 倾向选择当前资源利用率更低的节点，从而尽量“用上”更多设备
func (r *ResourceSpreadScore) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := r.nodeInfoFromSnapshot(nodeName)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	req := readPodRequests(state, pod)

	alloc := nodeInfo.Allocatable
	requested := nodeInfo.Requested

	allocCPU := alloc.MilliCPU
	allocMem := alloc.Memory
	if allocCPU <= 0 || allocMem <= 0 {
		return 0, framework.NewStatus(framework.Success, "zero allocatable")
	}

	usedCPU := requested.MilliCPU + req.milliCPU
	usedMem := requested.Memory + req.memory

	cpuUtil := float64(usedCPU) / float64(allocCPU)
	memUtil := float64(usedMem) / float64(allocMem)
	if cpuUtil < 0 {
		cpuUtil = 0
	}
	if memUtil < 0 {
		memUtil = 0
	}
	if cpuUtil > 1 {
		cpuUtil = 1
	}
	if memUtil > 1 {
		memUtil = 1
	}

	// 以 CPU/内存平均利用率作为综合利用率，利用率越低，得分越高
	util := (cpuUtil + memUtil) / 2
	score := int64((1 - util) * float64(framework.MaxNodeScore))
	if score < framework.MinNodeScore {
		score = framework.MinNodeScore
	} else if score > framework.MaxNodeScore {
		score = framework.MaxNodeScore
	}

	return score, framework.NewStatus(framework.Success, "")
}

func (r *ResourceSpreadScore) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func (r *ResourceSpreadScore) nodeInfoFromSnapshot(nodeName string) (*framework.NodeInfo, error) {
	return r.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
}

// NewResourceSpreadScore 创建插件实例
func NewResourceSpreadScore(_ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &ResourceSpreadScore{handle: handle}, nil
}

func getPodRequests(pod *v1.Pod) *podRequests {
	var cpuMilli int64
	var memBytes int64
	var initCPU int64
	var initMem int64
	for i := range pod.Spec.Containers {
		req := pod.Spec.Containers[i].Resources.Requests
		if req == nil {
			continue
		}
		if q, ok := req[v1.ResourceCPU]; ok {
			cpuMilli += q.MilliValue()
		}
		if q, ok := req[v1.ResourceMemory]; ok {
			memBytes += q.Value()
		}
	}
	for i := range pod.Spec.InitContainers {
		req := pod.Spec.InitContainers[i].Resources.Requests
		if req == nil {
			continue
		}
		var c int64
		var m int64
		if q, ok := req[v1.ResourceCPU]; ok {
			c = q.MilliValue()
		}
		if q, ok := req[v1.ResourceMemory]; ok {
			m = q.Value()
		}
		if c > initCPU {
			initCPU = c
		}
		if m > initMem {
			initMem = m
		}
	}
	if initCPU > cpuMilli {
		cpuMilli = initCPU
	}
	if initMem > memBytes {
		memBytes = initMem
	}
	if pod.Spec.Overhead != nil {
		if q, ok := pod.Spec.Overhead[v1.ResourceCPU]; ok {
			cpuMilli += q.MilliValue()
		}
		if q, ok := pod.Spec.Overhead[v1.ResourceMemory]; ok {
			memBytes += q.Value()
		}
	}
	return &podRequests{milliCPU: cpuMilli, memory: memBytes}
}

func readPodRequests(state *framework.CycleState, pod *v1.Pod) *podRequests {
	if state == nil {
		return getPodRequests(pod)
	}
	v, err := state.Read(cycleStateKey)
	if err != nil {
		return getPodRequests(pod)
	}
	req, ok := v.(*podRequests)
	if !ok || req == nil {
		return getPodRequests(pod)
	}
	return req
}
