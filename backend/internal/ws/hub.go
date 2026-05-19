package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type AgentConn struct {
	Conn *websocket.Conn
	Send chan []byte
	ID   string
}

type HubType struct {
	Agents     map[string]*AgentConn
	Mu         sync.RWMutex
	Broadcast  chan []byte
	Register   chan *AgentConn
	Unregister chan *AgentConn
}

var Hub = &HubType{
	Agents:     make(map[string]*AgentConn),
	Broadcast:  make(chan []byte),
	Register:   make(chan *AgentConn),
	Unregister: make(chan *AgentConn),
}

func (h *HubType) Lock()    { h.Mu.Lock() }
func (h *HubType) Unlock()  { h.Mu.Unlock() }
func (h *HubType) RLock()   { h.Mu.RLock() }
func (h *HubType) RUnlock() { h.Mu.RUnlock() }

func Run() {
	for {
		select {
		case conn := <-Hub.Register:
			Hub.Mu.Lock()
			Hub.Agents[conn.ID] = conn
			Hub.Mu.Unlock()
			log.Printf("Agent %s connected", conn.ID)
		case conn := <-Hub.Unregister:
			Hub.Mu.Lock()
			if _, ok := Hub.Agents[conn.ID]; ok {
				delete(Hub.Agents, conn.ID)
				close(conn.Send)
				log.Printf("Agent %s disconnected", conn.ID)
			}
			Hub.Mu.Unlock()
		case msg := <-Hub.Broadcast:
			Hub.Mu.RLock()
			for _, conn := range Hub.Agents {
				select {
				case conn.Send <- msg:
				default:
					close(conn.Send)
					delete(Hub.Agents, conn.ID)
				}
			}
			Hub.Mu.RUnlock()
		}
	}
}

func SendToAgent(agentID string, msg interface{}) error {
	Hub.Mu.RLock()
	defer Hub.Mu.RUnlock()
	if conn, ok := Hub.Agents[agentID]; ok {
		data, _ := json.Marshal(msg)
		conn.Send <- data
		return nil
	}
	return nil
}

func (ac *AgentConn) ReadPump() {
	defer func() {
		Hub.Unregister <- ac
		ac.Conn.Close()
	}()
	ac.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	ac.Conn.SetPongHandler(func(string) error { ac.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, msg, err := ac.Conn.ReadMessage()
		if err != nil {
			break
		}
		var packet map[string]interface{}
		json.Unmarshal(msg, &packet)
		log.Printf("Recv from %s: %v", ac.ID, packet)
	}
}

func (ac *AgentConn) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() { ticker.Stop(); ac.Conn.Close() }()
	for {
		select {
		case message := <-ac.Send:
			ac.Conn.WriteMessage(websocket.TextMessage, message)
		case <-ticker.C:
			ac.Conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}
