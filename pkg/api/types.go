package api

import "encoding/json"

type LoginResponse struct {
	SessionToken string `json:"session_token"`
	UserID       string `json:"user_id"`
	ExpiresAt    int64  `json:"expires_at"`
	Requires2FA  bool   `json:"requires_2fa,omitempty"`
}

type Workflow struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Status          string          `json:"status"`
	Definition      json.RawMessage `json:"definition"`
	TriggerConfig   json.RawMessage `json:"trigger_config"`
	NLPrompt        string          `json:"nl_prompt,omitempty"`
	AlertConfig     json.RawMessage `json:"alert_config,omitempty"`
	CreatedBy       *string         `json:"created_by,omitempty"`
	UpdatedBy       *string         `json:"updated_by,omitempty"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
	LastTriggeredAt *string         `json:"last_triggered_at,omitempty"`
	TriggerCount    int             `json:"trigger_count"`
}

type WorkflowDefinition struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

type WorkflowNode struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type WorkflowEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

type NodeData struct {
	Label       string `json:"label"`
	Integration string `json:"integration,omitempty"`
	ActionID    string `json:"action_id,omitempty"`
	SignalType  string `json:"signal_type,omitempty"`
}

type CreateWorkflowRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Definition    json.RawMessage `json:"definition"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
	NLPrompt      string          `json:"nl_prompt,omitempty"`
	AlertConfig   json.RawMessage `json:"alert_config,omitempty"`
}

type UpdateWorkflowRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Definition    json.RawMessage `json:"definition"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
	NLPrompt      string          `json:"nl_prompt,omitempty"`
	AlertConfig   json.RawMessage `json:"alert_config,omitempty"`
}

type GenerateWorkflowResponse struct {
	Success       bool            `json:"success"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Definition    json.RawMessage `json:"definition"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
	Explanation   string          `json:"explanation"`
	Error         *string         `json:"error,omitempty"`
}

type GeneratedWorkflow struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Definition    json.RawMessage `json:"definition"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
}

type WorkflowStats struct {
	TotalWorkflows  int                    `json:"total_workflows"`
	ActiveWorkflows int                    `json:"active_workflows"`
	TotalExecutions int                    `json:"total_executions"`
	StatusBreakdown map[string]int         `json:"status_breakdown"`
	DailyExecutions []DailyExecutionCount  `json:"daily_executions"`
}

type DailyExecutionCount struct {
	Date      string `json:"date"`
	Completed int    `json:"completed"`
	Failed    int    `json:"failed"`
	Running   int    `json:"running"`
	Waiting   int    `json:"waiting"`
}

type WorkflowExecution struct {
	ID            string          `json:"id"`
	WorkflowID    string          `json:"workflow_id"`
	TenantID      string          `json:"tenant_id"`
	IncidentID    string          `json:"incident_id,omitempty"`
	TriggerSignal json.RawMessage `json:"trigger_signal"`
	Status        string          `json:"status"`
	StartedAt     string          `json:"started_at"`
	CompletedAt   *string         `json:"completed_at,omitempty"`
	StepResults   json.RawMessage `json:"step_results"`
	ErrorMessage  string          `json:"error_message,omitempty"`
	CurrentStepID string          `json:"current_step_id,omitempty"`
}

type ExecutionListResponse struct {
	Executions []WorkflowExecution `json:"executions"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
}

type WorkflowApproval struct {
	ID           string          `json:"id"`
	ExecutionID  string          `json:"execution_id"`
	TenantID     string          `json:"tenant_id"`
	StepID       string          `json:"step_id"`
	ApprovalType string          `json:"approval_type"`
	Status       string          `json:"status"`
	RequestedAt  string          `json:"requested_at"`
	RespondedAt  *string         `json:"responded_at,omitempty"`
	RespondedBy  *string         `json:"responded_by,omitempty"`
	PRUrl        string          `json:"pr_url,omitempty"`
	ExpiresAt    *string         `json:"expires_at,omitempty"`
	Context      json.RawMessage `json:"context,omitempty"`
}

type WorkflowRequest struct {
	ID                string          `json:"id"`
	TenantID          string          `json:"tenant_id"`
	RequesterSlackUID string          `json:"requester_slack_user_id"`
	RequesterName     string          `json:"requester_name"`
	Source            string          `json:"source"`
	Prompt            string          `json:"prompt"`
	Category          string          `json:"category"`
	MatchedWorkflowID *string         `json:"matched_workflow_id,omitempty"`
	ExecutionID       *string         `json:"execution_id,omitempty"`
	Parameters        json.RawMessage `json:"parameters"`
	MissingParameters json.RawMessage `json:"missing_parameters"`
	SuggestedWorkflow json.RawMessage `json:"suggested_workflow,omitempty"`
	Status            string          `json:"status"`
	CreatedAt         string          `json:"created_at"`
	UpdatedAt         string          `json:"updated_at"`
}

type SuggestedWorkflow struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Definition  json.RawMessage `json:"definition"`
	NLPrompt    string          `json:"nl_prompt,omitempty"`
	CreatedAt   string          `json:"created_at"`
}

type Catalog struct {
	Signals      []SignalTemplate  `json:"signals"`
	Actions      []ActionTemplate  `json:"actions"`
	Integrations []IntegrationMeta `json:"integrations"`
}

type SignalTemplate struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	SignalType  string `json:"signal_type"`
}

type ActionTemplate struct {
	ID          string `json:"id"`
	Integration string `json:"integration"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type IntegrationMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IntegrationStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
}

// API Key types

type APIKey struct {
	ID         string  `json:"id"`
	TenantID   string  `json:"tenant_id"`
	UserID     string  `json:"user_id"`
	Name       string  `json:"name"`
	KeyPrefix  string  `json:"key_prefix"`
	Scopes     []string `json:"scopes"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
	RevokedAt  *string `json:"revoked_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type APIKeyWithRaw struct {
	APIKey
	RawKey string `json:"raw_key"`
}

type CreateAPIKeyRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn string   `json:"expires_in,omitempty"`
}
