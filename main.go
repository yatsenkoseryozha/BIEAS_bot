package main

import (
	"BIEAS_bot/models"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
	// init .env
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	err := godotenv.Load(filepath.Join(exPath, ".env"))
	if err != nil {
		log.Println("No .env file found")
	}

	// init DataBase
	dbUri := os.Getenv("DB_URI")
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
	dbName := os.Getenv("DB_NAME")
	db.Collections["banks"] = client.Database(dbName).Collection("banks")
	db.Collections["operations"] = client.Database(dbName).Collection("operations")

	// init Bot
	bot.Token = os.Getenv("BOT_TOKEN")
	bot.ReplyKeyboard = models.ReplyKeyboard{
		Keyboard:       [][]string{},
		Resize:         true,
		OneTime:        true,
		RemoveKeyboard: true,
	}

	fmt.Println("Инициализация прошла успешно! Бот готов к работе.")
}

func main() {
	http.HandleFunc("/"+bot.Token, func(rw http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}

		var update models.Update

		err = json.Unmarshal(body, &update)
		if err != nil {
			log.Println(err)
		}

		handleUpdate(update)
	})

	PORT := os.Getenv("PORT")
	http.ListenAndServe(":"+PORT, nil)
}
