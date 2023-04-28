package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
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
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	scale(clientset, "helloworld", "crccheck/hello-world", "latest", "50051", 5)
	// createStaticPod(clientset, "helloworld", "crccheck/hello-world", "latest", "50051")
	// fmt.Printf("%v\n", f)
	// scaleDown("helloworld")
}

func scale(clientset *kubernetes.Clientset, name string, image string, version string, port string, revisions int) {
	for i := 0; i < revisions; i++ {
		createStaticPod(clientset, name, image, version, port)
	}
}

func scaleUp(clientset *kubernetes.Clientset, name string, image string, version string, port string, revisions int) ([]string, error) {

	var podNames []string
	var e error
	var wg sync.WaitGroup
	for i := 0; i < revisions; i++ {
		wg.Add(1)
		go func() {
			podName, err := createStaticPod(clientset, name, image, version, port)
			if err != nil {
				e = err
			}
			podNames = append(podNames, podName)
			wg.Done()
		}()
	}
	wg.Wait()
	if e != nil {
		err := scaleDown(name)
		if err != nil {
			return nil, fmt.Errorf("error deleting partially scaled up %s function pods: %v", name, err)
		}
	}
	return podNames, e
}

func scaleDown(name string) error {

	files := []string{}
	err := filepath.Walk(podManifestDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasPrefix(info.Name(), name) {
			files = append(files, info.Name())
		}
		return nil
	})

	for _, file := range files {
		err = os.Remove(filepath.Join(podManifestDir, file))
		if err != nil {
			return fmt.Errorf("error deleting file: %v", err)
		}
	}
	return nil
}

func createStaticPod(clientset *kubernetes.Clientset, name string, image string, version string, port string) (string, error) {

	pod := name + "-" + randSeq(9) + "-" + randSeq(5)
	host, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("Unable to get hostname: %v", err)
	}
	podName := pod + "-" + host

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
      - containerPort: %s
`, pod, name, image, version, port)

	podManifestPath := filepath.Join(podManifestDir, pod+".yaml")

	startTime := time.Now()
	err = ioutil.WriteFile(podManifestPath, []byte(podManifest), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write pod manifest file: %v\n", err)
	}

	// Wait until pod is ready
	waitForPodReady(clientset, "default", podName)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	fmt.Printf("Static pod created with name %s in %v\n", podName, duration)

	return podName, nil
}

func invokeFunc(clientset *kubernetes.Clientset, namespace string, podName string) (*http.Response, error) {

	podIP, err := getPodIP(clientset, "default", podName)
	if err != nil {
		return &http.Response{}, err
	}

	// Invoke function on pod IP and port
	resp, err := http.Get(fmt.Sprintf("http://%s:%d", podIP, 8000))
	if err != nil {
		return &http.Response{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &http.Response{}, err
	}

	fmt.Printf("Function response%s\n", string(body))
	return resp, nil
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

func waitForPodReady(clientset *kubernetes.Clientset, namespace string, podName string) {

	for {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			continue
		}

		if pod.Status.Phase == corev1.PodRunning {
			break
		}

		time.Sleep(1 * time.Millisecond)
	}
}

func getPodIP(clientset *kubernetes.Clientset, namespace string, podName string) (string, error) {

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("unable to get pod IP: %v", err)
	}
	return pod.Status.PodIP, nil
}
