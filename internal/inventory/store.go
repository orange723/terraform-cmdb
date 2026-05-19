package inventory

import "sync"

// Machine is the normalized instance shape used by the UI and API.
type Machine struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Provider        string         `json:"provider"`
	ResourceType    string         `json:"resource_type"`
	ResourceName    string         `json:"resource_name"`
	ResourceAddress string         `json:"resource_address"`
	Region          string         `json:"region"`
	Zone            string         `json:"zone"`
	Status          string         `json:"status"`
	InstanceType    string         `json:"instance_type"`
	CPUCores        string         `json:"cpu_cores"`
	Memory          string         `json:"memory"`
	Disks           []Disk         `json:"disks,omitempty"`
	PrivateIPs      []string       `json:"private_ips"`
	PublicIPs       []string       `json:"public_ips"`
	Tags            map[string]any `json:"tags,omitempty"`
	Attributes      map[string]any `json:"attributes"`
}

type Disk struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	SizeGB string `json:"size_gb"`
}

type Snapshot struct {
	FileName     string    `json:"file_name"`
	SourceFiles  []string  `json:"source_files,omitempty"`
	Terraform    string    `json:"terraform"`
	RawResources int       `json:"raw_resources"`
	Machines     []Machine `json:"instances"`
	LastError    string    `json:"last_error,omitempty"`
}

type Store struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Replace(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot = snapshot
	s.snapshot.Machines = append([]Machine(nil), snapshot.Machines...)
	s.snapshot.SourceFiles = append([]string(nil), snapshot.SourceFiles...)
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := s.snapshot
	snapshot.Machines = append([]Machine(nil), s.snapshot.Machines...)
	snapshot.SourceFiles = append([]string(nil), s.snapshot.SourceFiles...)
	return snapshot
}
