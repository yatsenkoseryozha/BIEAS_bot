package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var store Store

func init() {
	// find .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("No .env file found")
	}

	// init DataBase
	store.CTX = context.TODO()
	dbUri, _ := os.LookupEnv("DB_URI")
	clientOptions := options.Client().ApplyURI(dbUri)
	client, err := mongo.Connect(store.CTX, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(store.CTX, nil)
	if err != nil {
		log.Fatal(err)
	}
	store.DataBase.Collections = make(map[string]*mongo.Collection)
	store.DataBase.Collections["banks"] = client.Database("general").Collection("Banks")
	store.DataBase.Collections["operations"] = client.Database("general").Collection("Operations")

	// init Bot
	botUrl := "https://api.telegram.org/bot"
	botToken, _ := os.LookupEnv("BOT_TOKEN")
	store.Bot.URI = botUrl + botToken
	store.Bot.ReplyKeyboard = ReplyKeyboard{
		Keyboard:       [][]string{},
		Resize:         true,
		OneTime:        true,
		RemoveKeyboard: true,
	}
}

func main() {
	offset := 0
	for {
		err := store.Bot.getUpdates(offset)
		if err != nil {
			log.Fatal(err)
		}

		for _, update := range store.Bot.GetUpdatesResp.Updates {
			if update.Message.Text == "/start" {
				// ---------------------------------------------------------------------------------- handle /start command
				bank := Bank{
					Account: update.Message.Chat.ChatId,
					Name:    "other",
					Balance: 0,
				}
				err = bank.create()
				if err != nil {
					log.Println(err)
					if err = store.Bot.sendMessage(
						update.Message.Chat.ChatId,
						"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
					); err != nil {
						log.Fatal(err)
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/cancel" {
				// --------------------------------------------------------------------------------- handle /cancel command
				store.Processing.deleteCommand(update.Message.Chat.ChatId)

				if err = store.Bot.sendMessage(
					update.Message.Chat.ChatId,
					"Что-нибудь ещё?",
				); err != nil {
					log.Fatal(err)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/create_bank" {
				// ---------------------------------------------------------------------------- handle /create_bank command
				store.Processing.addCommand(update.Message.Chat.ChatId, update.Message.Text, Extra{})

				if err = store.Bot.sendMessage(
					update.Message.Chat.ChatId,
					"Как хочешь назвать новую копилку? Напиши /cancel, если передумал",
				); err != nil {
					log.Fatal(err)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/destroy_bank" {
				// --------------------------------------------------------------------------- handle /destroy_bank command
				err = store.Bot.ReplyKeyboard.createKeyboard(update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)
					if err = store.Bot.sendMessage(
						update.Message.Chat.ChatId,
						"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
					); err != nil {
						log.Fatal(err)
					}
				} else {
					if len(store.Bot.ReplyKeyboard.Keyboard) == 0 {
						store.Bot.ReplyKeyboard.destroyKeyboard()

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"На твоем аккаунте нет ни одной копилки!",
						); err != nil {
							log.Fatal(err)
						}
					} else {
						store.Processing.addCommand(update.Message.Chat.ChatId, update.Message.Text, Extra{})

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"Какую копилку ты хочешь удалить? Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						store.Bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/income" || update.Message.Text == "/expense" {
				// ------------------------------------------------------------------- handle /income and /expense commands
				err = store.Bot.ReplyKeyboard.createKeyboard(update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)
					if err = store.Bot.sendMessage(
						update.Message.Chat.ChatId,
						"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
					); err != nil {
						log.Fatal(err)
					}
				} else {
					if len(store.Bot.ReplyKeyboard.Keyboard) == 0 {
						store.Bot.ReplyKeyboard.destroyKeyboard()

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"На твоем аккаунте нет ни одной копилки!",
						); err != nil {
							log.Fatal(err)
						}
					} else {
						store.Processing.addCommand(update.Message.Chat.ChatId, update.Message.Text, Extra{})

						var operation string
						if update.Message.Text == "/income" {
							operation = "увеличен"
						} else if update.Message.Text == "/expense" {
							operation = "уменьшен"
						}

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"Баланс какой копилки был "+operation+"? Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						store.Bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/get_balance" {
				// ---------------------------------------------------------------------------- handle /get_balance command
				err = store.Bot.ReplyKeyboard.createKeyboard(update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)
					if err = store.Bot.sendMessage(
						update.Message.Chat.ChatId,
						"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
					); err != nil {
						log.Fatal(err)
					}
				} else {
					if len(store.Bot.ReplyKeyboard.Keyboard) == 0 {
						store.Bot.ReplyKeyboard.destroyKeyboard()

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"На твоем аккаунте нет ни одной копилки!",
						); err != nil {
							log.Fatal(err)
						}
					} else {
						store.Processing.addCommand(update.Message.Chat.ChatId, update.Message.Text, Extra{})

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"Баланс какой копилки ты хочешь узнать? Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						store.Bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else {
				for _, command := range store.Processing.Commands {
					if command.Chat == update.Message.Chat.ChatId {
						if command.Command == "/create_bank" {
							// ------------------------------------------- handle update in /create_bank command processing
							bank := Bank{
								Account: update.Message.Chat.ChatId,
								Name:    update.Message.Text,
								Balance: 0,
							}

							err = bank.create()
							if err != nil {
								log.Println(err)
								if err.Error() == "Копилка с таким названием уже существует. Попробуй снова" {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Копилка с таким названием уже существует. Попробуй снова",
									); err != nil {
										log.Fatal(err)
									}
								} else {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
									); err != nil {
										log.Fatal(err)
									}

									store.Processing.deleteCommand(update.Message.Chat.ChatId)
								}
							} else {
								if err = store.Bot.sendMessage(
									update.Message.Chat.ChatId,
									"Копилка успешно создана!",
								); err != nil {
									log.Fatal(err)
								}

								store.Processing.deleteCommand(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						} else if command.Command == "/destroy_bank" {
							// ------------------------------------------ handle update in /destroy_bank command processing
							bank, err := store.DataBase.getDocument("banks", update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)
								if err.Error() == "Копилка с таким названием не найдена. Попробуй снова" {
									store.Bot.ReplyKeyboard.createKeyboard(update.Message.Chat.ChatId)

									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Копилка с таким названием не найдена. Попробуй снова",
									); err != nil {
										log.Fatal(err)
									}

									store.Bot.ReplyKeyboard.destroyKeyboard()
								} else {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
									); err != nil {
										log.Fatal(err)
									}

									store.Processing.deleteCommand(update.Message.Chat.ChatId)
								}
							} else {
								err = bank.destroy()
								if err != nil {
									log.Println(err)
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
									); err != nil {
										log.Fatal(err)
									}
								} else {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Копилка успешно удалена!",
									); err != nil {
										log.Fatal(err)
									}
								}

								store.Processing.deleteCommand(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						} else if command.Command == "/income" || command.Command == "/expense" {
							// ------------------------------------ handle update in /income or /expense command processing
							bank, err := store.DataBase.getDocument("banks", update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)
								if err.Error() == "Копилка с таким названием не найдена. Попробуй снова" {
									store.Bot.ReplyKeyboard.createKeyboard(update.Message.Chat.ChatId)

									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Копилка с таким названием не найдена. Попробуй снова",
									); err != nil {
										log.Fatal(err)
									}

									store.Bot.ReplyKeyboard.destroyKeyboard()
								} else {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
									); err != nil {
										log.Fatal(err)
									}

									store.Processing.deleteCommand(update.Message.Chat.ChatId)
								}
							} else {
								if err = store.Bot.sendMessage(
									update.Message.Chat.ChatId,
									"На какую сумму?",
								); err != nil {
									log.Fatal(err)
								}

								var commandToProcessing string
								if command.Command == "/income" {
									commandToProcessing = "/set_income_amount"
								} else if command.Command == "/expense" {
									commandToProcessing = "/set_expense_amount"
								}

								store.Processing.addCommand(
									update.Message.Chat.ChatId,
									commandToProcessing,
									Extra{
										Bank: bank,
									},
								)
							}
							// --------------------------------------------------------------------------------------------
						} else if command.Command == "/set_income_amount" || command.Command == "/set_expense_amount" {
							// -------------- handle update in /set_income_amount or /set_expense_amount command processing
							amount, err := strconv.Atoi(update.Message.Text)
							if err != nil {
								log.Println(err)
								if err = store.Bot.sendMessage(
									update.Message.Chat.ChatId,
									"Некорректное значение. Попробуй снова",
								); err != nil {
									log.Fatal(err)
								}
							} else {
								var updatedBank Bank
								if command.Command == "/set_income_amount" {
									updatedBank, err = command.Extra.Bank.updateBalance(amount, "income")
								} else if command.Command == "/set_expense_amount" {
									updatedBank, err = command.Extra.Bank.updateBalance(amount, "expense")
								}

								if err != nil {
									log.Println(err)
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
									); err != nil {
										log.Fatal(err)
									}
								} else {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Баланс копилки был успешно изменен! Текущий баланс: "+string(rune(updatedBank.Balance))+" руб.",
									); err != nil {
										log.Fatal(err)
									}
								}

								store.Processing.deleteCommand(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						} else if command.Command == "/get_balance" {
							// ------------------------------------------- handle update in /get_balance command processing
							bank, err := store.DataBase.getDocument("banks", update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)
								if err.Error() == "Копилка с таким названием не найдена. Попробуй снова" {
									store.Bot.ReplyKeyboard.createKeyboard(update.Message.Chat.ChatId)

									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Копилка с таким названием не найдена. Попробуй снова",
									); err != nil {
										log.Fatal(err)
									}

									store.Bot.ReplyKeyboard.destroyKeyboard()
								} else {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
									); err != nil {
										log.Fatal(err)
									}

									store.Processing.deleteCommand(update.Message.Chat.ChatId)
								}
							} else {
								if err = store.Bot.sendMessage(
									update.Message.Chat.ChatId,
									"Баланс копилки "+bank.Name+" составляет "+strconv.Itoa(bank.Balance)+" руб.",
								); err != nil {
									log.Fatal(err)
								}

								store.Processing.deleteCommand(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						}
					}
				}
			}

			fmt.Println(store.Processing.Commands)
			offset = update.UpdateId + 1
		}
	}
}
