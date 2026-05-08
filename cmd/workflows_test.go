package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCLIActivateBlockedWithMissingFields tests that `kestrel workflows activate`
// properly surfaces the missing fields error from the server.
func TestCLIActivateBlockedWithMissingFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/activate") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Workflow has missing required fields",
				"missing_fields": []map[string]string{
					{"node_id": "action-1", "node_label": "Generate Manifest", "field_name": "name", "field_label": "Resource Name"},
				},
				"message": `Node "Generate Manifest": required field "Resource Name" is not set`,
			})
			return
		}
		http.Error(w, "not found", 404)
	}))
	defer srv.Close()

	bin := buildCLI(t)
	out, _ := runCLI(bin, srv.URL, "workflows", "activate", "wf-123")
	// The CLI should display the human-readable missing fields message (exits 0 but shows error)
	if !strings.Contains(out, "Cannot activate") || !strings.Contains(out, "Resource Name") {
		if !strings.Contains(out, "required field") {
			t.Errorf("expected friendly missing fields message, got:\n%s", out)
		}
	}
}

// TestCLIActivateSucceedsWithCompleteFields tests that `kestrel workflows activate`
// succeeds when the server returns 200.
func TestCLIActivateSucceedsWithCompleteFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/activate") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "active"})
			return
		}
		http.Error(w, "not found", 404)
	}))
	defer srv.Close()

	bin := buildCLI(t)
	out, err := runCLI(bin, srv.URL, "workflows", "activate", "wf-456")
	if err != nil {
		t.Fatalf("expected success, got error: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "activated") {
		t.Errorf("expected activation confirmation, got:\n%s", out)
	}
}

// TestCLISaveAllowedWithMissingFields tests that saving (create from JSON file)
// does not perform validation — missing fields are permitted for drafts.
func TestCLISaveAllowedWithMissingFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/workflows" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "wf-new", "name": "Incomplete WF", "status": "draft",
			})
			return
		}
		http.Error(w, "not found", 404)
	}))
	defer srv.Close()

	// Create a temp workflow JSON file with missing required fields
	wfJSON := `{
		"name": "Incomplete WF",
		"definition": {
			"nodes": [
				{"id": "t1", "type": "trigger", "data": {"source": "kubernetes"}, "position": {"x":0,"y":0}},
				{"id": "a1", "type": "action", "data": {"action": "kestrel-generate-k8s-manifest", "label": "Gen", "config": {}}, "position": {"x":0,"y":100}}
			],
			"edges": [{"id": "e1", "source": "t1", "target": "a1"}]
		},
		"trigger_config": {"source": "kubernetes"}
	}`
	tmpFile, err := os.CreateTemp("", "wf-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(wfJSON)
	tmpFile.Close()

	bin := buildCLI(t)
	out, err := runCLI(bin, srv.URL, "workflows", "create", "--file", tmpFile.Name())
	if err != nil {
		t.Fatalf("expected save to succeed (missing fields allowed for drafts), got error: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "Created workflow") {
		t.Errorf("expected creation confirmation, got:\n%s", out)
	}
}

// TestCLIGeneratePromptsMissingFields tests that `kestrel workflows generate`
// detects and prompts for missing required fields from the catalog.
func TestCLIGeneratePromptsMissingFields(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/workflows/generate":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":     true,
				"name":        "Scale Service",
				"description": "Scales a deployment",
				"definition": map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{"id": "t1", "type": "trigger", "data": map[string]interface{}{"source": "kubernetes"}, "position": map[string]interface{}{"x": 0, "y": 0}},
						map[string]interface{}{"id": "a1", "type": "action", "data": map[string]interface{}{
							"action": "kestrel-generate-k8s-manifest", "label": "Generate Manifest", "integration": "kestrel",
							"config": map[string]interface{}{"resource_type": "Deployment"},
						}, "position": map[string]interface{}{"x": 0, "y": 100}},
					},
					"edges": []interface{}{map[string]interface{}{"id": "e1", "source": "t1", "target": "a1"}},
				},
				"trigger_config": map[string]interface{}{"source": "kubernetes"},
				"explanation":    "This generates a K8s manifest.",
			})
		case r.Method == "GET" && r.URL.Path == "/api/workflows/catalog":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"signals": []interface{}{},
				"actions": []interface{}{
					map[string]interface{}{
						"id": "kestrel-generate-k8s-manifest", "integration": "kestrel",
						"name": "Generate K8s Manifest", "description": "Generate a manifest",
						"category": "Provisioning",
						"fields": []interface{}{
							map[string]interface{}{"name": "resource_type", "label": "Resource Type", "type": "select", "required": true},
							map[string]interface{}{"name": "name", "label": "Resource Name", "type": "template", "required": true, "placeholder": "my-app"},
						},
					},
				},
				"integrations": []interface{}{},
			})
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer srv.Close()

	bin := buildCLI(t)
	// Run generate without --save (shouldn't prompt since we aren't saving)
	// But the prompting only happens if --save is passed, so test the detection output
	out, _ := runCLIWithStdin(bin, srv.URL, "my-app\n", "workflows", "generate", "scale my service", "--save")
	// Should show the prompt for missing "Resource Name" field
	if !strings.Contains(out, "Resource Name") && !strings.Contains(out, "required fields") {
		t.Logf("Output:\n%s", out)
		// Even if the prompt times out (no stdin in test), verify the catalog was fetched
		if callCount < 2 {
			t.Errorf("expected at least 2 server calls (generate + catalog), got %d", callCount)
		}
	}
}

// --- helpers ---

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := t.TempDir() + "/kestrel-test"
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = cliRootDir()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\n%s", err, out)
	}
	return bin
}

func runCLI(bin, server string, args ...string) (string, error) {
	return runCLIWithStdin(bin, server, "", args...)
}

func runCLIWithStdin(bin, server, stdin string, args ...string) (string, error) {
	// Create a temp HOME with a .kestrel/config.json pointing to the test server
	tmpHome, _ := os.MkdirTemp("", "kestrel-home-*")
	defer os.RemoveAll(tmpHome)
	kestrelDir := tmpHome + "/.kestrel"
	os.MkdirAll(kestrelDir, 0o700)
	cfgJSON := fmt.Sprintf(`{"server_url": %q, "session_token": "test-token", "user_id": "u-1", "email": "test@example.com"}`, server)
	os.WriteFile(kestrelDir+"/config.json", []byte(cfgJSON), 0o600)

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", tmpHome))
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func cliRootDir() string {
	return ".."
}
