package bridge

import (
	"github.com/tjclement/gossdp"
	"net"
	"log"
)

func NewUPnPServer() *UPnPServer {
	return &UPnPServer{}
}

type UPnPServer struct {

}

func (server UPnPServer) Start() {
	ssdp, _ := gossdp.NewSsdp(server)
	defer ssdp.Stop()

	ssdp.AdvertiseServer(gossdp.AdvertisableServer{
		ServiceType: "urn:schemas-upnp-org:device:basic:1",
		DeviceUuid: "2f402f80-da50-11e1-9b23-00178829d301",
		Location: "http://" + GetOutboundIP().String() + ":8080/description.xml",
		MaxAge: 5,
		CustomResponseHeaders: map[string]string{
			"hue-bridgeid": "001788FFFE29D301",
		},
	})

	ssdp.Start()
}

func (server UPnPServer) NotifyAlive(message gossdp.AliveMessage) {
	//fmt.Println(message)
}

func (server UPnPServer) NotifyBye(message gossdp.ByeMessage) {
	//fmt.Println(message)
}

func (server UPnPServer) Response(message gossdp.ResponseMessage) {
	//fmt.Println(message)
}

// Get preferred outbound (local) ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}