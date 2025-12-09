//go:build sandbox
// +build sandbox

package renderer_test

import (
	"strings"
	"testing"

	"envseed/internal/parser"
	"envseed/internal/renderer"
	"envseed/internal/testsupport"
)

// [EVT-MEP-1]
func TestRender_MultiLevelEvaluation(t *testing.T) {
	testsupport.RequireSandbox(t)

	input := strings.ReplaceAll(`OUT=$(printf "%s|%s|%s" "$(printf %s "<pass://alpha>")" "$(printf %s "$(printf %s "<pass://beta>")")" "{{BT}}printf %s "<pass://gamma>"{{BT}}")`+"\n", "{{BT}}", "`")

	resolver := externalResolver{
		"alpha": "LEVEL-A",
		"beta":  "LEVEL-B",
		"gamma": "LEVEL-C",
	}

	rendered, err := renderer.RenderString(input, resolver)
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	values, err := renderer.ExportSandboxCapture(rendered, []string{"OUT"})
	if err != nil {
		t.Fatalf("sandbox execution failed: %v", err)
	}
	if got := values["OUT"]; got != "LEVEL-A|LEVEL-B|LEVEL-C" {
		t.Fatalf("OUT value = %q, want %q", got, "LEVEL-A|LEVEL-B|LEVEL-C")
	}
}

// [EVT-MEP-3]
func TestRender_RealisticScenario(t *testing.T) {
	testsupport.RequireSandbox(t)
	input := `# Example service configuration
API_ENDPOINT="https://<pass://host>/v1"
AUTH_HEADER="Bearer <pass://key>"
SCRIPT=$(printf "%s" "<pass://script|allow_newline>")
MESSAGE="<pass://message|allow_tab>"
`
	resolver := externalResolver{
		"host":    "internal.example.org",
		"key":     "s3cr3t$token",
		"script":  "deploy\nnext",
		"message": "tab\tok",
	}
	rendered, err := renderer.RenderString(input, resolver)
	if err != nil {
		t.Fatalf("RenderString error: %v", err)
	}
	if _, err := parser.Parse(rendered); err != nil {
		t.Fatalf("re-parse failure: %v", err)
	}
	if err := testsupport.BashValidate(rendered); err != nil {
		t.Fatalf("bash -n validation failed: %v", err)
	}
	values, err := renderer.ExportSandboxCapture(rendered, []string{"API_ENDPOINT", "AUTH_HEADER", "SCRIPT", "MESSAGE"})
	if err != nil {
		t.Fatalf("sandbox capture failed: %v", err)
	}
	want := map[string]string{
		"API_ENDPOINT": "https://internal.example.org/v1",
		"AUTH_HEADER":  "Bearer s3cr3t$token",
		"SCRIPT":       "deploy\nnext",
		"MESSAGE":      "tab\tok",
	}
	if diff := compareStringMaps(values, want); diff != "" {
		t.Fatalf("sandbox values mismatch:\n%s", diff)
	}
}
