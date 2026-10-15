package domain

import (
	"context"
	"fmt"
)

// AgentActivity is one agent's reports: when the latest came, and how many came since a moment.
type AgentActivity struct {
	Agent string `json:"agent"`
	// Last is the time of its latest report, as FormatTime writes it.
	Last string `json:"last"`
	// Since counts its reports at or after the moment AgentActivity was asked about.
	Since int `json:"since"`
}

// AgentActivity lists every agent that ever reported, the most recently active first. The poller
// that files pull request reports under GitHubAgent is not an agent a person connected, so it is
// left out. since is a timestamp as FormatTime writes it.
func (c *Core) AgentActivity(ctx context.Context, since string) ([]AgentActivity, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT agent, MAX(created_at), SUM(created_at >= ?) FROM reports WHERE agent <> ? AND agent <> '' "+
			"GROUP BY agent ORDER BY MAX(created_at) DESC, agent", since, GitHubAgent)
	if err != nil {
		return nil, fmt.Errorf("agent activity: %w", err)
	}
	return collect(rows, func(s scanner) (AgentActivity, error) {
		var a AgentActivity
		err := s.Scan(&a.Agent, &a.Last, &a.Since)
		return a, err
	})
}
