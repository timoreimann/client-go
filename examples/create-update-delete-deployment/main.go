/*
Copyright 2017 The Kubernetes Authors.

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

// Note: the example only works with the code within the same release/branch.
package main

import (
	"flag"
	"fmt"
	"path/filepath"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

const deploymentName = "demo-deployment"

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	v1beta1DeploymentsClient := clientset.AppsV1beta1().Deployments(apiv1.NamespaceDefault)

	pp := metav1.DeletePropagationBackground
	err = v1beta1DeploymentsClient.Delete(deploymentName, &metav1.DeleteOptions{
		GracePeriodSeconds: int64Ptr(0),
		PropagationPolicy:  &pp,
	})

	switch {
	case err == nil:
		fmt.Println("Deleted deployment.")
	case err != nil:
		if !errors.IsNotFound(err) {
			panic(fmt.Sprintf("Failed to delete deployment: %s", err))
		}
		fmt.Println("Deployment does not exist.")
	}

	deployment := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	resultV1, err := v1beta1DeploymentsClient.Create(deployment)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q in version %q.\n", resultV1.GetObjectMeta().GetName(), resultV1.TypeMeta.APIVersion)

	// Update Deployment
	fmt.Println("Updating deployment...")

	v1beta2DeploymentsClient := clientset.AppsV1beta2().Deployments(apiv1.NamespaceDefault)

	// Retrieve the latest version of Deployment before attempting update
	resultV2, getErr := v1beta2DeploymentsClient.Get(deploymentName, metav1.GetOptions{})
	if getErr != nil {
		panic(fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := v1beta2DeploymentsClient.Get(deploymentName, metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
		}

		var updateErr error
		resultV2, updateErr = v1beta2DeploymentsClient.Update(result)
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("Update failed: %v", retryErr))
	}

	fmt.Printf("Updated deployment %q in version %q.\n", resultV2.GetObjectMeta().GetName(), resultV2.TypeMeta.APIVersion)
}

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
