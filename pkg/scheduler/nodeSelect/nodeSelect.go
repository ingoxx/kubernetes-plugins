package nodeSelect

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const PluginName = "MyCustomFilter"
const AllowedLabelKey = "scheduler.alpha.io/allowed"

// MyCustomFilter 实现了 Filter 接口
type MyCustomFilter struct{}

// New 初始化插件
func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {

	// obj 用于接收配置参数 (如果有的话)
	return &MyCustomFilter{}, nil
}

// Name 返回插件的名称
func (m *MyCustomFilter) Name() string {
	return PluginName
}

// Filter 是核心方法，用于检查 Pod 是否可以在 Node 上运行
func (m *MyCustomFilter) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// 获取当前正在检查的 Node
	node := nodeInfo.Node()
	if node == nil {
		return framework.NewStatus(framework.Error, "node not found")
	}

	// 1. 检查 Node 是否有我们需要的标签
	if value, ok := node.Labels[AllowedLabelKey]; ok && value == "true" {
		// 标签存在且值为 "true"，允许调度到这个节点
		return framework.NewStatus(framework.Success, "")
	}

	// 2. 如果标签不匹配，则拒绝调度到这个节点
	// framework.Unschedulable 表示 Pod 无法被调度到这个节点
	reason := fmt.Sprintf("node %s does not have label %s=true", node.Name, AllowedLabelKey)
	return framework.NewStatus(framework.Unschedulable, reason)
}
