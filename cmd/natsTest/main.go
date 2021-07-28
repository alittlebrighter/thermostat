package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/alittlebrighter/thermostat/models"
	nats "github.com/nats-io/nats.go"
)

const (
	natsSub = "otto.sensor.temperature.current"
)

var (
	ConfigPath         = "/etc/thermostat.conf"
	ThermometerSvcName = "thermometer"
	natsURL            = nats.DefaultURL
)

func main() {
	flag.StringVar(&ConfigPath, "config", ConfigPath, "Path to the configuration file to use.")
	flag.StringVar(&ThermometerSvcName, "thermometerSvc", ThermometerSvcName, "mDNS service name for the thermometer.")
	flag.StringVar(&natsURL, "natsUrl", natsURL, "Url for NATS instance to connect to.")
	flag.Parse()

	log.Println("Starting thermostat.")

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Println("Could not connect to message bus.")
	}
	defer nc.Close()
	log.Println("Connected to NATS.")

	nc.Subscribe(natsSub, func(m *nats.Msg) {
		update := new(models.SensorUpdate)
		if err := json.Unmarshal(m.Data, update); err != nil {
			log.Println("could not parse update from NATS")
		}

		log.Println("got update from NATS: value:", update.Value)
	})

	log.Println("Starting web server.")
	log.Fatal(http.ListenAndServe("localhost:9999", nil))
}
