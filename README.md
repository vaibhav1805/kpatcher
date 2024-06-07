## Kpatcher
For faster & concurrent bulk patching of k8s resources.

## Setup
```
git clone https://github.com/vaibhav1805/kpatcher.git
cd kpatcher
npm install
```

## Usage Guide
1. Create a `js` file defining logic for filtering the resources meant to undergo patch. (refer `samples/filter.js`)
2. Create a `json` file with patch data.

```
Options:
  -t, --type <type>                  Resource type
  -n, --name <name>                  Resource name
  -ns, --namespace <namespace>       Namespace
  -A, --all-namespaces               Patch all namespaces
  -b, --batch-size <batchSize>       Batch size
  -p, --patch-file <file>            File with patch data (JSON format)
  -f, --filter <file>                JavaScript file for filtering resources
  --resource-name-same-as-namespace  Use the namespace as the resource name
  -h, --help                         display help for command
```

*Patch a specific resource in a namespace*
```
node index.js -t deployment -ns default -n app -b 2 -p patch.json
```

*Patch a reource all namspaces*
```
node index.js -t deployment -A -n app -b 5 -p patch.json -f filter.js
```

```
node index.js -t deployment -A -n app -b 5 -p patch.json -f filter.js --resource-name-same-as-namespace
```

### Filter
For filtering relevant resources you need to write a `js` file which contains logic to filter the resource. Template for the file should look like below

```
function filter(resource) {
    // Your filter logic
}
  
// Export the filter function
filter;
```

*Example*
Filter deployements with label `type:service` & ` spec.revisionHistoryLimit=2`

```
function filter(resource) {
    const spec = resource.spec;
    const labels = resource.metadata.labels;
    return (labels && labels["type"] === "service") && (spec && spec["revisionHistoryLimit"] == 2);
}
  
// Export the filter function
filter;
```