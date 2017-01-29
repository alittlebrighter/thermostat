package main

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	lowTemp  float64 = 69
	highTemp         = 80

	errorTolerance = 3

	tempCheckInterval = 1 * time.Minute
)

func TempCToF(tempC float64) float64 {
	return tempC*9/5 + 32
}

func TempFToC(tempF float64) float64 {
	return (tempF - 32) * 5 / 9
}

type Thermostat struct {
	LowTemp, HighTemp, overshoot, TargetTemp float64
	pollInterval                             time.Duration
	errors                                   uint8
	control                                  *Controller
	thermometer                              *Thermometer
	Events                                   *RingBuffer
}

func NewThermostat(controls *Controller, meter *Thermometer, low, high, overshoot float64) *Thermostat {
	return &Thermostat{control: controls, thermometer: meter, LowTemp: low, HighTemp: high, overshoot: overshoot, Events: NewRingBuffer(60)}
}

type TemperatureReading struct {
	Temperature float64
	Units       string
	Error       string
}

func (stat *Thermostat) ReadTemperature() (float64, error) {
	resp, err := http.Get("http://pi2/temperature")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	tempReading := new(TemperatureReading)
	err = json.Unmarshal(body, tempReading)
	if err != nil {
		return 0, err
	}

	return TempCToF(tempReading.Temperature), nil
}

type EventLog struct {
	AmbientTemperature float64
	Units              string
	Direction          ThermoDirection
}

type RingBuffer struct {
	buffer []*EventLog
	index  uint
}

func NewRingBuffer(size uint) *RingBuffer {
	return &RingBuffer{buffer: make([]*EventLog, size)}
}

func (buf *RingBuffer) Add(item *EventLog) {
	if buf.index == uint(len(buf.buffer)) {
		buf.index = 0
	}
	buf.buffer[buf.index] = item
	buf.index = buf.index + 1
}

func (buf *RingBuffer) GetAll() []*EventLog {
	return append(buf.buffer[buf.index:], buf.buffer[:buf.index]...)
}

func (buf *RingBuffer) GetLast() *EventLog {
	if buf.index == 0 {
		return buf.buffer[len(buf.buffer)-1]
	}

	return buf.buffer[buf.index-1]
}

func (stat *Thermostat) ProcessTemperatureReading(tempF float64) {
	log.Printf("Current Temperature (F): %f\n", tempF)
	switch {
	case (stat.control.Direction == Heating && tempF > stat.LowTemp+stat.overshoot) || (stat.control.Direction == Cooling && tempF < stat.HighTemp-stat.overshoot):
		log.Println("turning OFF")
		stat.control.Off()
	case tempF < stat.LowTemp:
		log.Println("turning on HEAT")
		stat.control.Heat()
	case tempF > stat.HighTemp:
		log.Println("turning on COOL")
		stat.control.Cool()
	default:
		log.Println("doing NOTHING")
	}

	stat.Events.Add(&EventLog{AmbientTemperature: tempF, Units: "Fahrenheit", Direction: stat.control.Direction})
}

func (stat *Thermostat) HandleError() {
	stat.errors++

	if stat.errors > errorTolerance {
		stat.control.Off()
		stat.errors = 0
	}
}

func (stat *Thermostat) Run() {
	ticker := time.NewTicker(tempCheckInterval)
	for {
		<-ticker.C

		tempF, err := stat.ReadTemperature()
		if err != nil {
			log.Println("Error reading Temperature: " + err.Error())
			stat.HandleError()
		} else {
			stat.ProcessTemperatureReading(tempF)
		}
	}
}

func main() {
	log.Println("Starting thermostat.")

	log.Println("Setting up controller.")
	control, err := NewController(16, 20, 21)
	if err != nil {
		log.Fatalln("Error starting controller: " + err.Error())
	}
	defer control.Shutdown()

	thermostat := NewThermostat(control, nil, lowTemp, highTemp, 2)

	tempF, err := thermostat.ReadTemperature()
	if err != nil {
		log.Println("Error reading Temperature: " + err.Error())
	} else {
		thermostat.ProcessTemperatureReading(tempF)
	}

	go thermostat.Run()

	tmpl, _ := template.New("windowForm").Parse(`<!DOCTYPE html>
<html>
<head>
    <title>Thermostat</title>
</head>
<body>
	<h2>{{.Events.GetLast.AmbientTemperature}}&#176; {{.Events.GetLast.Units}}</h2>
	<h3 style="text-transform: uppercase;">{{.Events.GetLast.Direction}}</h3>
    <form action="/thermostat/window" method="POST">
		<label for="high">High Temp</label>
        <input name="high" type="text" value="{{.HighTemp}}" />
		<br>
		<label for="low">Low Temp</label>
        <input name="low" type="text" value="{{.LowTemp}}" />
		<br>
        <button>Submit</button>
    </form>
</body>
</html>`)

	http.HandleFunc("/window", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			low, lowErr := strconv.ParseFloat(r.PostFormValue("low"), 64)
			if lowErr != nil {
				low = thermostat.LowTemp
			}

			high, highErr := strconv.ParseFloat(r.PostFormValue("high"), 64)
			if highErr != nil {
				high = thermostat.HighTemp
			}

			if low < high {
				log.Printf("Setting new temperature window. Low: %f, High: %f\n", low, high)
				thermostat.LowTemp = low
				thermostat.HighTemp = high
			}
		}
		log.Printf("Last Event: %f, %s", thermostat.Events.GetLast().AmbientTemperature, thermostat.Events.GetLast().Direction.String())

		tmpl.Execute(w, thermostat)
	})

	log.Println("Starting web server.")
	log.Fatal(http.ListenAndServe("127.0.0.1:9000", nil))
}
