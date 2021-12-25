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

// --------------------------------------------------------------------------------------- bot errors names
const NO_BANKS = "NO_BANKS"
const BANK_NAME_IS_EXIST = "BANK_NAME_IS_EXIST"
const BANK_NOT_FOUND = "BANK_NOT_FOUND"
const INCORRECT_VALUE = "INCORRECT_VALUE"
const UNEXPECTED_ERROR = "UNEXPECTED_ERROR"

// --------------------------------------------------------------------------------------------------------

var ctx = context.TODO()
var db DataBase
var bot Bot
var processing Processing

func init() {
	// find .env file
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)
	err := godotenv.Load(filepath.Join(exPath, ".env"))
	if err != nil {
		log.Fatal("No .env file found")
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
	db.Collections["banks"] = client.Database("general").Collection("Banks")
	db.Collections["operations"] = client.Database("general").Collection("Operations")

	// init Bot
	botUrl := "https://api.telegram.org/bot"
	botToken, _ := os.LookupEnv("BOT_TOKEN")
	bot.URI = botUrl + botToken
	bot.ReplyKeyboard = ReplyKeyboard{
		Keyboard:       [][]string{},
		Resize:         true,
		OneTime:        true,
		RemoveKeyboard: true,
	}
	bot.Errors = map[string]Error{
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

	fmt.Println("Инициализация прошла успешно! Бот готов к работе.")
}

func main() {
	offset := 0
	for {
		err := bot.getUpdates(offset)
		if err != nil {
			log.Fatal(err)
		}

		for _, update := range bot.GetUpdatesResp.Updates {
			if update.Message.Text == "/start" {
				// ---------------------------------------------------------------------------------- handle /start command
				processing.destroyProcess(update.Message.Chat.ChatId)

				finded, err := db.findAccout(update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if finded == true {
						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Балуешься?",
						); err != nil {
							log.Fatal(err)
						}
					} else {
						processing.createProcess(update.Message.Chat.ChatId, "/create_bank", Extra{})

						if err = bot.sendMessage(
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
				processing.destroyProcess(update.Message.Chat.ChatId)

				if err = bot.sendMessage(
					update.Message.Chat.ChatId,
					"Для работый с ботом используй одну из следующих команд:\n"+
						"/create_bank - создать копилку\n"+
						"/destroy_bank - удалить копилку\n"+
						"/income - увеличить баланс копилки\n"+
						"/expense - уменьшить баланс копилки\n"+
						"/get_balance - узнать баланс копилки",
				); err != nil {
					log.Fatal(err)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/create_bank" {
				// ---------------------------------------------------------------------------- handle /create_bank command
				processing.createProcess(update.Message.Chat.ChatId, update.Message.Text, Extra{})

				if err = bot.sendMessage(
					update.Message.Chat.ChatId,
					"Как хочешь назвать новую копилку? Напиши /cancel, если передумал",
				); err != nil {
					log.Fatal(err)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/destroy_bank" {
				// --------------------------------------------------------------------------- handle /destroy_bank command
				processing.destroyProcess(update.Message.Chat.ChatId)

				err = bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					bot.ReplyKeyboard.destroyKeyboard()

					err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if len(bot.ReplyKeyboard.Keyboard) == 0 {
						bot.ReplyKeyboard.destroyKeyboard()

						err = bot.sendError(update.Message.Chat.ChatId, NO_BANKS)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						processing.createProcess(update.Message.Chat.ChatId, update.Message.Text, Extra{})

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Какую копилку ты хочешь удалить? Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/income" || update.Message.Text == "/expense" {
				// ------------------------------------------------------------------- handle /income and /expense commands
				processing.destroyProcess(update.Message.Chat.ChatId)

				err = bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					bot.ReplyKeyboard.destroyKeyboard()

					err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if len(bot.ReplyKeyboard.Keyboard) == 0 {
						bot.ReplyKeyboard.destroyKeyboard()

						err = bot.sendError(update.Message.Chat.ChatId, NO_BANKS)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						processing.createProcess(update.Message.Chat.ChatId, update.Message.Text, Extra{})

						var operation string
						if update.Message.Text == "/income" {
							operation = "увеличим"
						} else if update.Message.Text == "/expense" {
							operation = "уменьшим"
						}

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Баланс какой копилки "+operation+"? Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/get_balance" {
				// ---------------------------------------------------------------------------- handle /get_balance command
				processing.destroyProcess(update.Message.Chat.ChatId)

				err = bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					bot.ReplyKeyboard.destroyKeyboard()

					err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if len(bot.ReplyKeyboard.Keyboard) == 0 {
						bot.ReplyKeyboard.destroyKeyboard()

						err = bot.sendError(update.Message.Chat.ChatId, NO_BANKS)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						processing.createProcess(update.Message.Chat.ChatId, update.Message.Text, Extra{})

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Баланс какой копилки ты хочешь узнать? Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else {
				for _, process := range processing.Processes {
					if process.Chat == update.Message.Chat.ChatId {
						if process.Command == "/create_bank" {
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
									err = bot.sendError(update.Message.Chat.ChatId, BANK_NAME_IS_EXIST)
									if err != nil {
										log.Fatal(err)
									}
								} else {
									err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
										log.Fatal(err)
									}

									processing.destroyProcess(update.Message.Chat.ChatId)
								}
							} else {
								if err = bot.sendMessage(
									update.Message.Chat.ChatId,
									"Копилка успешно создана!",
								); err != nil {
									log.Fatal(err)
								}

								processing.destroyProcess(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						} else if process.Command == "/destroy_bank" {
							// ------------------------------------------ handle update in /destroy_bank command processing
							bank, err := db.getBank(update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)

								if err.Error() == BANK_NOT_FOUND {
									bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)

									err = bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									bot.ReplyKeyboard.destroyKeyboard()
								} else {
									err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
										log.Fatal(err)
									}

									processing.destroyProcess(update.Message.Chat.ChatId)
								}
							} else {
								err = bank.destroy()
								if err != nil {
									log.Println(err)

									err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
										log.Fatal(err)
									}
								} else {
									if err = bot.sendMessage(
										update.Message.Chat.ChatId,
										"Копилка успешно удалена!",
									); err != nil {
										log.Fatal(err)
									}
								}

								processing.destroyProcess(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						} else if process.Command == "/income" || process.Command == "/expense" {
							// ------------------------------------ handle update in /income or /expense command processing
							bank, err := db.getBank(update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)
								if err.Error() == BANK_NOT_FOUND {
									bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)

									err = bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									bot.ReplyKeyboard.destroyKeyboard()
								} else {
									err = bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									processing.destroyProcess(update.Message.Chat.ChatId)
								}
							} else {
								err = bot.sendMessage(update.Message.Chat.ChatId, "На какую сумму?")
								if err != nil {
									log.Fatal(err)
								}

								var commandToProcessing string
								if process.Command == "/income" {
									commandToProcessing = "/set_income_amount"
								} else if process.Command == "/expense" {
									commandToProcessing = "/set_expense_amount"
								}

								processing.createProcess(
									update.Message.Chat.ChatId,
									commandToProcessing,
									Extra{
										Bank: bank,
									},
								)
							}
							// --------------------------------------------------------------------------------------------
						} else if process.Command == "/set_income_amount" || process.Command == "/set_expense_amount" {
							// -------------- handle update in /set_income_amount or /set_expense_amount command processing
							amount, err := strconv.Atoi(update.Message.Text)
							if err != nil {
								log.Println(err)
								err = bot.sendError(update.Message.Chat.ChatId, INCORRECT_VALUE)
								if err != nil {
									log.Fatal(err)
								}
							} else {
								operation := Operation{
									Account: update.Message.Chat.ChatId,
									Amout:   amount,
								}

								if process.Command == "/set_income_amount" {
									operation.Operation = "income"
								} else if process.Command == "/set_expense_amount" {
									operation.Operation = "expense"
								}

								err = operation.makeOparetion(&process.Extra.Bank)
								if err != nil {
									log.Println(err)

									err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
										log.Fatal(err)
									}
								} else {
									if err = bot.sendMessage(
										update.Message.Chat.ChatId,
										"Баланс копилки был успешно изменен! Текущий баланс: "+
											strconv.Itoa(process.Extra.Bank.Balance)+" руб.",
									); err != nil {
										log.Fatal(err)
									}
								}

								processing.destroyProcess(update.Message.Chat.ChatId)
							}
							// --------------------------------------------------------------------------------------------
						} else if process.Command == "/get_balance" {
							// ------------------------------------------- handle update in /get_balance command processing
							bank, err := db.getBank(update.Message.Chat.ChatId, update.Message.Text)
							if err != nil {
								log.Println(err)
								if err.Error() == BANK_NOT_FOUND {
									bot.ReplyKeyboard.createKeyboard("banks", update.Message.Chat.ChatId)

									err = bot.sendError(update.Message.Chat.ChatId, BANK_NOT_FOUND)
									if err != nil {
										log.Fatal(err)
									}

									bot.ReplyKeyboard.destroyKeyboard()
								} else {
									err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
									if err != nil {
										log.Fatal(err)
									}

									processing.destroyProcess(update.Message.Chat.ChatId)
								}
							} else {
								if err = bot.sendMessage(
									update.Message.Chat.ChatId,
									"Баланс копилки "+bank.Name+" составляет "+strconv.Itoa(bank.Balance)+" руб.",
								); err != nil {
									log.Fatal(err)
								}

								processing.destroyProcess(update.Message.Chat.ChatId)
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
