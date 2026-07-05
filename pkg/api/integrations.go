package api

import (
	"fmt"
	"net/url"
)

// --- Generic integration endpoints (token integrations) ---

// ConnectIntegration POSTs credentials to a token integration's connect endpoint.
func (c *Client) ConnectIntegration(path string, body map[string]interface{}) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.post(path, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DisconnectIntegration POSTs to a token integration's disconnect endpoint.
func (c *Client) DisconnectIntegration(path string) error {
	return c.post(path, nil, nil)
}

// TestIntegration POSTs to a token integration's test endpoint.
func (c *Client) TestIntegration(path string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.post(path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Tribal knowledge sources (Confluence, Jira, Linear, Notion, Glean) ---

type TribalKnowledgeSource struct {
	ID               string `json:"id"`
	SourceType       string `json:"source_type"`
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	BaseURL          string `json:"base_url,omitempty"`
	ConnectionStatus string `json:"connection_status,omitempty"`
}

func (c *Client) ListKnowledgeSources() ([]TribalKnowledgeSource, error) {
	var out struct {
		Sources []TribalKnowledgeSource `json:"sources"`
	}
	if err := c.get("/api/tribal-knowledge/sources", nil, &out); err != nil {
		return nil, err
	}
	return out.Sources, nil
}

func (c *Client) CreateKnowledgeSource(body map[string]interface{}) (*TribalKnowledgeSource, error) {
	var out struct {
		Message string                `json:"message"`
		Source  TribalKnowledgeSource `json:"source"`
	}
	if err := c.post("/api/tribal-knowledge/sources", body, &out); err != nil {
		return nil, err
	}
	return &out.Source, nil
}

func (c *Client) DeleteKnowledgeSource(id string) error {
	return c.delete("/api/tribal-knowledge/sources/" + url.PathEscape(id))
}

func (c *Client) TestKnowledgeSource(id string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.post("/api/tribal-knowledge/sources/"+url.PathEscape(id)+"/test", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- OAuth integrations (GitHub, GitLab, Slack) ---

// GetGitHubInstallURL returns the GitHub App installation URL.
func (c *Client) GetGitHubInstallURL() (string, error) {
	var out struct {
		InstallationURL string `json:"installation_url"`
		AppName         string `json:"app_name"`
	}
	if err := c.get("/api/tenant/github/connect", nil, &out); err != nil {
		return "", err
	}
	return out.InstallationURL, nil
}

// GetGitLabAuthURL returns the GitLab OAuth authorization URL.
func (c *Client) GetGitLabAuthURL() (string, error) {
	var out struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	if err := c.get("/api/tenant/gitlab/connect", nil, &out); err != nil {
		return "", err
	}
	return out.AuthorizationURL, nil
}

// GetSlackInstallURL returns the Slack app installation URL.
func (c *Client) GetSlackInstallURL() (string, error) {
	var out struct {
		InstallURL string `json:"install_url"`
	}
	if err := c.get("/api/integrations/slack/install-url", nil, &out); err != nil {
		return "", err
	}
	return out.InstallURL, nil
}

// --- Kubernetes cluster onboarding ---

type OnboardClusterRequest struct {
	ClusterName string `json:"cluster_name"`
	Description string `json:"description,omitempty"`
	SafeApply   bool   `json:"safe_apply,omitempty"`
}

type MTLSCertificates struct {
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
	CACert     string `json:"ca_cert"`
}

type OnboardClusterResponse struct {
	AccessToken      string            `json:"access_token"`
	ClusterID        string            `json:"cluster_id"`
	ClusterName      string            `json:"cluster_name"`
	ExpiresIn        int64             `json:"expires_in"`
	TokenType        string            `json:"token_type"`
	ChartVersion     string            `json:"chart_version"`
	ChartName        string            `json:"chart_name,omitempty"`
	MTLSCertificates *MTLSCertificates `json:"mtls_certificates,omitempty"`
}

func (c *Client) OnboardCluster(req OnboardClusterRequest) (*OnboardClusterResponse, error) {
	var out OnboardClusterResponse
	if err := c.post("/api/operator/onboard-cluster", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- AWS ---

type AWSBootstrapResponse struct {
	ExternalID     string `json:"external_id"`
	CFNLaunchURL   string `json:"cfn_launch_url"`
	CLICreateCmd   string `json:"cli_create_cmd"`
	CLIOutputCmd   string `json:"cli_output_cmd"`
	KestrelAccount string `json:"kestrel_account_id"`
	Region         string `json:"region"`
}

func (c *Client) AWSBootstrap(region string) (*AWSBootstrapResponse, error) {
	body := map[string]string{"region": region}
	var out AWSBootstrapResponse
	if err := c.post("/api/integrations/aws/bootstrap", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AWSVerify(roleARN, externalID, region string) (map[string]interface{}, error) {
	body := map[string]string{
		"role_arn":    roleARN,
		"external_id": externalID,
		"region":      region,
	}
	var out map[string]interface{}
	if err := c.post("/api/integrations/aws/verify", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- OCI ---

func (c *Client) OCISetupInstructions() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.get("/api/integrations/oci/setup-instructions", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type OCIVerifyRequest struct {
	TenancyOCID    string `json:"tenancy_ocid"`
	UserOCID       string `json:"user_ocid"`
	Fingerprint    string `json:"fingerprint"`
	PrivateKey     string `json:"private_key"`
	Region         string `json:"region"`
	ConnectionName string `json:"connection_name,omitempty"`
}

func (c *Client) OCIVerify(req OCIVerifyRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.post("/api/integrations/oci/verify", &req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Server URL (for printing browser-flow instructions) ---

// BaseURL returns the configured server URL.
func (c *Client) BaseURL() string { return c.baseURL }

// FormatIntegrationPage returns a full URL to an in-app integration page.
func (c *Client) FormatIntegrationPage(path string) string {
	return fmt.Sprintf("%s%s", c.baseURL, path)
}
