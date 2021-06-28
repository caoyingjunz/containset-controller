/*
Copyright 2021 The Pixiu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package imageset

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	appsv1alpha1 "github.com/caoyingjunz/pixiu/pkg/apis/apps/v1alpha1"
	appsClient "github.com/caoyingjunz/pixiu/pkg/client/clientset/versioned/typed/apps/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetHostname returns env's hostname if 'hostnameOverride' is empty; otherwise, return 'hostnameOverride'.
func GetHostName(hostnameOverride string) (string, error) {
	hostName := hostnameOverride

	if len(hostName) == 0 {
		hostName = os.Getenv("NODE_NAME")
	}

	// Trim whitespaces first to avoid getting an empty hostname
	// For linux, the hostname is read from file /proc/sys/kernel/hostname directly
	hostName = strings.TrimSpace(hostName)
	if len(hostName) == 0 {
		return "", fmt.Errorf("empty hostname is invaild")
	}

	return strings.ToLower(hostName), nil
}

const (
	dockerSocket = "/var/run/docker.sock"
	dockerHost   = "unix://" + dockerSocket

	containerdSocket = "/run/containerd/containerd.sock"
)

// isExistingSocket checks if path exists and is domain socket
func isExistingSocket(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.Mode()&os.ModeSocket != 0
}

func updateImageSetStatus(c appsClient.ImageSetInterface, ims *appsv1alpha1.ImageSet, newStatus appsv1alpha1.ImageSetStatus) (*appsv1alpha1.ImageSet, error) {
	if ims.Status.Image == newStatus.Image &&
		ims.Status.ObservedGeneration == ims.Generation &&
		reflect.DeepEqual(ims.Status.Nodes, newStatus.Nodes) {
		return ims, nil
	}

	newStatus.ObservedGeneration = ims.Generation
	ims.Status = newStatus

	return c.UpdateStatus(context.TODO(), ims, metav1.UpdateOptions{})
}

func calculateImageSetStatus(c appsClient.ImageSetInterface, name, hostName, imageRef string, err error) appsv1alpha1.ImageSetStatus {
	ims, err := c.Get(context.TODO(), name, metav1.GetOptions{})

	nodes := ims.Status.Nodes
	if nodes == nil {
		nodes = make([]appsv1alpha1.ImageSetNodes, 0)
	}

	sn := appsv1alpha1.ImageSetNodes{
		LastUpdateTime: metav1.Now(),
		NodeName:       hostName,
		ImageId:        imageRef,
	}
	if err != nil {
		sn.Message = err.Error()
	}

	index := -1
	for i, node := range nodes {
		if node.NodeName == hostName {
			index = i
			break
		}
	}

	if index == -1 {
		nodes = append(nodes, sn)
	} else {
		nodes[index] = sn
	}

	newStatus := appsv1alpha1.ImageSetStatus{
		Image: ims.Spec.Image,
		Nodes: nodes,
	}

	return newStatus
}
