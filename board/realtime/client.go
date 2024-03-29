package realtime

import (
	"log"

	"github.com/desertbit/glue"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/tryanzu/core/core/user"
	"github.com/tryanzu/core/deps"
	"gopkg.in/mgo.v2/bson"
)

// Client contains a message to be broadcasted to a channel
type Client struct {
	Raw      *glue.Socket
	Channels map[string]*glue.Channel
	User     *user.User
	Read     chan socketEvent
}

func (c *Client) readWorker() {
	for e := range c.Read {
		switch e.Event {
		case "auth":
			token, exists := e.Params["token"].(string)
			if !exists {
				log.Println("[REALTIME] Could not authenticate socket client: missing token")
				continue
			}

			signed, err := jwt.Parse(token, func(passed_token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil {
				log.Println("[REALTIME] Could not parse socket client token: ", err)
				continue
			}

			claims := signed.Claims.(jwt.MapClaims)
			usr, err := user.FindId(deps.Container, bson.ObjectIdHex(claims["user_id"].(string)))
			if err != nil {
				log.Println("[REALTIME] Could not find user from socket token: ", err)
				continue
			}

			c.User = &usr
			event := socketEvent{
				Event: "auth:my",
				Params: map[string]interface{}{
					"user": usr,
				},
			}
			c.SafeWrite(event.encode())

		case "auth:clean":
			c.User = nil
			c.SafeWrite(socketEvent{
				Event: "auth:cleaned",
			}.encode())

		case "listen":
			channel, exists := e.Params["chan"].(string)
			if !exists {
				log.Println("Could not join channel: missing id")
				continue
			}

			c.Channels[channel] = c.Raw.Channel(channel)
			c.SafeWrite(socketEvent{
				Event: "listen:ready",
				Params: map[string]interface{}{
					"chan": channel,
				},
			}.encode())
		case "unlisten":
			channel, exists := e.Params["chan"].(string)
			if !exists {
				log.Println("Could not remove channel: missing id")
				continue
			}

			delete(c.Channels, channel)

			c.SafeWrite(socketEvent{
				Event: "unlisten:ready",
				Params: map[string]interface{}{
					"chan": channel,
				},
			}.encode())
		}
	}
}

func (c *Client) SafeWrite(data string) {
	if c.Raw != nil {
		c.Raw.Write(data)
	}
}

func (c *Client) send(packed []M) {
	for _, m := range packed {
		if m.Channel == "" {
			c.SafeWrite(m.Content)
			continue
		}
		if m.Channel[0:4] == "user" {
			if c.User == nil {
				continue
			}
			id := bson.ObjectIdHex(m.Channel[5:])
			if id.Valid() == false {
				log.Println("Invalid userId in packed messages sending. Chan:", m.Channel)
				continue
			}
			if id == c.User.Id {
				c.SafeWrite(m.Content)
			}
		}

		if c, exists := c.Channels[m.Channel]; exists {
			c.Write(m.Content)
		}
	}
}
