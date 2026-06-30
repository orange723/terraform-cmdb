package inventory

// InstancesPayload builds the JSON response shape for machine asset APIs.
func InstancesPayload(snapshot Snapshot) map[string]any {
	return map[string]any{
		"file_name":     snapshot.FileName,
		"terraform":     snapshot.Terraform,
		"raw_resources": snapshot.RawResources,
		"count":         len(snapshot.Machines),
		"source_files":  snapshot.SourceFiles,
		"instances":     snapshot.Machines,
	}
}
