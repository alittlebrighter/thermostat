package thermometer

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/alittlebrighter/thermostat/util"
)

// JSONWebService reads temperature values from a remote location through a JSON API call over http.
type JSONWebService struct {
	client  *http.Client
	request *http.Request
}

// NewJSONWebService constructs a JSONWebService.
func NewJSONWebService(endpoint string) (*JSONWebService, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	thermometer := &JSONWebService{client: http.DefaultClient, request: req}

	var resp *http.Response
	resp, err = thermometer.client.Do(req)
	if err != nil || resp.StatusCode > 300 {
		return nil, errors.New("Could not connect to thermometer web service.")
	}

	return thermometer, nil
}

// ReadTemperature calls out to the configured web service to obtain a temperature reading.
func (meter *JSONWebService) ReadTemperature() (float64, util.TemperatureUnits, error) {
	resp, err := meter.client.Do(meter.request)
	defer resp.Body.Close()
	if err != nil {
		return 0, util.Celsius, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, util.Celsius, err
	}

	tempReading := new(TemperatureReading)
	err = json.Unmarshal(body, tempReading)
	if err != nil {
		return 0, util.Celsius, err
	}

	return tempReading.Explode()
}

// Explode returns the elements of a TemperatureReading into individual parameters.
func (r *TemperatureReading) Explode() (float64, util.TemperatureUnits, error) {
	var err error
	if r.Error == "<nil>" {
		err = nil
	} else {
		err = errors.New(r.Error)
	}
	return r.Temperature, r.Units, err
}

// Shutdown exists for the JSONWebService purely to satisfy the Thermometer interface
func (meter *JSONWebService) Shutdown() {}

// TemperatureReading exists to make deserialization of the web service call easier.
type TemperatureReading struct {
	Temperature float64
	Units       util.TemperatureUnits
	Error       string
}
