package ws

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/pbkdf2"
	"io"
	"log"
	"time"
)

type Client struct {
	Conn     *websocket.Conn
	Message  chan *Message
	ID       string `json:"id"`
	RoomID   string `json:"roomId"`
	Username string `json:"username"`
	Type     string `json:"Type"`
}

type RemoveClient struct {
	Removal *Client
	Deleted *Client
}

type Message struct {
	Content  string    `json:"content"`
	RoomID   string    `json:"roomId"`
	Username string    `json:"username"`
	Time     time.Time `json:"time"`
}

func GenerateKey(first_in, second_in []byte, iterations int) []byte {
	key := pbkdf2.Key(first_in, second_in, iterations, 32, sha256.New)
	return key
}

func Encrypt(key []byte, message []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nil, nonce, message, nil)
	encryptedMessage := append(nonce, ciphertext...)
	return hex.EncodeToString(encryptedMessage), nil
}

func Decrypt(key []byte, encryptedMessage string) ([]byte, error) {
	encryptedData, err := hex.DecodeString(encryptedMessage)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(encryptedData) < aesGCM.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := encryptedData[:aesGCM.NonceSize()], encryptedData[aesGCM.NonceSize():]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (c *Client) writeMessage() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		message, ok := <-c.Message
		message.Time = time.Now()
		if !ok {
			return
		}
		encryptedMessage, err := Encrypt(GenerateKey([]byte(c.ID), []byte(c.Username), 10000), []byte(message.Content))
		if err != nil {
			log.Printf("error encrypting message: %v", err)
			continue
		}
		c.Conn.WriteJSON(encryptedMessage)
	}
}

func (c *Client) readMessage(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		var encryptedMessage string
		err := c.Conn.ReadJSON(&encryptedMessage)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		plaintext, err := Decrypt(GenerateKey([]byte(c.ID), []byte(c.Username), 10000), encryptedMessage)
		if err != nil {
			log.Printf("error decrypting message: %v", err)
			continue
		}

		tmp := <- c.Message
		msg := &Message{
			Content:  string(plaintext),
			RoomID:   c.RoomID,
			Username: c.Username,
			Time:     tmp.Time,
		}
		hub.Broadcast <- msg
	}
}
