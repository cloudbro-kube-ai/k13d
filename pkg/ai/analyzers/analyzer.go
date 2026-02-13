package analyzers

import "context"

// Severity levels for findings.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Finding represents a single diagnostic finding.
type Finding struct {
	Analyzer    string   `json:"analyzer"`
	Resource    string   `json:"resource"` // e.g., "Pod/default/my-pod"
	Severity    Severity `json:"severity"`
	Title       string   `json:"title"`       // Short description
	Details     string   `json:"details"`     // Detailed explanation
	Suggestions []string `json:"suggestions"` // Remediation suggestions
}

// ResourceInfo contains Kubernetes resource data for analysis.
type ResourceInfo struct {
	Kind       string
	Name       string
	Namespace  string
	Status     string
	Conditions []Condition
	Events     []Event
	Containers []ContainerInfo
	Labels     map[string]string
	Raw        map[string]interface{} // Raw resource data
}

// Condition represents a Kubernetes resource condition.
type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// Event represents a Kubernetes event.
type Event struct {
	Type    string
	Reason  string
	Message string
	Count   int32
}

// ContainerInfo holds container status information.
type ContainerInfo struct {
	Name         string
	Ready        bool
	RestartCount int32
	State        string
	Reason       string
	Message      string
	ExitCode     int32
	Image        string
}

// Analyzer interface for pluggable diagnostic checks.
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, resource *ResourceInfo) []Finding
}

// Registry holds all registered analyzers.
type Registry struct {
	analyzers []Analyzer
}

// NewRegistry creates an empty analyzer registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds an analyzer to the registry.
func (r *Registry) Register(a Analyzer) {
	r.analyzers = append(r.analyzers, a)
}

// AnalyzeAll runs all registered analyzers against a resource.
func (r *Registry) AnalyzeAll(ctx context.Context, resource *ResourceInfo) []Finding {
	if resource == nil {
		return nil
	}
	var findings []Finding
	for _, a := range r.analyzers {
		findings = append(findings, a.Analyze(ctx, resource)...)
	}
	return findings
}

// ValidateCross runs cross-resource validation across multiple resources.
func (r *Registry) ValidateCross(ctx context.Context, resources []*ResourceInfo) []Finding {
	cv := &CrossValidator{}
	return cv.ValidateCross(ctx, resources)
}

// DefaultRegistry creates a registry with all built-in analyzers.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(&PodAnalyzer{})
	r.Register(&ServiceAnalyzer{})
	r.Register(&DeploymentAnalyzer{})
	r.Register(&NodeAnalyzer{})
	return r
}
