package main

import (
	"github.com/tjclement/go-amazon-echo-hue-bridge"
)

func main() {
	devices := []bridge.HueDevice {
		bridge.NewEspDimmerChannel("Living Room 1", "http://1.1.1.1/", 13, bridge.TYPE_DIMMABLE),
	}

	web := bridge.NewWebServer(devices)
	upnp := bridge.NewUPnPServer()
	go web.Start()
	upnp.Start()
}