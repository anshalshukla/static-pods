package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const podManifestDir = "/etc/kubernetes/manifests"

func main() {
	static("helloworld", "crccheck/hello-world", "latest", "50051")
}

func static(name string, image string, version string, port string) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	podName := name + "-" + randSeq(9) + "-" + randSeq(5)
	startTime := time.Now()
	podManifest := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
spec:
  containers:
  - name: %s
    image: %s:%s
    imagePullPolicy: IfNotPresent
    ports:
      - name: h2c
        containerPort: %s
`, podName, name, image, version, port)

	podManifestPath := filepath.Join(podManifestDir, podName+".yaml")

	err = ioutil.WriteFile(podManifestPath, []byte(podManifest), 0644)
	if err != nil {
		fmt.Printf("Failed to write pod manifest file: %v\n", err)
		panic(err)
	}

	fmt.Printf("Created pod manifest file at %s\n", podManifestPath)

	// Wait until pod is ready
	podName = podName + "-node-0.anshal-155872.ntu-cloud-pg0.cloudlab.umass.edu"
	waitForPodReady(clientset, podName)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	fmt.Printf("Static pod created with name %s in %v\n", podName, duration)

	podIP, err := getPodIP(clientset, "default", podName)
	if err != nil {
		panic(err)
	}

	// Invoke function on pod IP and port
	resp, err := http.Get(fmt.Sprintf("http://%s:%d", podIP, 8000))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Function response%s\n", string(body))
}

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func waitForPodReady(clientset *kubernetes.Clientset, podName string) {
	for {
		pod, err := clientset.CoreV1().Pods("default").Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			continue
		}
		pod, err = clientset.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
		if err != nil {
			panic(err)
		}

		if pod.Status.Phase == corev1.PodRunning {
			break
		}

		time.Sleep(1 * time.Millisecond)
	}
}

func getPodIP(clientset *kubernetes.Clientset, namespace string, name string) (string, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Status.PodIP, nil
}
