package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"cli/pkg/config"
)

type Client struct {
	baseURL    string
	token      string
	apiKey     string
	httpClient *http.Client
}

func NewFromConfig(cfg *config.Config) (*Client, error) {
	if !cfg.IsLoggedIn() {
		return nil, fmt.Errorf("not logged in — run `kestrel login` or `kestrel auth` first")
	}
	return &Client{
		baseURL: cfg.ServerURL,
		token:   cfg.SessionToken,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

func NewUnauthenticated(serverURL string) *Client {
	return &Client{
		baseURL: serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login authenticates with email/password and returns the session token.
func (c *Client) Login(email, password string) (*LoginResponse, error) {
	body := map[string]string{"email": email, "password": password}
	var resp LoginResponse
	if err := c.post("/api/login", body, &resp); err != nil {
		return nil, err
	}
	c.token = resp.SessionToken
	return &resp, nil
}

// ValidateSession checks if the current session is still valid.
func (c *Client) ValidateSession() error {
	return c.get("/api/validate-session", nil, nil)
}

// --- Workflow CRUD ---

func (c *Client) ListWorkflows(status string) ([]Workflow, error) {
	path := "/api/workflows"
	if status != "" {
		path += "?status=" + url.QueryEscape(status)
	}
	var out []Workflow
	if err := c.get(path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetWorkflow(id string) (*Workflow, error) {
	var out Workflow
	if err := c.get("/api/workflows/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateWorkflow(req CreateWorkflowRequest) (*Workflow, error) {
	var out Workflow
	if err := c.post("/api/workflows", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateWorkflow(id string, req UpdateWorkflowRequest) (*Workflow, error) {
	var out Workflow
	if err := c.put("/api/workflows/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteWorkflow(id string) error {
	return c.delete("/api/workflows/" + id)
}

func (c *Client) ActivateWorkflow(id string) error {
	return c.post("/api/workflows/"+id+"/activate", nil, nil)
}

func (c *Client) PauseWorkflow(id string) error {
	return c.post("/api/workflows/"+id+"/pause", nil, nil)
}

func (c *Client) DuplicateWorkflow(id, newName string) (*Workflow, error) {
	body := map[string]string{"name": newName}
	var out Workflow
	if err := c.post("/api/workflows/"+id+"/duplicate", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) TestWorkflow(id string) (*WorkflowExecution, error) {
	var out WorkflowExecution
	if err := c.post("/api/workflows/"+id+"/test", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Generation ---

func (c *Client) GenerateWorkflow(prompt string) (*GenerateWorkflowResponse, error) {
	body := map[string]string{"prompt": prompt}
	var out GenerateWorkflowResponse
	if err := c.post("/api/workflows/generate", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Stats ---

func (c *Client) GetWorkflowStats() (*WorkflowStats, error) {
	var out WorkflowStats
	if err := c.get("/api/workflows/stats", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Executions ---

func (c *Client) ListExecutions(workflowID string, page, pageSize int) (*ExecutionListResponse, error) {
	path := fmt.Sprintf("/api/workflows/%s/executions?page=%d&page_size=%d", workflowID, page, pageSize)
	var out ExecutionListResponse
	if err := c.get(path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetExecution(id string) (*WorkflowExecution, error) {
	var out WorkflowExecution
	if err := c.get("/api/workflow-executions/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CancelExecution(id string) error {
	return c.post("/api/workflow-executions/"+id+"/cancel", nil, nil)
}

// --- Approvals ---

func (c *Client) ListPendingApprovals() ([]WorkflowApproval, error) {
	var out []WorkflowApproval
	if err := c.get("/api/workflow-approvals/pending", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ApproveStep(id string) error {
	return c.post("/api/workflow-approvals/"+id+"/approve", nil, nil)
}

func (c *Client) ApproveStepWithJustification(id, justification string) error {
	body := map[string]string{"justification": justification}
	return c.post("/api/workflow-approvals/"+id+"/approve", body, nil)
}

func (c *Client) RejectStep(id string) error {
	return c.post("/api/workflow-approvals/"+id+"/reject", nil, nil)
}

// --- Requests ---

func (c *Client) ListWorkflowRequests() ([]WorkflowRequest, error) {
	var out struct {
		Requests []WorkflowRequest `json:"requests"`
		Total    int               `json:"total"`
	}
	if err := c.get("/api/workflow-requests", nil, &out); err != nil {
		return nil, err
	}
	return out.Requests, nil
}

func (c *Client) WorkflowRequestAction(id, action string) error {
	return c.post("/api/workflow-requests/"+id+"/"+action, nil, nil)
}

func (c *Client) TriggerWorkflowRequest(prompt, requesterName string) (map[string]interface{}, error) {
	body := map[string]string{
		"prompt":         prompt,
		"source":         "cli",
		"requester_name": requesterName,
	}
	var out map[string]interface{}
	if err := c.post("/api/workflow-requests", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Suggestions ---

func (c *Client) ListSuggestions() ([]SuggestedWorkflow, error) {
	var out []SuggestedWorkflow
	if err := c.get("/api/workflows/suggestions", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Catalog ---

func (c *Client) GetCatalog() (*Catalog, error) {
	var out Catalog
	if err := c.get("/api/workflows/catalog", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Integrations ---

func (c *Client) GetIntegrationsStatus() ([]IntegrationStatus, error) {
	var out []IntegrationStatus
	if err := c.get("/api/workflows/integrations/status", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- HTTP helpers ---

func (c *Client) get(path string, _ interface{}, out interface{}) error {
	return c.do("GET", path, nil, out)
}

func (c *Client) post(path string, body interface{}, out interface{}) error {
	return c.do("POST", path, body, out)
}

func (c *Client) put(path string, body interface{}, out interface{}) error {
	return c.do("PUT", path, body, out)
}

func (c *Client) delete(path string) error {
	return c.do("DELETE", path, nil, nil)
}

func (c *Client) do(method, path string, body interface{}, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else if c.token != "" {
		req.Header.Set("X-Session-Token", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := string(respBody)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// API Key management

func (c *Client) ListAPIKeys() ([]APIKey, error) {
	var keys []APIKey
	return keys, c.do("GET", "/api/api-keys", nil, &keys)
}

func (c *Client) CreateAPIKey(req CreateAPIKeyRequest) (*APIKeyWithRaw, error) {
	var key APIKeyWithRaw
	return &key, c.do("POST", "/api/api-keys", req, &key)
}

func (c *Client) RevokeAPIKey(id string) error {
	return c.do("POST", "/api/api-keys/"+id+"/revoke", nil, nil)
}

func (c *Client) DeleteAPIKey(id string) error {
	return c.do("DELETE", "/api/api-keys/"+id, nil, nil)
}
