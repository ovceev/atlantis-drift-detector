package exporter

import (
	"encoding/csv"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var errorGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "drift_detector_error_count",
		Help: "Number of error occurrences in drift detector.",
	},
)

var driftedGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "drift_detector_drifted_count",
		Help: "Number of drifted occurrences in drift detector.",
	},
)

var noChangesGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "drift_detector_no_changes_count",
		Help: "Number of no changes occurrences in drift detector.",
	},
)

func init() {
	prometheus.MustRegister(errorGauge)
	prometheus.MustRegister(driftedGauge)
	prometheus.MustRegister(noChangesGauge)
}

func UpdateMetricsFromCSV(folderPath string) error {
	// Initialize counters
	var errorCount, driftedCount, noChangesCount float64

	// Read all files in the folder
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".csv") {
			csvPath := filepath.Join(folderPath, file.Name())

			errCount, driftCount, noChangeCount, err := processCSV(csvPath)
			if err != nil {
				return err
			}

			// Aggregate counts
			errorCount += errCount
			driftedCount += driftCount
			noChangesCount += noChangeCount
		}
	}

	// Set the gauge values
	errorGauge.Set(errorCount)
	driftedGauge.Set(driftedCount)
	noChangesGauge.Set(noChangesCount)

	return nil
}

func processCSV(filename string) (errorCount, driftedCount, noChangesCount float64, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, 0, 0, err
	}

	for _, record := range records {
		switch record[1] {
		case "error":
			errorCount++
		case "drifted":
			driftedCount++
		case "No changes":
			noChangesCount++
		}
	}

	return errorCount, driftedCount, noChangesCount, nil
}
