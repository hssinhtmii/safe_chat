package ws

import "log"

type Room struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Clients map[string]*Client `json:"clients"`
	Owner   string             `json:"owner"`
}

type Hub struct {
	Rooms      map[string]*Room
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
	Delete     chan *RemoveClient
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message, 5),
		Delete:     make(chan *RemoveClient),
	}
}

func (h *Hub) Run() {
	for {
		// add user in room
		select {
		// getting user information
		case cl := <-h.Register:
			// room does exist?
			if _, ok := h.Rooms[cl.RoomID]; ok {
				// room informaation
				r := h.Rooms[cl.RoomID]
				if _, ok := r.Clients[cl.ID]; !ok {
					r.Clients[cl.ID] = cl
				}
			}
			// exit the user from the room
		case cl := <-h.Unregister:
			// room does exist?
			if _, ok := h.Rooms[cl.RoomID]; ok {
				// user exist?
				if _, ok := h.Rooms[cl.RoomID].Clients[cl.ID]; ok {
					// is there any user?
					if len(h.Rooms[cl.RoomID].Clients) != 0 {
						h.Broadcast <- &Message{
							Content:  "user left the chat",
							RoomID:   cl.RoomID,
							Username: cl.Username,
						}
					}
					delete(h.Rooms[cl.RoomID].Clients, cl.ID)
					close(cl.Message)
				}
			}
			// send message to all users in room
		case m := <-h.Broadcast:
			if _, ok := h.Rooms[m.RoomID]; ok {
				// send for all clients
				for _, cl := range h.Rooms[m.RoomID].Clients {
					cl.Message <- m
				}
			}
		case dl := <-h.Delete:
			if _, ok := h.Rooms[dl.Removal.RoomID]; ok {
				if dl.Removal.Type == "member" {
					log.Fatal("Do Not Access")
				} else if dl.Removal.Type == "admin" {
					if dl.Deleted.Type == "owner" {
						log.Fatal("Do Not Access")
					}
				} else {
					delete(h.Rooms[dl.Deleted.RoomID].Clients, dl.Deleted.ID)
				}
			}
		}
	}
}
