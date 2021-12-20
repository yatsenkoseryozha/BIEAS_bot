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
