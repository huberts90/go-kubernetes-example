package main

import (
	"context"
	"time"

	"github.com/huberts90/go-k3d-nfs/internal"
	"github.com/huberts90/go-k3d-nfs/internal/config"
	"github.com/huberts90/go-k3d-nfs/internal/constants"
	"github.com/huberts90/go-k3d-nfs/internal/k8s"

	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func main() {
	logger := internal.NewZapLogger()
	config.Load(logger, "config/config.yaml")
	logger.Info("Creating pod", zap.String("POD", config.GetString("signerPod.namespace")))

	podNamespace := config.GetString("signerPod.namespace")
	podController, err := k8s.NewPodController(podNamespace)
	if err != nil {
		logger.Error("Failed to create PodController", zap.Error(err))
		return
	}

	podName := "signer-pod"
	const volumeMountRemotePath = "/mnt/pv1"
	watcher, err := podController.Create(
		logger,
		context.Background(),
		podName,
		"alpine:3.14.0",
		volumeMountRemotePath,
		"echo 'Hello world from signer pod'",
	)
	if err != nil {
		logger.Error("Failed to create pod", zap.Error(err))
		return
	}
	logger.Info("Pod created", zap.String(constants.NAMESPACE, podNamespace), zap.String(constants.NAME, podName))
	defer func() {
		// Stop watching the pod and delete the pod
		watcher.Stop()
		err = podController.Delete(context.Background(), podName)
		if err != nil {
			logger.Error("Failed to delete pod", zap.String(constants.NAMESPACE, podNamespace), zap.String(constants.NAME, podName), zap.Error(err))
		}
		logger.Info("Pod deleted", zap.String(constants.NAMESPACE, podNamespace), zap.String(constants.NAME, podName))
	}()

	isRunning := false
	chRunning := make(chan bool)

	go func() {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				continue // we don't care about those events
			case watch.Modified:
				pod, ok := event.Object.(*core.Pod)
				if !ok {
					logger.Error("Event is not for a pod. Check your watch interface.")
					continue
				}
				if pod.Name != podName {
					logger.Error("Event is not for the right pod. Check your field selector.", zap.String("Pod name", pod.Name))
					continue
				}

				ilogger := logger.With(
					zap.String("Pod Phase", string(pod.Status.Phase)),
					zap.String("Pod Message", pod.Status.Message),
					zap.String("Pod Reason", string(pod.Status.Reason)))

				if len(pod.Status.ContainerStatuses) == 0 {
					ilogger.Debug("Pod event without a container status.")
					continue
				}
				if len(pod.Status.ContainerStatuses) > 1 {
					ilogger.Error("Unexpected container")
					continue
				}

				if pod.Status.ContainerStatuses[0].State.Terminated != nil {
					terminated := pod.Status.ContainerStatuses[0].State.Terminated
					ilogger.Info("Container Terminated",
						zap.Int32("Exit code", terminated.ExitCode),
						zap.String("Terminated Reason", terminated.Reason),
						zap.String("Terminated Message", terminated.Message))

					break // we can stop watching
				} else if pod.Status.ContainerStatuses[0].State.Running != nil {
					ilogger.Info("Container running")
					if !isRunning {
						isRunning = true
						chRunning <- true
					}
				}
			case watch.Deleted:
				logger.Info("Pod was deleted")
			default:
				logger.Error("Unexpected event type", zap.String("Event type", string(event.Type)))
			}
			if event.Type == watch.Added {
				continue
			}
		}
	}()

	// Wait for the pod to run
	<-chRunning

	// The pod is running, it has some fixed time to terminate
	timer := time.NewTimer(config.GetDurationDefault("signerPod.timeout", 30*time.Second))

	// Wait for pod terminates or timeout
	select {
	case <-timer.C:
		logger.Info("Timeout. Deleting pod")
	}

	logger.Info("Task completed")
}
