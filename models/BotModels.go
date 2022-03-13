package models

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------- BOT MODELS
type Bot struct {
	Token         string
	ReplyKeyboard ReplyKeyboard
}

func (bot *Bot) SendMessage(chat int, text string) error {
	options := "?chat_id=" + strconv.Itoa(chat) + "&text=" + text

	keyboardJSON, err := json.Marshal(bot.ReplyKeyboard)
	if err != nil {
		return err
	}

	options += "&reply_markup=" + string(keyboardJSON)

	resp, err := http.Get("https://api.telegram.org/bot" + bot.Token + "/sendMessage" + options)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Updates Models ------------------------------------------------------------
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

// ReplyKeyboard Models ------------------------------------------------------
type ReplyKeyboard struct {
	Keyboard       [][]string `json:"keyboard"`
	Resize         bool       `json:"resize_keyboard"`
	OneTime        bool       `json:"one_time_keyboard"`
	RemoveKeyboard bool       `json:"remove_keyboard"`
}

func (rk *ReplyKeyboard) Create(buttons []string) {
	var keyboardRow []string

	for _, button := range buttons {
		keyboardRow = append(keyboardRow, button)

		if len(keyboardRow) >= 3 {
			rk.Keyboard = append(rk.Keyboard, keyboardRow)
			keyboardRow = []string{}
		}
	}

	if len(keyboardRow) > 0 {
		rk.Keyboard = append(rk.Keyboard, keyboardRow)
	}

	rk.RemoveKeyboard = false
}

func (rk *ReplyKeyboard) Destroy() {
	rk.Keyboard = [][]string{}
	rk.RemoveKeyboard = true
}
