package domain

import "time"

type Project struct {
	Id             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	RepositoryUrl  string    `json:"repositoryUrl"`
	Stack          string    `json:"stack"`
	WorkServerId   string    `json:"workServerId"`
	DeployServerId string    `json:"deployServerId"`
	CreatedAt      time.Time `json:"createdAt"`
}
type Agent struct {
	Id           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Description  string   `json:"description"`
	Skills       []string `json:"skills"`
	ProjectId    *string  `json:"projectId"`
	ServerId     string   `json:"serverId"`
	Status       string   `json:"status"`
	Instructions string   `json:"instructions"`
}
type ServerConfig struct {
	Provider          string            `json:"provider"`
	Endpoint          string            `json:"endpoint"`
	Model             string            `json:"model"`
	FallbackModel     string            `json:"fallbackModel"`
	CredentialRef     string            `json:"credentialRef"`
	ModelOverrides    map[string]string `json:"modelOverrides"`
	DailyBudget       float64           `json:"dailyBudget"`
	MaxConcurrency    int               `json:"maxConcurrency"`
	TimeoutMinutes    int               `json:"timeoutMinutes"`
	MaxRetries        int               `json:"maxRetries"`
	RequireApproval   bool              `json:"requireApproval"`
	IsolatedWorkspace bool              `json:"isolatedWorkspace"`
	StopOnBudget      bool              `json:"stopOnBudget"`
}
type Server struct {
	Id       string       `json:"id"`
	Name     string       `json:"name"`
	Host     string       `json:"host"`
	Role     string       `json:"role"`
	Region   string       `json:"region"`
	Status   string       `json:"status"`
	Config   ServerConfig `json:"config"`
	Metrics  Metrics      `json:"metrics"`
	LastSeen *time.Time   `json:"lastSeen"`
}
type Metrics struct {
	CpuPercent    float64     `json:"cpuPercent"`
	MemoryUsedGb  float64     `json:"memoryUsedGb"`
	MemoryTotalGb float64     `json:"memoryTotalGb"`
	DiskUsedGb    float64     `json:"diskUsedGb"`
	DiskTotalGb   float64     `json:"diskTotalGb"`
	Containers    []Container `json:"containers"`
}
type Container struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
}
type Task struct {
	Id             string        `json:"id"`
	Title          string        `json:"title"`
	Brief          string        `json:"brief"`
	Criteria       string        `json:"criteria"`
	ProjectId      string        `json:"projectId"`
	AgentId        string        `json:"agentId"`
	WorkServerId   string        `json:"workServerId"`
	DeployServerId string        `json:"deployServerId"`
	Priority       string        `json:"priority"`
	Status         string        `json:"status"`
	Budget         float64       `json:"budget"`
	Progress       int           `json:"progress"`
	Model          string        `json:"model"`
	Provider       string        `json:"provider"`
	Result         string        `json:"result"`
	Error          string        `json:"error"`
	Branch         string        `json:"branch"`
	CreatedAt      time.Time     `json:"createdAt"`
	Spent          float64       `json:"spent"`
	LeaseId        string        `json:"leaseId,omitempty"`
	Execution      TaskExecution `json:"execution"`
}
type TaskExecution struct {
	Model          string `json:"model"`
	Endpoint       string `json:"endpoint"`
	CredentialRef  string `json:"credentialRef"`
	Instructions   string `json:"instructions"`
	RepositoryUrl  string `json:"repositoryUrl"`
	TimeoutMinutes int    `json:"timeoutMinutes"`
	MaxRetries     int    `json:"maxRetries"`
}
type TaskUpdate struct {
	Status   string  `json:"status"`
	Progress int     `json:"progress"`
	Result   string  `json:"result"`
	Error    string  `json:"error"`
	Spent    float64 `json:"spent"`
	LeaseId  string  `json:"leaseId"`
}
type Approval struct {
	Id        string    `json:"id"`
	TaskId    string    `json:"taskId"`
	ServerId  string    `json:"serverId"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}
type ApprovalDecision struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}
type Message struct {
	Id        string    `json:"id"`
	TaskId    string    `json:"taskId"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}
type Event struct {
	Id        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}
type Document struct {
	Id        string `json:"id"`
	ProjectId string `json:"projectId"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}
type RunnerIdentity struct {
	ServerId string `json:"serverId"`
}
type PairToken struct {
	Token    string `json:"token"`
	ServerId string `json:"serverId"`
}
type TokenHash struct {
	TokenHash string `json:"tokenHash"`
}
type ContainerCommand struct {
	Id          string    `json:"id"`
	ServerId    string    `json:"serverId"`
	ContainerId string    `json:"containerId"`
	Action      string    `json:"action"`
	Status      string    `json:"status"`
	Result      string    `json:"result"`
	CreatedAt   time.Time `json:"createdAt"`
}
type Settings struct {
	WorkspaceName string  `json:"workspaceName"`
	Timezone      string  `json:"timezone"`
	DailyBudget   float64 `json:"dailyBudget"`
	MonthlyBudget float64 `json:"monthlyBudget"`
}
