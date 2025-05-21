# todoist-rss

Takes Todoist tasks with the `@rss` label (or configurable with the `LABEL` env
var) and turns them into an RSS feed.

Completes the task after being scraped, with the assumption that your feed
reader stores the RSS items. The first link in the content of a task will be
extracted. Tested against Miniflux.

Make sure to provide a Todoist access token with the `TOKEN` environment
variable.
