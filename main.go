package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const NO_BANKS = "NO_BANKS"
const BANK_NAME_IS_EXIST = "BANK_NAME_IS_EXIST"
const BANK_NOT_FOUND = "BANK_NOT_FOUND"
const INCORRECT_VALUE = "INCORRECT_VALUE"
const UNEXPECTED_ERROR = "UNEXPECTED_ERROR"

var store Store

func init() {
	// find .env file
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	err := godotenv.Load(filepath.Join(exPath, ".env"))
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
	store.Bot.Errors = map[string]Error{
		NO_BANKS: {
			Message: "На твоем аккаунте нет ни одной копилки!",
		},
		BANK_NAME_IS_EXIST: {
			Message: "Копилка с таким названием уже существует. Попробуй снова",
		},
		BANK_NOT_FOUND: {
			Message: "Копилка с таким названием не найдена. Попробуй снова",
		},
		INCORRECT_VALUE: {
			Message: "Некорректное значение. Попробуй снова",
		},
		UNEXPECTED_ERROR: {
			Message: "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @iss53",
		},
	}
}

func main() {
	fmt.Println("Инициализация прошла успешно! Бот готов к работе.")

	offset := 0
	for {
		err := store.Bot.getUpdates(offset)
		if err != nil {
			log.Fatal(err)
		}

		for _, update := range store.Bot.GetUpdatesResp.Updates {
			if update.Message.Text == "/start" {
				// ---------------------------------------------------------------------------------- handle /start command
				store.Processing.deleteCommand(update.Message.Chat.ChatId)

				finded, err := store.DataBase.findAccout(update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if finded == true {
						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"Балуешься?",
						); err != nil {
							log.Fatal(err)
						}
					} else {
						store.Processing.addCommand(update.Message.Chat.ChatId, "/create_bank", Extra{})

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"Привет! Давай создадим для тебя копилку. Какое название дадим ей?",
						); err != nil {
							log.Fatal(err)
						}
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
				store.Processing.deleteCommand(update.Message.Chat.ChatId)

				err = store.Bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					store.Bot.ReplyKeyboard.destroyKeyboard()

					err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if len(store.Bot.ReplyKeyboard.Keyboard) == 0 {
						store.Bot.ReplyKeyboard.destroyKeyboard()

						err = store.Bot.sendError(update.Message.Chat.ChatId, NO_BANKS)
						if err != nil {
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
				store.Processing.deleteCommand(update.Message.Chat.ChatId)

				err = store.Bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					store.Bot.ReplyKeyboard.destroyKeyboard()

					err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if len(store.Bot.ReplyKeyboard.Keyboard) == 0 {
						store.Bot.ReplyKeyboard.destroyKeyboard()

						err = store.Bot.sendError(update.Message.Chat.ChatId, NO_BANKS)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						store.Processing.addCommand(update.Message.Chat.ChatId, update.Message.Text, Extra{})

						var operation string
						if update.Message.Text == "/income" {
							operation = "увеличим"
						} else if update.Message.Text == "/expense" {
							operation = "уменьшим"
						}

						if err = store.Bot.sendMessage(
							update.Message.Chat.ChatId,
							"Баланс какой копилки "+operation+"? Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						store.Bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/get_balance" {
				// ---------------------------------------------------------------------------- handle /get_balance command
				store.Processing.deleteCommand(update.Message.Chat.ChatId)

				err = store.Bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					store.Bot.ReplyKeyboard.destroyKeyboard()

					err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if len(store.Bot.ReplyKeyboard.Keyboard) == 0 {
						store.Bot.ReplyKeyboard.destroyKeyboard()

						err = store.Bot.sendError(update.Message.Chat.ChatId, NO_BANKS)
						if err != nil {
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

								if err.Error() == BANK_NAME_IS_EXIST {
									err = store.Bot.sendError(update.Message.Chat.ChatId, BANK_NAME_IS_EXIST)
									if err != nil {
										log.Fatal(err)
									}
								} else {
									err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
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
							bank, err := store.DataBase.getBank(update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)

								if err.Error() == BANK_NOT_FOUND {
									store.Bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)

									err = store.Bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									store.Bot.ReplyKeyboard.destroyKeyboard()
								} else {
									err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
										log.Fatal(err)
									}

									store.Processing.deleteCommand(update.Message.Chat.ChatId)
								}
							} else {
								err = bank.destroy()
								if err != nil {
									log.Println(err)

									err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
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
							bank, err := store.DataBase.getBank(update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)
								if err.Error() == BANK_NOT_FOUND {
									store.Bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)

									err = store.Bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									store.Bot.ReplyKeyboard.destroyKeyboard()
								} else {
									err = store.Bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									store.Processing.deleteCommand(update.Message.Chat.ChatId)
								}
							} else {
								err = store.Bot.sendMessage(update.Message.Chat.ChatId, "На какую сумму?")
								if err != nil {
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
								err = store.Bot.sendError(update.Message.Chat.ChatId, INCORRECT_VALUE)
								if err != nil {
									log.Fatal(err)
								}
							} else {
								operation := Operation{
									Account: update.Message.Chat.ChatId,
									Amout:   amount,
								}

								if command.Command == "/set_income_amount" {
									operation.Operation = "income"
								} else if command.Command == "/set_expense_amount" {
									operation.Operation = "expense"
								}

								err = operation.makeOparetion(&command.Extra.Bank)
								if err != nil {
									log.Println(err)

									err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
										log.Fatal(err)
									}
								} else {
									if err = store.Bot.sendMessage(
										update.Message.Chat.ChatId,
										"Баланс копилки был успешно изменен! Текущий баланс: "+
											strconv.Itoa(command.Extra.Bank.Balance)+" руб.",
									); err != nil {
										log.Fatal(err)
									}
								}

								store.Processing.deleteCommand(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						} else if command.Command == "/get_balance" {
							// ------------------------------------------- handle update in /get_balance command processing
							bank, err := store.DataBase.getBank(update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)
								if err.Error() == BANK_NOT_FOUND {
									store.Bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)

									err = store.Bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									store.Bot.ReplyKeyboard.destroyKeyboard()
								} else {
									err = store.Bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
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

			log.Println(update.Message.Chat.Username + " say: " + update.Message.Text)
			offset = update.UpdateId + 1
		}
	}
}
