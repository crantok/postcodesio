// Package postcodesio wraps the postcodes.io geocoding service.
package postcodesio

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Error is used to represent any error in this package.
type Error string

// The various errors that might be encountered.
const (
	NotFound        Error = "Postcode Not Found"
	BadRequest      Error = "Bad Request"
	ServerError     Error = "Server Error"
	NoResults       Error = "No Results"
	MultipleResults Error = "Multiple Results"
	InvalidError    Error = "Invalid Error"
)

func errorFromHTTPCode(code int) Error {
	switch code {
	case 400:
		return BadRequest
	case 404:
		return NotFound
	case 500:
		return ServerError
	default:
		return InvalidError
	}
}

func (e Error) Error() string {
	switch e {
	case NotFound:
		return "postcodes.io could not find the requested information (404)"
	case BadRequest:
		return "postcodes.io rejected the request (400)"
	case ServerError:
		return "postcodes.io encountered an error (500)"
	case NoResults:
		return "postcodes.io returned no results for the request"
	case MultipleResults:
		return "postcodes.io returned multiple results for the request"
	default:
		return "Invalid Error: Please report to poscodesio Go package maintainer"
	}
}

const (
	baseURL = "https://api.postcodes.io"
)

type geocodeResult struct {
	Postcode                   string  `json:"postcode"`
	Quality                    int     `json:"quality"`
	Eastings                   int     `json:"eastings"`
	Northings                  int     `json:"northings"`
	Country                    string  `json:"country"`
	Nhs_ha                     string  `json:"nhs_ha"`
	Longitude                  float64 `json:"longitude"`
	Latitude                   float64 `json:"latitude"`
	Parliamentary_constituency string  `json:"parliamentary_constituency"`
	European_electoral_region  string  `json:"european_electoral_region"`
	Primary_care_trust         string  `json:"primary_care_trust"`
	Region                     string  `json:"region"`
	Lsoa                       string  `json:"lsoa"`
	Msoa                       string  `json:"msoa"`
	Incode                     string  `json:"incode"`
	Outcode                    string  `json:"outcode"`
	Admin_district             string  `json:"admin_district"`
	Parish                     string  `json:"parish"`
	Admin_county               string  `json:"admin_county"`
	Admin_ward                 string  `json:"admin_ward"`
	Ccg                        string  `json:"ccg"`
	Nuts                       string  `json:"nuts"`
	Codes                      struct {
		Admin_district string `json:"admin_district"`
		Admin_county   string `json:"admin_county"`
		Admin_ward     string `json:"admin_ward"`
		Parish         string `json:"parish"`
		Ccg            string `json:"ccg"`
		Nuts           string `json:"nuts"`
	} `json:"codes"`
}

func geocodeURL(postcode string) (result string, err error) {
	// Not returning errors

	uri, err := url.ParseRequestURI(baseURL + "/postcodes/" + postcode)
	if err != nil {
		return
	}

	return uri.String(), nil
}

func decodePayload(r *http.Response) (geocodeResult, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return geocodeResult{},
			errors.New("could not read http response body: " + err.Error())
	}

	payload := struct {
		Status int
		Result geocodeResult
		Error  string
	}{}

	jsonDecoder := json.NewDecoder(bytes.NewBuffer(body))
	err = jsonDecoder.Decode(&payload)

	if err != nil {
		if err == io.EOF {
			err = nil
		}
	}

	// Did the decoded json contain an error message?
	if err == nil && payload.Error != "" {
		err = errors.New(payload.Error)
	}

	// Did the decoded json include a non 200 status? This would be surprising
	// given that the response status should be checked before calling this
	// function.
	if err == nil && payload.Status != 200 {
		err = errorFromHTTPCode(int(payload.Status))
	}

	return payload.Result, err
}

func decorateGeocodingError(err error) error {
	return errors.New("postcodes.io: could not geocode postcode: " + err.Error())
}

// GeoPoint contains a geographical location as lat/lon coordinates.
type GeoPoint struct {
	Longitude float64
	Latitude  float64
}

// Geocode returns the geographic coordinates of the given UK postcode.
func Geocode(postcode string) (pt GeoPoint, err error) {

	url, err := geocodeURL(postcode)
	if err != nil {
		err = decorateGeocodingError(err)
		return
	}

	r, err := http.Get(url)
	if err != nil {
		err = decorateGeocodingError(err)
		return
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		err = decorateGeocodingError(errorFromHTTPCode(r.StatusCode))
		return
	}

	result, err := decodePayload(r)
	if err != nil {
		err = decorateGeocodingError(err)
	}

	return GeoPoint{Longitude: result.Longitude, Latitude: result.Latitude}, err
}
