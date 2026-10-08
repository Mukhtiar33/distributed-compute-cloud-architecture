package scanner

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	// MaxFileSize is the maximum size of a single decompressed file (25MB)
	MaxFileSize = 25 * 1024 * 1024
	// MaxTotalSize is the maximum total decompressed size (100MB)
	MaxTotalSize = 100 * 1024 * 1024
	// MaxCompressionRatio is the maximum allowed compression ratio
	MaxCompressionRatio = 100
)

// ScanResult represents the outcome of a static scan.
type ScanResult struct {
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
}

// Scanner performs static analysis on job payloads.
type Scanner struct {
	vulnDB        *VulnerabilityDB
	patternChecks []PatternCheck
}

// NewScanner creates a new static scanner with default checks.
func NewScanner() *Scanner {
	return &Scanner{
		vulnDB:        NewVulnerabilityDB(),
		patternChecks: defaultPatternChecks(),
	}
}

// ScanZip scans a zip file containing a job payload.
func (s *Scanner) ScanZip(zipData []byte) (*ScanResult, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	var allReasons []string
	var totalUncompressed int64

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Check for zip bomb
		totalUncompressed += int64(file.UncompressedSize64)
		if totalUncompressed > MaxTotalSize {
			return &ScanResult{
				Passed:  false,
				Reasons: []string{fmt.Sprintf("zip bomb: total uncompressed size %d exceeds limit %d", totalUncompressed, MaxTotalSize)},
			}, nil
		}

		// Check compression ratio
		if file.CompressedSize64 > 0 {
			ratio := float64(file.UncompressedSize64) / float64(file.CompressedSize64)
			if ratio > MaxCompressionRatio {
				return &ScanResult{
					Passed:  false,
					Reasons: []string{fmt.Sprintf("zip bomb: compression ratio %.1fx exceeds limit %dx", ratio, MaxCompressionRatio)},
				}, nil
			}
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", file.Name, err)
		}

		// Limit reader to prevent zip bombs
		limitedReader := io.LimitReader(rc, MaxFileSize)
		data, err := io.ReadAll(limitedReader)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file.Name, err)
		}

		// Check if file was truncated (zip bomb)
		if int64(len(data)) >= MaxFileSize {
			return &ScanResult{
				Passed:  false,
				Reasons: []string{fmt.Sprintf("file %s exceeds maximum size %d", file.Name, MaxFileSize)},
			}, nil
		}

		// Check dependencies
		if isDependencyFile(file.Name) {
			reasons := s.vulnDB.CheckFile(file.Name, data)
			allReasons = append(allReasons, reasons...)
		}

		// Check source code patterns
		if isSourceFile(file.Name) {
			reasons := s.checkPatterns(file.Name, data)
			allReasons = append(allReasons, reasons...)
		}
	}

	if len(allReasons) > 0 {
		return &ScanResult{Passed: false, Reasons: allReasons}, nil
	}

	return &ScanResult{Passed: true}, nil
}

// ScanManifest validates the manifest's entrypoint exists in the zip.
func (s *Scanner) ScanManifest(zipData []byte, entrypoint string) (*ScanResult, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	for _, file := range zipReader.File {
		if file.Name == entrypoint {
			return &ScanResult{Passed: true}, nil
		}
	}

	return &ScanResult{
		Passed:  false,
		Reasons: []string{fmt.Sprintf("entrypoint '%s' not found in payload", entrypoint)},
	}, nil
}

func (s *Scanner) checkPatterns(filename string, data []byte) []string {
	var reasons []string
	content := string(data)

	for _, check := range s.patternChecks {
		if check.Pattern.MatchString(content) {
			reasons = append(reasons, fmt.Sprintf("%s: %s", filename, check.Description))
		}
	}

	return reasons
}

func isDependencyFile(name string) bool {
	depFiles := []string{
		"requirements.txt",
		"package.json",
		"go.mod",
		"Pipfile",
		"pom.xml",
		"build.gradle",
		"Gemfile",
		"Cargo.toml",
	}
	for _, df := range depFiles {
		if strings.HasSuffix(name, df) {
			return true
		}
	}
	return false
}

func isSourceFile(name string) bool {
	sourceExts := []string{".py", ".js", ".ts", ".go", ".java", ".rb", ".sh", ".bat", ".ps1"}
	for _, ext := range sourceExts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// PatternCheck represents a static code pattern check.
type PatternCheck struct {
	Pattern     *regexp.Regexp
	Description string
}

func defaultPatternChecks() []PatternCheck {
	return []PatternCheck{
		{
			Pattern:     regexp.MustCompile(`(?i)(eval|exec)\s*\(\s*["'` + "`" + `]`),
			Description: "Potential code injection via eval/exec",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)os\.system\s*\(`),
			Description: "Direct system command execution",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)subprocess\.(call|run|Popen)`),
			Description: "Subprocess execution detected",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(base64|codecs)\.(decode|b64decode|decodebytes)`),
			Description: "Base64 decoding (potential obfuscation)",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)__import__\s*\(\s*["'` + "`" + `]os["'` + "`" + `]\s*\)`),
			Description: "Dynamic import of os module",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)ctypes\.CDLL`),
			Description: "Dynamic library loading via ctypes",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)Runtime\.getRuntime\(\)\.exec`),
			Description: "Java runtime exec",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)child_process\.exec`),
			Description: "Node.js child_process exec",
		},
	}
}

// VulnerabilityDB is a simple in-memory vulnerability database.
type VulnerabilityDB struct {
	vulnerabilities map[string][]Vulnerability
}

// Vulnerability represents a known vulnerable package version.
type Vulnerability struct {
	Package  string   `json:"package"`
	Versions []string `json:"versions"`
	Severity string   `json:"severity"`
	Summary  string   `json:"summary"`
}

// NewVulnerabilityDB creates a vulnerability database with known vulnerable packages.
func NewVulnerabilityDB() *VulnerabilityDB {
	return &VulnerabilityDB{
		vulnerabilities: map[string][]Vulnerability{
			"requests": {
				{
					Package:  "requests",
					Versions: []string{"2.25.0", "2.25.1", "2.24.0"},
					Severity: "HIGH",
					Summary:  "CVE-2023-32681: Information disclosure via Proxy-Authorization header",
				},
			},
			"django": {
				{
					Package:  "django",
					Versions: []string{"3.2.0", "3.2.1", "3.2.2", "3.2.3", "3.2.4", "3.2.5"},
					Severity: "CRITICAL",
					Summary:  "CVE-2021-33203: Potential directory traversal via django.utils.archive.extract",
				},
			},
			"flask": {
				{
					Package:  "flask",
					Versions: []string{"1.0.0", "1.0.1", "1.0.2", "1.0.3", "1.1.0"},
					Severity: "MEDIUM",
					Summary:  "CVE-2023-30861: Possible session cookie exposure",
				},
			},
			"lodash": {
				{
					Package:  "lodash",
					Versions: []string{"4.17.20", "4.17.19", "4.17.18", "4.17.17", "4.17.16", "4.17.15"},
					Severity: "HIGH",
					Summary:  "CVE-2021-23337: Command injection via template function",
				},
			},
			"express": {
				{
					Package:  "express",
					Versions: []string{"4.17.1", "4.17.0", "4.16.1", "4.16.0"},
					Severity: "MEDIUM",
					Summary:  "CVE-2022-24999: Open redirect in express",
				},
			},
		},
	}
}

// CheckFile checks a dependency file for known vulnerabilities.
func (db *VulnerabilityDB) CheckFile(filename string, data []byte) []string {
	var reasons []string
	content := string(data)

	switch {
	case strings.HasSuffix(filename, "requirements.txt"):
		reasons = append(reasons, db.checkPythonRequirements(content)...)
	case strings.HasSuffix(filename, "package.json"):
		reasons = append(reasons, db.checkPackageJSON(content)...)
	case strings.HasSuffix(filename, "go.mod"):
		reasons = append(reasons, db.checkGoMod(content)...)
	}

	return reasons
}

func (db *VulnerabilityDB) checkPythonRequirements(content string) []string {
	var reasons []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse package name and version
		parts := strings.Split(line, "==")
		if len(parts) != 2 {
			continue
		}

		pkgName := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])

		if vulns, ok := db.vulnerabilities[pkgName]; ok {
			for _, vuln := range vulns {
				for _, vulnVersion := range vuln.Versions {
					if version == vulnVersion {
						reasons = append(reasons, fmt.Sprintf(
							"VULNERABILITY: %s==%s — %s (%s)",
							pkgName, version, vuln.Summary, vuln.Severity,
						))
					}
				}
			}
		}
	}

	return reasons
}

func (db *VulnerabilityDB) checkPackageJSON(content string) []string {
	var reasons []string

	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return reasons
	}

	for pkgName, version := range pkg.Dependencies {
		// Strip version prefixes like ^, ~, >=, etc.
		version = strings.TrimPrefix(version, "^")
		version = strings.TrimPrefix(version, "~")
		version = strings.TrimPrefix(version, ">=")
		version = strings.TrimPrefix(version, ">")
		version = strings.TrimPrefix(version, "v")

		if vulns, ok := db.vulnerabilities[pkgName]; ok {
			for _, vuln := range vulns {
				for _, vulnVersion := range vuln.Versions {
					if version == vulnVersion {
						reasons = append(reasons, fmt.Sprintf(
							"VULNERABILITY: %s@%s — %s (%s)",
							pkgName, version, vuln.Summary, vuln.Severity,
						))
					}
				}
			}
		}
	}

	return reasons
}

func (db *VulnerabilityDB) checkGoMod(content string) []string {
	var reasons []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || line == "module" || line == "go" {
			continue
		}

		// Parse "require" lines
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pkgName := parts[0]
		version := strings.TrimPrefix(parts[1], "v")

		if vulns, ok := db.vulnerabilities[pkgName]; ok {
			for _, vuln := range vulns {
				for _, vulnVersion := range vuln.Versions {
					if version == vulnVersion {
						reasons = append(reasons, fmt.Sprintf(
							"VULNERABILITY: %s@%s — %s (%s)",
							pkgName, version, vuln.Summary, vuln.Severity,
						))
					}
				}
			}
		}
	}

	return reasons
}
