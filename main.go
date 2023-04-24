package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
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
	static("nginx")
}

func static(revision string) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	podName := revision + "-" + randSeq(9) + "-" + randSeq(5)
	// podName := revision
	imageName := revision

	namespace := "static"
	// Create the new namespace
	_, _ = clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})

	startTime := time.Now()
	// pod := &corev1.Pod{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      podName,
	// 		Namespace: namespace,
	// 	},
	// 	Spec: corev1.PodSpec{
	// 		Containers: []corev1.Container{
	// 			{
	// 				Name:  revision,
	// 				Image: imageName,
	// 				Ports: []corev1.ContainerPort{
	// 					{
	// 						Name:          "http",
	// 						ContainerPort: 8080,
	// 						Protocol:      corev1.ProtocolTCP,
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	// // Serialize the pod to YAML
	// podYaml, err := serializeObject(pod)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// // Write the pod YAML to a file
	// err = ioutil.WriteFile("my-static-pod.yaml", []byte(podYaml), 0644)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// pod, err = clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	// if err != nil {
	// 	panic(err)
	// }

	imageTag := "latest"

	podManifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
spec:
  containers:
  - name: %s
    image: %s:%s
`, podName, imageName, imageName, imageTag)

	podManifestPath := filepath.Join(podManifestDir, podName+".yaml")

	err = ioutil.WriteFile(podManifestPath, []byte(podManifest), 0644)
	if err != nil {
		fmt.Printf("Failed to write pod manifest file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created pod manifest file at %s\n", podManifestPath)

	// podName = podName + "-node-0.anshal-155406.ntu-cloud-pg0.utah.cloudlab.us"
	// pod, err := clientset.CoreV1().Pods("default").Get(context.Background(), podName, metav1.GetOptions{})
	// if err != nil {
	// 	panic(err.Error())
	// }

	// Wait until pod is ready
	podName = podName + "-node-0.anshal-155406.ntu-cloud-pg0.utah.cloudlab.us"
	waitForPodReady(clientset, podName)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	fmt.Printf("Static pod created with name %s in %v\n", podName, duration)

	// podIP, err := getPodIP(clientset, namespace, podName)
	// if err != nil {
	// 	panic(err)
	// }

	// // Invoke function on pod IP and port
	// resp, err := http.Get(fmt.Sprintf("http://%s:%d", podIP, 8080))
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()

	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Printf("Function response%s\n", string(body))
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

// func serializeObject(obj interface{}) (string, error) {
// 	serializer := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: true, Pretty: true})
// 	result := ""
// 	err := serializer.Encode(obj, &result)
// 	return result, err
// }
