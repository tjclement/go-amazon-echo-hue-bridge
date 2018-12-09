package bridge

import (
	"net/http"
	"io"
	"github.com/julienschmidt/httprouter"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"encoding/json"
)

type WebServer struct {
	server *http.Server
	devices []HueDevice
}

func NewWebServer(devices []HueDevice) *WebServer {
	server := &WebServer{}
	server.devices = devices
	return server
}

func (web WebServer) Start() error {
	router := httprouter.New()
	router.GET("/description.xml", web.handleDescription)
	router.GET("/api/:user/lights", web.handleListLights)
	router.GET("/api/:user/lights/:id", web.handleShowLight)
	router.GET("/api/:user/lights/:id/state", web.handleShowLightState)
	router.PUT("/api/:user/lights/:id/state", web.handleSetLightState)
	router.NotFound = NotFound{}

	web.server = &http.Server{Addr: ":8080", Handler: router}
	return web.server.ListenAndServe()
}

func (web WebServer) Stop()  {
	web.server.Close();
}

type NotFound struct{}

func (n NotFound) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Not found!")
	fmt.Printf("Got request (%s) from: %s\n", r.URL, r.RemoteAddr)
}

func (web WebServer) handleDescription(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Printf("Got description request from: %s\n", r.RemoteAddr)

	b, _ := ioutil.ReadFile("description.xml")
	contents := strings.Replace(string(b), "##URLBASE##", GetOutboundIP().String() + ":8080", 2)
	w.Header().Add("Content-Type", "application/xml")
	io.WriteString(w, contents)
}

func (web WebServer) handleListLights(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Printf("Got list request from: %s\n", r.RemoteAddr)
	devicesList := "{\n"
	for key, value := range web.devices {
		devicesList += fmt.Sprintf(`"%d": %s`, key+1, value)
		if len(web.devices) > 1 && key + 1 < len(web.devices) {
			devicesList += ",\n"
		} else {
			devicesList += "\n"
		}
	}
	devicesList += "}"
	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, devicesList)
}

func (web WebServer) handleShowLight(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	index, err := strconv.ParseUint(p.ByName("id"), 10, 8)

	if err != nil || len(web.devices) < int(index - 1) {
		io.WriteString(w, "Invalid request: malformed light ID")
		return
	}

	fmt.Printf("Got show request from: %s for light %d\n", r.RemoteAddr, index)

	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, web.devices[index-1].String())
}

func (web WebServer) handleShowLightState(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Printf("Got show state request from: %s\n", r.RemoteAddr)
	index, err := strconv.ParseUint(p.ByName("id"), 10, 8)

	if err != nil || len(web.devices) < int(index - 1) {
		io.WriteString(w, "Invalid request: malformed light ID")
		return
	}

	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, fmt.Sprintf("{%s}", web.devices[index-1].State().String()))
}

func (web WebServer) handleSetLightState(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Printf("Got set state request from: %s\n", r.RemoteAddr)
	index, err := strconv.ParseUint(p.ByName("id"), 10, 8)

	if err != nil || len(web.devices) < int(index - 1) {
		io.WriteString(w, "Invalid request: malformed light ID")
		return
	}

	command := parseEchoCommand(r.Body)

	device := web.devices[index-1]
	state := HueState{On: command.On, Brightness: command.Brightness}
	device.SetState(state)

	fmt.Println("Resp")
	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, fmt.Sprintf(
		`[{"success":{"/lights/%d/state/bri":%d}}, {"success":{"/lights/%d/state/on":%t}}]`,
		index, command.Brightness, index, command.On))
}

func parseEchoCommand(input io.ReadCloser) EchoCommand {
	body, err := ioutil.ReadAll(input)
	defer input.Close()

	if err != nil {
		panic(err)
	}

	cont := string(body)
	fmt.Println(cont)
	var command EchoCommand
	err = json.Unmarshal(body, &command)

	if err != nil {
		panic(err)
	}

	if command.On && command.Brightness == 0 {
		command.Brightness = 255
	}

	return command
}

type EchoCommand struct {
	On bool `json:"on"`
	Brightness uint8 `json:"bri"`
}