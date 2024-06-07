function filter(resource) {
    const spec = resource.spec;
    const labels = resource.metadata.labels;
    return (labels && labels["type"] === "service") && (spec && spec["revisionHistoryLimit"] == 2);
}
  
// Export the filter function
filter;