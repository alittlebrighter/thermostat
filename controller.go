package main

import (
	"time"

	"github.com/stianeikeland/go-rpio"
)

type Control uint8

const (
	Off Control = iota
	Fan
	Heat
	Cool
)

const (
	on  = rpio.Low
	off = rpio.High
)

func StartController(heatPin int, coolPin int, fanPin int) (chan Control, error) {
	err := rpio.Open()
	if err != nil {
		return nil, err
	}

	heat := rpio.Pin(heatPin)
	cool := rpio.Pin(coolPin)
	fan := rpio.Pin(fanPin)

	heat.Output()
	cool.Output()
	fan.Output()
	fan.Write(off)
	heat.Write(off)
	cool.Write(off)

	commands := make(chan Control)
	fanCancel := make(chan bool)
	go func() {
		defer rpio.Close()
		defer fan.Write(off)
		defer heat.Write(off)
		defer cool.Write(off)

		fanCoolingDown := false

		for cmd := range commands {
			switch cmd {
			case Off:
				heat.Write(off)
				cool.Write(off)
				go fanCooldown(fan, fanCancel, &fanCoolingDown)
			case Fan:
				fan.Write(on)
				heat.Write(off)
				cool.Write(off)
				if fanCoolingDown {
					fanCancel <- true
				}
			case Heat:
				fan.Write(on)
				heat.Write(on)
				cool.Write(off)
				if fanCoolingDown {
					fanCancel <- true
				}
			case Cool:
				fan.Write(on)
				cool.Write(on)
				heat.Write(off)
				if fanCoolingDown {
					fanCancel <- true
				}
			}
		}
		return
	}()

	return commands, nil
}

func fanCooldown(fan rpio.Pin, cancel chan bool, coolingDown *bool) {
	*coolingDown = true

	timer := time.NewTimer(1 * time.Minute)
	select {
	case <-timer.C:
	case <-cancel:
	}
	timer.Stop()
	fan.Write(off)

	*coolingDown = false
}
