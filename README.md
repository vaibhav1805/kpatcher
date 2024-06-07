## Kpatcher
For filtering & patching k8s resources at scale.

## Run Locally
```shell
git clone https://github.com/vaibhav1805/kpatcher.git
cd kpatcher
```

### Generating Sample Data
```shell
./samples/manifest/create-resources.sh deployment.yaml 1
```

## Usage Guide
1. Create a `js` file defining logic for filtering the resources meant to undergo patch. (refer `samples/filter.js`)
2. Create a `json` file with patch data.  (refer `samples/patch.json`)

```
Options:
  -r, --resource <resource>          Resource group
  -p, --patch <file>                 File with patch data (JSON format)
  -f, --filter <file>                JavaScript file for filtering resources
  -h, --help                         display help for command
```

```shell
go run cmd/main.go --resource=deployments.v1.apps --filter=./samples/filter.js --patch=./samples/patch.json
```

### Filter
For filtering relevant resources you need to write a `js` file which contains logic to filter the resource. Template for the file should look like below
```
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