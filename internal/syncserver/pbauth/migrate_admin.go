package pbauth

import (
	"github.com/pocketbase/pocketbase/core"
)

// superuserSignInsPerMinute is how many superuser password sign-ins one address may try in a
// minute. The admin UI (/_/) is public since 2026-10-22, so its password is guessed from anywhere.
const superuserSignInsPerMinute = 5

func init() {
	core.AppMigrations.Register(limitSuperuserSignIn, nil, "1790000100_adhd_superuser_rate_limit.go")
}

// limitSuperuserSignIn adds the superusers' sign-in to the rate limits, next to the users' one.
func limitSuperuserSignIn(txApp core.App) error {
	settings := txApp.Settings()
	label := core.CollectionNameSuperusers + ":authWithPassword"
	for _, rule := range settings.RateLimits.Rules {
		if rule.Label == label {
			return nil
		}
	}
	settings.RateLimits.Enabled = true
	settings.RateLimits.Rules = append(settings.RateLimits.Rules,
		core.RateLimitRule{Label: label, Duration: 60, MaxRequests: superuserSignInsPerMinute})
	return txApp.Save(settings)
}
