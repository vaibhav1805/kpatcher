package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/go-node"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Command-line arguments
	var kubeconfig string
	var resource string
	var filterFile string
	var patchFile string

	flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "Path to the kubeconfig file")
	flag.StringVar(&resource, "resource", "", "Resource type (e.g., deployments.v1.apps)")
	flag.StringVar(&filterFile, "filter", "", "Path to the filter JavaScript file")
	flag.StringVar(&patchFile, "patch", "", "Path to the patch JSON file")
	flag.Parse()

	if resource == "" || filterFile == "" || patchFile == "" {
		fmt.Println("resource, filter, and patch are required arguments")
		return
	}

	// Load patch JSON file
	patch, err := ioutil.ReadFile(patchFile)
	if err != nil {
		panic(err.Error())
	}

	// Load kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Parse the resource type
	parts := strings.Split(resource, ".")
	if len(parts) != 3 {
		fmt.Println("resource should be in the format <resource>.<version>.<group>")
		return
	}

	resourceType := parts[0]
	resourceVersion := parts[1]
	resourceGroup := parts[2]

	// Create a GVR (GroupVersionResource)
	gvr := schema.GroupVersionResource{
		Group:    resourceGroup,
		Version:  resourceVersion,
		Resource: resourceType,
	}

	// List all resources in all namespaces
	resourceClient := dynamicClient.Resource(gvr)
	list, err := resourceClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, item := range list.Items {
		item.SetManagedFields([]metav1.ManagedFieldsEntry{})
		annotations := item.GetAnnotations()
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		item.SetAnnotations(annotations)
		item.Object["status"] = map[string]interface{}{}
	}

	// Convert list items to JSON
	listData, err := json.Marshal(list.Items)
	if err != nil {
		panic(err.Error())
	}

	// Load and execute the JavaScript filter file
	filterScript, err := ioutil.ReadFile(filterFile)
	if err != nil {
		panic(err.Error())
	}

	// Create a new JavaScript VM
	vm := node.New(&node.Options{OnError: func(msg string) {
		log.Printf("failed to create VM, %s", msg)
	}})

	// Define the filter function in JavaScript
	jsFilterScript := string(filterScript)

	// Execute the filter function
	jsCode := jsFilterScript + `
		filterResources(JSON.parse(` + "`" + string(listData) + "`" + `));
	`

	// Run the JavaScript code and get the result
	result := vm.Run(jsCode)
	if result.Error() != nil {
		panic(result.Error())
	}

	// Parse the filtered result
	var filteredItems []map[string]interface{}
	if err := json.Unmarshal([]byte(result.String()), &filteredItems); err != nil {
		panic(err.Error())
	}

	// Apply the patch to each filtered resource
	for _, item := range filteredItems {
		name := item["metadata"].(map[string]interface{})["name"].(string)
		namespace := item["metadata"].(map[string]interface{})["namespace"].(string)

		_, err := resourceClient.Namespace(namespace).Patch(context.TODO(), name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("Patched resource %q in namespace %q of type %q.\n", name, namespace, resource)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
