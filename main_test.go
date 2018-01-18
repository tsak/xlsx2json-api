package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"os"
	"bytes"
	"mime/multipart"
	"path/filepath"
	"io"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.PanicLevel)

	os.Exit(m.Run())
}

func TestWelcome(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(Welcome)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if rr.Body.String() != welcomeTemplate {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), welcomeTemplate)
	}
}

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

func testUpload(t *testing.T, testname string, paramName string, path string, expected string, expectedStatusCode int) {
	t.Run(testname, func(t *testing.T) {
		req, err := newfileUploadRequest("/", nil, paramName, path)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ReceiveFile)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != expectedStatusCode {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, expectedStatusCode)
		}

		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expected)
		}
	})
}

func TestReceiveFile(t *testing.T) {
	expected := `{"name":"sample.xlsx","spreadsheets":[{"name":"Sheet 1","columns":["Column0","Column1","Column2","Column3","Column4"],"rows":[["1","2","3","4","5"],["a","b","c","d","e"]]},{"name":"Sheet 2","columns":["Column0","Column1","Column2"],"rows":[["1","2","3"],["a","b","c"]]}]}` + "\n"
	testUpload(t, "sample.xlsx", "file", "testfiles/sample.xlsx", expected, 200)

	expected2 := `{"name":"empty.xlsx","spreadsheets":[{"name":"Sheet 1","columns":null,"rows":null}]}` + "\n"
	testUpload(t, "empty.xlsx", "file", "testfiles/empty.xlsx", expected2, 200)

	expected3 := `{"http_error_code":500,"http_error":"Internal Server Error","message":"Invalid XLSX file"}` + "\n"
	testUpload(t, "wrong.csv", "file", "testfiles/wrong.csv", expected3, 500)

	expected4 := `{"http_error_code":500,"http_error":"Internal Server Error","message":"No file upload found. Please send as param named 'file'."}` + "\n"
	testUpload(t, "wrong param name", "upload", "testfiles/sample.xlsx", expected4, 500)
}
