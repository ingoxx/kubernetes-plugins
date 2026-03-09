package demo

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller 结构体：包含了我们在架构图中提到的核心组件
type Controller struct {
	indexer  cache.Indexer                   // Local Cache (本地缓存)
	informer cache.Controller                // Informer (包含内部的 Reflector 和 DeltaFIFO)
	queue    workqueue.RateLimitingInterface // WorkQueue (限速工作队列)
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
	}
}

// === 第三阶段：调谐逻辑 (Reconcile) ===
// 这里的业务逻辑是：对比期望状态与实际状态，并执行操作
func (c *Controller) syncHandler(key string) error {
	// 1. 从 Local Cache (Indexer) 中获取资源对象，而不是去请求 K8s APIServer
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		fmt.Printf("从缓存获取对象失败: %s\n", err.Error())
		return err
	}

	if !exists {
		// 场景C (删除)：在缓存中找不到，说明资源已经被删除了
		fmt.Printf("[调谐 Act] Pod %s 已经被删除，执行清理逻辑...\n", key)
	} else {
		// 场景A/B (新建/更新)：获取到了资源，执行调谐逻辑
		pod := obj.(*corev1.Pod)
		fmt.Printf("[调谐 Act] 发现 Pod 变更 -> Namespace: %s, Name: %s, 当前状态: %s\n",
			pod.Namespace, pod.Name, pod.Status.Phase)
		// ⚠️ 这里就是你的 Operator 领域知识大展身手的地方！
		// 比如：判断如果是 Redis CR，就去调用 Redis API 分配 Slots
	}
	return nil
}

// 消费者 Worker：不断从 WorkQueue 取出任务
func (c *Controller) processNextItem() bool {
	// 5. 从 WorkQueue 中取出一个 Key
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// 告诉队列这个 key 我们处理完了
	defer c.queue.Done(key)

	// 6. 调用调谐函数
	err := c.syncHandler(key.(string))
	if err == nil {
		// 10. 处理成功，将 key 从队列里“遗忘”（重置重试次数）
		c.queue.Forget(key)
		return true
	}

	// 如果处理失败，则重新放回队列，稍后重试 (限速退避机制)
	fmt.Printf("处理失败 %v, 重新入队: %v\n", key, err)
	c.queue.AddRateLimited(key)
	return true
}

// 启动 Worker
func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

// Controller 的启动入口
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer c.queue.ShutDown()

	fmt.Println("正在启动 Controller...")
	// 启动 Informer (底层会启动 Reflector 去 APIServer List&Watch 资源)
	go c.informer.Run(stopCh)

	// 等待 Local Cache 同步完成 (确保 List 完成，DeltaFIFO 里的数据都存入了 Indexer)
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		fmt.Println("等待缓存同步超时")
		return
	}
	fmt.Println("本地缓存同步完成！启动 Workers 开始处理...")

	// 启动多个 Worker 协程，不断消费队列
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	fmt.Println("Controller 停止")
}

//func main() {
//	// 加载 ~/.kube/config 配置文件
//	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
//	if err != nil {
//		panic(err)
//	}
//	// 创建 Clientset 与 APIServer 通信
//	clientset, err := kubernetes.NewForConfig(config)
//	if err != nil {
//		panic(err)
//	}
//
//	// === 第一阶段：初始化 Informer 机制 ===
//	// 创建 SharedInformerFactory，它封装了对 APIServer 的 List & Watch 逻辑
//	factory := informers.NewSharedInformerFactory(clientset, 0)
//	podInformer := factory.Core().V1().Pods().Informer()
//
//	// 初始化一个有速率限制的 WorkQueue
//	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
//
//	// === 第二阶段：注册事件处理器 (Event Handlers) ===
//	// 当 Informer 从 DeltaFIFO 取出事件后，会触发这里注册的回调函数
//	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
//		AddFunc: func(obj interface{}) {
//			// 生成 Key (例如: default/my-pod)
//			key, err := cache.MetaNamespaceKeyFunc(obj)
//			if err == nil {
//				// 任务入队 (Enqueue)
//				fmt.Printf("[事件 Event] 监听到 Pod 被添加: %s\n", key)
//				queue.Add(key)
//			}
//		},
//		UpdateFunc: func(old interface{}, new interface{}) {
//			key, err := cache.MetaNamespaceKeyFunc(new)
//			if err == nil {
//				// 任务入队 (Enqueue)
//				queue.Add(key)
//			}
//		},
//		DeleteFunc: func(obj interface{}) {
//			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
//			if err == nil {
//				// 任务入队 (Enqueue)
//				fmt.Printf("[事件 Event] 监听到 Pod 被删除: %s\n", key)
//				queue.Add(key)
//			}
//		},
//	})
//
//	// 实例化我们的 Controller
//	controller := NewController(queue, podInformer.GetIndexer(), podInformer)
//
//	// 启动 Controller
//	stopCh := make(chan struct{})
//	defer close(stopCh)
//	go controller.Run(1, stopCh)
//
//	// 阻塞主线程
//	select {}
//}
