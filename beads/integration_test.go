//go:build integration

package beads

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIntegrationReadyWithRealBD(t *testing.T) {
	repo, bdPath, client := setupIntegrationRepo(t)
	runIntegrationCommand(t, repo, bdPath,
		"create", "Ready issue",
		"--labels", "sdk",
		"--priority", "1",
	)

	issues, err := client.Ready(context.Background(), WithAll(), WithLabel("sdk"))
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if !slices.ContainsFunc(issues, func(issue Issue) bool {
		return strings.HasPrefix(issue.ID, "bgotest-") &&
			issue.Title == "Ready issue" &&
			issue.Status == "open"
	}) {
		t.Fatalf("Ready issues did not include created issue: %+v", issues)
	}
}

func TestIntegrationBDJSONEnvelopeEnvWithRealBD(t *testing.T) {
	t.Setenv("BD_JSON_ENVELOPE", "1")
	repo, bdPath, client := setupIntegrationRepo(t)
	id := strings.TrimSpace(runIntegrationCommand(t, repo, bdPath,
		"create", "Envelope env issue",
		"--labels", "sdk",
		"--priority", "1",
		"--silent",
	))

	ready, err := client.Ready(context.Background(), WithAll(), WithLabel("sdk"))
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if !slices.ContainsFunc(ready, func(issue Issue) bool { return issue.ID == id }) {
		t.Fatalf("Ready with BD_JSON_ENVELOPE=1 did not include %s: %+v", id, ready)
	}

	shown, err := client.Show(context.Background(), id)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if shown.ID != id || shown.Title != "Envelope env issue" {
		t.Fatalf("Show with BD_JSON_ENVELOPE=1 returned unexpected issue: %+v", shown)
	}
}

func TestIntegrationRealBDErrorClassification(t *testing.T) {
	repo, bdPath, client := setupIntegrationRepo(t)
	id := strings.TrimSpace(runIntegrationCommand(t, repo, bdPath,
		"create", "Error classification issue",
		"--priority", "1",
		"--silent",
	))

	_, err := client.Show(context.Background(), "bgotest-does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Show missing error = %v, want ErrNotFound", err)
	}

	err = client.Transition(context.Background(), id, "not_a_status")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Transition invalid status error = %v, want ErrValidation", err)
	}
}

func TestIntegrationFullMethodSetWithRealBD(t *testing.T) {
	repo, bdPath, client := setupIntegrationRepo(t)

	blockerID := strings.TrimSpace(runIntegrationCommand(t, repo, bdPath,
		"create", "Blocker issue",
		"--labels", "sdk",
		"--priority", "1",
		"--silent",
	))
	dependentID := strings.TrimSpace(runIntegrationCommand(t, repo, bdPath,
		"create", "Dependent issue",
		"--labels", "sdk",
		"--deps", "blocks:"+blockerID,
		"--silent",
	))

	ready, err := client.Ready(context.Background(), WithAll(), WithLabel("sdk"))
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if ready == nil {
		t.Fatal("Ready returned nil slice")
	}
	if !slices.ContainsFunc(ready, func(issue Issue) bool { return issue.ID == dependentID }) {
		t.Fatalf("Ready did not include unblocked issue %s: %+v", dependentID, ready)
	}
	if slices.ContainsFunc(ready, func(issue Issue) bool { return issue.ID == blockerID }) {
		t.Fatalf("Ready included blocked issue %s: %+v", blockerID, ready)
	}

	listed, err := client.List(context.Background(), WithAll(), WithLabel("sdk"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if listed == nil {
		t.Fatal("List returned nil slice")
	}
	blocker, ok := findIssue(listed, blockerID)
	if !ok {
		t.Fatalf("List did not include blocker %s: %+v", blockerID, listed)
	}
	if !slices.ContainsFunc(blocker.Dependencies, func(dep Dependency) bool {
		return dep.DependsOnID == dependentID && dep.Type == "blocks"
	}) {
		t.Fatalf("blocker dependencies did not include dependent %s: %+v", dependentID, blocker.Dependencies)
	}

	shown, err := client.Show(context.Background(), dependentID)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if shown.ID != dependentID || shown.Title != "Dependent issue" || shown.Status != "open" {
		t.Fatalf("Show returned unexpected issue: %+v", shown)
	}

	if err := client.Comment(context.Background(), dependentID, "integration note"); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	commented, err := client.Show(context.Background(), dependentID)
	if err != nil {
		t.Fatalf("Show after Comment: %v", err)
	}
	if !strings.Contains(string(commented.RawJSON), "integration note") {
		t.Fatalf("Show RawJSON after Comment did not include note: %s", commented.RawJSON)
	}

	if err := client.Transition(context.Background(), dependentID, "in_progress"); err != nil {
		t.Fatalf("Transition: %v", err)
	}
	transitioned, err := client.Show(context.Background(), dependentID)
	if err != nil {
		t.Fatalf("Show after Transition: %v", err)
	}
	if transitioned.Status != "in_progress" {
		t.Fatalf("status after Transition = %q, want in_progress", transitioned.Status)
	}

	if err := client.Close(context.Background(), dependentID, "integration complete"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	closed, err := client.Show(context.Background(), dependentID)
	if err != nil {
		t.Fatalf("Show after Close: %v", err)
	}
	if closed.Status != "closed" {
		t.Fatalf("status after Close = %q, want closed", closed.Status)
	}
}

func setupIntegrationRepo(t *testing.T) (repo string, bdPath string, client *Client) {
	t.Helper()
	bdPath = requireIntegrationBinary(t, "bd")
	gitPath := requireIntegrationBinary(t, "git")

	repo = t.TempDir()
	runIntegrationCommand(t, repo, gitPath, "init")
	runIntegrationCommand(t, repo, bdPath,
		"init",
		"--non-interactive",
		"--skip-agents",
		"--skip-hooks",
		"--prefix", "bgotest",
		"--database", "bgotest",
	)

	var err error
	client, err = NewClient(
		WithBinary(bdPath),
		WithDataDir(filepath.Join(repo, ".beads")),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return repo, bdPath, client
}

func findIssue(issues []Issue, id string) (Issue, bool) {
	for _, issue := range issues {
		if issue.ID == id {
			return issue, true
		}
	}
	return Issue{}, false
}

func requireIntegrationBinary(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not found on PATH", name)
	}
	return path
}

func runIntegrationCommand(t *testing.T, dir, binary string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", binary, strings.Join(args, " "), err, out)
	}
	return string(out)
}
