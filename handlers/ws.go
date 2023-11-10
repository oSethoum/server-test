package handlers

import (
	"app/utils"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type (
	Action = string
	Kind   = string

	InMessage struct {
		Action Action   `json:"action,omitempty"`
		Events []string `json:"events,omitempty"`
	}

	OutMessage struct {
		Kind  Kind   `json:"kind,omitempty"`
		Event string `json:"event,omitempty"`
		Error any    `json:"error,omitempty"`
		Data  any    `json:"data,omitempty"`
	}

	BusMessage struct {
		Event string `json:"event,omitempty"`
		Data  any    `json:"data,omitempty"`
	}
)

var (
	ActionSubscribe      = Action("subscribe")
	ActionUnsubscribe    = Action("unsubscribe")
	ActionUnsubscribeAll = Action("unsubscribeAll")
	ActionDisconnect     = Action("disconnect")

	KindConnected    = Kind("connected")
	KindDisconnected = Kind("disconnected")
	KindNotify       = Kind("notify")
	KindWarning      = Kind("warning")
	KindError        = Kind("error")

	mutex       = sync.Mutex{}
	subscribers = make(map[string]map[*websocket.Conn]bool, 0)
)

func subscribe(events []string, c *websocket.Conn) {
	mutex.Lock()
	for _, event := range events {
		m, ok := subscribers[event]
		if !ok {
			m = make(map[*websocket.Conn]bool)
		}
		m[c] = true
		subscribers[event] = m
	}
	mutex.Unlock()
}

func unsubscribe(events []string, c *websocket.Conn) {
	mutex.Lock()
	for _, event := range events {
		m, ok := subscribers[event]
		if ok {
			delete(m, c)
			if len(m) == 0 {
				delete(subscribers, event)
			} else {
				subscribers[event] = m
			}
		}
	}
	mutex.Unlock()
}

func Subscription(c *websocket.Conn) {
	c.WriteJSON(&OutMessage{
		Kind: KindConnected,
	})
	events := []string{}
	for {
		m := new(InMessage)
		err := c.ReadJSON(m)
		if err != nil {
			if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) {
				println("closed unexpectedly")
				unsubscribe(events, c)
				return
			}
			c.WriteJSON(OutMessage{
				Kind:  KindError,
				Error: err.Error(),
			})
			continue
		}
		switch m.Action {
		case ActionSubscribe:
			var appended = []string{}
			events, appended = utils.AppendValues(events, m.Events...)
			subscribe(appended, c)
		case ActionUnsubscribe:
			var removed = []string{}
			events, removed = utils.RemoveValues(events, m.Events...)
			unsubscribe(removed, c)
		case ActionUnsubscribeAll:
			unsubscribe(events, c)
			events = []string{}
		case ActionDisconnect:
			unsubscribe(events, c)
			return
		}
	}
}

func Broadcast(event string, data any) {
	mutex.Lock()
	if m, ok := subscribers[event]; ok {
		for ws := range m {
			go ws.WriteJSON(OutMessage{
				Kind:  KindNotify,
				Event: event,
				Data:  data,
			})
		}
	}
	mutex.Unlock()
}
