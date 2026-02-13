package anonymizer

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestAnonymizeIPAddresses(t *testing.T) {
	a := New(true)

	input := "Pod is running on node 10.0.1.5 and connects to 192.168.1.100"
	result := a.Anonymize(input)

	if strings.Contains(result, "10.0.1.5") {
		t.Error("IP 10.0.1.5 was not anonymized")
	}
	if strings.Contains(result, "192.168.1.100") {
		t.Error("IP 192.168.1.100 was not anonymized")
	}
	if !strings.Contains(result, "<IP_") {
		t.Error("expected IP placeholder in output")
	}

	// Round-trip
	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeSameValueSamePlaceholder(t *testing.T) {
	a := New(true)

	input := "Connect from 10.0.0.1 to 10.0.0.2, then back to 10.0.0.1"
	result := a.Anonymize(input)

	// 10.0.0.1 appears twice, 10.0.0.2 once
	// Same value gets same placeholder, so we should have exactly 2 unique IP mappings
	if a.MappingCount() != 2 {
		t.Errorf("expected 2 mappings, got %d", a.MappingCount())
	}

	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeDockerImages(t *testing.T) {
	a := New(true)

	input := "Image: registry.example.com/org/myapp:v1.2.3 is pulling"
	result := a.Anonymize(input)

	if strings.Contains(result, "registry.example.com") {
		t.Error("docker image was not anonymized")
	}
	if !strings.Contains(result, "<IMAGE_") {
		t.Error("expected IMAGE placeholder in output")
	}

	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeDockerImageWithSHA(t *testing.T) {
	a := New(true)

	sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	input := fmt.Sprintf("Image: gcr.io/project/app@sha256:%s", sha)
	result := a.Anonymize(input)

	if strings.Contains(result, "gcr.io") {
		t.Error("docker image with sha was not anonymized")
	}

	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeURLs(t *testing.T) {
	a := New(true)

	input := "Webhook: https://api.example.com/v1/hooks and http://internal.svc:8080/health"
	result := a.Anonymize(input)

	if strings.Contains(result, "api.example.com") {
		t.Error("HTTPS URL was not anonymized")
	}
	if strings.Contains(result, "internal.svc") {
		t.Error("HTTP URL was not anonymized")
	}
	if !strings.Contains(result, "<URL_") {
		t.Error("expected URL placeholder in output")
	}

	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeEmails(t *testing.T) {
	a := New(true)

	input := "Contact admin@example.com or dev+k8s@company.org for support"
	result := a.Anonymize(input)

	if strings.Contains(result, "admin@example.com") {
		t.Error("email was not anonymized")
	}
	if strings.Contains(result, "dev+k8s@company.org") {
		t.Error("email with + was not anonymized")
	}
	if !strings.Contains(result, "<EMAIL_") {
		t.Error("expected EMAIL placeholder in output")
	}

	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeTokens(t *testing.T) {
	a := New(true)

	token := "abcdefghijklmnopqrstuvwxyz1234567890ABCD"
	input := fmt.Sprintf("Authorization: Bearer %s", token)
	result := a.Anonymize(input)

	if strings.Contains(result, token) {
		t.Error("token was not anonymized")
	}
	if !strings.Contains(result, "<TOKEN_") {
		t.Error("expected TOKEN placeholder in output")
	}

	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeDisabled(t *testing.T) {
	a := New(false)

	input := "IP: 10.0.0.1, URL: https://example.com, email: user@test.com"
	result := a.Anonymize(input)

	if result != input {
		t.Errorf("disabled anonymizer should be no-op, got: %s", result)
	}

	result = a.Deanonymize(input)
	if result != input {
		t.Errorf("disabled deanonymize should be no-op, got: %s", result)
	}
}

func TestReset(t *testing.T) {
	a := New(true)

	a.Anonymize("IP: 10.0.0.1")
	if a.MappingCount() == 0 {
		t.Error("expected mappings after anonymize")
	}

	a.Reset()
	if a.MappingCount() != 0 {
		t.Errorf("expected 0 mappings after reset, got %d", a.MappingCount())
	}
}

func TestAnonymizeMixedContent(t *testing.T) {
	a := New(true)

	input := `Pod nginx-7f89b4c5d-xk2p9 is running on 10.244.0.5.
Image: docker.io/library/nginx:1.25 pulled successfully.
Webhook configured: https://hooks.slack.com/services/T00/B00/abc123.
Contact: ops@company.com for issues.`

	result := a.Anonymize(input)

	// Verify sensitive data is masked
	if strings.Contains(result, "10.244.0.5") {
		t.Error("IP was not anonymized")
	}
	if strings.Contains(result, "docker.io/library/nginx") {
		t.Error("image was not anonymized")
	}
	if strings.Contains(result, "hooks.slack.com") {
		t.Error("URL was not anonymized")
	}
	if strings.Contains(result, "ops@company.com") {
		t.Error("email was not anonymized")
	}

	// Verify non-sensitive text is preserved
	if !strings.Contains(result, "Pod") {
		t.Error("non-sensitive text 'Pod' was removed")
	}
	if !strings.Contains(result, "pulled successfully") {
		t.Error("non-sensitive text was removed")
	}

	// Round-trip
	restored := a.Deanonymize(result)
	if restored != input {
		t.Errorf("round-trip failed:\n  got:  %s\n  want: %s", restored, input)
	}
}

func TestAnonymizeNoSensitiveData(t *testing.T) {
	a := New(true)

	input := "Pod is running. Deployment is healthy."
	result := a.Anonymize(input)

	if result != input {
		t.Errorf("text without sensitive data should be unchanged, got: %s", result)
	}
	if a.MappingCount() != 0 {
		t.Errorf("expected 0 mappings for non-sensitive text, got %d", a.MappingCount())
	}
}

func TestAnonymizeEmptyString(t *testing.T) {
	a := New(true)

	result := a.Anonymize("")
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

func TestThreadSafety(t *testing.T) {
	a := New(true)

	inputs := []string{
		"IP: 10.0.0.1",
		"IP: 10.0.0.2",
		"URL: https://example.com/path",
		"Email: user@test.com",
		"IP: 172.16.0.1",
		"URL: http://internal.svc:8080",
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			input := inputs[idx%len(inputs)]
			result := a.Anonymize(input)
			_ = a.Deanonymize(result)
		}(i)
	}
	wg.Wait()

	// If we get here without a race condition, the test passes
	if a.MappingCount() == 0 {
		t.Error("expected some mappings after concurrent anonymization")
	}
}

func TestDeanonymizePreservesUnknownPlaceholders(t *testing.T) {
	a := New(true)

	// Deanonymize text that has placeholders we didn't create
	input := "Result: <UNKNOWN_1> is not mapped"
	result := a.Deanonymize(input)

	if result != input {
		t.Errorf("unknown placeholders should be preserved, got: %s", result)
	}
}

func TestAnonymizeConsistentAcrossCalls(t *testing.T) {
	a := New(true)

	// First call
	result1 := a.Anonymize("Connect to 10.0.0.1")
	// Second call with same IP
	result2 := a.Anonymize("Also reaching 10.0.0.1")

	// Extract the placeholder used for 10.0.0.1
	// Both should use the same placeholder
	if a.MappingCount() != 1 {
		t.Errorf("expected 1 mapping for repeated value, got %d", a.MappingCount())
	}

	// Verify both results use the same placeholder
	placeholder := "<IP_1>"
	if !strings.Contains(result1, placeholder) {
		t.Errorf("first result should contain %s, got: %s", placeholder, result1)
	}
	if !strings.Contains(result2, placeholder) {
		t.Errorf("second result should contain %s, got: %s", placeholder, result2)
	}
}

func TestMappingCountAccuracy(t *testing.T) {
	a := New(true)

	if a.MappingCount() != 0 {
		t.Error("new anonymizer should have 0 mappings")
	}

	a.Anonymize("IP: 10.0.0.1 and 10.0.0.2")
	if a.MappingCount() != 2 {
		t.Errorf("expected 2 mappings, got %d", a.MappingCount())
	}

	a.Anonymize("IP: 10.0.0.1") // same IP, no new mapping
	if a.MappingCount() != 2 {
		t.Errorf("expected still 2 mappings, got %d", a.MappingCount())
	}

	a.Anonymize("Email: user@test.com")
	if a.MappingCount() != 3 {
		t.Errorf("expected 3 mappings, got %d", a.MappingCount())
	}
}
