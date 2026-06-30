package server

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"terraform-cmdb/internal/inventory"
)

func TestHandleUploadKeepsSnapshotOnParseError(t *testing.T) {
	store := inventory.NewStore()
	store.Replace(inventory.Snapshot{
		FileName: "states (1 files)",
		Machines: []inventory.Machine{
			{ID: "i-existing", Name: "existing-host"},
		},
	})

	server := New(store, Config{StateDir: t.TempDir()})
	app := server.App()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("state", "broken.tfstate")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.WriteString(part, `{`); err != nil {
		t.Fatalf("write form body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	snapshot := store.Snapshot()
	if len(snapshot.Machines) != 1 {
		t.Fatalf("expected existing snapshot to be preserved, got %d machines", len(snapshot.Machines))
	}
	if snapshot.Machines[0].ID != "i-existing" {
		t.Fatalf("expected existing machine id i-existing, got %q", snapshot.Machines[0].ID)
	}
}
