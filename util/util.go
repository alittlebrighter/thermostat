package util

import (
	"time"

	"github.com/alittlebrighter/thermostat/controller"
)

type TemperatureUnits string

const (
	Celsius    TemperatureUnits = "Celsius"
	Fahrenheit                  = "Fahrenheit"
)

// TempCToF converts temperature degrees from Celsius to Fahrenheit
func TempCToF(tempC float64) float64 {
	return tempC*9/5 + 32
}

// TempFToC converts temperature degrees from Fahrenheit to Celsius
func TempFToC(tempF float64) float64 {
	return (tempF - 32) * 5 / 9
}

type ClockTime time.Time

func (t *ClockTime) UnmarshalJSON(data []byte) error {
	realTime, err := time.Parse(`"`+time.Kitchen+`"`, string(data))
	*t = ClockTime(realTime)
	return err
}

func (t *ClockTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.Kitchen)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, time.Kitchen)
	b = append(b, '"')
	return b, nil
}

func (t ClockTime) Hour() int {
	return time.Time(t).Hour()
}

func (t ClockTime) Minute() int {
	return time.Time(t).Minute()
}

func (t ClockTime) AppendFormat(dat []byte, format string) []byte {
	return time.Time(t).AppendFormat(dat, format)
}

type EventLog struct {
	AmbientTemperature float64                    `json:"ambientTemperature"`
	Units              TemperatureUnits           `json:"units"`
	Direction          controller.ThermoDirection `json:"direction"`
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
