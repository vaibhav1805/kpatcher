package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tidwall/go-node"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	TypeFilter       = "filter"
	TypePatch        = "patch"
	TypePatchDynamic = "dynamic"
)

type JSRunner struct {
	VM               node.VM
	logger           *log.Logger
	filterFile       string
	patchFile        string
	dynamicPatchFile string
}

func NewJSRunner(logger *log.Logger, filterFile, patchFile, dynamicPatchFile string) *JSRunner {
	vm := node.New(&node.Options{OnError: func(msg string) {
		logger.Panic("failed to create JavaScript VM")
	}})

	return &JSRunner{
		VM:               vm,
		logger:           logger,
		filterFile:       filterFile,
		patchFile:        patchFile,
		dynamicPatchFile: dynamicPatchFile,
	}
}

func (j *JSRunner) readScript(file string) ([]byte, error) {
	script, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return script, nil
}

func (j *JSRunner) executeCode(codeType string, data string) (string, error) {
	code := ""
	switch codeType {
	case TypeFilter:
		file := j.filterFile
		script, err := j.readScript(file)
		if err != nil {
			return "", err
		}
		code = string(script) + `
			filterResources(JSON.parse(` + "`" + data + "`" + `));
		`
	case TypePatchDynamic:
		// Todo add logic for data loader
		file := j.dynamicPatchFile
		script, err := j.readScript(file)
		if err != nil {
			return "", err
		}
		code = string(script) + `
			createDynamicPatch(` + "'" + data + "'" + `);
		`
	case TypePatch:
		file := j.patchFile
		script, err := j.readScript(file)
		if err != nil {
			return "", err
		}
		return string(script), nil
	}

	result := j.VM.Run(code)
	if result.Error() != nil {
		return "", result.Error()
	}
	return result.String(), nil
}

type KPatcher struct {
	ctx       context.Context
	logger    *log.Logger
	client    dynamic.NamespaceableResourceInterface
	gvr       schema.GroupVersionResource
	jsRunner  *JSRunner
	patch     []byte
	patchType string
	batchSize int
}

func NewKPatcher() (*KPatcher, error) {
	var (
		kubeconfig       string
		resource         string
		filterFile       string
		patchFile        string
		dynamicPatchFile string
		batchSize        int
	)

	logger := log.New(os.Stderr, "kpatcher", log.LstdFlags)

	flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "Path to the kubeconfig file")
	flag.StringVar(&resource, "resource", "", "Resource type (e.g., deployments.v1.apps)")
	flag.StringVar(&filterFile, "filter", "", "Path to the filter JavaScript file")
	flag.StringVar(&patchFile, "patch", "", "Path to the patch JSON file")
	flag.StringVar(&dynamicPatchFile, "dynamic-patch", "", "Path to the patch Javascript file which returns patch json")
	flag.IntVar(&batchSize, "batch-size", 5, "Number of resources to patch concurrently in each batch")
	flag.Parse()

	if resource == "" || filterFile == "" || (patchFile == "" && dynamicPatchFile == "") {
		fmt.Println("resource, filter, and patch/dynamic-patch required arguments")
		return &KPatcher{}, errors.New("missing required arguments")
	}

	patchType := TypePatch
	if dynamicPatchFile != "" {
		patchType = TypePatchDynamic
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
		log.Print("resource should be in the format <resource>.<version>.<group>")
		return &KPatcher{}, errors.New("invalid resource format")
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

	// Initialise JS runner
	jsRunner := NewJSRunner(logger, filterFile, patchFile, dynamicPatchFile)

	return &KPatcher{
		ctx:       context.TODO(),
		logger:    logger,
		client:    resourceClient,
		gvr:       gvr,
		jsRunner:  jsRunner,
		patchType: patchType,
		batchSize: batchSize,
	}, nil
}

func (k *KPatcher) list() (*unstructured.UnstructuredList, error) {
	list, err := k.client.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Remove manged stuff which messes up json un/marshalling
	for _, item := range list.Items {
		item.SetManagedFields([]metav1.ManagedFieldsEntry{})
		annotations := item.GetAnnotations()
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		item.SetAnnotations(annotations)
		item.Object["status"] = map[string]interface{}{}
	}
	return list, nil
}

func (k *KPatcher) filter(list *unstructured.UnstructuredList) ([]map[string]interface{}, error) {
	listData, err := json.Marshal(list.Items)
	if err != nil {
		panic(err.Error())
	}
	result, err := k.jsRunner.executeCode(TypeFilter, string(listData))
	if err != nil {
		return nil, err
	}

	var filteredItems []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &filteredItems); err != nil {
		panic(err.Error())
	}

	return filteredItems, nil
}

func (k *KPatcher) execute() error {
	list, err := k.list()
	if err != nil {
		return err
	}

	filteredItems, err := k.filter(list)
	if err != nil {
		return err
	}

	// Apply the patch to each filtered resource
	var wg sync.WaitGroup
	sem := make(chan struct{}, k.batchSize)

	for _, item := range filteredItems {
		wg.Add(1)
		sem <- struct{}{}

		go func(item map[string]interface{}) {
			defer wg.Done()
			defer func() { <-sem }()

			name := item["metadata"].(map[string]interface{})["name"].(string)
			namespace := item["metadata"].(map[string]interface{})["namespace"].(string)

			patch, err := k.jsRunner.executeCode(k.patchType, name)
			if err != nil {
				log.Print(fmt.Errorf("failed to create patch for the resource: %s, error: %w", name, err))
			}

			_, err = k.client.Namespace(namespace).Patch(context.TODO(), name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
			if err != nil {
				log.Print(fmt.Errorf("failed to patch the resource: %s, error: %w", name, err))
			}
			fmt.Printf("Patched resource %q in namespace %q of type %q.\n", name, namespace, k.gvr.Resource)
		}(item)
		wg.Wait()
	}
	return nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func main() {
	patcher, err := NewKPatcher()
	if err != nil {
		log.Fatalf("failed to create patcher, %v", err)
	}

	if err = patcher.execute(); err != nil {
		log.Fatalf("failed to patch resource, %v", err)
	}
}
