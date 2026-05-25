package terraformstate

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"sort"
	"strconv"
	"strings"

	"terraform-cmdb/internal/inventory"

	"gopkg.in/yaml.v3"
)

type ParseResult struct {
	Terraform    string
	RawResources int
	Machines     []inventory.Machine
}

type resourceEntry struct {
	Address    string
	Provider   string
	Type       string
	Name       string
	IndexKey   any
	Attributes map[string]any
}

type stateFile struct {
	Version   int             `json:"version"`
	Terraform string          `json:"terraform_version"`
	Resources []stateResource `json:"resources"`
	Values    *stateValues    `json:"values"`
}

type stateValues struct {
	RootModule valueModule `json:"root_module"`
}

type valueModule struct {
	Resources    []valueResource `json:"resources"`
	ChildModules []valueModule   `json:"child_modules"`
}

type valueResource struct {
	Address string         `json:"address"`
	Mode    string         `json:"mode"`
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Values  map[string]any `json:"values"`
}

type stateResource struct {
	Mode      string          `json:"mode"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Provider  string          `json:"provider"`
	Instances []stateInstance `json:"instances"`
}

type stateInstance struct {
	IndexKey   any            `json:"index_key,omitempty"`
	Attributes map[string]any `json:"attributes"`
}

func Parse(content []byte) (ParseResult, error) {
	var state stateFile
	if err := json.Unmarshal(content, &state); err != nil {
		return ParseResult{}, fmt.Errorf("Terraform state JSON 解析失败: %w", err)
	}

	resources := collectResources(state)
	allResources := collectAllResources(state)
	machines := buildMachines(resources)
	linkPublicIPs(resources, machines)
	enrichVSphereHostRegions(allResources, machines)

	sort.SliceStable(machines, func(i, j int) bool {
		left := strings.ToLower(machines[i].Provider + machines[i].Name + machines[i].ResourceAddress)
		right := strings.ToLower(machines[j].Provider + machines[j].Name + machines[j].ResourceAddress)
		return left < right
	})

	return ParseResult{
		Terraform:    state.Terraform,
		RawResources: len(resources),
		Machines:     machines,
	}, nil
}

func collectResources(state stateFile) []resourceEntry {
	return collectResourcesByMode(state, false)
}

func collectAllResources(state stateFile) []resourceEntry {
	return collectResourcesByMode(state, true)
}

func collectResourcesByMode(state stateFile, includeData bool) []resourceEntry {
	var resources []resourceEntry
	for _, resource := range state.Resources {
		if !isSupportedMode(resource.Mode, includeData) {
			continue
		}
		for _, instance := range resource.Instances {
			if len(instance.Attributes) == 0 {
				continue
			}
			resources = append(resources, resourceEntry{
				Address:    resourceAddress(resource.Type, resource.Name, instance.IndexKey),
				Provider:   resource.Provider,
				Type:       resource.Type,
				Name:       resource.Name,
				IndexKey:   instance.IndexKey,
				Attributes: instance.Attributes,
			})
		}
	}

	if state.Values != nil {
		collectValueResources(state.Values.RootModule, &resources, includeData)
	}
	return resources
}

func collectValueResources(module valueModule, resources *[]resourceEntry, includeData bool) {
	for _, resource := range module.Resources {
		if !isSupportedMode(resource.Mode, includeData) || len(resource.Values) == 0 {
			continue
		}
		address := resource.Address
		if address == "" {
			address = resourceAddress(resource.Type, resource.Name, nil)
		}
		*resources = append(*resources, resourceEntry{
			Address:    address,
			Type:       resource.Type,
			Name:       resource.Name,
			Attributes: resource.Values,
		})
	}

	for _, child := range module.ChildModules {
		collectValueResources(child, resources, includeData)
	}
}

func buildMachines(resources []resourceEntry) []inventory.Machine {
	var machines []inventory.Machine
	for _, resource := range resources {
		if !isMachineResource(resource.Type) {
			continue
		}
		machines = append(machines, buildMachine(resource))
	}
	return machines
}

func resourceAddress(resourceType, resourceName string, indexKey any) string {
	address := resourceType + "." + resourceName
	if indexKey != nil {
		address = fmt.Sprintf("%s[%q]", address, fmt.Sprint(indexKey))
	}
	return address
}

func buildMachine(resource resourceEntry) inventory.Machine {
	attrs := resource.Attributes
	privateIPs := collectStringsByKeys(attrs, "private_ip", "private_ip_address", "private_ips", "private_ip_addresses")
	if resource.Type == "vsphere_virtual_machine" {
		privateIPs = append(privateIPs, vspherePrivateIPs(attrs)...)
	}

	machine := inventory.Machine{
		ID:              firstString(attrs, "id", "instance_id", "vm_id", "server_id", "urn", "arn"),
		Name:            firstString(attrs, "name", "instance_name", "computer_name", "hostname", "display_name", "vm_name"),
		Provider:        normalizeProvider(resource.Provider, resource.Type),
		ResourceType:    resource.Type,
		ResourceName:    resource.Name,
		ResourceAddress: resource.Address,
		Region:          firstString(attrs, "region", "location"),
		Zone:            firstString(attrs, "availability_zone", "zone", "placement_availability_zone"),
		Status:          firstString(attrs, "instance_state", "status", "power_state", "vm_state"),
		InstanceType:    firstString(attrs, "instance_type", "instance_class", "machine_type", "flavor_name", "size", "vm_size"),
		CPUCores:        firstString(attrs, "cpu", "cpus", "num_cpu", "num_cpus", "num_cores", "cpu_core_count", "core_count", "cores", "vcpu", "vcpus", "vcpu_count"),
		Memory:          firstMemory(attrs),
		Disks:           collectDisks(attrs),
		PrivateIPs:      uniqueStrings(privateIPs),
		PublicIPs:       uniqueStrings(collectStringsByKeys(attrs, "public_ip", "public_ip_address", "public_ips", "public_ip_addresses", "ipv4_address", "access_ip_v4")),
		Tags:            firstMap(attrs, "tags", "labels", "metadata"),
		Attributes:      maps.Clone(attrs),
	}

	if machine.Name == "" && machine.Tags != nil {
		machine.Name = anyString(machine.Tags["Name"])
		if machine.Name == "" {
			machine.Name = anyString(machine.Tags["name"])
		}
	}
	if machine.Name == "" {
		machine.Name = machine.ResourceName
	}

	machine.PrivateIPs = filterIPs(machine.PrivateIPs, true)
	machine.PublicIPs = filterIPs(machine.PublicIPs, false)
	return machine
}

func vspherePrivateIPs(attrs map[string]any) []string {
	privateIPs := collectStringsByKeys(attrs, "default_ip_address", "guest_ip_addresses")

	for _, config := range vsphereGuestInfoConfigs(attrs) {
		for _, key := range []string{"guestinfo.metadata", "guestinfo.userdata", "guestinfo.vendordata", "metadata", "userdata", "user_data"} {
			payload := firstString(config, key)
			if payload == "" {
				continue
			}

			encoding := firstString(config, key+".encoding")
			decoded := decodeVSphereGuestInfo(payload, encoding)
			privateIPs = append(privateIPs, privateIPsFromCloudInit(decoded)...)
		}
	}

	return uniqueStrings(privateIPs)
}

func enrichVSphereHostRegions(resources []resourceEntry, machines []inventory.Machine) {
	hostNamesByID := vsphereHostNamesByID(resources)
	for idx := range machines {
		if machines[idx].ResourceType != "vsphere_virtual_machine" {
			continue
		}
		host := vsphereMachineHost(machines[idx].Attributes, hostNamesByID)
		if host != "" {
			machines[idx].Region = host
		}
	}
}

func vsphereHostNamesByID(resources []resourceEntry) map[string]string {
	hosts := map[string]string{}
	for _, resource := range resources {
		if resource.Type != "vsphere_host" {
			continue
		}

		name := firstString(resource.Attributes, "name", "host_name")
		if name == "" {
			continue
		}
		for _, id := range collectStringsByKeys(resource.Attributes, "id", "moid", "host_system_id") {
			id = strings.TrimSpace(id)
			if id != "" {
				hosts[id] = name
			}
		}
	}
	return hosts
}

func vsphereMachineHost(attrs map[string]any, hostNamesByID map[string]string) string {
	hostID := firstString(attrs, "host_system_id", "host_id", "esxi_host_id")
	if hostID != "" {
		if name := hostNamesByID[hostID]; name != "" {
			return name
		}
	}

	return firstNonEmpty(
		firstString(attrs, "host_system_name", "host_name", "esxi_host", "esxi_host_name"),
		hostID,
	)
}

func vsphereGuestInfoConfigs(attrs map[string]any) []map[string]any {
	configs := []map[string]any{attrs}
	for _, key := range []string{"extra_config", "vapp"} {
		for _, config := range mapsFromAny(attrs[key]) {
			configs = append(configs, config)
			for _, properties := range mapsFromAny(config["properties"]) {
				configs = append(configs, properties)
			}
		}
	}
	return configs
}

func decodeVSphereGuestInfo(value, encoding string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	normalizedEncoding := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(encoding), " ", ""))
	if strings.Contains(normalizedEncoding, "base64") {
		decoded, err := decodeBase64Text(value)
		if err != nil {
			return value
		}
		if strings.Contains(normalizedEncoding, "gzip") {
			if unzipped, err := gunzip(decoded); err == nil {
				decoded = unzipped
			}
		}
		return string(decoded)
	}

	if decoded, err := decodeBase64Text(value); err == nil && looksLikeCloudInit(string(decoded)) {
		return string(decoded)
	}
	return value
}

func decodeBase64Text(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("base64 decode failed")
}

func gunzip(value []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(value))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func privateIPsFromCloudInit(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	decoder := yaml.NewDecoder(strings.NewReader(content))
	var privateIPs []string
	for {
		var document any
		if err := decoder.Decode(&document); err != nil {
			if err == io.EOF {
				break
			}
			return nil
		}
		privateIPs = append(privateIPs, privateIPsFromCloudInitValue(document, nil, 0)...)
	}
	return filterIPs(privateIPs, true)
}

func privateIPsFromCloudInitValue(value any, path []string, depth int) []string {
	if depth > 4 {
		return nil
	}

	var privateIPs []string
	switch typed := value.(type) {
	case map[string]any:
		for key, childValue := range typed {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			childPath := append(path, lowerKey)
			if isCloudInitIPKey(lowerKey, childPath) {
				privateIPs = append(privateIPs, flattenStrings(childValue)...)
			}
			privateIPs = append(privateIPs, privateIPsFromCloudInitValue(childValue, childPath, depth+1)...)
		}
	case map[any]any:
		converted := make(map[string]any, len(typed))
		for key, childValue := range typed {
			converted[fmt.Sprint(key)] = childValue
		}
		privateIPs = append(privateIPs, privateIPsFromCloudInitValue(converted, path, depth+1)...)
	case []any:
		for _, childValue := range typed {
			privateIPs = append(privateIPs, privateIPsFromCloudInitValue(childValue, path, depth+1)...)
		}
	case string:
		if looksLikeCloudInit(typed) {
			privateIPs = append(privateIPs, privateIPsFromCloudInit(typed)...)
		}
	}
	return privateIPs
}

func isCloudInitIPKey(key string, path []string) bool {
	if pathContainsAny(path, "nameserver", "nameservers", "gateway", "gateway4", "gateway6", "route", "routes") {
		return false
	}
	if key == "addresses" {
		return true
	}
	if key == "address" || key == "ip" || key == "ip_address" || key == "ipv4_address" {
		return pathContainsAny(path, "network", "ethernets", "interfaces", "subnets", "network_config")
	}
	return false
}

func pathContainsAny(path []string, needles ...string) bool {
	for _, part := range path {
		for _, needle := range needles {
			if strings.Contains(part, needle) {
				return true
			}
		}
	}
	return false
}

func looksLikeCloudInit(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.Contains(value, "\n") {
		return false
	}
	lower := strings.ToLower(value)
	return strings.Contains(lower, "network:") ||
		strings.Contains(lower, "addresses:") ||
		strings.Contains(lower, "address:") ||
		strings.Contains(lower, "#cloud-config")
}

func linkPublicIPs(resources []resourceEntry, machines []inventory.Machine) {
	machineByID := map[string][]int{}
	for idx, machine := range machines {
		for _, id := range machineIdentityCandidates(machine) {
			machineByID[id] = append(machineByID[id], idx)
		}
	}

	publicIPsByID := map[string][]string{}
	type publicIPLink struct {
		machineIDs []string
		publicIPs  []string
		publicIDs  []string
	}
	var links []publicIPLink

	for _, resource := range resources {
		if !isPublicIPResource(resource.Type) && !isPublicIPAssociationResource(resource.Type) {
			continue
		}

		publicIPs := publicIPsFromAttributes(resource.Attributes)
		publicIDs := publicIPIdentityCandidates(resource.Attributes)
		machineIDs := publicIPMachineCandidates(resource.Attributes)
		for _, id := range publicIDs {
			publicIPsByID[id] = appendUnique(publicIPsByID[id], publicIPs...)
		}
		if len(machineIDs) > 0 {
			links = append(links, publicIPLink{
				machineIDs: machineIDs,
				publicIPs:  publicIPs,
				publicIDs:  publicIDs,
			})
		}
	}

	for _, link := range links {
		publicIPs := append([]string(nil), link.publicIPs...)
		for _, id := range link.publicIDs {
			publicIPs = appendUnique(publicIPs, publicIPsByID[id]...)
		}
		if len(publicIPs) == 0 {
			continue
		}

		for _, machineID := range link.machineIDs {
			for _, machineIdx := range machineByID[machineID] {
				machines[machineIdx].PublicIPs = appendUnique(machines[machineIdx].PublicIPs, publicIPs...)
			}
		}
	}
}

func machineIdentityCandidates(machine inventory.Machine) []string {
	candidates := []string{machine.ID, machine.ResourceAddress}
	candidates = append(candidates, collectStringsByKeys(machine.Attributes,
		"id",
		"instance_id",
		"vm_id",
		"server_id",
		"arn",
		"urn",
		"self_link",
		"network_interface_id",
		"primary_network_interface_id",
	)...)
	return uniqueStrings(candidates)
}

func publicIPIdentityCandidates(attrs map[string]any) []string {
	return uniqueStrings(collectStringsByKeys(attrs,
		"id",
		"allocation_id",
		"eip_id",
		"address_id",
		"public_ip_id",
		"floating_ip_id",
	))
}

func publicIPMachineCandidates(attrs map[string]any) []string {
	return uniqueStrings(collectStringsByKeys(attrs,
		"instance",
		"instance_id",
		"server_id",
		"vm_id",
		"resource_id",
		"associated_instance_id",
		"bound_instance_id",
		"network_interface_id",
		"primary_network_interface_id",
	))
}

func publicIPsFromAttributes(attrs map[string]any) []string {
	return filterIPs(collectStringsByKeys(attrs,
		"public_ip",
		"public_ip_address",
		"public_ips",
		"public_ip_addresses",
		"ip_address",
		"ipv4_address",
		"floating_ip_address",
		"eip_address",
		"internet_ip",
		"address",
	), false)
}

func isPublicIPResource(resourceType string) bool {
	lower := strings.ToLower(resourceType)
	return strings.Contains(lower, "_eip") ||
		strings.Contains(lower, "eip_") ||
		strings.Contains(lower, "elastic_ip") ||
		strings.Contains(lower, "public_ip") ||
		strings.Contains(lower, "floatingip") ||
		strings.Contains(lower, "floating_ip")
}

func isPublicIPAssociationResource(resourceType string) bool {
	lower := strings.ToLower(resourceType)
	return strings.Contains(lower, "eip_association") ||
		strings.Contains(lower, "public_ip_association") ||
		strings.Contains(lower, "floatingip_associate") ||
		strings.Contains(lower, "floating_ip_associate")
}

func isManaged(mode string) bool {
	return mode == "" || mode == "managed"
}

func isSupportedMode(mode string, includeData bool) bool {
	if isManaged(mode) {
		return true
	}
	return includeData && mode == "data"
}

func isMachineResource(resourceType string) bool {
	known := map[string]bool{
		"aws_instance":                       true,
		"alicloud_instance":                  true,
		"tencentcloud_instance":              true,
		"azurerm_linux_virtual_machine":      true,
		"azurerm_windows_virtual_machine":    true,
		"azurerm_virtual_machine":            true,
		"google_compute_instance":            true,
		"huaweicloud_compute_instance":       true,
		"volcengine_ecs_instance":            true,
		"ucloud_instance":                    true,
		"baiducloud_instance":                true,
		"openstack_compute_instance_v2":      true,
		"vsphere_virtual_machine":            true,
		"linode_instance":                    true,
		"digitalocean_droplet":               true,
		"hcloud_server":                      true,
		"proxmox_vm_qemu":                    true,
		"proxmox_virtual_environment_vm":     true,
		"cloudstack_instance":                true,
		"flexibleengine_compute_instance_v2": true,
	}
	if known[resourceType] {
		return true
	}

	lower := strings.ToLower(resourceType)
	return strings.Contains(lower, "_instance") ||
		strings.Contains(lower, "_virtual_machine") ||
		strings.Contains(lower, "_server") ||
		strings.Contains(lower, "_droplet")
}

func normalizeProvider(provider, resourceType string) string {
	provider = strings.TrimPrefix(provider, "provider[\"")
	if idx := strings.Index(provider, "\"]"); idx >= 0 {
		provider = provider[:idx]
	}
	if idx := strings.LastIndex(provider, "/"); idx >= 0 {
		provider = provider[idx+1:]
	}
	provider = strings.TrimSpace(provider)
	if provider != "" {
		return provider
	}

	prefixes := []string{"aws", "alicloud", "tencentcloud", "azurerm", "google", "huaweicloud", "volcengine", "ucloud", "baiducloud", "openstack", "vsphere", "linode", "digitalocean", "hcloud", "proxmox", "cloudstack"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(resourceType, prefix+"_") {
			return prefix
		}
	}
	return "unknown"
}

func firstString(attrs map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := anyString(attrs[key]); value != "" {
			return value
		}
	}
	return ""
}

func firstMap(attrs map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		switch typed := attrs[key].(type) {
		case map[string]any:
			return typed
		case map[string]string:
			result := make(map[string]any, len(typed))
			for k, v := range typed {
				result[k] = v
			}
			return result
		}
	}
	return nil
}

func formatMemory(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "gb") || strings.Contains(lower, "gib") || strings.HasSuffix(lower, "g") {
		return formatNumber(trimMemoryUnit(value))
	}
	if strings.Contains(lower, "mb") || strings.Contains(lower, "mib") || strings.HasSuffix(lower, "m") {
		if mb, err := strconv.ParseFloat(trimMemoryUnit(value), 64); err == nil {
			return formatGigabytes(mb / 1024)
		}
		return value
	}
	if mb, err := strconv.ParseFloat(value, 64); err == nil {
		return formatGigabytes(mb / 1024)
	}
	return value
}

func formatGigabytes(gb float64) string {
	if gb <= 0 {
		return ""
	}
	return formatNumber(fmt.Sprintf("%.1f", gb))
}

func formatNumber(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", number), "0"), ".")
}

func trimMemoryUnit(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, "gib")
	value = strings.TrimSuffix(value, "gb")
	value = strings.TrimSuffix(value, "g")
	value = strings.TrimSuffix(value, "mib")
	value = strings.TrimSuffix(value, "mb")
	value = strings.TrimSuffix(value, "m")
	return strings.TrimSpace(value)
}

func firstMemory(attrs map[string]any) string {
	for _, key := range []string{"memory", "memory_size", "memory_mb", "memory_gb", "ram", "ram_size"} {
		value := firstString(attrs, key)
		if value == "" {
			continue
		}
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "_gb") {
			return formatNumber(value)
		}
		if strings.Contains(lowerKey, "_mb") {
			if mb, err := strconv.ParseFloat(value, 64); err == nil {
				return formatGigabytes(mb / 1024)
			}
			return value
		}
		return formatMemory(value)
	}
	return ""
}

func collectDisks(attrs map[string]any) []inventory.Disk {
	var disks []inventory.Disk
	addDisk := func(name, diskType, size string) {
		size = strings.TrimSpace(size)
		if size == "" || size == "0" {
			return
		}
		disks = append(disks, inventory.Disk{
			Name:   name,
			Type:   diskType,
			SizeGB: normalizeDiskSize(size),
		})
	}

	addDisk("system", firstString(attrs, "system_disk_category", "root_disk_type"), firstString(attrs, "system_disk_size", "root_disk_size"))
	addDisk("boot", firstString(attrs, "boot_disk_type"), firstString(attrs, "boot_disk_size"))
	addDisk("disk", firstString(attrs, "disk_type", "volume_type"), firstString(attrs, "disk_size", "volume_size", "storage_size"))

	for _, key := range []string{"root_block_device", "boot_disk", "system_disk", "os_disk"} {
		for _, diskAttrs := range mapsFromAny(attrs[key]) {
			addDisk(firstNonEmpty(firstString(diskAttrs, "name"), key), firstString(diskAttrs, "type", "disk_type", "volume_type", "category"), firstString(diskAttrs, "size", "disk_size", "volume_size", "disk_size_gb"))
		}
	}

	for _, key := range []string{"data_disks", "data_disk", "ebs_block_device", "attached_disk", "disk", "disks"} {
		for idx, diskAttrs := range mapsFromAny(attrs[key]) {
			name := firstString(diskAttrs, "name", "device_name", "disk_name")
			if name == "" {
				name = fmt.Sprintf("data-%d", idx+1)
			}
			addDisk(name, firstString(diskAttrs, "type", "disk_type", "volume_type", "category"), firstString(diskAttrs, "size", "disk_size", "volume_size", "disk_size_gb"))
		}
	}

	return dedupeDisks(disks)
}

func mapsFromAny(value any) []map[string]any {
	switch typed := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return []map[string]any{typed}
	case []any:
		var out []map[string]any
		for _, item := range typed {
			out = append(out, mapsFromAny(item)...)
		}
		return out
	default:
		return nil
	}
}

func normalizeDiskSize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "gb") || strings.Contains(lower, "gib") || strings.Contains(lower, "tb") || strings.Contains(lower, "tib") {
		return value
	}
	return value + " GB"
}

func dedupeDisks(disks []inventory.Disk) []inventory.Disk {
	seen := map[string]bool{}
	var out []inventory.Disk
	for _, disk := range disks {
		key := disk.Name + "|" + disk.Type + "|" + disk.SizeGB
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, disk)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func collectStringsByKeys(value any, keys ...string) []string {
	keySet := make(map[string]bool, len(keys))
	for _, key := range keys {
		keySet[key] = true
	}

	var out []string
	var walk func(any, string)
	walk = func(current any, key string) {
		if keySet[key] {
			out = append(out, flattenStrings(current)...)
		}

		switch typed := current.(type) {
		case map[string]any:
			for childKey, childValue := range typed {
				walk(childValue, childKey)
			}
		case []any:
			for _, childValue := range typed {
				walk(childValue, "")
			}
		}
	}
	walk(value, "")
	return out
}

func flattenStrings(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return []string{typed}
	case []string:
		return typed
	case []any:
		var out []string
		for _, item := range typed {
			out = append(out, flattenStrings(item)...)
		}
		return out
	case map[string]any:
		var out []string
		for _, item := range typed {
			out = append(out, flattenStrings(item)...)
		}
		return out
	default:
		if text := anyString(typed); text != "" {
			return []string{text}
		}
	}
	return nil
}

func anyString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	case bool:
		return fmt.Sprint(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func filterIPs(values []string, private bool) []string {
	var out []string
	for _, value := range values {
		ip := normalizeIP(value)
		if ip == "" {
			continue
		}
		parsed := net.ParseIP(ip)
		if private && parsed.IsPrivate() {
			out = append(out, ip)
		}
		if !private && !parsed.IsPrivate() {
			out = append(out, ip)
		}
	}
	return uniqueStrings(out)
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	if ip, _, err := net.ParseCIDR(value); err == nil {
		return ip.String()
	}
	return ""
}

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
