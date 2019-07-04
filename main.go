package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/caarlos0/env"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"
)

const (
	jsonMimeType = "application/json"
	xlsxMimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
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
<h1>XSLX to JSON to XLSX REST API</h1>
<form action="/" method="post" enctype="multipart/form-data">
    <input type="file" name="file" accept="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/json">
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
	router.HandleFunc("/", ReceiveFile).Methods("POST").Headers("Content-Type", jsonMimeType)
	router.HandleFunc("/", ReceiveFile).Methods("POST")
	router.HandleFunc("/json2xlsx", ReceiveJson).Methods("POST").Headers("Content-Type", jsonMimeType)

	srv := &http.Server{
		Addr:         cfg.ApiHost + ":" + cfg.ApiPort,
		Handler:      addCorsHeader(router),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func addCorsHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}

func Welcome(writer http.ResponseWriter, request *http.Request) {
	request.Header.Get("Content-Type")
	writer.Write([]byte(welcomeTemplate))
}

func ReceiveFile(writer http.ResponseWriter, request *http.Request) {
	file, fileHeader, err := request.FormFile("file")
	if err != nil {
		handleError(writer, errors.New("No file upload found. Please send as param named 'file'."))
		return
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		handleError(writer, err)
		return
	}

	fileContentType := fileHeader.Header.Get("Content-Type")

	if fileContentType == jsonMimeType {
		handleJson(writer, buf.Bytes())
	} else {
		handleXlsx(writer, buf.Bytes(), fileHeader)
	}
}

func ReceiveJson(writer http.ResponseWriter, request *http.Request) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		handleError(writer, err)
	}
	handleJson(writer, body)
}

func handleJson(writer http.ResponseWriter, payload []byte) {
	workbook := Workbook{}
	err := json.Unmarshal(payload, &workbook)
	if err != nil {
		handleError(writer, err)
	}

	log.WithFields(log.Fields{
		"name":       workbook.Name,
		"num_sheets": len(workbook.Sheets),
	}).Info("JSON -> XLSX")

	file := xlsx.NewFile()

	boldStyle := xlsx.NewStyle()
	boldFont := xlsx.NewFont(10, "Arial")
	boldFont.Bold = true
	boldStyle.Font = *boldFont
	boldStyle.ApplyFont = true

	for _, sheet := range workbook.Sheets {
		s, _ := file.AddSheet(sheet.Name)
		h := s.AddRow()
		for _, col := range sheet.Columns {
			c := h.AddCell()
			c.SetValue(col)
			c.SetStyle(boldStyle)
		}
		for _, row := range sheet.Rows {
			r := s.AddRow()
			for _, cell := range row {
				c := r.AddCell()
				c.Value = cell
			}
		}
	}

	writer.Header().Set("Content-Disposition", "attachment; filename="+workbook.Name)
	writer.Header().Add("Content-type", xlsxMimeType)
	file.Write(writer)
}

func handleXlsx(writer http.ResponseWriter, payload []byte, header *multipart.FileHeader) {
	xlFile, err := xlsx.OpenBinary(payload)
	if err != nil {
		handleError(writer, errors.New("Invalid XLSX file"))
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

	log.WithField("file", header.Filename).Info("XLSX -> JSON")

	writer.Header().Add("Content-type", jsonMimeType)
	json.NewEncoder(writer).Encode(workbook)
}

func handleError(writer http.ResponseWriter, err error) {
	log.WithField("error", err).Error("Error")
	writer.Header().Add("Content-type", jsonMimeType)

	jsonError := JsonError{
		Code:      500,
		HttpError: http.StatusText(500),
		Message:   err.Error(),
	}

	jsonErrorPayload, _ := json.Marshal(jsonError)

	http.Error(writer, string(jsonErrorPayload), 500)
}
