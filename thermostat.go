package thermostat

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alittlebrighter/thermostat/controller"
	tmeter "github.com/alittlebrighter/thermostat/thermometer"
	"github.com/alittlebrighter/thermostat/util"
)

var ErrTempReading = errors.New("could not read temperature")

type Config struct {
	Thermostat  *Thermostat
	Controller  struct{ Pins struct{ Fan, Cool, Heat int } }
	Thermometer struct{ Type, Endpoint string }
}

// Thermostat is the primary struct that contains all of the data required to operate a smart thermostat system.
type Thermostat struct {
	Modes                 `json:"modes"`
	DefaultMode           string                `json:"defaultMode"`
	Schedule              []*ScheduleEvent      `json:"schedule"`
	Overshoot             float64               `json:"overshoot"`
	PollInterval          time.Duration         `json:"pollInterval"`
	MinFan                time.Duration         `json:"minFan"`
	LastFan               time.Time             `json:"lastFan"`
	MaxErrors, errorCount uint8                 `json:"maxErrors"`
	UnitPreference        util.TemperatureUnits `json:"unitPreference"`

	control               controller.Controller
	thermometer           tmeter.Thermometer
	Events                *util.RingBuffer `json:"events"`
}

// Modes are a collection of Windows referenced by a string label/key
type Modes map[string]*Window

// Window defines low and high temperatures
type Window struct {
	LowTemp  float64 `json:"low"`
	HighTemp float64 `json:"high"`
}

// ScheduleEvent defines a block of time from Start to End on the specified Days each week when the specified
// mode (ModeName) should be applied.
type ScheduleEvent struct {
	Days     []time.Weekday `json:"days"`
	ModeName string         `json:"mode"`
	Start    util.ClockTime `json:"start"`
	End      util.ClockTime `json:"end"`
}

func (stat *Thermostat) SetController(c controller.Controller) {
	stat.control = c
}

func (stat *Thermostat) SetThermometer(t tmeter.Thermometer) {
	stat.thermometer = t
}

// CurrentTemperatureWindow calculates what the current desired low and high temperatures should be based
// on the configured modes and schedule.
func (stat *Thermostat) CurrentTemperatureWindow(t time.Time) *Window {
	for _, spec := range stat.Schedule {
		if _, ok := stat.Modes[spec.ModeName]; !ok {
			continue
		}

		dayMatch := false
		for _, day := range spec.Days {
			if t.Weekday() == day {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			continue
		}

		switch {
		case t.Hour() < spec.Start.Hour() || t.Hour() > spec.End.Hour():
			fallthrough
		case t.Hour() == spec.Start.Hour() && t.Minute() < spec.Start.Minute():
			fallthrough
		case t.Hour() == spec.End.Hour() && t.Minute() > spec.End.Minute():
			continue
		default:
			return stat.Modes[spec.ModeName]
		}
	}

	return stat.Modes[stat.DefaultMode]
}

// ProcessTemperatureReading takes a temperature reading and the units the reading was measured at and determines
// what commands to send to the HVAC controller to keep the temperature inside of the configured range.
func (stat *Thermostat) ProcessTemperatureReading(ambientTemp float64, units util.TemperatureUnits) {
	var temp float64
	if string(units) == string(util.Celsius) && string(stat.UnitPreference) != string(util.Celsius) {
		temp = util.TempCToF(ambientTemp)
	} else if string(units) == string(util.Fahrenheit) && string(stat.UnitPreference) != string(util.Fahrenheit) {
		temp = util.TempFToC(ambientTemp)
	} else {
		temp = ambientTemp
	}

	now := time.Now()

	window := stat.CurrentTemperatureWindow(now)

	log.Printf("Current Temperature (%s): %f, Target: %f to %f, Last Fan: %v",
		stat.UnitPreference,
		temp,
		window.LowTemp,
		window.HighTemp,
		stat.LastFan.Local().Format(time.RFC3339),
	)
	switch {
	case (stat.control.Direction() == controller.Heating && temp > window.LowTemp+stat.Overshoot) /* done heating */ ||
		(stat.control.Direction() == controller.Cooling && temp < window.HighTemp-stat.Overshoot) /* done cooling */ ||
		(time.Duration(stat.MinFan).Nanoseconds() > 0 &&
			stat.control.Direction() == controller.Fan &&
			time.Since(stat.LastFan) > 0 &&
			time.Since(stat.LastFan) <= (time.Duration(1)*time.Hour)-stat.MinFan) /* done running fan */ :
		log.Println("turning OFF")
		stat.control.Off()
		stat.LastFan = now
	case temp < window.LowTemp:
		log.Println("turning on HEAT")
		stat.control.Heat()
		stat.LastFan = now
	case temp > window.HighTemp:
		log.Println("turning on COOL")
		stat.control.Cool()
		stat.LastFan = now
	case time.Duration(stat.MinFan).Nanoseconds() > 0 &&
		time.Since(stat.LastFan) > (time.Duration(1)*time.Hour)-stat.MinFan:
		log.Println("turning on FAN")
		stat.control.Fan()
		stat.LastFan = now.Add(time.Duration(stat.MinFan))
	default:
		log.Println("doing NOTHING")
	}

	stat.Events.Add(&util.EventLog{
		AmbientTemperature: temp,
		Units:              stat.UnitPreference,
		Direction:          stat.control.Direction(),
		Timestamp:          time.Now(),
	})
}

// HandleError manages errors received from temperature readings to make sure the system does not stay on in the event of
// not being able to acquire a temperature reading.
func (stat *Thermostat) HandleError() {
	stat.errorCount++

	if stat.errorCount > stat.MaxErrors {
		stat.control.Off()
		stat.errorCount = 0
	}
}

// Run starts the main event loop to run the thermostat.
func (stat *Thermostat) Run(ctx context.Context) {
	// we want to do something right away
	temp, units, err := stat.thermometer.ReadTemperature()
	if err != nil {
		log.Println("Error reading Temperature: " + err.Error())
		stat.Events.Add(&util.EventLog{AmbientTemperature: -1, Units: stat.UnitPreference, Direction: stat.control.Direction()})
		stat.HandleError()
	} else {
		stat.ProcessTemperatureReading(temp, units)
	}

	ticker := time.NewTicker(time.Duration(stat.PollInterval))
	for {
		select {
		case <-ticker.C:
			lastEvent := stat.Events.GetLast()
			if lastEvent != nil && time.Now().Add(time.Duration(-1)*stat.PollInterval).Before(lastEvent.Timestamp) {
				continue
			}

			temp, units, err := stat.thermometer.ReadTemperature()
			if err != nil {
				log.Println("Error reading Temperature: " + err.Error())
				stat.Events.Add(&util.EventLog{AmbientTemperature: -1, Units: stat.UnitPreference, Direction: stat.control.Direction()})
				stat.HandleError()
				continue
			}
			stat.ProcessTemperatureReading(temp, units)
		case <-ctx.Done():
			return
		}
	}
}

// Validate checks that a thermostat has a valid configuration and returns a string explaining any issues.  An empty string denotes a valid configuration.
func (stat *Thermostat) Validate() string {
	if _, ok := stat.Modes[stat.DefaultMode]; !ok {
		return "DefaultMode definition not found!"
	}

	for key, window := range stat.Modes {
		if window.LowTemp >= window.HighTemp {
			return fmt.Sprintf("%s mode is not valid.", key)
		}
	}

	for i, spec := range stat.Schedule {
		if time.Time(spec.Start).Unix() >= time.Time(spec.End).Unix() {
			return fmt.Sprintf("Schedule entry #%d not valid.", i+1)
		} else if _, ok := stat.Modes[spec.ModeName]; !ok {
			return fmt.Sprintf("Schedule entry #%d not valid.", i+1)
		}
	}

	return ""
}
