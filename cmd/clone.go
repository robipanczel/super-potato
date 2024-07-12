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
	Short: "Clone and rename dashboard and saved searches, replacing old prefix with new prefix value",
	Long: `Download dashboard based on the given URL
	and create copies of it with the newPrefix prefix in the name
	Ideally the dashboard should be the stating dashboard.`,
	Run: runClone,
}

var dashboard string
var oldPrefix string
var newPrefix string
var prefixSeparator string

func init() {
	cloneCmd.Flags().StringVarP(&dashboard, "dashboard", "d", "", "URL of the dashboard from which to clone from (required)")
	cloneCmd.Flags().StringVarP(&oldPrefix, "oldPrefix", "r", "", "Prefix of the current dashboard and its saved searches")
	cloneCmd.Flags().StringVarP(&newPrefix, "newPrefix", "f", "", "ID of the feature that the clone dashboard is for")
	cloneCmd.MarkFlagRequired("dashboard")

	prefixSeparator = "_"

	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) {
	slog.Info("clone command initiated", "dashboard", dashboard, "newPrefix", newPrefix)

	savedSearches := make([]string, 0, 10)
	savedSearches = append(savedSearches, dashboard)
	processedSavedSearches := 0

	for len(savedSearches) != processedSavedSearches {
		savedSearchId := savedSearches[processedSavedSearches]
		processedSavedSearches = processedSavedSearches + 1

		xmlString, err := ReadXml(savedSearchId)
		if err != nil {
			slog.Error("failed to load xml",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

		foundReportNames := FindAllSavedSearchIds(xmlString, dashboard, savedSearchId)
		slog.Info("reports found",
			"dashboard", dashboard,
			"savedSearchId", savedSearchId,
			"linked reports", foundReportNames,
		)
		savedSearches = append(savedSearches, foundReportNames...)

		withNewPrefix := ReplaceOldPrefixWithNewPrefix(savedSearchId, oldPrefix, newPrefix, prefixSeparator)
		xmlString = ReplaceOldPrefixWithNewPrefixInXml(xmlString, oldPrefix, newPrefix, prefixSeparator, foundReportNames)

		err = WriteXml(withNewPrefix, xmlString)
		if err != nil {
			slog.Error("failed to upload cloned xml",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

	}
	slog.Info("clone command done", "dashboard", dashboard, "newPrefix", newPrefix, "generated reports", savedSearches)
}

func FindAllSavedSearchIds(xmlString string, dashboard string, savedSearchId string) (savedSearchIds []string) {
	re := regexp.MustCompile(`savedsearch\s([a-z1-9]|\_|\-)*[^\s]`)
	foundSavedSearchCmds := re.FindAllString(xmlString, -1)
	savedSearchIds = make([]string, 0)
	for _, foundSavedSearchCmd := range foundSavedSearchCmds {
		_, after, found := strings.Cut(foundSavedSearchCmd, "savedsearch")
		if !found {
			slog.Error("failed to parse report name",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
			)
		}

		cleanedSavedSearchId := strings.TrimSpace(after)

		if slices.Contains(savedSearchIds, cleanedSavedSearchId) {
			continue
		}

		savedSearchIds = append(savedSearchIds, cleanedSavedSearchId)
	}

	return savedSearchIds
}

func ReplaceOldPrefixWithNewPrefixInXml(xmlString, oldPrefix, newPrefix, separator string, savedSearchIds []string) string {
	for _, savedSearchId := range savedSearchIds {
		withNewPrefix := ReplaceOldPrefixWithNewPrefix(savedSearchId, oldPrefix, newPrefix, separator)
		xmlString = strings.ReplaceAll(xmlString, savedSearchId, withNewPrefix)
	}
	return xmlString
}

func ReplaceOldPrefixWithNewPrefix(savedSearchId, oldPrefix, newPrefix, separator string) string {
	withoutPrefix := strings.TrimPrefix(savedSearchId, oldPrefix)
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
