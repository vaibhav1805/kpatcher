const k8s = require('@kubernetes/client-node');
const { Command } = require('commander');
const async = require('async');
const fs = require('fs');
const vm = require('vm');

const program = new Command();
program
  .requiredOption('-t, --type <type>', 'Resource type')
  .option('-n, --name <name>', 'Resource name')
  .option('-ns, --namespace <namespace>', 'Namespace')
  .option('-A, --all-namespaces', 'Patch all namespaces')
  .requiredOption('-b, --batch-size <batchSize>', 'Batch size', parseInt)
  .requiredOption('-p, --patch-file <file>', 'File with patch data (JSON format)')
  .option('-f, --filter <file>', 'JavaScript file for filtering resources')
  .option('--resource-name-same-as-namespace', 'Use the namespace as the resource name');

program.parse(process.argv);

const { type, name, namespace, allNamespaces, batchSize, patchFile, filter, resourceNameSameAsNamespace } = program.opts();

const kc = new k8s.KubeConfig();
kc.loadFromDefault();

const k8sApiClient = kc.makeApiClient(k8s.KubernetesObjectApi);
const coreApi = kc.makeApiClient(k8s.CoreV1Api);

const loadPatchData = (filePath) => {
  const data = fs.readFileSync(filePath, 'utf8');
  return JSON.parse(data);
};

const patchData = loadPatchData(patchFile);

const patchResource = async (resourceType, resourceName, resourceNamespace, patch) => {
    console.log(`Patching ${resourceType} ${resourceName} in namespace ${resourceNamespace}`);
//   const options = {
//     headers: { 'Content-type': k8s.PatchUtils.PATCH_FORMAT_JSON_PATCH },
//   };

//   try {
//     await k8sApiClient.patchNamespacedCustomObject(
//       'apps',
//       'v1',
//       resourceNamespace,
//       resourceType,
//       resourceName,
//       patch,
//       undefined,
//       undefined,
//       undefined,
//       options
//     );
//     console.log(`Patched ${resourceType} ${resourceName} in namespace ${resourceNamespace}`);
//   } catch (error) {
//     console.error(`Failed to patch ${resourceType} ${resourceName} in namespace ${resourceNamespace}: ${error}`);
//   }
};

const getAllNamespaces = async () => {
  try {
    const res = await coreApi.listNamespace();
    return res.body.items.map((ns) => ns.metadata.name);
  } catch (error) {
    console.error('Failed to fetch namespaces:', error);
    return [];
  }
};

const getApiClientForResourceType = (resourceType) => {
  switch (resourceType.toLowerCase()) {
    case 'deployment':
      return kc.makeApiClient(k8s.AppsV1Api);
    case 'service':
      return kc.makeApiClient(k8s.CoreV1Api);
    case 'pod':
      return kc.makeApiClient(k8s.CoreV1Api);
    // Add more resource types and their corresponding API clients as needed
    default:
      throw new Error(`Unsupported resource type: ${resourceType}`);
  }
};

const getResources = async (resourceType, namespace) => {
  const client = getApiClientForResourceType(resourceType);

  switch (resourceType.toLowerCase()) {
    case 'deployment':
      return client.listNamespacedDeployment(namespace).then((res) => res.body.items);
    case 'service':
      return client.listNamespacedService(namespace).then((res) => res.body.items);
    case 'pod':
      return client.listNamespacedPod(namespace).then((res) => res.body.items);
    // Add more resource types and their corresponding methods as needed
    default:
      throw new Error(`Unsupported resource type: ${resourceType}`);
  }
};

const loadFilterFunction = (filePath) => {
  const code = fs.readFileSync(filePath, 'utf8');
  const script = new vm.Script(code);
  const sandbox = { filter: null };
  script.runInNewContext(sandbox);
  return sandbox.filter;
};

const createBatches = (array, batchSize) => {
  const batches = [];
  for (let i = 0; i < array.length; i += batchSize) {
    batches.push(array.slice(i, i + batchSize));
  }
  return batches;
};

const patchResources = async (namespaces, filterFunction) => {
  let allResources = [];

  // Collect resources from all specified namespaces
  for (const ns of namespaces) {
    const resources = await getResources(type, ns);
    const resourceJSON = JSON.parse(JSON.stringify(resources));

    const filteredResources = resourceJSON.filter(filterFunction);
    allResources.push(...filteredResources);
  }

  const resourceBatches = createBatches(allResources, batchSize);

  for (const batch of resourceBatches) {
    console.log(batch)
    await async.each(batch, async (resource) => {
      const resourceNamespace = resource.metadata.namespace;
      const resourceName = resourceNameSameAsNamespace ? resourceNamespace : resource.metadata.name;
      await patchResource(type, resourceName, resourceNamespace, patchData);
    });
  }
};

(async () => {
  let namespaces = [];
  if (allNamespaces) {
    namespaces = await getAllNamespaces();
  } else if (namespace) {
    namespaces = [namespace];
  } else {
    console.error('Either --namespace or --all-namespaces must be specified.');
    return;
  }

  let filterFunction = () => true; // Default filter: no filtering
  if (filter) {
    try {
      filterFunction = loadFilterFunction(filter);
    } catch (error) {
      console.error('Failed to load filter function:', error);
      return;
    }
  }

  await patchResources(namespaces, filterFunction);
})();
