package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/kata/internal/db"
)

func TestReadyIssues_FiltersOutClosed(t *testing.T) {
	d, ctx, p := setupTestProject(t)
	open := makeIssue(t, ctx, d, p.ID, "open", "tester")
	closed := makeIssue(t, ctx, d, p.ID, "closed", "tester")
	_, _, _, err := d.CloseIssue(ctx, closed.ID, "done", "tester", "", nil)
	require.NoError(t, err)

	got := readyNumbers(t, ctx, d, p.ID)
	assert.Contains(t, got, open.ShortID)
	assert.NotContains(t, got, closed.ShortID)
}

func TestReadyIssues_ExcludesIssuesBlockedByOpenBlocker(t *testing.T) {
	d, ctx, p := setupTestProject(t)
	blocker := makeIssue(t, ctx, d, p.ID, "blocker", "tester")
	blocked := makeIssue(t, ctx, d, p.ID, "blocked", "tester")
	standalone := makeIssue(t, ctx, d, p.ID, "standalone", "tester")
	makeLink(ctx, t, d, p.ID, blocker.ID, blocked.ID, "blocks")

	got := readyNumbers(t, ctx, d, p.ID)
	assert.Contains(t, got, blocker.ShortID, "blocker is ready (not blocked itself)")
	assert.Contains(t, got, standalone.ShortID, "standalone is ready")
	assert.NotContains(t, got, blocked.ShortID, "blocked is not ready while blocker is open")
}

func TestReadyIssues_ClosedBlockerUnblocksDownstream(t *testing.T) {
	d, ctx, p := setupTestProject(t)
	blocker := makeIssue(t, ctx, d, p.ID, "blocker", "tester")
	blocked := makeIssue(t, ctx, d, p.ID, "blocked", "tester")
	makeLink(ctx, t, d, p.ID, blocker.ID, blocked.ID, "blocks")
	_, _, _, err := d.CloseIssue(ctx, blocker.ID, "done", "tester", "", nil)
	require.NoError(t, err)

	got := readyNumbers(t, ctx, d, p.ID)
	assert.Contains(t, got, blocked.ShortID, "blocked is ready once blocker closes")
}

// readyNumbers fetches ready issues for projectID and returns their short IDs.
//
//nolint:revive // test helper: t *testing.T conventionally precedes ctx.
func readyNumbers(t *testing.T, ctx context.Context, d *db.DB, projectID int64) []string {
	t.Helper()
	rows, err := d.ReadyIssues(ctx, projectID, 0)
	require.NoError(t, err)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ShortID)
	}
	return out
}

func TestReadyIssuesGlobal_ReturnsIssuesAcrossProjects(t *testing.T) {
	d, ctx, p1 := setupTestProject(t)
	p2, err := d.CreateProject(ctx, "second-project")
	require.NoError(t, err)

	a := makeIssue(t, ctx, d, p1.ID, "in p1", "tester")
	b := makeIssue(t, ctx, d, p2.ID, "in p2", "tester")

	rows, err := d.ReadyIssuesGlobal(ctx, 0)
	require.NoError(t, err)

	got := map[string]string{}
	for _, r := range rows {
		got[r.Issue.ShortID] = r.ProjectName
	}
	assert.Equal(t, p1.Name, got[a.ShortID])
	assert.Equal(t, "second-project", got[b.ShortID])
}

func TestReadyIssuesGlobal_ExcludesArchivedProjects(t *testing.T) {
	d, ctx, p1 := setupTestProject(t)
	p2, err := d.CreateProject(ctx, "to-archive")
	require.NoError(t, err)

	keep := makeIssue(t, ctx, d, p1.ID, "keep", "tester")
	hidden := makeIssue(t, ctx, d, p2.ID, "hidden", "tester")

	_, _, err = d.RemoveProject(ctx, db.RemoveProjectParams{
		ProjectID: p2.ID,
		Actor:     "tester",
		Force:     true,
	})
	require.NoError(t, err)

	rows, err := d.ReadyIssuesGlobal(ctx, 0)
	require.NoError(t, err)

	got := map[string]bool{}
	for _, r := range rows {
		got[r.Issue.ShortID] = true
	}
	assert.True(t, got[keep.ShortID], "issue in active project is returned")
	assert.False(t, got[hidden.ShortID], "issue in archived project is excluded")
}

func TestReadyIssuesGlobal_ExcludesBlockedIssues(t *testing.T) {
	d, ctx, p := setupTestProject(t)
	blocker := makeIssue(t, ctx, d, p.ID, "blocker", "tester")
	blocked := makeIssue(t, ctx, d, p.ID, "blocked", "tester")
	makeLink(ctx, t, d, p.ID, blocker.ID, blocked.ID, "blocks")

	rows, err := d.ReadyIssuesGlobal(ctx, 0)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, r := range rows {
		got[r.Issue.ShortID] = true
	}
	assert.True(t, got[blocker.ShortID])
	assert.False(t, got[blocked.ShortID])
}
