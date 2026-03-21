package web

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func buildPodSecurityDetails(spec corev1.PodSpec) map[string]interface{} {
	podSC := spec.SecurityContext
	podSeccomp := describeSeccompProfile(nil)
	podRunAsNonRoot := "inherit"
	podRunAsUser := interface{}(nil)
	podFSGroup := interface{}(nil)

	if podSC != nil {
		podSeccomp = describeSeccompProfile(podSC.SeccompProfile)
		podRunAsNonRoot = describeBoolSetting(podSC.RunAsNonRoot, "inherit")
		podRunAsUser = derefInt64Value(podSC.RunAsUser)
		podFSGroup = derefInt64Value(podSC.FSGroup)
	}

	serviceAccount := spec.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = "default"
	}

	containerItems := make([]map[string]interface{}, 0, len(spec.Containers))
	warnings := make([]string, 0)
	privilegedCount := 0
	allowPrivilegeEscalationCount := 0
	readOnlyRootFSMissingCount := 0
	addedCapabilitiesCount := 0
	nonRootMissingCount := 0
	unconfinedCount := 0

	for _, container := range spec.Containers {
		sc := container.SecurityContext
		privileged := false
		allowPrivilegeEscalation := "inherit"
		readOnlyRootFS := "inherit"
		runAsNonRoot := podRunAsNonRoot
		runAsUser := interface{}(nil)
		seccompProfile := podSeccomp
		capAdd := []string{}
		capDrop := []string{}

		if sc != nil {
			if sc.Privileged != nil {
				privileged = *sc.Privileged
			}
			allowPrivilegeEscalation = describeBoolSetting(sc.AllowPrivilegeEscalation, "inherit")
			readOnlyRootFS = describeBoolSetting(sc.ReadOnlyRootFilesystem, "inherit")
			if sc.RunAsNonRoot != nil {
				runAsNonRoot = describeBoolSetting(sc.RunAsNonRoot, "inherit")
			}
			if user := derefInt64Value(sc.RunAsUser); user != nil {
				runAsUser = user
			}
			if sc.SeccompProfile != nil {
				seccompProfile = describeSeccompProfile(sc.SeccompProfile)
			}
			if sc.Capabilities != nil {
				capAdd = capabilityStrings(sc.Capabilities.Add)
				capDrop = capabilityStrings(sc.Capabilities.Drop)
			}
		}

		if privileged {
			privilegedCount++
		}
		if allowPrivilegeEscalation == "true" {
			allowPrivilegeEscalationCount++
		}
		if readOnlyRootFS != "true" {
			readOnlyRootFSMissingCount++
		}
		if runAsNonRoot != "true" {
			nonRootMissingCount++
		}
		if strings.EqualFold(seccompProfile, "Unconfined") || strings.EqualFold(seccompProfile, "Unset") {
			unconfinedCount++
		}
		if len(capAdd) > 0 {
			addedCapabilitiesCount++
		}

		containerItems = append(containerItems, map[string]interface{}{
			"name":                     container.Name,
			"image":                    container.Image,
			"privileged":               privileged,
			"allowPrivilegeEscalation": allowPrivilegeEscalation,
			"readOnlyRootFilesystem":   readOnlyRootFS,
			"runAsNonRoot":             runAsNonRoot,
			"runAsUser":                runAsUser,
			"seccompProfile":           seccompProfile,
			"capabilitiesAdd":          capAdd,
			"capabilitiesDrop":         capDrop,
		})
	}

	if spec.HostNetwork {
		warnings = append(warnings, "Host network is enabled.")
	}
	if spec.HostPID {
		warnings = append(warnings, "Host PID namespace is enabled.")
	}
	if spec.HostIPC {
		warnings = append(warnings, "Host IPC namespace is enabled.")
	}
	if privilegedCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d container(s) run as privileged.", privilegedCount))
	}
	if allowPrivilegeEscalationCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d container(s) allow privilege escalation.", allowPrivilegeEscalationCount))
	}
	if addedCapabilitiesCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d container(s) add Linux capabilities.", addedCapabilitiesCount))
	}
	if unconfinedCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d container(s) do not use a restricted seccomp profile.", unconfinedCount))
	}
	if nonRootMissingCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d container(s) are not explicitly non-root.", nonRootMissingCount))
	}
	if readOnlyRootFSMissingCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d container(s) do not enable readOnlyRootFilesystem.", readOnlyRootFSMissingCount))
	}

	return map[string]interface{}{
		"posture":                            securityPostureLabel(warnings),
		"podSeccompProfile":                  podSeccomp,
		"podRunAsNonRoot":                    podRunAsNonRoot,
		"podRunAsUser":                       podRunAsUser,
		"podFSGroup":                         podFSGroup,
		"serviceAccount":                     serviceAccount,
		"automountServiceAccount":            describeBoolSetting(spec.AutomountServiceAccountToken, "default"),
		"hostNetwork":                        spec.HostNetwork,
		"hostPID":                            spec.HostPID,
		"hostIPC":                            spec.HostIPC,
		"warnings":                           warnings,
		"containers":                         containerItems,
		"privilegedContainers":               privilegedCount,
		"containersAllowPrivilegeEscalation": allowPrivilegeEscalationCount,
		"nonRootUnsetOrFalse":                nonRootMissingCount,
		"readOnlyRootFSMissing":              readOnlyRootFSMissingCount,
		"containersWithAddedCaps":            addedCapabilitiesCount,
		"containersWithWeakSeccomp":          unconfinedCount,
	}
}

func describeSeccompProfile(profile *corev1.SeccompProfile) string {
	if profile == nil {
		return "Unset"
	}
	switch profile.Type {
	case corev1.SeccompProfileTypeRuntimeDefault:
		return "RuntimeDefault"
	case corev1.SeccompProfileTypeUnconfined:
		return "Unconfined"
	case corev1.SeccompProfileTypeLocalhost:
		if profile.LocalhostProfile != nil && *profile.LocalhostProfile != "" {
			return "Localhost/" + *profile.LocalhostProfile
		}
		return "Localhost"
	default:
		return string(profile.Type)
	}
}

func describeBoolSetting(value *bool, fallback string) string {
	if value == nil {
		return fallback
	}
	if *value {
		return "true"
	}
	return "false"
}

func capabilityStrings(caps []corev1.Capability) []string {
	items := make([]string, 0, len(caps))
	for _, cap := range caps {
		items = append(items, string(cap))
	}
	return items
}

func derefInt64Value(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func securityPostureLabel(warnings []string) string {
	switch {
	case len(warnings) == 0:
		return "Hardened"
	case len(warnings) <= 2:
		return "Needs review"
	default:
		return "Elevated"
	}
}
