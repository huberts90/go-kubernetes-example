package k8s

import (
	"context"
	"fmt"

	"github.com/huberts90/go-k3d-nfs/internal/config"

	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	typed "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
)

// PodController manages Kubernetes pods (create, delete) in a namespace
type PodController struct {
	namespace    string
	podInterface typed.PodInterface
}

func NewPodController(
	namespace string,
) (*PodController, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to create InCluster configuration: %w", err)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create clientset: %w", err)
	}

	return &PodController{
		namespace:    namespace,
		podInterface: clientset.CoreV1().Pods(namespace),
	}, nil
}

// Create creates a Kubernetes pod
func (n *PodController) Create(
	logger *zap.Logger,
	ctx context.Context,
	podName string,
	imageName string,
	volumeMountPath string, // the path within the pod where the volume is mounted (e.g. /mnt/pv1)
	shellScript string, // the content of the shell script to run in the pod
) (watch.Interface, error) {
	const KEY_CONF_VOLUME_NAME = "keyconfig"
	_, err := n.podInterface.Create(
		ctx,
		&core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      podName,
				Namespace: n.namespace,
			},
			Spec: core.PodSpec{
				ServiceAccountName: config.GetString("signerPod.serviceAccount"),
				ImagePullSecrets: []core.LocalObjectReference{
					{
						Name: config.GetString("signerPod.imagePullSecret"),
					},
				},
				RestartPolicy: core.RestartPolicyNever,
				Containers: []core.Container{
					{
						Name:  "signer",
						Image: imageName,
						Command: []string{
							"sh",
						},
						Args: []string{
							"-c",
							shellScript,
						},
						VolumeMounts: []core.VolumeMount{
							{
								Name:      KEY_CONF_VOLUME_NAME,
								MountPath: "/etc/config",
								ReadOnly:  true,
							},
						},
					},
				},
				Volumes: []core.Volume{
					{
						Name: KEY_CONF_VOLUME_NAME,
						VolumeSource: core.VolumeSource{
							ConfigMap: &core.ConfigMapVolumeSource{
								LocalObjectReference: core.LocalObjectReference{
									Name: config.GetString("signerPod.keyConfigmap"),
								},
							},
						},
					},
				},
			},
		},
		meta.CreateOptions{},
	)
	if err != nil {
		return nil, err
	}

	watcher, err := n.podInterface.Watch(
		ctx,
		meta.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.namespace=%s,metadata.name=%s", n.namespace, podName),
		})
	if err != nil {
		return nil, err
	}

	return watcher, err
}

// Delete deletes a Kubernetes pod
func (n *PodController) Delete(
	ctx context.Context,
	podName string,
) error {
	return n.podInterface.Delete(ctx, podName, meta.DeleteOptions{})
}
