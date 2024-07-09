package cmd

import "testing"

func TestReadXml_ExistingXml(t *testing.T) {
	data, err := ReadXml("splunk_example_1.xml")

	if err != nil {
		t.Errorf("should have read the file, but found: %s", err)
	}

	if len(data) <= 0 {
		t.Errorf("parsed string xml should be longer then 0 length")
	}
}
func TestReadXml_NotExistingXml(t *testing.T) {
	_, err := ReadXml("foo.xml")
	if err == nil {
		t.Errorf("should have thrown an error if file does not exits")
	}
}
