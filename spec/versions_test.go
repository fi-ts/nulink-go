package spec

import (
	"encoding/json"
	"os"
	"testing"
)

// TestVersionsAgree ensures the OpenAPI spec and the validations file
// describe the same nulink version. Both files are unversioned by name, so
// the version inside the files is the only source of truth.
func TestVersionsAgree(t *testing.T) {
	var specFile struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	var validationsFile struct {
		APIVersion string `json:"api_version"`
	}

	readJSON(t, "openapi.json", &specFile)
	readJSON(t, "validations.json", &validationsFile)

	if specFile.Info.Version == "" {
		t.Fatal("openapi.json: info.version is empty")
	}
	if validationsFile.APIVersion == "" {
		t.Fatal("validations.json: api_version is empty")
	}
	if specFile.Info.Version != validationsFile.APIVersion {
		t.Errorf("version mismatch: openapi.json info.version=%q, validations.json api_version=%q",
			specFile.Info.Version, validationsFile.APIVersion)
	}
}

func readJSON(t *testing.T, name string, v any) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
}
