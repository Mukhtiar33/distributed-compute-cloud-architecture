package validation

import (
	"archive/zip"
	"bytes"
	"testing"
)

func createZipOutput(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	w.Close()
	return buf.Bytes()
}

func TestValidateWorkerLocal_Pass(t *testing.T) {
	output := createZipOutput(map[string]string{
		"result.txt": "test result",
	})

	expected := ExpectedOutput{
		Files:   []string{"result.txt"},
		MinSize: 10,
	}

	result := ValidateWorkerLocal(output, expected)
	if !result.Passed {
		t.Errorf("expected stage 1 to pass, got: %s", result.Message)
	}
}

func TestValidateWorkerLocal_EmptyOutput(t *testing.T) {
	expected := ExpectedOutput{}
	result := ValidateWorkerLocal([]byte{}, expected)
	if result.Passed {
		t.Error("expected stage 1 to fail for empty output")
	}
}

func TestValidateWorkerLocal_MissingFile(t *testing.T) {
	output := createZipOutput(map[string]string{
		"other.txt": "test",
	})

	expected := ExpectedOutput{
		Files: []string{"result.txt"},
	}

	result := ValidateWorkerLocal(output, expected)
	if result.Passed {
		t.Error("expected stage 1 to fail for missing file")
	}
}

func TestValidateWorkerLocal_SizeTooSmall(t *testing.T) {
	output := createZipOutput(map[string]string{
		"result.txt": "tiny",
	})

	expected := ExpectedOutput{
		MinSize: 1000,
	}

	result := ValidateWorkerLocal(output, expected)
	if result.Passed {
		t.Error("expected stage 1 to fail for size below minimum")
	}
}

func TestValidateWorkerLocal_SizeTooLarge(t *testing.T) {
	// Use non-compressible data to ensure zip file is large enough
	data := make([]byte, 2000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	output := createZipOutput(map[string]string{
		"result.txt": string(data),
	})

	// Set MaxSize to 100 bytes — the zip file with 2000 bytes of data will be larger
	expected := ExpectedOutput{
		MaxSize: 100,
	}

	result := ValidateWorkerLocal(output, expected)
	if result.Passed {
		t.Error("expected stage 1 to fail for size above maximum")
	}
}

func TestValidateWorkerLocal_InvalidZip(t *testing.T) {
	expected := ExpectedOutput{}
	result := ValidateWorkerLocal([]byte("not a zip"), expected)
	if result.Passed {
		t.Error("expected stage 1 to fail for invalid zip")
	}
}

func TestValidateControlPlane_Pass(t *testing.T) {
	output := createZipOutput(map[string]string{
		"result.txt": "test result",
	})

	expected := ExpectedOutput{
		Files:   []string{"result.txt"},
		MinSize: 10,
	}

	result := ValidateControlPlane(output, expected)
	if !result.Passed {
		t.Errorf("expected stage 2 to pass, got: %s", result.Message)
	}
}

func TestValidateControlPlane_ZipBomb(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, _ := w.Create("bomb.txt")
	data := make([]byte, 10*1024*1024)
	f.Write(data)
	w.Close()

	expected := ExpectedOutput{}
	result := ValidateControlPlane(buf.Bytes(), expected)
	if result.Passed {
		t.Error("expected stage 2 to fail for zip bomb")
	}
}

func TestValidateControlPlane_InvalidZip(t *testing.T) {
	expected := ExpectedOutput{}
	result := ValidateControlPlane([]byte("not a zip"), expected)
	if result.Passed {
		t.Error("expected stage 2 to fail for invalid zip")
	}
}

func TestValidatePreDelivery_Pass(t *testing.T) {
	output := createZipOutput(map[string]string{
		"result.txt": "test result",
	})

	expected := ExpectedOutput{}
	result := ValidatePreDelivery(output, expected)
	if !result.Passed {
		t.Errorf("expected stage 3 to pass, got: %s", result.Message)
	}
}

func TestValidatePreDelivery_InvalidZip(t *testing.T) {
	expected := ExpectedOutput{}
	result := ValidatePreDelivery([]byte("not a zip"), expected)
	if result.Passed {
		t.Error("expected stage 3 to fail for invalid zip")
	}
}

func TestValidateAll_AllPass(t *testing.T) {
	output := createZipOutput(map[string]string{
		"result.txt": "test result",
	})

	expected := ExpectedOutput{
		Files:   []string{"result.txt"},
		MinSize: 10,
	}

	results := ValidateAll(output, expected)
	for _, r := range results {
		if !r.Passed {
			t.Errorf("expected all stages to pass, stage %d failed: %s", r.Stage, r.Message)
		}
	}
}

func TestValidateAll_Stage2Fails(t *testing.T) {
	output := createZipOutput(map[string]string{
		"other.txt": "test",
	})

	expected := ExpectedOutput{
		Files: []string{"result.txt"},
	}

	results := ValidateAll(output, expected)
	if results[0].Passed {
		t.Error("expected stage 1 to fail")
	}
	if results[1].Passed {
		t.Error("expected stage 2 to fail")
	}
}

func TestDetectContentType_Zip(t *testing.T) {
	data := []byte{0x50, 0x4B, 0x03, 0x04}
	ct := detectContentType(data)
	if ct != "application/zip" {
		t.Errorf("expected application/zip, got %s", ct)
	}
}

func TestDetectContentType_Gzip(t *testing.T) {
	data := []byte{0x1F, 0x8B, 0x08, 0x00}
	ct := detectContentType(data)
	if ct != "application/gzip" {
		t.Errorf("expected application/gzip, got %s", ct)
	}
}

func TestDetectContentType_Unknown(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03}
	ct := detectContentType(data)
	if ct != "application/octet-stream" {
		t.Errorf("expected application/octet-stream, got %s", ct)
	}
}
