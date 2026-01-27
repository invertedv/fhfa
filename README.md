
## package fhfa

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/fhfa)](https://goreportcard.com/report/github.com/invertedv/fhfw)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/fhfa?tab=overview)

### Purpose

The purpose of this package is to make the FHFA house price indices available in a format that is fast and
readily usable within Go. 

### Details

This package will pull the data from the FHFA web site and find the HPI for individual geos and dates. The data can also be read from the FHFA XLSX files directly, if they have already been downloaded. Once loaded, the package provides access to series.


The indices are available [here](https://www.fhfa.gov/data/hpi/datasets?tab=quarterly-data).


The XLSX format is chosen since not all of the data is available as a CSV but all are as XLSX.
