package cmd

import (
	"fmt"
	"os"
	"testing"
)

func TestReadXml_ExistingXml(t *testing.T) {
	data, err := ReadXml("./tests", "splunk_example_1")

	if err != nil {
		t.Errorf("should have read the file, but found: %s", err)
	}

	if len(data) <= 0 {
		t.Errorf("parsed string xml should be longer then 0 length")
	}
}

func TestReadXml_NotExistingXml(t *testing.T) {
	_, err := ReadXml("./tests", "barfoo")
	if err == nil {
		t.Errorf("should have thrown an error if file does not exits")
	}
}

func A() {
	path, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(path)
}

func TestReplaceOldPrefixWithNewPrefix(t *testing.T) {
	separator := "_"

	testCases := []struct {
		foundReportName string
		oldPrefix       string
		newPrefix       string
		expectedResult  string
	}{
		{
			foundReportName: "dashboard_name",
			oldPrefix:       "prod",
			newPrefix:       "dev",
			expectedResult:  "dev_dashboard_name",
		},
		{
			foundReportName: "prod_dashboard_name",
			oldPrefix:       "prod_",
			newPrefix:       "dev_",
			expectedResult:  "dev__dashboard_name",
		},
		{
			foundReportName: "dev_dashboard_name",
			oldPrefix:       "prod",
			newPrefix:       "dev",
			expectedResult:  "dev_dev_dashboard_name",
		},
	}

	for _, testCase := range testCases {
		result := ReplaceOldPrefixWithNewPrefix(testCase.foundReportName, testCase.oldPrefix, testCase.newPrefix, separator)
		if result != testCase.expectedResult {
			t.Errorf("ReplaceOldPrefixWithNewPrefix(%s, %s, %s) = %s, expected %s",
				testCase.foundReportName, testCase.oldPrefix, testCase.newPrefix, result, testCase.expectedResult)
		}
	}
}

func TestReplaceOldPrefixWithNewPrefixInXml(t *testing.T) {
	separator := "_"
	xmlString := `
<dashboard>
	<panel>
		<savedsearch>prod_dashboard_name</savedsearch>
		<savedsearch>prod_report_1</savedsearch>
		<savedsearch>prodreport_2</savedsearch>
	</panel>
</dashboard>
`
	oldPrefix := "prod"
	newPrefix := "dev"
	reportNames := []string{"prod_dashboard_name", "prod_report_1", "prodreport_2"}
	expectedResult := `
<dashboard>
	<panel>
		<savedsearch>dev_dashboard_name</savedsearch>
		<savedsearch>dev_report_1</savedsearch>
		<savedsearch>dev_report_2</savedsearch>
	</panel>
</dashboard>
`

	result := ReplaceOldPrefixWithNewPrefixInSplunkQuery(xmlString, oldPrefix, newPrefix, separator, reportNames)
	if result != expectedResult {
		t.Errorf("ReplaceOldPrefixWithNewPrefixInXml() = %s, expected %s", result, expectedResult)
	}
}

func TestCreateBackupDir(t *testing.T) {
	rootBackupPath := "../temp"

	createdBackupPath, err := CreateBackupDir(rootBackupPath)
	if err != nil {
		t.Errorf("CreateBackupDir returned an error: %s", err)
	}

	// Verify that the backup directory was created
	backupDirExists, err := exists(createdBackupPath)
	if err != nil {
		t.Errorf("Failed to check if backup directory exists: %s", err)
	}
	if !backupDirExists {
		t.Errorf("Backup directory was not created")
	}

	// Cleanup: Remove the backup directory
	err = os.RemoveAll(createdBackupPath)
	if err != nil {
		t.Errorf("Failed to remove backup directory: %s", err)
	}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
