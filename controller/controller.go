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
