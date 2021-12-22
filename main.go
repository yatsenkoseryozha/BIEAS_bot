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
				if previousCommand == "/createBank" {
					bank := Bank{
						Account: update.Message.Chat.ChatId,
						Owner:   update.Message.Chat.Username,
						Name:    update.Message.Text,
						Balance: 0,
					}
					bank.createBank()
					previousCommand = ""
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

				if update.Message.Text == "/createBank" {
					sendMessage(botUri, update.Message.Chat.ChatId, "Как хочешь назвать новую копилку?")
					previousCommand = "/createBank"
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
