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
		log.Fatal("No .env file found")
		// MAKE PANIC HERE
	}

	dbUri, _ := os.LookupEnv("DB_URI")
	clientOptions := options.Client().ApplyURI(dbUri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
		// MAKE PANIC HERE
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
		// MAKE PANIC HERE
	}

	collection = client.Database("general").Collection("Banks")
}

func main() {
	botUrl := "https://api.telegram.org/bot"
	botToken, _ := os.LookupEnv("BOT_TOKEN")
	botUri := botUrl + botToken

	var replyKeyboard = ReplyKeyboard{
		Keyboard:       [][]string{},
		Resize:         true,
		OneTime:        true,
		RemoveKeyboard: true,
	}
	var previousCommand string

	offset := 0
	for {
		updates, err := getUpdates(botUri, offset)
		if err != nil {
			log.Fatal(err)
			// MAKE PANIC HERE
		}

		for _, update := range updates {
			if previousCommand != "" && string(update.Message.Text[0]) != "/" {
				if previousCommand == "/create_bank" {
					nameIsValid := true

					banks, err := collection.Find(ctx, bson.M{"account": update.Message.Chat.ChatId})
					if err != nil {
						log.Fatal(err)
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						for banks.Next(ctx) {
							var bank bson.M
							if err = banks.Decode(&bank); err != nil {
								log.Fatal(err)
								// MAKE PANIC HERE
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
							err = bank.createBank()
							if err != nil {
								log.Fatal(err)
								sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
							} else {
								previousCommand = ""
								sendMessage(botUri, update.Message.Chat.ChatId, "Копилка успешно создана!", &replyKeyboard)
							}
						} else {
							sendMessage(botUri, update.Message.Chat.ChatId, "Копилка с таким названием уже существует. Попробуй другое", &replyKeyboard)
						}
					}
					defer banks.Close(ctx)
				}

				if previousCommand == "/destroy_bank" {
					replyKeyboard.destroyBanksKeyboard()

					previousCommand = ""

					_, err = collection.DeleteOne(ctx, bson.M{"name": update.Message.Text})
					if err != nil {
						log.Fatal(err)
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						sendMessage(botUri, update.Message.Chat.ChatId, "Копилка успешно удалена!", &replyKeyboard)
					}
				}
			} else {
				if update.Message.Text == "/start" {
					bank := Bank{
						Account: update.Message.Chat.ChatId,
						Owner:   update.Message.Chat.Username,
						Name:    "other",
						Balance: 0,
					}
					err = bank.createBank()
					if err != nil {
						log.Fatal(err)
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					}
				}

				if update.Message.Text == "/cancel" {
					replyKeyboard.destroyBanksKeyboard()

					previousCommand = ""
					sendMessage(botUri, update.Message.Chat.ChatId, "Что-нибудь ещё?", &replyKeyboard)
				}

				if update.Message.Text == "/create_bank" {
					replyKeyboard.destroyBanksKeyboard()

					previousCommand = "/create_bank"
					sendMessage(botUri, update.Message.Chat.ChatId, "Как хочешь назвать новую копилку? Если передумал, напиши /cancel", &replyKeyboard)
				}

				if update.Message.Text == "/destroy_bank" {
					err = replyKeyboard.createBanksKeyboard(update.Message.Chat.ChatId)
					if err != nil {
						log.Fatal(err)
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						if len(replyKeyboard.Keyboard) > 0 {
							previousCommand = "/destroy_bank"
							sendMessage(botUri, update.Message.Chat.ChatId, "Какую копилку ты хочешь удалить? Если передумал, напиши /cancel", &replyKeyboard)
						} else {
							sendMessage(botUri, update.Message.Chat.ChatId, "Нет копилок, которые ты мог бы удалить", &replyKeyboard)
						}
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

func sendMessage(botUri string, chatId int, text string, keyboard *ReplyKeyboard) error {
	options := "?chat_id=" + strconv.Itoa(chatId) + "&text=" + text

	keyboardJSON, err := json.Marshal(keyboard)
	if err != nil {
		return err
	}

	options += "&reply_markup=" + string(keyboardJSON)

	resp, err := http.Get(botUri + "/sendMessage" + options)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
