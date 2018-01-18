package main

import (
	"github.com/tealeg/xlsx"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/caarlos0/env"
	log "github.com/sirupsen/logrus"
	"net/http"
	"bytes"
	"io"
	"time"
	"errors"
)

type config struct {
	ApiHost string `env:"API_HOST" envDefault:"localhost"`
	ApiPort string `env:"API_PORT" envDefault:"8000"`
	Debug   bool   `env:"DEBUG" envDefault:"false"`
}

type Workbook struct {
	Name   string        `json:"name"`
	Sheets []Spreadsheet `json:"spreadsheets"`
}

type Spreadsheet struct {
	Name    string     `json:"name"`
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type JsonError struct {
	Code      int    `json:"http_error_code"`
	HttpError string `json:"http_error"`
	Message   string `json:"message"`
}

const welcomeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>XSLX2JSON API</title>
</head>
<body>
<h1>XSLX to JSON REST API</h1>
<form action="/" method="post" enctype="multipart/form-data">
    <input type="file" name="file" accept="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet">
    <button type="submit">Upload</button>
</form>
</body>
</html>`

func main() {
	// Read environment config
	cfg := config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Error(err)
	}

	// Set debug logging if DEBUG=true
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	log.WithFields(log.Fields{
		"ApiHost": cfg.ApiHost,
		"ApiPort": cfg.ApiPort,
		"Debug":   cfg.Debug,
	}).Debug("Environment config")

	router := mux.NewRouter()
	if cfg.Debug {
		router.HandleFunc("/", Welcome).Methods("GET")
	}
	router.HandleFunc("/", ReceiveFile).Methods("POST")

	srv := &http.Server{
		Addr: cfg.ApiHost+":"+cfg.ApiPort,
		Handler: router,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func Welcome(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte(welcomeTemplate))
}

func ReceiveFile(writer http.ResponseWriter, request *http.Request) {
	file, header, err := request.FormFile("file")
	if err != nil {
		handleError(writer, request, errors.New("No file upload found. Please send as param named 'file'."))
		return
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		handleError(writer, request, err)
		return
	}

	xlFile, err := xlsx.OpenBinary(buf.Bytes())
	if err != nil {
		handleError(writer, request, errors.New("Invalid XLSX file"))
		return
	}

	workbook := Workbook{
		Name: header.Filename,
	}
	for _, s := range xlFile.Sheets {
		spreadsheet := Spreadsheet{
			Name: s.Name,
		}
		for i, r := range s.Rows {
			var row []string
			for _, c := range r.Cells {
				row = append(row, c.String())
			}
			if i == 0 {
				spreadsheet.Columns = row
			} else {
				spreadsheet.Rows = append(spreadsheet.Rows, row)
			}
		}
		workbook.Sheets = append(workbook.Sheets, spreadsheet)
	}

	log.WithField("file", header.Filename).Info("Converted")

	writer.Header().Add("Content-type", "application/json")
	json.NewEncoder(writer).Encode(workbook)
}

func handleError(writer http.ResponseWriter, request *http.Request, err error) {
	log.WithField("error", err).Error("Error")
	writer.Header().Add("Content-type", "application/json")

	jsonError := JsonError{
		Code:      500,
		HttpError: http.StatusText(500),
		Message:   err.Error(),
	}

	jsonErrorPayload, _ := json.Marshal(jsonError)

	http.Error(writer, string(jsonErrorPayload), 500)
}
