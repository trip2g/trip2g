package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int             `json:"message_id"`
	Chat      Chat            `json:"chat"`
	Text      string          `json:"text"`
	Entities  []MessageEntity `json:"entities"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type MessageEntity struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	CustomEmojiID string `json:"custom_emoji_id"`
}

type SendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	apiURL := "https://api.telegram.org/bot" + token

	log.Println("Emoji bot started. Send custom emoji to get codes.")

	offset := 0
	for {
		updates, err := getUpdates(apiURL, offset)
		if err != nil {
			log.Printf("Error getting updates: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			// Debug: log full message
			log.Printf("Message text: %q", update.Message.Text)
			log.Printf("Entities count: %d", len(update.Message.Entities))
			for i, entity := range update.Message.Entities {
				log.Printf("Entity %d: type=%s, custom_emoji_id=%s", i, entity.Type, entity.CustomEmojiID)
			}

			if update.Message.Text == "" && len(update.Message.Entities) == 0 {
				continue
			}

			// Check for custom emoji entities
			var emojiCodes []string
			for _, entity := range update.Message.Entities {
				if entity.Type == "custom_emoji" && entity.CustomEmojiID != "" {
					code := fmt.Sprintf("![emoji](tg://emoji?id=%s)", entity.CustomEmojiID)
					emojiCodes = append(emojiCodes, code)
				}
			}

			if len(emojiCodes) > 0 {
				response := "Obsidian markdown:\n\n"
				for _, code := range emojiCodes {
					response += code + "\n"
				}
				response += "\nTemplater snippet:\n```\n"
				for _, code := range emojiCodes {
					response += code + "\n"
				}
				response += "```"

				err := sendMessage(apiURL, update.Message.Chat.ID, response)
				if err != nil {
					log.Printf("Error sending message: %v", err)
				}
			} else {
				// No custom emoji found
				err := sendMessage(apiURL, update.Message.Chat.ID, "Send me a message with custom emoji, and I'll give you the code!")
				if err != nil {
					log.Printf("Error sending message: %v", err)
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func getUpdates(apiURL string, offset int) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30", apiURL, offset)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result.Result, nil
}

func sendMessage(apiURL string, chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", apiURL)

	payload := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
