# embd [![Build Status](https://travis-ci.org/alittlebrighter/embd.svg?branch=master)](https://travis-ci.org/alittlebrighter/embd) [![GoDoc](http://godoc.org/github.com/alittlebrighter/embd?status.png)](http://godoc.org/github.com/alittlebrighter/embd)

**embd** is a hardware abstraction layer (HAL) for embedded systems.

It allows you to start your hardware hack on easily available hobby boards (like the Raspberry Pi, BeagleBone Black, etc.) by giving you staight forward access to the board's capabilities as well as a plethora of **sensors** (like accelerometers, gyroscopes, thermometers, etc.) and **controllers** (PWM generators, digital-to-analog convertors) for which we have written drivers. And when things get serious, you dont have to throw away the code. You carry forward the effort onto more custom designed boards where the HAL abstraction of EMBD will save you precious time.

Development supported and sponsored by [**SoStronk**](https://www.sostronk.com) and [**ThoughtWorks**](http://www.thoughtworks.com/) 

Also, you might be interested in: [Why Golang?](https://github.com/alittlebrighter/embd/wiki/Why-Go)

[Blog post introducing EMBD](http://kidoman.io/framework/embd.html)

## Getting Started

After installing Go* and setting up your [GOPATH](http://golang.org/doc/code.html#GOPATH), create your first .go file. We'll call it ```simpleblinker.go```.

```go
package main

import (
	"time"

	"github.com/alittlebrighter/embd"
	_ "github.com/alittlebrighter/embd/host/rpi" // This loads the RPi driver
)

func main() {
	for {
		embd.LEDToggle("LED0")
		time.Sleep(250 * time.Millisecond)
	}
}
```

Then install the EMBD package (go1.2 and greater is required):

	$ go get github.com/alittlebrighter/embd

Build the binary*:

	$ export GOOS=linux
	$ export GOARCH=arm
	$ go build simpleblinker.go

Copy the cross-compiled binary to your RaspberryPi*:

	$ scp simpleblinker pi@192.168.2.2:~

Then run the program with ```sudo```*:

	$ sudo ./simpleblinker

**You will now see the green LED (next to the always on power LED) blink every 1/4 sec.**

**<nowiki>*</nowiki> Notes**

* Please install the cross compilers. Mac users: ```brew install go --cross-compile-common```. [goxc](https://github.com/laher/goxc) can be a big help as well
* We are instructing the ```go``` compiler to create a binary which will run on the RaspberryPi processor
* Assuming your RaspberryPi has an IP address of ```192.168.2.2```. Substitute as necessary
* ```sudo``` (root) permission is required as we are controlling the hardware by writing to special files
* This sample program is optimized for brevity and does not clean up after itself. Click here to see the [full version](https://github.com/alittlebrighter/embd/blob/master/samples/fullblinker.go)

## Getting Help

Join the [mailing list](https://groups.google.com/forum/#!forum/go-embd)

## Platforms Supported

* [RaspberryPi](http://www.raspberrypi.org/) (including [A+](http://www.raspberrypi.org/products/model-a-plus/) and [B+](http://www.raspberrypi.org/products/model-b-plus/))
* [RaspberryPi 2](http://www.raspberrypi.org/)
* [BeagleBone Black](http://beagleboard.org/Products/BeagleBone%20Black)
* [Intel Edison](http://www.intel.com/content/www/us/en/do-it-yourself/galileo-maker-quark-board.html) **coming soon**
* [Radxa](http://radxa.com/) **coming soon**
* [Cubietruck](http://www.cubietruck.com/) **coming soon**
* Bring Your Own **coming soon**

## The command line tool

	go get github.com/alittlebrighter/embd/embd

will install a command line utility ```embd``` which will allow you to quickly get started with prototyping. The binary should be available in your ```$GOPATH/bin```. However, to be able to run this on a ARM based device, you will need to build it with ```GOOS=linux``` and ```GOARCH=arm``` environment variables set.

But, since I am feeling so generous, a prebuilt/tested version is available for direct download and deployment [here](https://dl.dropboxusercontent.com/u/6727135/Binaries/embd/linux-arm/embd).

For example, if you run ```embd detect``` on a **BeagleBone Black**:

	root@beaglebone:~# embd detect

	detected host BeagleBone Black (rev 0)

Run ```embd``` without any arguments to discover the various commands supported by the utility.

## How to use the framework

Package **embd** provides a hardware abstraction layer for doing embedded programming
on supported platforms like the Raspberry Pi and BeagleBone Black. Most of the examples below
will work without change (i.e. the same binary) on all supported platforms. How cool is that?

Although samples are all present in the [samples](https://github.com/alittlebrighter/embd/tree/master/samples) folder,
we will show a few choice examples here.

Use the **LED** driver to toggle LEDs on the BBB:

```go
import "github.com/alittlebrighter/embd"
import _ "github.com/alittlebrighter/embd/host/all"
...
embd.InitLED()
defer embd.CloseLED()
...
led, err := embd.NewLED(3)
...
led.Toggle()
```

Even shorter when quickly trying things out:

```go
import "github.com/alittlebrighter/embd"
import _ "github.com/alittlebrighter/embd/host/all"
...
embd.InitLED()
defer embd.CloseLED()
...
embd.ToggleLED(3)
```

**3** is the same as **USR3** for all intents and purposes. The driver is smart enough to figure all this out.

BBB + **PWM**:

```go
import "github.com/alittlebrighter/embd"
import _ "github.com/alittlebrighter/embd/host/all"
...
embd.InitGPIO()
defer embd.CloseGPIO()
...
pwm, _ := embd.NewPWMPin("P9_14")
defer pwm.Close()
...
pwm.SetDuty(1000)
```

Control **GPIO** pins on the RaspberryPi / BeagleBone Black:

```go
import "github.com/alittlebrighter/embd"
import _ "github.com/alittlebrighter/embd/host/all"
...
embd.InitGPIO()
defer embd.CloseGPIO()
...
embd.SetDirection(10, embd.Out)
embd.DigitalWrite(10, embd.High)
```

Could also do:

```go
import "github.com/alittlebrighter/embd"
import _ "github.com/alittlebrighter/embd/host/all"
...
embd.InitGPIO()
defer embd.CloseGPIO()
...
pin, err := embd.NewDigitalPin(10)
...
pin.SetDirection(embd.Out)
pin.Write(embd.High)
```

Or read data from the **Bosch BMP085** barometric sensor:

```go
import "github.com/alittlebrighter/embd"
import "github.com/alittlebrighter/embd/sensor/bmp085"
import _ "github.com/alittlebrighter/embd/host/all"
...
bus := embd.NewI2CBus(1)
...
baro := bmp085.New(bus)
...
temp, err := baro.Temperature()
altitude, err := baro.Altitude()
```

Even find out the heading from the **LSM303** magnetometer:

```go
import "github.com/alittlebrighter/embd"
import "github.com/alittlebrighter/embd/sensor/lsm303"
import _ "github.com/alittlebrighter/embd/host/all"
...
bus := embd.NewI2CBus(1)
...
mag := lsm303.New(bus)
...
heading, err := mag.Heading()
```

The above two examples depend on **I2C** and therefore will work without change on almost all
platforms.

## Protocols Supported

* **Digital GPIO** [Documentation](http://godoc.org/github.com/alittlebrighter/embd#DigitalPin)
* **Analog GPIO** [Documentation](http://godoc.org/github.com/alittlebrighter/embd#AnalogPin)
* **PWM** [Documentation](http://godoc.org/github.com/alittlebrighter/embd#PWMPin)
* **I2C** [Documentation](http://godoc.org/github.com/alittlebrighter/embd#I2CBus)
* **LED** [Documentation](http://godoc.org/github.com/alittlebrighter/embd#LED)
* **SPI** [Documentation](http://godoc.org/github.com/alittlebrighter/embd#SPIBus)

## Sensors Supported

* **TMP006** Thermopile sensor [Documentation](http://godoc.org/github.com/alittlebrighter/embd/sensor/tmp006), [Datasheet](http://www.adafruit.com/datasheets/tmp006.pdf), [Userguide](http://www.adafruit.com/datasheets/tmp006ug.pdf)

* **BMP085** Barometric pressure sensor [Documentation](http://godoc.org/github.com/alittlebrighter/embd/sensor/bmp085), [Datasheet](https://www.sparkfun.com/datasheets/Components/General/BST-BMP085-DS000-05.pdf)

* **BMP180** Barometric pressure sensor [Documentation](http://godoc.org/github.com/alittlebrighter/embd/sensor/bmp180), [Datasheet](http://www.adafruit.com/datasheets/BST-BMP180-DS000-09.pdf)

* **LSM303** Accelerometer and magnetometer [Documentation](http://godoc.org/github.com/alittlebrighter/embd/sensor/lsm303), [Datasheet](https://www.sparkfun.com/datasheets/Sensors/Magneto/LSM303%20Datasheet.pdf)

* **L3GD20** Gyroscope [Documentation](http://godoc.org/github.com/alittlebrighter/embd/sensor/l3gd20), [Datasheet](http://www.adafruit.com/datasheets/L3GD20.pdf)

* **US020** Ultrasonic proximity sensor [Documentation](http://godoc.org/github.com/alittlebrighter/embd/sensor/us020), [Product Page](http://www.digibay.in/sensor/object-detection-and-proximity?product_id=239)

* **BH1750FVI** Luminosity sensor [Documentation](http://godoc.org/github.com/alittlebrighter/embd/sensor/bh1750fvi), [Datasheet](http://www.elechouse.com/elechouse/images/product/Digital%20light%20Sensor/bh1750fvi-e.pdf)

## Interfaces

* **Keypad(4x3)** [Product Page](http://www.adafruit.com/products/419#Learn)

## Controllers

* **PCA9685** 16-channel, 12-bit PWM Controller with I2C protocol [Documentation](http://godoc.org/github.com/alittlebrighter/embd/controller/pca9685), [Datasheet](http://www.adafruit.com/datasheets/PCA9685.pdf), [Product Page](http://www.adafruit.com/products/815)

* **MCP4725** 12-bit DAC [Documentation](http://godoc.org/github.com/alittlebrighter/embd/controller/mcp4725), [Datasheet](http://www.adafruit.com/datasheets/mcp4725.pdf), [Product Page](http://www.adafruit.com/products/935)

* **ServoBlaster** RPi PWM/PCM based PWM controller [Documentation](http://godoc.org/github.com/alittlebrighter/embd/controller/servoblaster), [Product Page](https://github.com/richardghirst/PiBits/tree/master/ServoBlaster)

## Convertors

* **MCP3008** 8-channel, 10-bit ADC with SPI protocol, [Datasheet](https://www.adafruit.com/datasheets/MCP3008.pdf)

## Contributing

We look forward to your pull requests, but contributions which abide by the [guidelines](https://github.com/alittlebrighter/embd/blob/master/CONTRIBUTING.md) will get a free beer!

File an [issue](https://github.com/alittlebrighter/embd/issues), open a [pull request](https://github.com/alittlebrighter/embd/pulls). We are waiting.

## About

EMBD is affectionately designed/developed by Karan Misra ([kidoman](https://github.com/kidoman)), Kunal Powar ([kunalpowar](https://github.com/kunalpowar)) and [FRIENDS](https://github.com/alittlebrighter/embd/blob/master/AUTHORS). We also have a list of [CONTRIBUTORS](https://github.com/alittlebrighter/embd/blob/master/CONTRIBUTORS).
