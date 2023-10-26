package main

import (
	"atlantis-drift-detector/config"
	"atlantis-drift-detector/drift"
	"atlantis-drift-detector/exporter"
	"atlantis-drift-detector/server"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

var driftMutex sync.Mutex
var isDriftRunning bool

func main() {

	log.SetFormatter(&log.JSONFormatter{})
	log.Info("starting drift detector")

	// Get environment variables
	repoAllowlist, ghAppSlug, ghAppId, ghAppKeyFile, ghInstallationId, cronExpression, mergeKubeconfigs := config.InitEnvs()

	// Merge kubeconfigs
	if mergeKubeconfigs == "true" {
		config.CreateKubeconfig()
	}

	err := exporter.UpdateMetricsFromCSV("/csv/data")
	if err != nil {
		log.Warnf("error reading CSV: %s", err)
		return
	}

	// Start web server as a go routine
	go server.Run()

	// Schedule cron job to detect drift
	c := cron.New()
	_, err = c.AddFunc(cronExpression, func() {
		driftMutex.Lock()

		if isDriftRunning {
			driftMutex.Unlock()
			return
		}

		isDriftRunning = true
		driftMutex.Unlock()

		log.Debug("running DetectDrift function")
		drift.DetectDrift(strings.Split(repoAllowlist, ","), ghAppSlug, ghAppId, ghAppKeyFile, ghInstallationId)

		driftMutex.Lock()
		isDriftRunning = false
		driftMutex.Unlock()

		err := exporter.UpdateMetricsFromCSV("/csv/data")
		if err != nil {
			log.Warnf("error reading CSV: %s", err)
			return
		}
		time.Sleep(1 * time.Hour)
	})
	if err != nil {
		log.Warnf("Error scheduling cron job: %s", err)
		return
	}
	c.Start()

	select {}
}
