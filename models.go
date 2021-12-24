package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	CTX        context.Context
	DataBase   DataBase
	Bot        Bot
	Processing Processing
}

// ---------------------------------------------------------------------------
// ----------------------------------------------------------- DATABASE MODELS
type DataBase struct {
	Collections map[string]*mongo.Collection
}

func (db *DataBase) getDocuments(collection string, chat int) (*mongo.Cursor, error) {
	documents, err := db.Collections[collection].Find(store.CTX, bson.M{"account": chat})
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func (db *DataBase) getBank(chat int, name string) (Bank, error) {
	var bank Bank
	db.Collections["banks"].FindOne(store.CTX, bson.M{"account": chat, "name": name}).Decode(&bank)
	if bank.Name == "" {
		return bank, errors.New("Копилка с таким названием не найдена. Попробуй снова")
	}

	return bank, nil
}

func (db *DataBase) findAccout(chat int) (bool, error) {
	documents, err := db.getDocuments("banks", chat)
	if err != nil {
		return false, err
	}
	defer documents.Close(store.CTX)

	if documents.RemainingBatchLength() > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

// Bank Models ---------------------------------------------------------------
type Bank struct {
	Id        string `json:"id" bson:"id"`
	Account   int    `json:"account" bson:"account"`
	Name      string `json:"name" bson:"name"`
	Balance   int    `json:"balance" bson:"balance"`
	CreatedAt string `json:"created_at" bson:"created_at"`
	UpdatedAt string `json:"updated_at" bson:"updated_at"`
}

func (bank *Bank) create() error {
	var findedBank Bank
	store.DataBase.Collections["banks"].FindOne(store.CTX, bank).Decode(&findedBank)
	if findedBank.Name == bank.Name {
		return errors.New("Копилка с таким названием уже существует. Попробуй снова")
	}

	id, err := gonanoid.New()
	if err != nil {
		return err
	}

	bank.Id = id

	bank.CreatedAt = time.Now().String()
	bank.UpdatedAt = time.Now().String()

	_, err = store.DataBase.Collections["banks"].InsertOne(store.CTX, bank)
	if err != nil {
		return err
	}

	return nil
}

func (bank *Bank) destroy() error {
	_, err := store.DataBase.Collections["banks"].DeleteOne(store.CTX, bank)
	if err != nil {
		return err
	}

	return nil
}

func (bank *Bank) updateBalance(amount int, operation string) error {
	if operation == "income" {
		bank.Balance = bank.Balance + amount
	} else if operation == "expense" {
		bank.Balance = bank.Balance - amount
	}

	bank.UpdatedAt = time.Now().String()

	after := options.After
	options := &options.FindOneAndUpdateOptions{ReturnDocument: &after}
	err := store.DataBase.Collections["banks"].FindOneAndUpdate(
		store.CTX,
		bson.M{"id": bank.Id},
		bson.M{
			"$set": bson.M{
				"balance":    bank.Balance,
				"updated_at": bank.UpdatedAt,
			},
		},
		options,
	).Decode(&bank)
	if err != nil {
		return err
	}

	return nil
}

type Operation struct {
	Account   int    `json:"account" bson:"account"`
	Bank      string `json:"bank" bson:"bank"`
	Operation string `json:"operation" bson:"operation"`
	Amout     int    `json:"amount" bson:"amount"`
	CreatedAt string `json:"created_at" bson:"created_at"`
}

func (operation *Operation) makeOparetion(bank *Bank) error {
	operation.Bank = bank.Id
	operation.CreatedAt = time.Now().String()

	_, err := store.DataBase.Collections["operations"].InsertOne(store.CTX, operation)
	if err != nil {
		return err
	}

	err = bank.updateBalance(operation.Amout, operation.Operation)
	if err != nil {
		return nil
	}

	return nil
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------- BOT MODELS
type Bot struct {
	URI            string
	GetUpdatesResp GetUpdatesResp
	ReplyKeyboard  ReplyKeyboard
}

func (bot *Bot) getUpdates(offset int) error {
	resp, err := http.Get(bot.URI + "/getUpdates" + "?offset=" + strconv.Itoa(offset))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	json.Unmarshal(body, &bot.GetUpdatesResp)

	return nil
}

func (bot *Bot) sendMessage(chat int, text string) error {
	options := "?chat_id=" + strconv.Itoa(chat) + "&text=" + text

	keyboardJSON, err := json.Marshal(store.Bot.ReplyKeyboard)
	if err != nil {
		return err
	}

	options += "&reply_markup=" + string(keyboardJSON)

	resp, err := http.Get(store.Bot.URI + "/sendMessage" + options)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Updates Models ------------------------------------------------------------
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

// ReplyKeyboard Models ------------------------------------------------------
type ReplyKeyboard struct {
	Keyboard       [][]string `json:"keyboard"`
	Resize         bool       `json:"resize_keyboard"`
	OneTime        bool       `json:"one_time_keyboard"`
	RemoveKeyboard bool       `json:"remove_keyboard"`
}

func (rk *ReplyKeyboard) createKeyboard(collection string, chat int) error {
	var keyboardRow []string

	documents, err := store.DataBase.getDocuments(collection, chat)
	if err != nil {
		return err
	}
	defer documents.Close(store.CTX)

	for documents.Next(store.CTX) {
		var document bson.M
		err = documents.Decode(&document)
		if err != nil {
			return err
		}

		keyboardRow = append(keyboardRow, document["name"].(string))

		if len(keyboardRow) >= 3 {
			rk.Keyboard = append(rk.Keyboard, keyboardRow)
			keyboardRow = []string{}
		}
	}

	if len(keyboardRow) > 0 {
		rk.Keyboard = append(rk.Keyboard, keyboardRow)
	}

	rk.RemoveKeyboard = false

	return nil
}

func (rk *ReplyKeyboard) destroyKeyboard() {
	rk.Keyboard = [][]string{}
	rk.RemoveKeyboard = true
}

// ---------------------------------------------------------------------------
// --------------------------------------------------------- PROCESSING MODELS
type Processing struct {
	Commands []Command
}

type Command struct {
	Chat    int
	Command string
	Extra   Extra
}

type Extra struct {
	Bank   Bank
	Amount int
}

func (proc *Processing) addCommand(chat int, command string, extra Extra) {
	proc.deleteCommand(chat)

	proc.Commands = append(proc.Commands, Command{
		Chat:    chat,
		Command: command,
		Extra:   extra,
	})
}

func (proc *Processing) deleteCommand(chat int) {
	for index, command := range proc.Commands {
		if command.Chat == chat {
			proc.Commands[index] = proc.Commands[len(proc.Commands)-1]
			proc.Commands = proc.Commands[:len(proc.Commands)-1]

			break
		}
	}
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
