package binPacking

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"strconv"
)

// BinPackScoreName 是插件的名称
const BinPackScoreName = "DynamicBinPackScore"

// RealtimeCPUUtilizationAnnotation AnnotationKey 假定 Node Annotations 中存储了实时 CPU 利用率百分比
const RealtimeCPUUtilizationAnnotation = "custom.scheduler.io/realtime-cpu-percent"

type DynamicBinPackScore struct {
	// framework.Handle 允许插件访问调度器内部状态和 API 客户端
	handle framework.Handle
}

func (b *DynamicBinPackScore) Name() string {
	return BinPackScoreName
}

// Score 评分阶段：计算 Node 的得分
func (b *DynamicBinPackScore) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {

	// 1. 获取 NodeInfo
	nodeInfo, err := b.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		// 如果找不到 Node 信息，返回错误状态
		return 0, framework.AsStatus(err)
	}
	node := nodeInfo.Node()

	// 2. 从 Node Annotations 中获取实时 CPU 利用率
	utilizationStr, ok := node.Annotations[RealtimeCPUUtilizationAnnotation]
	if !ok {
		// 如果没有实时数据（例如，监控 Agent 故障或 Node 不支持），
		// 为了不影响调度，我们返回一个中性分数（例如 50分）
		return framework.MaxNodeScore / 2, framework.NewStatus(framework.Success, "Missing utilization data, returning neutral score")
	}

	// 3. 将字符串解析为浮点数或整数 (假设是 0-100 的百分比)
	// 生产环境中应有更健壮的错误处理
	utilizationFloat, err := strconv.ParseFloat(utilizationStr, 64)
	if err != nil {
		// 解析失败，返回中性分数
		return framework.MaxNodeScore / 2, framework.NewStatus(framework.Success, "Failed to parse utilization data, returning neutral score")
	}

	// 4. 计算得分 (Bin Packing 策略)
	// 目标：利用率越高，得分越高。
	// 将 0-100% 的利用率直接映射到 [0, framework.MaxNodeScore] (即 [0, 100])
	score := int64(utilizationFloat)

	// 5. 确保分数在有效范围内 [0, 100]
	if score < framework.MinNodeScore {
		score = framework.MinNodeScore
	} else if score > framework.MaxNodeScore {
		score = framework.MaxNodeScore
	}

	// 调试输出（生产环境中应避免过度日志记录）
	// fmt.Printf("Node %s utilization: %f%%, Score: %d\n", nodeName, utilizationFloat, score)

	return score, framework.NewStatus(framework.Success, "")
}

// ScoreExtensions 接口
func (b *DynamicBinPackScore) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// NewDynamicBinPackScore 是工厂函数 (框架要求)
func NewDynamicBinPackScore(_ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &DynamicBinPackScore{handle: handle}, nil
}
