package apis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/Mariiana15/serverutils"
	"github.com/gorilla/websocket"
)

type WebsocketHQA struct {
	Protocol string   `json:"protocol"`
	Token    string   `json:"token"`
	Hosts    []string `json:"hosts"`
}

var upgrader = websocket.Upgrader{ReadBufferSize: 1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 60000, // <-- add this line
}

type Message struct {
	Message Params `json:"message"`
	Token   string `json:"token"`
	Type    string `json:"type"`
	State   string `json:"state"`
	User    string `json:"userId"`
}
type Params struct {
	Token    string `json:"token"`
	ObjectId string `json:"objectId"`
	Body     string `json:"body"`
}

type ErrorWS struct {
	Message string `json:"message"`
	State   string `json:"state"`
}

func (ws *WebsocketHQA) NewWebSocketHQA() {

	path, _ := filepath.Abs("../configuration/config.json")
	file, _ := ioutil.ReadFile(path)
	var result map[string]interface{}
	json.Unmarshal([]byte(file), &result)
	byteData, _ := json.Marshal(result["websocket"])
	json.Unmarshal(byteData, &ws)
}

func handlerExceptionWS(ws *websocket.Conn, message string, state string) {
	err := ErrorWS{message, state}
	ws.WriteJSON(err)
	ws.Close()
}

func NewWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {

	var wsHQA WebsocketHQA
	wsHQA.NewWebSocketHQA()

	upgrader.Subprotocols = append(upgrader.Subprotocols, protocolBD)
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return serverutils.Contains(wsHQA.Hosts, r.Header.Get("origin"))
	}

	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		handlerExceptionWS(ws, err.Error(), serverutils.StateKO)
		log.Printf("Error: %v", err)
		return nil, err
	}

	log.Println("Connected!")
	listenMessageWS(ws)

	return ws, nil
}

func SendMessageJsonWS(ws *websocket.Conn, message string) {

	if err := ws.WriteJSON(message); err != nil {
		handlerExceptionWS(ws, err.Error(), serverutils.StateKO)
		log.Printf("Error: %v", err)
	}
}

func SendMessageWS(ws *websocket.Conn, message string) {

	byteData, _ := json.Marshal(message)
	if err := ws.WriteMessage(1, byteData); err != nil {
		handlerExceptionWS(ws, err.Error(), serverutils.StateKO)
		log.Printf("Error: %v", err)
	}

}

func GetMessageJsonWS(ws *websocket.Conn) Message {

	defer ws.Close()
	message := listenMessageWS(ws)
	return message
}

func listenMessageWS(ws *websocket.Conn) Message {
	var message Message
	for {
		_, message2, _ := ws.ReadMessage()
		//err := ws.ReadJSON(&message)
		err := json.Unmarshal([]byte(message2), &message)
		if err != nil {
			SendMessageJsonWS(ws, fmt.Sprintf("{\"error\": \"%v\"}", serverutils.MsgDataCorrupt))
			ws.Close()
			break
		}
		err = serverutils.TokenValidWS(message.Token)
		if err != nil {
			SendMessageJsonWS(ws, fmt.Sprintf("{\"error\": \"%v\"}", err.Error()))
			ws.Close()
			break
		}
		acc, err2 := serverutils.ExtractTokenMetadataWS(message.Token)
		if err2 != nil {
			SendMessageJsonWS(ws, fmt.Sprintf("{\"error\": \"%v\"}", err.Error()))
			ws.Close()
			break
		}
		message.User = acc.UserId

		if message.State != "" {
			log.Println("Validate---!")
			Dispatcher(ws, message)
			break
		}
	}
	return message
}
