// Package agent is the built-in assistant behind the Now screen's ask field. It runs an Eino
// ChatModelAgent whose tools go through internal/domain, keeps the conversation in memory, and
// records what each change did so Undo can revert exactly that.
package agent
