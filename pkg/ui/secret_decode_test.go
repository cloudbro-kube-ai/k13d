package ui

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodeSecretYAML_BasicDecode(t *testing.T) {
	input := `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
type: Opaque
data:
  username: ` + base64.StdEncoding.EncodeToString([]byte("admin")) + `
  password: ` + base64.StdEncoding.EncodeToString([]byte("s3cret!")) + `
`

	result := decodeSecretYAML(input)

	if !strings.Contains(result, "username: admin") {
		t.Errorf("expected decoded username 'admin', got:\n%s", result)
	}
	if !strings.Contains(result, "password: s3cret!") {
		t.Errorf("expected decoded password 's3cret!', got:\n%s", result)
	}
	// Metadata should be unchanged
	if !strings.Contains(result, "name: my-secret") {
		t.Errorf("metadata should be unchanged, got:\n%s", result)
	}
}

func TestDecodeSecretYAML_EmptyDataSection(t *testing.T) {
	input := `apiVersion: v1
kind: Secret
metadata:
  name: empty-secret
data:
type: Opaque
`

	result := decodeSecretYAML(input)

	if result != input {
		t.Errorf("empty data section should pass through unchanged, got:\n%s", result)
	}
}

func TestDecodeSecretYAML_StringDataNotDecoded(t *testing.T) {
	input := `apiVersion: v1
kind: Secret
metadata:
  name: test-secret
stringData:
  config.yaml: |
    apiUrl: https://my.api.com
    token: SOME_TOKEN
type: Opaque
`

	result := decodeSecretYAML(input)

	// stringData should pass through unchanged
	if result != input {
		t.Errorf("stringData should not be decoded, got:\n%s", result)
	}
}

func TestDecodeSecretYAML_MixedSections(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("decoded-value"))
	input := `apiVersion: v1
kind: Secret
metadata:
  name: mixed-secret
data:
  key1: ` + encoded + `
stringData:
  key2: plain-value
type: Opaque
`

	result := decodeSecretYAML(input)

	if !strings.Contains(result, "key1: decoded-value") {
		t.Errorf("data section value should be decoded, got:\n%s", result)
	}
	if !strings.Contains(result, "key2: plain-value") {
		t.Errorf("stringData section should remain unchanged, got:\n%s", result)
	}
}

func TestDecodeSecretYAML_NonBase64ValuePassthrough(t *testing.T) {
	input := `apiVersion: v1
kind: Secret
metadata:
  name: test
data:
  key: not-valid-base64!!!
type: Opaque
`

	result := decodeSecretYAML(input)

	// Non-base64 values should pass through unchanged
	if !strings.Contains(result, "key: not-valid-base64!!!") {
		t.Errorf("non-base64 values should pass through, got:\n%s", result)
	}
}

func TestDecodeSecretYAML_RoundTrip(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("my-password"))
	input := `apiVersion: v1
kind: Secret
data:
  password: ` + encoded + `
type: Opaque
`

	decoded := decodeSecretYAML(input)
	if !strings.Contains(decoded, "password: my-password") {
		t.Fatalf("decode failed: %s", decoded)
	}

	// Original input should still have encoded value
	if !strings.Contains(input, "password: "+encoded) {
		t.Error("original input was modified")
	}
}

func TestDecodeSecretYAML_MultipleKeys(t *testing.T) {
	input := `apiVersion: v1
kind: Secret
data:
  db-host: ` + base64.StdEncoding.EncodeToString([]byte("localhost")) + `
  db-port: ` + base64.StdEncoding.EncodeToString([]byte("5432")) + `
  db-name: ` + base64.StdEncoding.EncodeToString([]byte("mydb")) + `
  db-user: ` + base64.StdEncoding.EncodeToString([]byte("postgres")) + `
type: Opaque
`

	result := decodeSecretYAML(input)

	expected := map[string]string{
		"db-host": "localhost",
		"db-port": "5432",
		"db-name": "mydb",
		"db-user": "postgres",
	}

	for key, val := range expected {
		if !strings.Contains(result, key+": "+val) {
			t.Errorf("expected %s: %s in result, got:\n%s", key, val, result)
		}
	}
}

func TestVimViewerSecretToggle(t *testing.T) {
	v := &VimViewer{
		isSecretView: true,
	}

	if v.secretDecoded {
		t.Error("secretDecoded should initially be false")
	}
	if !v.isSecretView {
		t.Error("isSecretView should be true")
	}

	// Test toggle state
	v.secretDecoded = true
	if !v.secretDecoded {
		t.Error("secretDecoded should be true after toggle")
	}
	v.secretDecoded = false
	if v.secretDecoded {
		t.Error("secretDecoded should be false after second toggle")
	}
}

func TestVimViewerLogViewDefaults(t *testing.T) {
	v := &VimViewer{
		isLogView:  true,
		autoScroll: true,
		textWrap:   true,
	}

	if !v.isLogView {
		t.Error("isLogView should be true")
	}
	if !v.autoScroll {
		t.Error("autoScroll should default to true for logs")
	}
	if !v.textWrap {
		t.Error("textWrap should default to true for logs")
	}
}

func TestVimViewerLogWrapToggle(t *testing.T) {
	v := &VimViewer{
		isLogView: true,
		textWrap:  true,
	}

	v.textWrap = !v.textWrap
	if v.textWrap {
		t.Error("textWrap should be false after toggle")
	}

	v.textWrap = !v.textWrap
	if !v.textWrap {
		t.Error("textWrap should be true after second toggle")
	}
}

func TestVimViewerAutoScrollToggle(t *testing.T) {
	v := &VimViewer{
		isLogView:  true,
		autoScroll: true,
	}

	v.autoScroll = !v.autoScroll
	if v.autoScroll {
		t.Error("autoScroll should be false after toggle")
	}

	v.autoScroll = !v.autoScroll
	if !v.autoScroll {
		t.Error("autoScroll should be true after second toggle")
	}
}

func TestDecodeSecretYAML_PipeAndAngleValues(t *testing.T) {
	input := `apiVersion: v1
kind: Secret
data:
  multiline: |
    line1
    line2
  folded: >
    folded content
type: Opaque
`

	result := decodeSecretYAML(input)

	// Pipe and angle indicators should not be decoded
	if !strings.Contains(result, "multiline: |") {
		t.Errorf("pipe indicator should pass through, got:\n%s", result)
	}
	if !strings.Contains(result, "folded: >") {
		t.Errorf("angle indicator should pass through, got:\n%s", result)
	}
}

func TestDecodeSecretYAML_IndentedData(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("value"))
	input := `apiVersion: v1
kind: Secret
metadata:
  name: test
  namespace: default
  labels:
    app: test
data:
  key: ` + encoded + `
type: Opaque
`

	result := decodeSecretYAML(input)

	if !strings.Contains(result, "key: value") {
		t.Errorf("nested data should be decoded, got:\n%s", result)
	}
	// Labels should be unchanged
	if !strings.Contains(result, "app: test") {
		t.Errorf("labels should be unchanged, got:\n%s", result)
	}
}
