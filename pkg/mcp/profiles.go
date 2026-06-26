package mcp

import (
	"errors"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

var (
	ErrProfileNotFound                = errors.New("profile not found")
	ErrCannotUninstallRequiredProfile = errors.New("cannot uninstall required profile")
)

// DevOps MCP profiles - pre-configured server bundles for common use cases
// Users can install/uninstall profiles to quickly set up their AI environment

// GetAvailableProfiles returns all available MCP profiles
func GetAvailableProfiles() []config.MCPProfile {
	return []config.MCPProfile{
		KubernetesProfile,
		DockerProfile,
		ShellProfile,
		GitHubProfile,
		AWSProfile,
		ArgoCDProfile,
		SequentialThinkingProfile,
		FullStackProfile,
	}
}

// KubernetesProfile - for K8s cluster management and inspection
var KubernetesProfile = config.MCPProfile{
	ID:       "k8s",
	Name:     "Kubernetes",
	Category: "kubernetes",
	Required: false,
	Description: "Kubernetes resource management and inspection tools. " +
		"List pods, deployments, services, view logs, apply manifests, and troubleshoot clusters.",
	Tags: []string{"kubernetes", "k8s", "cluster", "containerization"},
	Servers: []config.MCPServer{
		{
			Name:        "kubernetes",
			Command:     "npx",
			Args:        []string{"-y", "mcp-server-kubernetes"},
			Env:         map[string]string{"KUBECONFIG": ""},
			Description: "Kubernetes resource management and inspection tools",
			Enabled:     false,
		},
	},
}

// DockerProfile - for Docker container management
var DockerProfile = config.MCPProfile{
	ID:          "docker",
	Name:        "Docker",
	Category:    "containerization",
	Required:    false,
	Description: "Docker container and image management. Build, run, manage containers and orchestrate workloads.",
	Tags:        []string{"docker", "container", "containerization", "registry"},
	Servers: []config.MCPServer{
		{
			Name:        "docker",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-docker"},
			Description: "Docker container and image management",
			Enabled:     false,
		},
	},
}

// ShellProfile - for shell command execution and scripting
var ShellProfile = config.MCPProfile{
	ID:          "shell",
	Name:        "Shell & Bash",
	Category:    "devops",
	Required:    false,
	Description: "Execute shell commands and scripts. Run system commands, manage files, and automate tasks.",
	Tags:        []string{"shell", "bash", "scripting", "automation", "sysadmin"},
	Servers: []config.MCPServer{
		{
			Name:        "shell",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-bash"},
			Description: "Shell command execution and scripting support",
			Enabled:     false,
		},
	},
}

// GitHubProfile - for GitHub repository and code management
var GitHubProfile = config.MCPProfile{
	ID:       "github",
	Name:     "GitHub",
	Category: "vcs",
	Required: false,
	Description: "GitHub repository management, code search, and pull request operations. " +
		"Manage repositories, read code, create issues, and manage pull requests.",
	Tags: []string{"github", "git", "vcs", "source-control", "repository"},
	Servers: []config.MCPServer{
		{
			Name:        "github",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-github"},
			Env:         map[string]string{"GITHUB_TOKEN": ""},
			Description: "GitHub repository management and code operations",
			Enabled:     false,
		},
	},
}

// AWSProfile - for AWS cloud infrastructure management
var AWSProfile = config.MCPProfile{
	ID:       "aws",
	Name:     "AWS (Amazon Web Services)",
	Category: "cloud",
	Required: false,
	Description: "AWS cloud infrastructure management. Manage EC2, S3, RDS, Lambda, and other AWS services. " +
		"Requires AWS credentials configuration.",
	Tags: []string{"aws", "amazon", "cloud", "ec2", "s3", "lambda", "infrastructure"},
	Servers: []config.MCPServer{
		{
			Name:        "aws",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-aws"},
			Env:         map[string]string{"AWS_PROFILE": "default"},
			Description: "AWS cloud infrastructure management",
			Enabled:     false,
		},
	},
}

// ArgoCDProfile - for GitOps and Argo CD management
var ArgoCDProfile = config.MCPProfile{
	ID:       "argocd",
	Name:     "ArgoCD",
	Category: "kubernetes",
	Required: false,
	Description: "ArgoCD and GitOps application management. Deploy applications, manage sync status, " +
		"and handle application lifecycle through GitOps principles.",
	Tags: []string{"argocd", "gitops", "cd", "deployment", "application-management"},
	Servers: []config.MCPServer{
		{
			Name:        "argocd",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-argocd"},
			Env:         map[string]string{"ARGOCD_SERVER": "", "ARGOCD_AUTH_TOKEN": ""},
			Description: "ArgoCD GitOps application management",
			Enabled:     false,
		},
	},
}

// SequentialThinkingProfile - for complex reasoning and problem solving
var SequentialThinkingProfile = config.MCPProfile{
	ID:       "thinking",
	Name:     "Sequential Thinking",
	Category: "reasoning",
	Required: false,
	Description: "Enable step-by-step reasoning and problem decomposition. " +
		"Helps AI break down complex tasks into logical steps for better solutions.",
	Tags: []string{"reasoning", "thinking", "logic", "analysis", "decomposition"},
	Servers: []config.MCPServer{
		{
			Name:        "sequential-thinking",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
			Description: "Sequential thinking and step-by-step reasoning",
			Enabled:     false,
		},
	},
}

// FullStackProfile - comprehensive DevOps tooling (k8s + docker + shell + github)
var FullStackProfile = config.MCPProfile{
	ID:       "fullstack",
	Name:     "Full DevOps Stack",
	Category: "devops",
	Required: false,
	Description: "Complete DevOps toolkit combining Kubernetes, Docker, Shell, and GitHub tools. " +
		"Recommended for teams managing complex cloud-native infrastructure.",
	Tags: []string{"devops", "kubernetes", "docker", "github", "complete", "comprehensive"},
	Servers: []config.MCPServer{
		{
			Name:        "kubernetes",
			Command:     "npx",
			Args:        []string{"-y", "mcp-server-kubernetes"},
			Env:         map[string]string{"KUBECONFIG": ""},
			Description: "Kubernetes resource management and inspection tools",
			Enabled:     false,
		},
		{
			Name:        "docker",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-docker"},
			Description: "Docker container and image management",
			Enabled:     false,
		},
		{
			Name:        "shell",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-bash"},
			Description: "Shell command execution and scripting support",
			Enabled:     false,
		},
		{
			Name:        "github",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-github"},
			Env:         map[string]string{"GITHUB_TOKEN": ""},
			Description: "GitHub repository management and code operations",
			Enabled:     false,
		},
		{
			Name:        "sequential-thinking",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
			Description: "Sequential thinking and step-by-step reasoning",
			Enabled:     false,
		},
	},
}

// GetProfileByID returns a profile by its ID
func GetProfileByID(id string) *config.MCPProfile {
	for _, p := range GetAvailableProfiles() {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// IsProfileInstalled checks if a profile's servers are installed
func IsProfileInstalled(cfg *config.Config, profileID string) bool {
	profile := GetProfileByID(profileID)
	if profile == nil {
		return false
	}

	for _, profileServer := range profile.Servers {
		found := false
		for _, configServer := range cfg.MCP.Servers {
			if configServer.Name == profileServer.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// InstallProfile installs all servers from a profile to config
func InstallProfile(cfg *config.Config, profileID string) error {
	profile := GetProfileByID(profileID)
	if profile == nil {
		return ErrProfileNotFound
	}

	for _, server := range profile.Servers {
		cfg.AddMCPServer(server)
	}
	return nil
}

// UninstallProfile removes all servers from a profile (except required profiles)
func UninstallProfile(cfg *config.Config, profileID string) error {
	profile := GetProfileByID(profileID)
	if profile == nil {
		return ErrProfileNotFound
	}

	if profile.Required {
		return ErrCannotUninstallRequiredProfile
	}

	for _, server := range profile.Servers {
		cfg.RemoveMCPServer(server.Name)
	}
	return nil
}
