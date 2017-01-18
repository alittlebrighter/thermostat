package main

import (
	"time"

	"github.com/stianeikeland/go-rpio"
)

const (
	on  = rpio.Low
	off = rpio.High
)

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

const (
	None ThermoDirection = iota
	Heating
	Cooling
)

type Controller struct {
	fan, heat, cool rpio.Pin

	fanCoolingDown bool
	fanCancel      chan bool
	Direction      ThermoDirection
}

func NewController(heatPin int, coolPin int, fanPin int) (*Controller, error) {
	err := rpio.Open()
	if err != nil {
		return nil, err
	}

	c := new(Controller)
	c.Direction = None

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

func (c *Controller) fanCooldown() {
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

func (c *Controller) Off() {
	c.Direction = None

	c.heat.Write(off)
	c.cool.Write(off)
	go c.fanCooldown()
}

func (c *Controller) Fan() {
	c.Direction = None

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.heat.Write(off)
	c.cool.Write(off)
}

func (c *Controller) Heat() {
	c.Direction = Heating

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.cool.Write(off)
	c.heat.Write(on)
}

func (c *Controller) Cool() {
	c.Direction = Cooling

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.fan.Write(on)
	c.cool.Write(off)
	c.heat.Write(on)
}

func (c *Controller) Shutdown() {
	c.Direction = None

	if c.fanCoolingDown {
		c.fanCancel <- true
	}

	c.cool.Write(off)
	c.heat.Write(off)
	c.fan.Write(off)

	rpio.Close()
}
