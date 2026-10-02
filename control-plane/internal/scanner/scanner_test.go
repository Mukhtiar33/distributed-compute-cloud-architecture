package scanner

import (
	"archive/zip"
	"bytes"
	"testing"
)

func createZip(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	w.Close()
	return buf.Bytes()
}

func TestScanner_CleanPayload(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"main.py":          "print('hello world')",
		"requirements.txt": "flask==2.3.0\nrequests==2.28.0",
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected clean payload to pass, got reasons: %v", result.Reasons)
	}
}

func TestScanner_VulnerableDependency(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"main.py":          "print('hello')",
		"requirements.txt": "requests==2.25.0",
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if result.Passed {
		t.Error("expected vulnerable dependency to be flagged")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected at least one reason for rejection")
	}
}

func TestScanner_DangerousPattern(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"main.py": `import os
os.system('rm -rf /')`,
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if result.Passed {
		t.Error("expected dangerous pattern to be flagged")
	}
}

func TestScanner_EntrypointExists(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"main.py": "print('hello')",
	})

	result, err := s.ScanManifest(zipData, "main.py")
	if err != nil {
		t.Fatalf("ScanManifest failed: %v", err)
	}
	if !result.Passed {
		t.Error("expected entrypoint to be found")
	}
}

func TestScanner_EntrypointMissing(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"main.py": "print('hello')",
	})

	result, err := s.ScanManifest(zipData, "app.py")
	if err != nil {
		t.Fatalf("ScanManifest failed: %v", err)
	}
	if result.Passed {
		t.Error("expected missing entrypoint to be flagged")
	}
}

func TestScanner_InvalidZip(t *testing.T) {
	s := NewScanner()
	_, err := s.ScanZip([]byte("not a zip file"))
	if err == nil {
		t.Error("expected error for invalid zip")
	}
}

func TestScanner_EmptyZip(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if !result.Passed {
		t.Error("expected empty zip to pass (no threats found)")
	}
}

func TestScanner_MultipleVulnerabilities(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"requirements.txt": "requests==2.25.0\ndjango==3.2.0\nflask==1.0.0",
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if result.Passed {
		t.Error("expected multiple vulnerabilities to be flagged")
	}
	if len(result.Reasons) < 3 {
		t.Errorf("expected at least 3 reasons, got %d", len(result.Reasons))
	}
}

func TestScanner_PackageJSONVulnerability(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"package.json": `{"dependencies": {"lodash": "4.17.20"}}`,
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if result.Passed {
		t.Error("expected vulnerable lodash to be flagged")
	}
}

func TestScanner_CleanPackageJSON(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"package.json": `{"dependencies": {"lodash": "4.17.21", "express": "4.18.0"}}`,
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected clean package.json to pass, got: %v", result.Reasons)
	}
}

func TestScanner_Base64Obfuscation(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"main.py": `import base64
data = base64.b64decode("aGVsbG8=")`,
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	if result.Passed {
		t.Error("expected base64 obfuscation to be flagged")
	}
}

func TestScanner_GoModVulnerability(t *testing.T) {
	s := NewScanner()
	zipData := createZip(map[string]string{
		"go.mod": "module example.com/test\n\ngo 1.21\n\nrequire github.com/dgrijalva/jwt-go v3.2.0",
	})

	result, err := s.ScanZip(zipData)
	if err != nil {
		t.Fatalf("ScanZip failed: %v", err)
	}
	// This should pass since we don't have this in our vuln DB yet
	if !result.Passed {
		t.Errorf("expected go.mod to pass, got: %v", result.Reasons)
	}
}

func TestVulnerabilityDB_CheckPythonRequirements(t *testing.T) {
	db := NewVulnerabilityDB()
	content := "requests==2.25.0\nflask==2.3.0\ndjango==3.2.0"

	reasons := db.checkPythonRequirements(content)
	if len(reasons) < 2 {
		t.Errorf("expected at least 2 vulnerability reasons, got %d", len(reasons))
	}
}

func TestVulnerabilityDB_CheckPackageJSON(t *testing.T) {
	db := NewVulnerabilityDB()
	content := `{"dependencies": {"lodash": "4.17.20", "express": "4.17.1"}}`

	reasons := db.checkPackageJSON(content)
	if len(reasons) < 2 {
		t.Errorf("expected at least 2 vulnerability reasons, got %d", len(reasons))
	}
}

func TestVulnerabilityDB_CleanDependencies(t *testing.T) {
	db := NewVulnerabilityDB()
	content := "requests==2.28.0\nflask==2.3.0"

	reasons := db.checkPythonRequirements(content)
	if len(reasons) != 0 {
		t.Errorf("expected no vulnerabilities, got %d: %v", len(reasons), reasons)
	}
}
