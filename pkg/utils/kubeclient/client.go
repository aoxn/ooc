/*
Copyright 2016 The Kubernetes Authors.

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
package kubeclient

import (
	"context"
	"fmt"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClientSetFromFile returns a ready-to-use client from a KubeConfig file
func ClientSetFromFile(path string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load admin kubeconfig [%v]", err)
	}
	return ToClientSet(config)
}

// ToClientSet converts a KubeConfig object to a client
func ToClientSet(config *clientcmdapi.Config) (*kubernetes.Clientset, error) {
	clientConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client configuration from kubeconfig: %v", err)
	}

	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %v", err)
	}
	return client, nil
}

// CreateOrUpdateSecret creates a Secret if the target resource doesn't exist. If the resource exists already, this function will update the resource instead.
func CreateOrUpdateSecret(client kubernetes.Interface, secret *v1.Secret) error {
	if _, err := client.CoreV1().Secrets(secret.ObjectMeta.Namespace).Create(context.TODO(),secret,metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create secret: %v", err)
		}

		if _, err := client.CoreV1().Secrets(secret.ObjectMeta.Namespace).Update(context.TODO(),secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("unable to update secret: %v", err)
		}
	}
	return nil
}

// CreateOrUpdateSecret creates a Secret if the target resource doesn't exist. If the resource exists already, this function will update the resource instead.
func CreateOrUpdateConfigMap(client kubernetes.Interface, cm *v1.ConfigMap) error {

	if _, err := client.CoreV1().ConfigMaps(cm.Namespace).Create(context.TODO(),cm,metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create configmap: %v", err)
		}

		if _, err := client.CoreV1().ConfigMaps(cm.ObjectMeta.Namespace).Update(context.TODO(),cm, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("unable to update configmap: %v", err)
		}
	}
	return nil
}

func EnsureNamespace(client kubernetes.Interface, namespace string) error {
	// should with retry.
	if _, err := client.CoreV1().
		Namespaces().
		Create(
			context.TODO(),
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
			metav1.CreateOptions{},
		); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func LoadClientFromConfig(config *clientcmdapi.Config) (kubernetes.Interface, error) {
	rest, err := clientcmd.BuildConfigFromKubeconfigGetter(
		"",
		func() (*clientcmdapi.Config, error) {
			return config, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(rest)
}

func FindSecret(client kubernetes.Interface, namespace, name string) (*v1.Secret, bool, error) {

	secret, err := client.
		CoreV1().
		Secrets(namespace).
		Get(context.TODO(),name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return secret, true, nil
}

func FindMasters(client kubernetes.Interface) ([]v1.Node, bool, error) {

	nodes, err := client.
		CoreV1().Nodes().List(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: "node-role.kubernetes.io/master",
			},
		)
	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return nodes.Items, true, nil
}

func FindConfigMap(client kubernetes.Interface, namespace, name string) (*v1.ConfigMap, bool, error) {

	cm, err := client.
		CoreV1().
		ConfigMaps(namespace).
		Get(context.TODO(),name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return cm, true, nil
}
