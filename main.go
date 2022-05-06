package main

import (
	"BIEAS_bot/models"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ctx = context.TODO()
var db models.DataBase
var bot models.Bot
var processing models.Processing

func init() {
	// find .env file
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	err := godotenv.Load(filepath.Join(exPath, ".env"))
	if err != nil {
		log.Println("No .env file found")
	}

	// init DataBase
	dbUri, _ := os.LookupEnv("DB_URI")
	clientOptions := options.Client().ApplyURI(dbUri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	db.Collections = make(map[string]*mongo.Collection)
	dbName, _ := os.LookupEnv("DB_NAME")
	db.Collections["banks"] = client.Database(dbName).Collection("banks")
	db.Collections["operations"] = client.Database(dbName).Collection("operations")

	// init Bot
	botUrl := "https://api.telegram.org/bot"
	botToken, _ := os.LookupEnv("BOT_TOKEN")
	bot.URI = botUrl + botToken
	bot.ReplyKeyboard = models.ReplyKeyboard{
		Keyboard:       [][]string{},
		Resize:         true,
		OneTime:        true,
		RemoveKeyboard: true,
	}

	fmt.Println("Инициализация прошла успешно! Бот готов к работе.")
}

func main() {
	offset := 0
	for {
		err := bot.GetUpdates(offset)
		if err != nil {
			log.Fatal(err)
		}

		for _, update := range bot.GetUpdatesResp.Updates {
			handler(update)

			log.Println(update.Message.Chat.Username + " say: " + update.Message.Text)
			offset = update.UpdateId + 1
		}
	}
}
