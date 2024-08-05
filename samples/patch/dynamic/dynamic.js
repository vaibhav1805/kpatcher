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
