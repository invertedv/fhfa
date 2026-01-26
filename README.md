
## package fhfa

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/fhfa)](https://goreportcard.com/report/github.com/invertedv/fhfw)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/fhfa?tab=overview)


The fhfa package makes the non-seasonally adjusted FHFA house price index available. It will pull the data from the web and has methods to find the HPI for individual geos and dates. The data can be read from the FHFA XLSX files if they have already been downloaded. The data is loaded
into the map HPIdata.  The key to the map is the geo name (or code, in the case of MSAs). The value of the map is the struct HPIgeo.  The HPI method of HPIgeo returns the house price index for a given date (specified as an int in the format CCYYQ).


The normal flow is:

1. Get the URL of the series you want by calling URLs if pulling the data from the web.
2. Fetch the data from the web (or load from a local file).
3. Parse the data into an HPIdata map.
4. Use the HPI method to find the HPI for any date for a given geo.


The indices are available [here](https://www.fhfa.gov/data/hpi/datasets?tab=quarterly-data).
