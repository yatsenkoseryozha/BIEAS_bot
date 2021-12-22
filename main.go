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
			if previousCommand != "" {
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
						sendMessage(botUri, update.Message.Chat.ChatId, "Копилка успешно создана!")
					} else {
						sendMessage(botUri, update.Message.Chat.ChatId, "Копилка с таким названием уже существует. Попробуй другое")
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
					bank.createBank()
				}

				if update.Message.Text == "/create_bank" {
					sendMessage(botUri, update.Message.Chat.ChatId, "Как хочешь назвать новую копилку?")
					previousCommand = "/create_bank"
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

func sendMessage(botUri string, chatId int, text string) error {
	resp, err := http.Get(botUri + "/sendMessage" + "?chat_id=" + strconv.Itoa(chatId) + "&text=" + text)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
