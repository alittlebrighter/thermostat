package controller

import (
	"time"

	"github.com/stianeikeland/go-rpio"
)

const (
	on  = rpio.Low
	off = rpio.High
)

// CentralController holds all of the data necessary to run a central HVAC system.
type CentralController struct {
	fan, heat, cool rpio.Pin

	fanCoolingDown bool
	fanCancel      chan bool
	direction      ThermoDirection
}

// NewCentralController initializes the controller for a central HVAC system.
func NewCentralController(heatPin int, coolPin int, fanPin int) (*CentralController, error) {
	err := rpio.Open()
	if err != nil {
		return nil, err
	}

	c := new(CentralController)
	c.direction = None

	c.heat = rpio.Pin(heatPin)
	c.cool = rpio.Pin(coolPin)
	c.fan = rpio.Pin(fanPin)

	c.heat.Output()
	c.cool.Output()
	c.fan.Output()

	c.fan.Write(off)
	c.heat.Write(off)
	c.cool.Write(off)

	c.fanCancel = make(chan bool)

	return c, nil
}

// Direction is a getter for the direction of the HVAC system.
func (c *CentralController) Direction() ThermoDirection {
	return c.direction
}

func (c *CentralController) fanCooldown() {
	c.fanCoolingDown = true

	timer := time.NewTimer(1 * time.Minute)
	select {
	case <-timer.C:
	case <-c.fanCancel:
	}
	timer.Stop()
	c.fan.Write(off)

	c.fanCoolingDown = false
}

// Off shuts down all HVAC components.
func (c *CentralController) Off() {
	c.heat.Write(off)
	c.cool.Write(off)
	if c.Direction() == Heating || c.Direction() == Cooling {
		go c.fanCooldown()
	} else {
		c.fan.Write(off)
	}

	c.direction = None
}

// Fan turns on the central fan while shutting down heating and cooling elements.
func (c *CentralController) Fan() {
	c.direction = Fan

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.heat.Write(off)
	c.cool.Write(off)
}

// Heat turns on the heating element and central fan.
func (c *CentralController) Heat() {
	c.direction = Heating

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.cool.Write(off)
	c.heat.Write(on)
}

// Cool turns on the air conditioner and central fan.
func (c *CentralController) Cool() {
	c.direction = Cooling

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.cool.Write(on)
	c.heat.Write(off)
}

// Shutdown turns off all HVAC components and closes the GPIO connection.
func (c *CentralController) Shutdown() {
	c.direction = None

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.cool.Write(off)
	c.heat.Write(off)
	c.fan.Write(off)

	rpio.Close()
}
