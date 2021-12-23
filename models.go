package main

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
