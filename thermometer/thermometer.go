package thermometer

import (
	"github.com/alittlebrighter/thermostat/util"
)

// Thermometer defines the basic functions needed of a thermometer.
type Thermometer interface {
	ReadTemperature() (float64, util.TemperatureUnits, error)
	Shutdown()
}

// NewLocal returns a pointer to a local thermometer instance that can be used.
func NewLocal() (Thermometer, error) {
	return NewMCP9808()
}

// NewRemote returns a pointer to a thermometer service hosted remotely.
func NewRemote(endpoint string) (Thermometer, error) {
	return NewJSONWebService(endpoint)
}
