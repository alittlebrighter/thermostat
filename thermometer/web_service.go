package thermometer

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/alittlebrighter/thermostat/util"
)

type JSONWebService struct {
	client  *http.Client
	request *http.Request
}

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

func (meter *JSONWebService) Shutdown() {}

type TemperatureReading struct {
	Temperature float64
	Units       util.TemperatureUnits
	Error       string
}

func (r *TemperatureReading) Explode() (float64, util.TemperatureUnits, error) {
	var err error
	if r.Error == "<nil>" {
		err = nil
	} else {
		err = errors.New(r.Error)
	}
	return r.Temperature, r.Units, err
}
