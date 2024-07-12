/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone dashboard and create copies with featureId prefix",
	Long: `Download dashboard based on the given URL
	and create copies of it with the featureId prefix in the name
	Ideally the dashboard should be the stating dashboard.`,
	Run: runClone,
}

var dashboard string
var reportPrefix string
var featureId string
var separator string

func init() {
	cloneCmd.Flags().StringVarP(&dashboard, "dashboard", "d", "", "URL of the dashboard from which to clone from (required)")
	cloneCmd.Flags().StringVarP(&reportPrefix, "reportPrefix", "r", "", "Prefix of the dashboard and its reports")
	cloneCmd.Flags().StringVarP(&featureId, "featureId", "f", "", "ID of the feature that the clone dashboard is for")
	cloneCmd.MarkFlagRequired("dashboard")

	separator = "_"

	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) {
	slog.Info("clone command initiated", "dashboard", dashboard, "featureId", featureId)

	reportNames := make([]string, 0, 10)
	reportNames = append(reportNames, dashboard)
	processedReportCounter := 0

	for len(reportNames) != processedReportCounter {
		currentReportName := reportNames[processedReportCounter]
		processedReportCounter = processedReportCounter + 1

		xmlString, err := ReadXml(currentReportName)
		if err != nil {
			slog.Error("failed to load xml",
				"dashboard", dashboard,
				"name", currentReportName,
				"error", err,
			)
			panic(err)
		}

		foundReportNames := CollectLinkedReportsFromXmlString(xmlString, dashboard, currentReportName, reportPrefix)
		slog.Info("reports found",
			"dashboard", dashboard,
			"current", currentReportName,
			"linked reports", foundReportNames,
		)
		reportNames = append(reportNames, foundReportNames...)

		withNewPrefix := ReplaceOldPrefixWithNewPrefix(currentReportName, reportPrefix, featureId, separator)
		xmlString = ReplaceOldPrefixWithNewPrefixInXml(xmlString, reportPrefix, featureId, separator, foundReportNames)

		err = WriteXml(withNewPrefix, xmlString)
		if err != nil {
			slog.Error("failed to upload cloned xml",
				"dashboard", dashboard,
				"current", currentReportName,
				"error", err,
			)
			panic(err)
		}

	}
	slog.Info("clone command done", "dashboard", dashboard, "featureId", featureId, "generated reports", reportNames)
}

func CollectLinkedReportsFromXmlString(xmlString string, dashboard string, currentReport string, reportPrefix string) (reportNames []string) {
	re := regexp.MustCompile(`savedsearch\s([a-z1-9]|\_|\-)*[^\s]`)
	linkedReportNames := re.FindAllString(xmlString, -1)
	reportNames = make([]string, 0)
	for _, linkedReportName := range linkedReportNames {
		_, after, found := strings.Cut(linkedReportName, "savedsearch")
		if !found {
			slog.Error("failed to parse report name",
				"dashboard", dashboard,
				"current", currentReport,
			)
		}

		trimmedLinkedReportName := strings.TrimSpace(after)

		if slices.Contains(reportNames, trimmedLinkedReportName) {
			continue
		}

		reportNames = append(reportNames, trimmedLinkedReportName)
	}

	return reportNames
}

func ReplaceOldPrefixWithNewPrefixInXml(xmlString, oldPrefix, newPrefix, separator string, reportNames []string) string {
	for _, foundReportName := range reportNames {
		withNewPrefix := ReplaceOldPrefixWithNewPrefix(foundReportName, oldPrefix, newPrefix, separator)
		xmlString = strings.ReplaceAll(xmlString, foundReportName, withNewPrefix)
	}
	return xmlString
}

func ReplaceOldPrefixWithNewPrefix(foundReportName, oldPrefix, newPrefix, separator string) string {
	withoutPrefix := strings.TrimPrefix(foundReportName, oldPrefix)
	withoutPrefixAndSeparator := strings.TrimPrefix(withoutPrefix, separator)
	withNewPrefix := fmt.Sprintf("%s%s%s", newPrefix, separator, withoutPrefixAndSeparator)
	return withNewPrefix
}

func ReadXml(name string) (xmlString string, err error) {
	data, err := os.ReadFile(filepath.Join(".", "tests", name))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func WriteXml(name string, xml string) error {
	err := os.WriteFile(filepath.Join("temp", name), []byte(xml), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
