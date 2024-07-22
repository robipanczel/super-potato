/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
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

// TODO: The following variables should be coming from a configuration file but also should be possible changed from commandline
var prefixSeparator string
var rootWorkDirPath string
var rootBackupPath string

func init() {
	cloneCmd.Flags().StringVarP(&dashboard, "dashboard", "d", "", "URL of the dashboard from which to clone from (required)")
	cloneCmd.MarkFlagRequired("dashboard")
	cloneCmd.Flags().StringVarP(&oldPrefix, "oldPrefix", "r", "", "Prefix of the current dashboard and its saved searches")
	cloneCmd.Flags().StringVarP(&newPrefix, "newPrefix", "f", "", "ID of the feature that the clone dashboard is for")

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

	savedSearchIds := make([]string, 0, 10)
	savedSearchIds = append(savedSearchIds, dashboard)
	processedSavedSearchIds := 0

	for len(savedSearchIds) != processedSavedSearchIds {
		savedSearchId := savedSearchIds[processedSavedSearchIds]
		processedSavedSearchIds = processedSavedSearchIds + 1

		splunkQuery, err := ReadXml(filepath.Join(".", "cmd", "tests"), savedSearchId)
		if err != nil {
			slog.Error("failed to load saved search",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

		err = WriteXml(createdBackupPath, savedSearchId, []byte(splunkQuery))
		if err != nil {
			slog.Error("failed to backup saved search",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

		savedSearchIdsFound := FindAllSavedSearchIds(splunkQuery, dashboard, savedSearchId)
		slog.Info("saved search found",
			"dashboard", dashboard,
			"savedSearchId", savedSearchId,
			"linked saved searches", savedSearchIdsFound,
		)
		savedSearchIds = append(savedSearchIds, savedSearchIdsFound...)

		newSavedSearchId := ReplaceOldPrefixWithNewPrefix(savedSearchId, oldPrefix, newPrefix, prefixSeparator)
		newSplunkQuery := ReplaceOldPrefixWithNewPrefixInSplunkQuery(splunkQuery, oldPrefix, newPrefix, prefixSeparator, savedSearchIdsFound)

		// TODO: Make this into an opt in feature for validation and testing
		err = WriteXml(rootWorkDirPath, newSavedSearchId, []byte(newSplunkQuery))
		if err != nil {
			slog.Error("failed to upload cloned xml",
				"dashboard", dashboard,
				"savedSearchId", savedSearchId,
				"error", err,
			)
			panic(err)
		}

		// TODO: Upload the newSavedSearchId and newSplunkQuery to Splunk
	}
	slog.Info("clone command done", "dashboard", dashboard, "newPrefix", newPrefix, "generated saved searches", savedSearchIds)
}

func FindAllSavedSearchIds(splunkQuery string, dashboard string, savedSearchId string) (savedSearchIds []string) {
	re := regexp.MustCompile(`savedsearch\s([a-z1-9]|\_|\-)*[^\s]`)
	foundSavedSearchCmds := re.FindAllString(splunkQuery, -1)
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

func ReplaceOldPrefixWithNewPrefixInSplunkQuery(splunkQuery, oldPrefix, newPrefix, separator string, savedSearchIds []string) string {
	for _, savedSearchId := range savedSearchIds {
		withNewPrefix := ReplaceOldPrefixWithNewPrefix(savedSearchId, oldPrefix, newPrefix, separator)
		splunkQuery = strings.ReplaceAll(splunkQuery, savedSearchId, withNewPrefix)
	}
	return splunkQuery
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

func WriteXml(filePath string, name string, splunkQuery []byte) error {
	err := os.WriteFile(filepath.Join(filePath, name), splunkQuery, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func GetRequest(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error making GET request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var response struct {
		LongString string `json:"long_string"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return response.LongString, nil
}

func PostRequest(url string, longString string) error {
	payload := struct {
		LongString string `json:"long_string"`
	}{
		LongString: longString,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling JSON payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error making POST request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	fmt.Printf("Response: %s\n", string(body))
	return nil
}
