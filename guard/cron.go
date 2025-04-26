package guard

import "github.com/robfig/cron/v3"

func CronScheduling(schedule string, containers []string) {
	c := cron.New()
	_, _ = c.AddFunc(schedule, func() {
		Pipe(containers)
	})
	c.Start()
}
