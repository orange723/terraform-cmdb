package terraformstate

import "testing"

func TestParseLinksPublicIPAssociationToMachine(t *testing.T) {
	state := []byte(`{
  "version": 4,
  "terraform_version": "1.9.0",
  "resources": [
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "attributes": {
            "id": "i-123",
            "instance_type": "t3.micro",
            "cpu_core_count": 2,
            "memory": 4096,
            "private_ip": "10.0.1.10",
            "root_block_device": [
              {
                "volume_size": 40,
                "volume_type": "gp3"
              }
            ],
            "tags": {"Name": "web-1"}
          }
        }
      ]
    },
    {
      "mode": "managed",
      "type": "aws_eip",
      "name": "web",
      "instances": [
        {
          "attributes": {
            "id": "eipalloc-123",
            "allocation_id": "eipalloc-123",
            "public_ip": "8.8.8.8"
          }
        }
      ]
    },
    {
      "mode": "managed",
      "type": "aws_eip_association",
      "name": "web",
      "instances": [
        {
          "attributes": {
            "allocation_id": "eipalloc-123",
            "instance_id": "i-123"
          }
        }
      ]
    }
  ]
}`)

	result, err := Parse(state)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(result.Machines) != 1 {
		t.Fatalf("machines length = %d, want 1", len(result.Machines))
	}

	machine := result.Machines[0]
	if machine.Name != "web-1" {
		t.Fatalf("machine name = %q, want web-1", machine.Name)
	}
	if len(machine.PublicIPs) != 1 || machine.PublicIPs[0] != "8.8.8.8" {
		t.Fatalf("public IPs = %#v, want [8.8.8.8]", machine.PublicIPs)
	}
	if machine.CPUCores != "2" {
		t.Fatalf("cpu cores = %q, want 2", machine.CPUCores)
	}
	if machine.Memory != "4" {
		t.Fatalf("memory = %q, want 4", machine.Memory)
	}
	if len(machine.Disks) != 1 || machine.Disks[0].SizeGB != "40 GB" || machine.Disks[0].Type != "gp3" {
		t.Fatalf("disks = %#v, want root 40 GB gp3", machine.Disks)
	}
}
