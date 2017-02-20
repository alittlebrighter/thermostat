package controller

import (
	"time"

	"github.com/stianeikeland/go-rpio"
)

const (
	on  = rpio.Low
	off = rpio.High
)

type CentralController struct {
	fan, heat, cool rpio.Pin

	fanCoolingDown bool
	fanCancel      chan bool
	direction      ThermoDirection
}

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

func (c *CentralController) Off() {
	c.direction = None

	c.heat.Write(off)
	c.cool.Write(off)
	go c.fanCooldown()
}

func (c *CentralController) Fan() {
	c.direction = Fan

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.heat.Write(off)
	c.cool.Write(off)
}

func (c *CentralController) Heat() {
	c.direction = Heating

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.cool.Write(off)
	c.heat.Write(on)
}

func (c *CentralController) Cool() {
	c.direction = Cooling

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.cool.Write(off)
	c.heat.Write(on)
}

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
