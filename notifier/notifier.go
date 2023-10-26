package notifier

import (
	"atlantis-drift-detector/config"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

func Notify(repo string, driftedProjects []string, errorProjects []string, freshProjects []string) {

	filePath, err := buildReportCSV(repo, driftedProjects, errorProjects, freshProjects)
	if err != nil {
		log.Warn("could not build report")
	}

	slackChannel, slackToken := config.InitSlackEnvs()
	err = sendReportToSlack(filePath, slackChannel, slackToken, repo, driftedProjects, errorProjects, freshProjects)
	if err != nil {
		log.Warnf("error sending slack message: %s", err)
	} else {
		log.Debug("report message was sent to slack")
	}
}

func buildReportCSV(repoFolder string, driftedProjects []string, errorProjects []string, freshProjects []string) (string, error) {

	log.Debug("building report csv")
	filename := "csv/data/" + repoFolder + "_report.csv"

	// Create a CSV file.
	file, err := os.Create(filename)
	if err != nil {
		log.Warnf("Could not create file: %s", err)
		return filename, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Write the data to the CSV file
	for _, driftedProject := range driftedProjects {
		row := []string{driftedProject, "drifted"}
		err := writer.Write(row)
		if err != nil {
			log.Warnf("error writing data to CSV: %s", err)
			return filename, err
		}
	}

	for _, errorProject := range errorProjects {
		row := []string{errorProject, "error"}
		err := writer.Write(row)
		if err != nil {
			log.Warnf("error writing data to CSV: %s", err)
			return filename, err
		}
	}

	for _, freshProject := range freshProjects {
		row := []string{freshProject, "No changes"}
		err := writer.Write(row)
		if err != nil {
			log.Warnf("error writing data to CSV: %s", err)
			return filename, err
		}
	}

	// Flush any buffered data to the underlying writer
	writer.Flush()
	log.Debug("CSV data written successfully")

	return filename, nil
}

func sendReportToSlack(filePath, slackChannel, slackToken, repo string, driftedProjects, errorProjects, freshProjects []string) error {

	if slackChannel == "" || slackToken == "" {
		err := fmt.Errorf("slack channel or token not set")
		log.Warnf("slack channel or token not set")
		return err
	}

	api := slack.New(slackToken)

	message := fmt.Sprintf("GM team!\nDrift report for `%s`\n:sos: Errors: %d\n:warning: Drifted: %d\n:white_check_mark: No changes: %d",
		repo,
		len(errorProjects),
		len(driftedProjects),
		len(freshProjects),
	)
	_, _, err := api.PostMessage(slackChannel, slack.MsgOptionText(message, false))
	if err != nil {
		return err
	}

	return nil
}
