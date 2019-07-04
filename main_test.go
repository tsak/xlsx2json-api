package main

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
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

func newfileUploadRequest(uri string, params map[string]string, paramName, path string, contentType string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, paramName, filepath.Base(path)))
	h.Set("Content-Type", contentType)
	part, err := writer.CreatePart(h)

	//part, err := writer.CreateFormFile(paramName, filepath.Base(path))
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

func testUpload(t *testing.T, testname string, paramName string, path string, expected string, expectedStatusCode int, contentType string) {
	t.Run(testname, func(t *testing.T) {
		req, err := newfileUploadRequest("/", nil, paramName, path, contentType)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Content-Type", contentType)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ReceiveFile)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != expectedStatusCode {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, expectedStatusCode)
		}

		log.Debug(len(rr.Body.String()))

		if expected != "" && rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expected)
		}
	})
}

func TestReceiveFile(t *testing.T) {
	// generic XLSX
	expected := `{"name":"sample.xlsx","spreadsheets":[{"name":"Sheet 1","columns":["Column0","Column1","Column2","Column3","Column4"],"rows":[["1","2","3","4","5"],["a","b","c","d","e"]]},{"name":"Sheet 2","columns":["Column0","Column1","Column2"],"rows":[["1","2","3"],["a","b","c"]]}]}` + "\n"
	testUpload(t, "sample.xlsx", "file", "testfiles/sample.xlsx", expected, 200, xlsxMimeType)

	// empty XLSX
	expected2 := `{"name":"empty.xlsx","spreadsheets":[{"name":"Sheet 1","columns":null,"rows":null}]}` + "\n"
	testUpload(t, "empty.xlsx", "file", "testfiles/empty.xlsx", expected2, 200, xlsxMimeType)

	// empty CSV file
	expected3 := `{"http_error_code":500,"http_error":"Internal Server Error","message":"Invalid XLSX file"}` + "\n"
	testUpload(t, "wrong.csv", "file", "testfiles/wrong.csv", expected3, 500, xlsxMimeType)

	// not sending as `file` in the POST body (also captures sending empty body)
	expected4 := `{"http_error_code":500,"http_error":"Internal Server Error","message":"No file upload found. Please send as param named 'file'."}` + "\n"
	testUpload(t, "wrong param name", "upload", "testfiles/sample.xlsx", expected4, 500, xlsxMimeType)

	// ZIP file renamed to XLSX
	expected5 := `{"http_error_code":500,"http_error":"Internal Server Error","message":"Invalid XLSX file"}` + "\n"
	testUpload(t, "wrong.xslx", "file", "testfiles/wrong.xslx", expected5, 500, xlsxMimeType)

	// JSON
	testUpload(t, "test.json", "file", "testfiles/test.json", "", 200, jsonMimeType)
}
