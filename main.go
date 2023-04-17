package main

import (
	"context"
	"fmt"
	"time"
    "net/http"


	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	podName := "helloworld-static"
	serviceName := "helloworld-static-service"
    servicePort := int32(80)
	imageName := "testcontainers/helloworld"

	namespace := "static"
	// Create the new namespace
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		fmt.Println(err)
	}

	startTime := time.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "my-container",
					Image: imageName,
				},
			},
		},
	}

	pod, err = clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	fmt.Printf("Static pod created with name %s in %v\n", pod.GetName(), duration)

	// Create the Nginx service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "nginx",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       servicePort,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}

	_, err = clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	duration = time.Now().Sub(startTime)
	fmt.Printf("Nginx service created with name %s in %v\n", service.GetName(), duration)

	// Invoke the Nginx service
	url := fmt.Sprintf("http://%s:%d", serviceName, servicePort)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	
	duration = time.Now().Sub(startTime)
	fmt.Println("Response status %v:", duration, resp.Status)
}
