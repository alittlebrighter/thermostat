package thermostat

import (
	"errors"
	"testing"
	"time"

	"github.com/alittlebrighter/thermostat/controller"
	"github.com/alittlebrighter/thermostat/util"
)

func TestProcessTemperatureReading(t *testing.T) {
	baseThermostat.Modes = map[string]*Window{"default": &Window{LowTemp: 69, HighTemp: 80}}

	// order of tests matters because the process takes direction into account and it's unlikely
	// and probably bad to run the AC directly following the heat
	baseThermostat.ProcessTemperatureReading(68.9, util.Celsius)
	if baseThermostat.control.Direction() != controller.Heating {
		t.Error("Failed to set direction to HEATING.")
	}

	baseThermostat.ProcessTemperatureReading(72.5, util.Celsius)
	if baseThermostat.control.Direction() != controller.None {
		t.Error("Failed to set direction to NONE.")
	}

	baseThermostat.ProcessTemperatureReading(82, util.Celsius)
	if baseThermostat.control.Direction() != controller.Cooling {
		t.Error("Failed to set direction to COOLING.")
	}
}

func TestHandleError(t *testing.T) {
	baseThermostat.Modes = map[string]*Window{"default": &Window{LowTemp: 69, HighTemp: 80}}
	baseThermostat.ProcessTemperatureReading(82, util.Celsius)
	baseThermostat.MaxErrors = 3

	for i := 0; uint8(i) < baseThermostat.MaxErrors; i++ {
		baseThermostat.HandleError()
		if baseThermostat.control.Direction() == controller.None {
			t.Error("Shut off HVAC after too few errors.")
		}
	}

	baseThermostat.HandleError()
	if baseThermostat.control.Direction() != controller.None {
		t.Error("Failed to shut off HVAC MaxErrors.")
	}
}

var baseThermostat = &Thermostat{
	Modes:          map[string]*Window{"default": &Window{LowTemp: 69, HighTemp: 80}},
	DefaultMode:    "default",
	Schedule:       []*ScheduleEvent{},
	Overshoot:      3,
	PollInterval:   util.Duration(time.Duration(1) * time.Minute),
	UnitPreference: util.Celsius,
	Events:         util.NewRingBuffer(1),
	control:        new(MockController),
	thermometer:    new(MockThermometer),
}

type MockController struct {
	direction controller.ThermoDirection
}

func (mc *MockController) Direction() controller.ThermoDirection {
	return mc.direction
}

func (mc *MockController) Off() {
	mc.direction = controller.None
}

func (mc *MockController) Fan() {
	mc.direction = controller.Fan
}

func (mc *MockController) Cool() {
	mc.direction = controller.Cooling
}

func (mc *MockController) Heat() {
	mc.direction = controller.Heating
}

func (mc *MockController) Shutdown() {}

type MockThermometer struct{}

func (mt *MockThermometer) ReadTemperature() (float64, util.TemperatureUnits, error) {
	return ambientTemp, util.Celsius, nil
}

func (mt *MockThermometer) Shutdown() {}

type MockErrorThermometer struct{}

func (mt *MockErrorThermometer) ReadTemperature() (float64, util.TemperatureUnits, error) {
	return ambientTemp, util.Celsius, errors.New("Temperature not available.")
}

func (mt *MockErrorThermometer) Shutdown() {}

var ambientTemp = 72.5
