function filterResources(resources) {
    let filtered =  resources.filter(resource => {
        return resource["metadata"]["annotations"] && resource["metadata"]["annotations"]["kpatch"] === "enabled";
    });
    return JSON.stringify(filtered)
}
