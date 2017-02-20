package controller

type Controller interface {
	Direction() ThermoDirection
	Off()
	Fan()
	Cool()
	Heat()
	Shutdown()
}

type ThermoDirection uint8

const (
	None ThermoDirection = iota
	Heating
	Cooling
)

func (d ThermoDirection) String() string {
	switch d {
	case Heating:
		return "heating"
	case Cooling:
		return "cooling"
	default:
		return "none"
	}
}

func (d ThermoDirection) MarshalText() (text []byte, err error) {
	return []byte(d.String()), nil
}
