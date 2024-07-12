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
	"time"

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
var rootWorkDirPath string
var rootBackupPath string

func init() {
	cloneCmd.Flags().StringVarP(&dashboard, "dashboard", "d", "", "URL of the dashboard from which to clone from (required)")
	cloneCmd.Flags().StringVarP(&oldPrefix, "oldPrefix", "r", "", "Prefix of the current dashboard and its saved searches")
	cloneCmd.Flags().StringVarP(&newPrefix, "newPrefix", "f", "", "ID of the feature that the clone dashboard is for")
	cloneCmd.MarkFlagRequired("dashboard")

	prefixSeparator = "_"
	rootWorkDirPath = filepath.Join(".", "temp")
	rootBackupPath = filepath.Join(".", "temp")

	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) {
	slog.Info("clone command initiated", "dashboard", dashboard, "newPrefix", newPrefix)

	createdBackupPath, err := CreateBackupDir(rootBackupPath)
	if err != nil {
		slog.Error("failed to create backup directory",
			"dashboard", dashboard,
			"error", err,
		)
		panic(err)
	}

	savedSearches := make([]string, 0, 10)
	savedSearches = append(savedSearches, dashboard)
	processedSavedSearches := 0

	for len(savedSearches) != processedSavedSearches {
		savedSearchId := savedSearches[processedSavedSearches]
		processedSavedSearches = processedSavedSearches + 1

		xmlString, err := ReadXml(filepath.Join(".", "cmd", "tests"), savedSearchId)
		if err != nil {
			slog.Error("failed to load saved search",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

		err = WriteXml(createdBackupPath, savedSearchId, []byte(xmlString))
		if err != nil {
			slog.Error("failed to backup saved search",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

		foundReportNames := FindAllSavedSearchIds(xmlString, dashboard, savedSearchId)
		slog.Info("saved search found",
			"dashboard", dashboard,
			"savedSearchId", savedSearchId,
			"linked saved searches", foundReportNames,
		)
		savedSearches = append(savedSearches, foundReportNames...)

		withNewPrefix := ReplaceOldPrefixWithNewPrefix(savedSearchId, oldPrefix, newPrefix, prefixSeparator)
		xmlString = ReplaceOldPrefixWithNewPrefixInXml(xmlString, oldPrefix, newPrefix, prefixSeparator, foundReportNames)

		//TODO: Make this into an opt in feature
		// This is for testing/validation
		err = WriteXml(rootWorkDirPath, withNewPrefix, []byte(xmlString))
		if err != nil {
			slog.Error("failed to upload cloned xml",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

	}
	slog.Info("clone command done", "dashboard", dashboard, "newPrefix", newPrefix, "generated saved searches", savedSearches)
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

func CreateBackupDir(rootBackupPath string) (createdBackupPath string, err error) {
	currentTime := time.Now()
	createdBackupPath = filepath.Join(
		rootBackupPath,
		fmt.Sprintf("%d-%d-%d-%d-%d-%d",
			currentTime.Year(),
			currentTime.Month(),
			currentTime.Day(),
			currentTime.Hour(),
			currentTime.Hour(),
			currentTime.Second(),
		),
	)

	err = os.Mkdir(createdBackupPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	return createdBackupPath, nil
}

func ReadXml(filePath string, name string) (xmlString string, err error) {
	data, err := os.ReadFile(filepath.Join(filePath, name))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func WriteXml(filePath string, name string, data []byte) error {
	err := os.WriteFile(filepath.Join(filePath, name), data, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
