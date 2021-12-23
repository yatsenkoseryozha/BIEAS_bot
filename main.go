package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var ctx = context.TODO()

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Print("No .env file found")
	}

	dbUri, _ := os.LookupEnv("DB_URI")
	clientOptions := options.Client().ApplyURI(dbUri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Println(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Println(err)
	}

	collection = client.Database("general").Collection("Banks")
}

func main() {
	botUrl := "https://api.telegram.org/bot"
	botToken, _ := os.LookupEnv("BOT_TOKEN")
	botUri := botUrl + botToken

	offset := 0
	var previousCommand string
	for {
		updates, err := getUpdates(botUri, offset)
		if err != nil {
			log.Println(err)
		}

		for _, update := range updates {
			if previousCommand != "" && string(update.Message.Text[0]) != "/" {
				if previousCommand == "/create_bank" {
					nameIsValid := true

					banks, err := collection.Find(ctx, bson.M{"account": update.Message.Chat.ChatId})
					if err != nil {
						log.Println(err)
					}
					defer banks.Close(ctx)

					for banks.Next(ctx) {
						var bank bson.M
						if err = banks.Decode(&bank); err != nil {
							log.Println(err)
						}
						if bank["name"] == update.Message.Text {
							nameIsValid = false
						}
					}

					if nameIsValid {
						bank := Bank{
							Account: update.Message.Chat.ChatId,
							Owner:   update.Message.Chat.Username,
							Name:    update.Message.Text,
							Balance: 0,
						}
						bank.createBank()
						previousCommand = ""
						sendMessage(botUri, update.Message.Chat.ChatId, "Копилка успешно создана!", ReplyKeyboard{Keyboard: [][]string{}})
					} else {
						sendMessage(botUri, update.Message.Chat.ChatId, "Копилка с таким названием уже существует. Попробуй другое", ReplyKeyboard{Keyboard: [][]string{}})
					}
				}

				if previousCommand == "/destroy_bank" {
					collection.DeleteOne(ctx, bson.M{"name": update.Message.Text})
					previousCommand = ""
					sendMessage(botUri, update.Message.Chat.ChatId, "Копилка успешно удалена!", ReplyKeyboard{Keyboard: [][]string{}})
				}
			} else {
				if update.Message.Text == "/start" {
					bank := Bank{
						Account: update.Message.Chat.ChatId,
						Owner:   update.Message.Chat.Username,
						Name:    "other",
						Balance: 0,
					}
					bank.createBank()
				}

				if update.Message.Text == "/cancel" {
					previousCommand = ""
					sendMessage(botUri, update.Message.Chat.ChatId, "Что-нибудь ещё?", ReplyKeyboard{Keyboard: [][]string{}})
				}

				if update.Message.Text == "/create_bank" {
					sendMessage(botUri, update.Message.Chat.ChatId, "Как хочешь назвать новую копилку? Если передумал, напиши /cancel", ReplyKeyboard{Keyboard: [][]string{}})
					previousCommand = "/create_bank"
				}

				if update.Message.Text == "/destroy_bank" {
					banks, err := collection.Find(ctx, bson.M{"account": update.Message.Chat.ChatId})
					if err != nil {
						log.Println(err)
					}
					defer banks.Close(ctx)

					replyKeyboard := ReplyKeyboard{
						Keyboard: [][]string{},
						Resize:   true,
						OneTime:  true,
					}
					var replyKeyboardRow []string

					for banks.Next(ctx) {
						var bank bson.M
						if err = banks.Decode(&bank); err != nil {
							log.Println(err)
						}

						if bank["name"] != "other" {
							replyKeyboardRow = append(replyKeyboardRow, bank["name"].(string))
						}

						if len(replyKeyboardRow) >= 3 {
							replyKeyboard.Keyboard = append(replyKeyboard.Keyboard, replyKeyboardRow)
							replyKeyboardRow = []string{}
						}
					}

					if len(replyKeyboardRow) > 0 {
						replyKeyboard.Keyboard = append(replyKeyboard.Keyboard, replyKeyboardRow)
					}

					if len(replyKeyboard.Keyboard) > 0 {
						sendMessage(botUri, update.Message.Chat.ChatId, "Какую копилку ты хочешь удалить? Если передумал, напиши /cancel", replyKeyboard)
						previousCommand = "/destroy_bank"
					} else {
						sendMessage(botUri, update.Message.Chat.ChatId, "Нет копилок, которые ты мог бы удалить", ReplyKeyboard{Keyboard: [][]string{}})
					}
				}
			}

			offset = update.UpdateId + 1
		}
	}
}

func getUpdates(botUri string, offset int) ([]Update, error) {
	resp, err := http.Get(botUri + "/getUpdates" + "?offset=" + strconv.Itoa(offset))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var getUpdatesResp *GetUpdatesResp

	json.Unmarshal(body, &getUpdatesResp)

	return getUpdatesResp.Updates, nil
}

func sendMessage(botUri string, chatId int, text string, keyboard ReplyKeyboard) error {
	options := "?chat_id=" + strconv.Itoa(chatId) + "&text=" + text

	if len(keyboard.Keyboard) == 0 {
		keyboard.RemoveKeyboard = true
	}

	keyboardJSON, err := json.Marshal(keyboard)
	if err != nil {
		log.Println(err)
	}

	options += "&reply_markup=" + string(keyboardJSON)

	resp, err := http.Get(botUri + "/sendMessage" + options)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
