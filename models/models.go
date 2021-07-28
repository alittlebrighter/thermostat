package models

import "github.com/alittlebrighter/thermostat/util"

type SensorUpdate struct {
	Location string      `json:"location"`
	Type     string      `json:"type"`
	Value    Temperature `json:"value"`
}

type Temperature struct {
	Degrees float64               `json:"degrees"`
	Unit    util.TemperatureUnits `json:"unit"`
}
