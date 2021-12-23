package main

import (
	"log"

	"go.mongodb.org/mongo-driver/bson"
)

// getUpdates Models
type GetUpdatesResp struct {
	Ok      bool     `json:"ok"`
	Updates []Update `json:"result"`
}

type Update struct {
	UpdateId int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessagId int    `json:"message_id"`
	Chat     Chat   `json:"chat"`
	Text     string `json:"text"`
}

type Chat struct {
	ChatId   int    `json:"id"`
	Username string `json:"username"`
}

// bank Model
type Bank struct {
	Account int
	Owner   string
	Name    string
	Balance int
}

func (bank *Bank) createBank() error {
	_, err := collection.InsertOne(ctx, bank)
	return err
}

// keyboard Models
type ReplyKeyboard struct {
	Keyboard       [][]string `json:"keyboard"`
	Resize         bool       `json:"resize_keyboard"`
	OneTime        bool       `json:"one_time_keyboard"`
	RemoveKeyboard bool       `json:"remove_keyboard"`
}

func (rk *ReplyKeyboard) createBanksKeyboard(chatId int, command string) error {
	var replyKeyboardRow []string

	banks, err := collection.Find(ctx, bson.M{"account": chatId})
	if err != nil {
		return err
	}
	defer banks.Close(ctx)

	for banks.Next(ctx) {
		var bank bson.M
		if err = banks.Decode(&bank); err != nil {
			log.Println(err)
		}

		if command == "/destroy_bank" {
			if bank["name"] != "other" {
				replyKeyboardRow = append(replyKeyboardRow, bank["name"].(string))
			}
		} else {
			replyKeyboardRow = append(replyKeyboardRow, bank["name"].(string))
		}

		if len(replyKeyboardRow) >= 3 {
			rk.Keyboard = append(rk.Keyboard, replyKeyboardRow)
			replyKeyboardRow = []string{}
		}
	}

	if len(replyKeyboardRow) > 0 {
		rk.Keyboard = append(rk.Keyboard, replyKeyboardRow)
	}

	rk.RemoveKeyboard = false

	return nil
}

func (rk *ReplyKeyboard) destroyBanksKeyboard() {
	rk.Keyboard = [][]string{}
	rk.RemoveKeyboard = true
}
