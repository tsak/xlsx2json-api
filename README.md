# XLSX 2 JSON API

[![Go Report Card](https://goreportcard.com/badge/github.com/tsak/xlsx2json-api)](https://goreportcard.com/report/github.com/tsak/xlsx2json-api)

A lightweight API that converts XLSX files into JSON. Try it on [xlsx2json.tsak.net](https://xlsx2json.tsak.net)

It will turn...

![tables are cool](tables.png)

...into...

```json
{
  "name" : "tables are cool.xlsx",
  "spreadsheets" : [
    {
      "name" : "Sheet 1",
      "columns" : [
        "Tables",
        "Are",
        "Cool"
      ],
      "rows" : [
        [
          "col 3 is",
          "right-aligned",
          "$1600"
        ],
        [
          "col 2 is",
          "centered",
          "$12"
        ],
        [
          "zebra stripes",
          "are neat",
          "$1"
        ]
      ]
    }
  ]
}
```

...and vice versa.

Simply send an XLSX file via `POST` and `multipart/form-data` request to `/`. Expected upload parameter name is `file`.

**Sample form**

```html
<!DOCTYPE html>
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
</html>
```

## Endpoints

### `POST /`

This either accepts an XLSX file or JSON file as part of a `multipart/form-data` request. It expects `name="file"`.

**Sample XLSX upload**

```
POST http://localhost:8000
Content-Type: multipart/form-data
-----------------------------1234567890
Content-Disposition: form-data; name="file"; filename="test.xlsx"
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet

...
```

### `POST /json2xlsx`

This endpoint accepts JSON if sent via `POST` and with `Content-Type: application/json`

```
POST http://localhost:8000/json2xlsx
Accept: */*
Cache-Control: no-cache
Content-Type: application/json

{}
```

The format has to match the format that is returned by turning XLSX to JSON.

## Environment variables

* `API_HOST` (default `localhost`)
* `API_PORT` (default `8000`)
* `DEBUG` (default `0`), use `DEBUG=1` to enable debug mode

## Test & build

```bash
go test && go build
```

## Run

### Local
Build, run the following command and then open [localhost:8000](http://localhost:8000).

```bash
DEBUG=1 ./xlsx2json-api
```

**Note:** If you run with `DEBUG=1`, it will serve a simple HTML form on [localhost:8000](http://localhost:8000) for
testing.

![HTML form](debug.png)

### Docker

Run the command below and then open [localhost:8000](http://localhost:8000).

```bash
docker run -p 8000:8000 tsak/xlsx2json-api
```

### As a systemd service

See [xlsx2json-api.service](xlsx2json-api.service) systemd service definition.

To install:

1. `adduser xlsx2json`
2. copy `xlsx2json-api` binary and `.env` file to `/home/xlsx2json`
3. place systemd service script in `/etc/systemd/system/`
4. `sudo systemctl enable xlsx2json-api.service`
5. `sudo systemctl start xlsx2json-api`
6. `sudo journalctl -f -u xlsx2json-api`

The last command will show if the service was started.
