# Kpatcher
For filtering & patching k8s resources at scale.

# Run Locally
```shell
git clone https://github.com/vaibhav1805/kpatcher.git
cd kpatcher
```

### Generating Sample Data
```shell
./samples/manifest/create-resources.sh deployment.yaml 5
```

# Usage Guide
1. Create a `js` file defining logic for filtering the resources meant to undergo patch. (refer `samples/filter.js`)
2. Create a `json` file with patch data.  (refer `samples/patch.json`)

```
Options:
  --resource <resource>          Resource group
  --patch <file>                 File with patch data (JSON format)
  --filter <file>                JavaScript file for filtering resources
  --dynamic-patch <file>         Path to the patch Javascript file which returns patch json
  --batch-size <int>             Number of resources to patch concurrently in each batch
  --help                         display help for command
```
## Static Patch
Applying a constant patch across all resources. Example usage
```shell
go run cmd/main.go --resource=deployments.v1.apps --filter=./samples/filter/filter.js --patch=./samples/patch/patch.json
```

## Dynamic Patch
You can write custom logic in a `javascript` file which can to patch. Implement your logic  inside `createDynamicPatch` function.
```javascript
function createDynamicPatch(deploymentName) {
    // Your filter logic
}
```
*Example*
```javascript
const deployments = {
    "deployment-1": 5,
    "deployment-2": 1,
    "deployment-3": 3,
    "deployment-4": 4
}

function createDynamicPatch(deploymentName) {
    try {
        let patch = {
            "spec": {
                "replicas": deployments[deploymentName]
            }
        }
        return JSON.stringify(patch)
    } catch (err) {
        console.error('Error parsing the JSON:', err);
    }
    return {}
}

```

### Filter
For filtering relevant resources you need to write a `js` file which contains logic to filter the resource. Template for the file should look like below
```javascript
function filterResources(resource) {
    // Your filter logic
}
```

*Example*
Filter deployments with label  `metadata.annotations["kpatch"] === "enabled"`
```
function filterResources(resources) {
    let filtered =  resources.filter(resource => {
        return resource["metadata"]["annotations"] && resource["metadata"]["annotations"]["kpatch"] === "enabled";
    });
    return JSON.stringify(filtered)
}
```