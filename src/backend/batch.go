package main

import (
	"react-golang/src/backend/config"
	"react-golang/src/backend/constants"
	"react-golang/src/backend/service"

	"github.com/robfig/cron/v3"
	"github.com/sarulabs/di"
)

type Batch struct {
	services *service.Service
	configs  *config.Config
	cron     *cron.Cron
}

func RunBatch(ioc di.Container) {
	batch := &Batch{
		services: ioc.Get(constants.CONTAINER_SERVICE).(*service.Service),
		configs:  config.GetInstance(),
		cron:     cron.New(),
	}

	batch.configs.WatchChanges(batch.restartCron)
	batch.startCron()
}

func (b *Batch) startCron() {
	b.cron.AddFunc(b.configs.CronSchedule, func() {
		b.services.Backup.Backup()
	})

	go func() {
		b.cron.Start()
		defer b.cron.Stop()
		select {}
	}()
}

func (b *Batch) restartCron() {
	b.cron.Stop()
	b.cron = cron.New()
	b.startCron()
}
