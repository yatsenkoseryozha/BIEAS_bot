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
	}

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

	var currentBank bson.M

	offset := 0
	for {
		updates, err := getUpdates(botUri, offset)
		if err != nil {
			log.Fatal(err)
		}

		for _, update := range updates {
			if previousCommand != "" && string(update.Message.Text[0]) != "/" {
				if previousCommand == "/create_bank" {
					nameIsValid := true

					banks, err := collection.Find(ctx, bson.M{"account": update.Message.Chat.ChatId})
					if err != nil {
						log.Println(err)

						previousCommand = ""
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						for banks.Next(ctx) {
							var bank bson.M
							err = banks.Decode(&bank)
							if err != nil {
								log.Println(err)
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
								log.Println(err)

								previousCommand = ""
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

					_, err = collection.DeleteOne(
						ctx,
						bson.M{
							"account": update.Message.Chat.ChatId,
							"name":    update.Message.Text,
						},
					)
					if err != nil {
						log.Println(err)

						previousCommand = ""
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						previousCommand = ""
						sendMessage(botUri, update.Message.Chat.ChatId, "Копилка успешно удалена!", &replyKeyboard)
					}
				}

				if previousCommand == "/income-to" || previousCommand == "/expense-to" {
					result := collection.FindOne(
						ctx, bson.M{
							"account": update.Message.Chat.ChatId,
							"name":    update.Message.Text,
						},
					)
					err = result.Decode(&currentBank)
					if err != nil {
						log.Println(err)

						previousCommand = ""
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					}
					if len(currentBank) == 0 {
						sendMessage(botUri, update.Message.Chat.ChatId, "Такой копилки не существует. Попробу снова", &replyKeyboard)
					} else {
						replyKeyboard.destroyBanksKeyboard()

						if previousCommand == "/income-to" {
							previousCommand = "/income-count"
						} else if previousCommand == "/expense-to" {
							previousCommand = "/expense-count"
						}
						sendMessage(botUri, update.Message.Chat.ChatId, "На какую сумму?", &replyKeyboard)
					}
				} else if previousCommand == "/income-count" || previousCommand == "/expense-count" {
					incomeCount, err := strconv.Atoi(update.Message.Text)
					if err != nil {
						log.Println(err)
						sendMessage(botUri, update.Message.Chat.ChatId, "Некорректное значение. Попробуй снова", &replyKeyboard)
					} else {
						after := options.After
						var result *mongo.SingleResult
						if previousCommand == "/income-count" {
							result = collection.FindOneAndUpdate(
								ctx,
								bson.M{
									"account": update.Message.Chat.ChatId,
									"name":    currentBank["name"].(string),
								},
								bson.M{
									"$set": bson.M{"balance": currentBank["balance"].(int32) + int32(incomeCount)},
								},
								&options.FindOneAndUpdateOptions{
									ReturnDocument: &after,
								},
							)
						} else if previousCommand == "/expense-count" {
							result = collection.FindOneAndUpdate(
								ctx,
								bson.M{
									"account": update.Message.Chat.ChatId,
									"name":    currentBank["name"].(string),
								},
								bson.M{
									"$set": bson.M{"balance": currentBank["balance"].(int32) - int32(incomeCount)},
								},
								&options.FindOneAndUpdateOptions{
									ReturnDocument: &after,
								},
							)
						}
						err = result.Decode(&currentBank)
						if err != nil {
							log.Println(err)

							previousCommand = ""
							sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
						} else {
							currentBalance := strconv.Itoa(int(currentBank["balance"].(int32)))

							previousCommand = ""
							sendMessage(botUri, update.Message.Chat.ChatId, "Баланс копилки был успешно изменен! Текущий баланс: "+currentBalance+" руб.", &replyKeyboard)
						}
					}
				}

				if previousCommand == "/get_balance" {
					replyKeyboard.destroyBanksKeyboard()

					result := collection.FindOne(
						ctx,
						bson.M{
							"account": update.Message.Chat.ChatId,
							"name":    update.Message.Text,
						},
					)
					err = result.Decode(&currentBank)
					if err != nil {
						log.Println(err)

						previousCommand = ""
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						currentBalance := strconv.Itoa(int(currentBank["balance"].(int32)))

						previousCommand = ""
						sendMessage(botUri, update.Message.Chat.ChatId, "Баланс копилки "+currentBank["name"].(string)+" составляет "+currentBalance+" руб.", &replyKeyboard)
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
					err := bank.createBank()
					if err != nil {
						log.Println(err)
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
					err = replyKeyboard.createBanksKeyboard(update.Message.Chat.ChatId, "/destroy_bank")
					if err != nil {
						log.Println(err)
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

				if update.Message.Text == "/income" || update.Message.Text == "/expense" {
					err = replyKeyboard.createBanksKeyboard(update.Message.Chat.ChatId, "/income")
					if err != nil {
						log.Println(err)
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						if update.Message.Text == "/income" {
							previousCommand = "/income-to"
							sendMessage(botUri, update.Message.Chat.ChatId, "Баланс какой копилки был увеличен? Если передумал, напиши /cancel", &replyKeyboard)
						} else if update.Message.Text == "/expense" {
							previousCommand = "/expense-to"
							sendMessage(botUri, update.Message.Chat.ChatId, "Баланс какой копилки был уменьшен? Если передумал, напиши /cancel", &replyKeyboard)
						}
					}
				}

				if update.Message.Text == "/get_balance" {
					err = replyKeyboard.createBanksKeyboard(update.Message.Chat.ChatId, "/get_balance")
					if err != nil {
						log.Println(err)
						sendMessage(botUri, update.Message.Chat.ChatId, "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53", &replyKeyboard)
					} else {
						previousCommand = "/get_balance"
						sendMessage(botUri, update.Message.Chat.ChatId, "Баланс какой копилки ты хочешь узнать? Если передумал, напиши /cancel", &replyKeyboard)
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
