package main

import (
	"fmt"

	"github.com/ingoxx/kubernetes-plugins/pkg/operator/demo"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

func main() {
	// 加载 ~/.kube/config 配置文件
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	// 创建 Clientset 与 APIServer 通信
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// === 第一阶段：初始化 Informer 机制 ===
	// 创建 SharedInformerFactory，它封装了对 APIServer 的 List & Watch 逻辑
	factory := informers.NewSharedInformerFactory(clientset, 0)
	podInformer := factory.Core().V1().Pods().Informer()

	// 初始化一个有速率限制的 WorkQueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// === 第二阶段：注册事件处理器 (Event Handlers) ===
	// 当 Informer 从 DeltaFIFO 取出事件后，会触发这里注册的回调函数
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// 生成 Key (例如: default/my-pod)
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				// 任务入队 (Enqueue)
				fmt.Printf("[事件 Event] 监听到 Pod 被添加: %s\n", key)
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				// 任务入队 (Enqueue)
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				// 任务入队 (Enqueue)
				fmt.Printf("[事件 Event] 监听到 Pod 被删除: %s\n", key)
				queue.Add(key)
			}
		},
	})

	// 实例化我们的 Controller
	controller := demo.NewController(queue, podInformer.GetIndexer(), podInformer)

	// 启动 Controller
	stopCh := make(chan struct{})
	defer close(stopCh)
	go controller.Run(1, stopCh)

	// 阻塞主线程
	select {}
}
