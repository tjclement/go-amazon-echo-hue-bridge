package bridge

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
	"math"
	"net"
)

const TIMEOUT = 15

type HueState struct {
	On         bool
	Brightness byte
	Hue        int16
	Saturation byte
	XY         []float32
	CT         int16
	Alert      string
	Effect     string
	ColorMode  string
	Reachable  bool
}

func (state HueState) String() string {
	var on string
	var reachable string

	if state.On {
		on = "true"
	} else {
		on = "false"
	}

	if state.Reachable {
		reachable = "true"
	} else {
		reachable = "false"
	}

	return fmt.Sprintf(`"state": {
"on": %s,
"bri": %d,
"hue": %d,
"sat": %d,
"ct": %d,
"alert": "%s",
"effect": "%s",
"reachable": %s
}`, on, state.Brightness, state.Hue, state.Saturation, state.CT,
	state.Alert, state.Effect, reachable)
}

type HueProperties struct {
	Name      string
	Type      string
	UniqueID   string
	ModelID   string
	SWVersion string
	Manufacturer string
}

//"on": true,
//"bri": 144,
//"hue": 13088,
//"sat": 212,
//"xy": [0.5128,0.4147],
//"ct": 467,
//"alert": "none",
//"effect": "none",
//"colormode": "xy",
//"reachable": true

type LightType int

const (
	TYPE_SWITCHABLE LightType = iota
	TYPE_DIMMABLE   LightType = iota
)

type HueDevice interface {
	State() HueState
	Props() HueProperties
	Update() error
	SetState(HueState) error
	String() string
}

type EspDimmerChannel struct {
	HueDevice
	// Full URI to device API, e.g. "http://192.168.1.200/"
	Address string
	// GPIO number of the target channel
	GPIO  int
	state HueState
	props HueProperties
	lightType LightType
}

func (dev EspDimmerChannel) State() HueState {
	return dev.state
}

func (dev EspDimmerChannel) Props() HueProperties {
	return dev.props
}

func (dev EspDimmerChannel) String() string {
	return fmt.Sprintf(`{
"name": "%s",
"type": "%s",
"modelid": "%s",
"uniqueid": "%s",
"swversion": "%s",
"manufacturername": "%s",
%s}`, dev.props.Name, dev.props.Type, dev.props.ModelID, dev.props.UniqueID, dev.props.SWVersion, dev.props.Manufacturer, dev.State())
}

func (esp EspDimmerChannel) Update() error {
	url := fmt.Sprintf("%sgetPwmDuty?gpio=%s", esp.Address, esp.GPIO)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		esp.state.Reachable = false
		return err
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		esp.state.Reachable = false
		return err
	}

	duty, err := strconv.ParseInt(string(contents), 10, 16)
	if err != nil {
		esp.state.Reachable = false
		return err
	}

	esp.state.Brightness = byte(duty / (1024 / 256))
	esp.state.Reachable = true
	if esp.state.Brightness > 0 {
		esp.state.On = true
	} else {
		esp.state.On = false
	}
	return nil
}

func (esp EspDimmerChannel) SetState(state HueState) error {
	url := ""
	newDuty := uint16(0)

	if esp.lightType == TYPE_DIMMABLE {
		newDuty = uint16(math.Ceil(float64(state.Brightness) * (float64(1023) / 255)))
		url = fmt.Sprintf("%sfadePwmDuty?gpio=%d&duty=%d", esp.Address, esp.GPIO, newDuty)
	} else {
		// Switch/toggle instead of fade
		newDuty = 1023
		if state.Brightness < 127 {
			newDuty = 0
		}
		url = fmt.Sprintf("%ssetPwmDuty?gpio=%d&duty=%d", esp.Address, esp.GPIO, newDuty)
	}

	fmt.Printf("Set to %d\n", newDuty)

	client := http.Client{Timeout: TIMEOUT * time.Second, Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
		Timeout:   TIMEOUT * time.Second,
		KeepAlive: TIMEOUT * time.Second,
		DualStack: true,
	}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       TIMEOUT * time.Second,
		TLSHandshakeTimeout:   TIMEOUT * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	contents, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	if !strings.Contains(string(contents), "Ok") {
		return fmt.Errorf("got erroneous response: %s", string(contents))
	}

	return nil
}

func NewEspDimmerChannel(name string, address string, gpio int, lightType LightType) *EspDimmerChannel {
	dimmer := &EspDimmerChannel{}
	dimmer.lightType = lightType
	dimmer.Address = address
	dimmer.GPIO = gpio
	dimmer.props.Name = name
	dimmer.props.Type = "Dimmable light"
	dimmer.props.UniqueID = "00:17:88:5E:D3:FF-01"
	dimmer.props.ModelID = "LCT010"
	dimmer.props.SWVersion = "66012040"
	dimmer.props.Manufacturer = "Philips"
	dimmer.state.XY = []float32{0.5, 0.5}
	dimmer.state.ColorMode = "hs"
	dimmer.state.Alert = "none"
	dimmer.state.Effect = "none"
	dimmer.state.Reachable = true
	return dimmer
}
