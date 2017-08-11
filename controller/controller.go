package controller

// Controller defines a struct that is capable of performing all of the necessary actions to change the temperature.
type Controller interface {
	Direction() ThermoDirection
	Off()
	Fan()
	Cool()
	Heat()
	Shutdown()
}

// ThermoDirection defines what a controller is currently doing.
type ThermoDirection uint8

const (
	None ThermoDirection = iota
	Heating
	Cooling
	Fan
)

func (d ThermoDirection) String() string {
	switch d {
	case Heating:
		return "heating"
	case Cooling:
		return "cooling"
	case Fan:
		return "fan"
	default:
		return "none"
	}
}

func (d ThermoDirection) MarshalText() (text []byte, err error) {
	return []byte(d.String()), nil
}

type Config struct {
	Pins struct{ Fan, Cool, Heat int }
}
