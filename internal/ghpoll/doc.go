// Package ghpoll follows the pull requests of tasks through the gh CLI the user is already signed
// in to, and files what it learns through internal/domain: the state on the task, and a report
// from GitHub when something needs attention. The app runs it; withdo mcp does not.
//
// The package has no wording of its own beyond the English summaries of the reports it files,
// which are stored in the database like any agent's.
package ghpoll
