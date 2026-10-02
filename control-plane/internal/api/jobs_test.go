package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
)

func TestSubmitJob_ValidSingle(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{"job_type":"single","environment":"python-3.11","entrypoint":"main.py","expected_output":"result.zip"}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["status"] != "intake_validated" {
		t.Errorf("expected status intake_validated, got %s", result["status"])
	}
}

func TestSubmitJob_ValidParallel(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{"job_type":"parallel","environment":"python-3.11","entrypoint":"process.py","inputs":["input1.csv","input2.csv"],"expected_output":"results.zip"}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestSubmitJob_MissingManifest(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSubmitJob_MissingZip(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{"job_type":"single","environment":"python-3.11","entrypoint":"main.py","expected_output":"result.zip"}`)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSubmitJob_InvalidManifest(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{"job_type":"invalid","environment":"python-3.11","entrypoint":"main.py","expected_output":"result.zip"}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["status"] != "rejected" {
		t.Errorf("expected status rejected, got %s", result["status"])
	}
}

func TestSubmitJob_ParallelMissingInputs(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{"job_type":"parallel","environment":"python-3.11","entrypoint":"process.py","expected_output":"results.zip"}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["status"] != "rejected" {
		t.Errorf("expected status rejected, got %s", result["status"])
	}
}

func TestSubmitJob_EmptyZip(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{"job_type":"single","environment":"python-3.11","entrypoint":"main.py","expected_output":"result.zip"}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte(""))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetJob_NotFound(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/jobs?id=nonexistent", nil)
	w := httptest.NewRecorder()

	handler.GetJob(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestListJobs(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	// Add a job
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{"job_type":"single","environment":"python-3.11","entrypoint":"main.py","expected_output":"result.zip"}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	handler.SubmitJob(w, req)

	// List jobs
	req = httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w = httptest.NewRecorder()
	handler.ListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 1 {
		t.Errorf("expected 1 job, got %d", len(result))
	}
}

func TestSubmitJob_InvalidJSON(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{invalid json}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["status"] != "rejected" {
		t.Errorf("expected status rejected, got %s", result["status"])
	}
}

func TestSubmitJob_MissingAllFields(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("manifest", `{}`)
	part, _ := writer.CreateFormFile("zip", "test.zip")
	part.Write([]byte("fake zip content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	errors, ok := result["validation_errors"].([]interface{})
	if !ok {
		t.Fatal("expected validation_errors to be an array")
	}
	if len(errors) < 3 {
		t.Errorf("expected at least 3 validation errors, got %d", len(errors))
	}
}

func TestSubmitJob_WrongMethod(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestGetJob_MissingID(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	w := httptest.NewRecorder()

	handler.GetJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSubmitJob_ContentTypeNotMultipart(t *testing.T) {
	store := jobs.NewStore()
	handler := NewJobHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
