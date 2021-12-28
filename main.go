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
	db.Collections["debts"] = client.Database("general").Collection("Debts")

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
	developer, _ := os.LookupEnv("DEVELOPER")
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
			Message: "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @" + developer,
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
							"Для работы с ботом используй одну из следующих команд:%0A"+
								"/create_bank - создать копилку%0A"+
								"/destroy_bank - удалить копилку%0A"+
								"/income - увеличить баланс копилки%0A"+
								"/expense - уменьшить баланс копилки%0A"+
								"/get_balance - узнать баланс копилки",
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
					"Что-нибудь ещё?",
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
			} else if update.Message.Text == "/destroy_bank" || update.Message.Text == "/get_balance" ||
				update.Message.Text == "/income" || update.Message.Text == "/expense" {
				// ------------------------------------ handle /destroy_bank or /get_balance or /income or /expense command
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

						var message string

						if update.Message.Text == "/destroy_bank" {
							message = "Какую копилку ты хочешь удалить?"
						} else if update.Message.Text == "/get_balance" {
							message = "Баланс какой копилки ты хочешь узнать?"
						} else if update.Message.Text == "/income" ||
							update.Message.Text == "/expense" {
							message = "Баланс какой копилки будем изменять?"
						}

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							message+" Напиши /cancel, если передумал",
						); err != nil {
							log.Fatal(err)
						}

						bot.ReplyKeyboard.destroyKeyboard()
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/create_debt" {
				// ---------------------------------------------------------------------------- handle /create_debt command
				processing.destroyProcess(update.Message.Chat.ChatId)

				err = bot.ReplyKeyboard.createKeyboard("debt_variants", update.Message.Chat.ChatId)
				if err != nil {
					log.Println(err)

					bot.ReplyKeyboard.destroyKeyboard()

					err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					processing.createProcess(update.Message.Chat.ChatId, update.Message.Text, Extra{})

					if err = bot.sendMessage(
						update.Message.Chat.ChatId,
						"Кто кому должен? Выбери подходящий вариант%0A"+
							"Напиши /cancel, если передумал",
					); err != nil {
						log.Fatal(err)
					}

					bot.ReplyKeyboard.destroyKeyboard()
				}
				// --------------------------------------------------------------------------------------------------------
			} else {
				var process Process

				for _, proc := range processing.Processes {
					if proc.Chat == update.Message.Chat.ChatId {
						process = proc
					}
				}

				if process.Command == "" {
					// -------------------------------------------------------------------------- handle unexpected message
					if err = bot.sendMessage(
						update.Message.Chat.ChatId,
						"Для работы с ботом используй одну из следующих команд:%0A"+
							"/create_bank - создать копилку%0A"+
							"/destroy_bank - удалить копилку%0A"+
							"/income - увеличить баланс копилки%0A"+
							"/expense - уменьшить баланс копилки%0A"+
							"/get_balance - узнать баланс копилки%0A",
					); err != nil {
						log.Fatal(err)
					}
					// -----------------------------------------------------------------------------------------------------
				} else if process.Command == "/create_bank" {
					// ------------------------------------------------ handle update in /create_bank command processing
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
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/destroy_bank" || process.Command == "/get_balance" ||
					process.Command == "/income" || process.Command == "/expense" {
					// -------- handle update in /destroy_bank or /get_balance or /income or /expense command processing
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
						if process.Command == "/destroy_bank" {
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

						if process.Command == "/get_balance" {
							if err = bot.sendMessage(
								update.Message.Chat.ChatId,
								"Баланс копилки "+bank.Name+" составляет "+strconv.Itoa(bank.Balance)+" руб.",
							); err != nil {
								log.Fatal(err)
							}

							processing.destroyProcess(update.Message.Chat.ChatId)
						}

						if process.Command == "/income" || process.Command == "/expense" {
							operation := Operation{
								Account:   update.Message.Chat.ChatId,
								Bank:      bank.Id,
								Operation: process.Command,
							}

							err = bot.sendMessage(update.Message.Chat.ChatId, "На какую сумму?")
							if err != nil {
								log.Fatal(err)
							}

							processing.createProcess(
								update.Message.Chat.ChatId,
								"/set_operation_amount",
								Extra{
									Bank:      bank,
									Operation: operation,
								},
							)
						}
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_operation_amount" {
					// --------------------------------------- handle update in /set_operation_amount command processing
					amount, err := strconv.Atoi(update.Message.Text)
					if err != nil {
						log.Println(err)

						err = bot.sendError(update.Message.Chat.ChatId, INCORRECT_VALUE)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						process.Extra.Operation.Amount = amount

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Добавь комментарий к операции",
						); err != nil {
							log.Fatal(err)
						}

						processing.createProcess(
							update.Message.Chat.ChatId,
							"/set_operation_comment",
							Extra{
								Bank:      process.Extra.Bank,
								Operation: process.Extra.Operation,
							},
						)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_operation_comment" {
					// -------------------------------------- handle update in /set_operation_comment command processing
					process.Extra.Operation.Comment = update.Message.Text

					err = process.Extra.Operation.createOparetion(
						&process.Extra.Bank)
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
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/create_debt" {
					// ------------------------------------------------ handle update in /create_debt command processing
					debt := Debt{
						Account: update.Message.Chat.ChatId,
						Whose:   update.Message.Text,
					}

					if err = bot.sendMessage(
						update.Message.Chat.ChatId,
						"Какую сумму?",
					); err != nil {
						log.Fatal(err)
					}

					processing.createProcess(
						update.Message.Chat.ChatId,
						"/set_debt_amount",
						Extra{
							Debt: debt,
						},
					)
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_debt_amount" {
					// -------------------------------------------- handle update in /set_debt_amount command processing
					amount, err := strconv.Atoi(update.Message.Text)
					if err != nil {
						log.Println(err)

						err = bot.sendError(update.Message.Chat.ChatId, INCORRECT_VALUE)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						process.Extra.Debt.Amount = amount

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Добавь комментарий",
						); err != nil {
							log.Fatal(err)
						}

						processing.createProcess(
							update.Message.Chat.ChatId,
							"/set_debt_comment",
							Extra{
								Debt: process.Extra.Debt,
							},
						)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_debt_comment" {
					// ------------------------------------------- handle update in /set_debt_comment command processing
					process.Extra.Debt.Comment = update.Message.Text

					err = process.Extra.Debt.createDebt()
					if err != nil {
						log.Println(err)

						err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Готово!",
						); err != nil {
							log.Fatal(err)
						}

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Я буду переодически напоминать тебе о нём",
						); err != nil {
							log.Fatal(err)
						}
					}

					processing.destroyProcess(update.Message.Chat.ChatId)
					// -------------------------------------------------------------------------------------------------
				}
			}

			log.Println(update.Message.Chat.Username + " say: " + update.Message.Text)
			offset = update.UpdateId + 1
		}
	}
}
