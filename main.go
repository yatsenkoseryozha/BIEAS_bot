package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
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
	db.Collections["banks"] = client.Database("general").Collection("banks")
	db.Collections["operations"] = client.Database("general").Collection("operations")
	// db.Collections["debts"] = client.Database("general").Collection("debts")

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
	bot.Errors = map[string]string{
		NO_BANKS:           "На твоем аккаунте нет ни одной копилки!",
		BANK_NAME_IS_EXIST: "Копилка с таким названием уже существует. Попробуй снова",
		BANK_NOT_FOUND:     "Копилка с таким названием не найдена. Попробуй снова",
		INCORRECT_VALUE:    "Некорректное значение. Попробуй снова",
		UNEXPECTED_ERROR:   "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @" + developer,
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
				processing.destroy(update.Message.Chat.ChatId)

				banks, err := db.getDocuments(
					"banks",
					bson.M{
						"account": update.Message.Chat.ChatId,
					},
				)
				defer banks.Close(ctx)

				if err != nil {
					log.Println(err)

					err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if banks.RemainingBatchLength() > 0 {
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
						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Привет! Давай создадим для тебя копилку. Какое название дадим ей?",
						); err != nil {
							log.Fatal(err)
						}

						processing.create(update.Message.Chat.ChatId, "/create_bank", Extra{})
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/cancel" {
				// --------------------------------------------------------------------------------- handle /cancel command
				processing.destroy(update.Message.Chat.ChatId)

				if err = bot.sendMessage(
					update.Message.Chat.ChatId,
					"Что-нибудь ещё?",
				); err != nil {
					log.Fatal(err)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/create_bank" {
				// ---------------------------------------------------------------------------- handle /create_bank command
				processing.destroy(update.Message.Chat.ChatId)

				if err = bot.sendMessage(
					update.Message.Chat.ChatId,
					"Как хочешь назвать новую копилку? Напиши /cancel, если передумал",
				); err != nil {
					log.Fatal(err)
				}

				processing.create(update.Message.Chat.ChatId, update.Message.Text, Extra{})
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/destroy_bank" || update.Message.Text == "/get_balance" ||
				update.Message.Text == "/income" || update.Message.Text == "/expense" {
				// ------------------------------------ handle /destroy_bank or /get_balance or /income or /expense command
				processing.destroy(update.Message.Chat.ChatId)

				var keyboardButtons []string

				banks, err := db.getDocuments(
					"banks",
					bson.M{
						"account": update.Message.Chat.ChatId,
					},
				)
				defer banks.Close(ctx)

				if err != nil {
					log.Println(err)

					err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
					if err != nil {
						log.Fatal(err)
					}
				} else {
					for banks.Next(ctx) {
						var bank Bank

						err = banks.Decode(&bank)
						if err != nil {
							log.Println(err)

							err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
							if err != nil {
								log.Fatal(err)
							}
						}

						keyboardButtons = append(keyboardButtons, bank.Name)
					}

					if len(keyboardButtons) == 0 {
						err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[NO_BANKS])
						if err != nil {
							log.Fatal(err)
						}
					} else {
						bot.ReplyKeyboard.create(keyboardButtons)

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

						bot.ReplyKeyboard.destroy()
						processing.create(
							update.Message.Chat.ChatId,
							update.Message.Text,
							Extra{
								Keyboard: keyboardButtons,
							},
						)
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == "/create_debt" {
				// ---------------------------------------------------------------------------- handle /create_debt command
				// processing.destroy(update.Message.Chat.ChatId)

				// err = bot.ReplyKeyboard.create("debt_variants", update.Message.Chat.ChatId)
				// if err != nil {
				// 	log.Println(err)

				// 	bot.ReplyKeyboard.destroy()

				// 	err = bot.sendError(update.Message.Chat.ChatId, UNEXPECTED_ERROR)
				// 	if err != nil {
				// 		log.Fatal(err)
				// 	}
				// } else {
				// 	processing.create(update.Message.Chat.ChatId, update.Message.Text, Extra{})

				// 	if err = bot.sendMessage(
				// 		update.Message.Chat.ChatId,
				// 		"Кто кому должен? Выбери подходящий вариант%0A"+
				// 			"Напиши /cancel, если передумал",
				// 	); err != nil {
				// 		log.Fatal(err)
				// 	}

				// 	bot.ReplyKeyboard.destroy()
				// }
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
					var bank Bank

					err = db.getDocument(
						"banks",
						bson.M{
							"account": update.Message.Chat.ChatId,
							"name":    update.Message.Text,
						},
					).Decode(&bank)
					if err != nil && err.Error() != "mongo: no documents in result" {
						log.Println(err)

						err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
						if err != nil {
							log.Fatal(err)
						}

						processing.destroy(update.Message.Chat.ChatId)
					} else if bank.Name != "" {
						log.Println(err)

						err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[BANK_NAME_IS_EXIST])
						if err != nil {
							log.Fatal(err)
						}
					} else {
						bank.Account = update.Message.Chat.ChatId
						bank.Name = update.Message.Text

						err = bank.create()
						if err != nil {
							log.Println(err)

							err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
							if err != nil {
								log.Fatal(err)
							}
						} else {
							if err = bot.sendMessage(
								update.Message.Chat.ChatId,
								"Копилка успешно создана!",
							); err != nil {
								log.Fatal(err)
							}
						}

						processing.destroy(update.Message.Chat.ChatId)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/destroy_bank" || process.Command == "/get_balance" ||
					process.Command == "/income" || process.Command == "/expense" {
					// -------- handle update in /destroy_bank or /get_balance or /income or /expense command processing
					var bank Bank

					if err = db.getDocument(
						"banks",
						bson.M{
							"account": update.Message.Chat.ChatId,
							"name":    update.Message.Text,
						},
					).Decode(&bank); err != nil {
						log.Println(err)

						if err.Error() == "mongo: no documents in result" {
							bot.ReplyKeyboard.create(process.Extra.Keyboard)

							err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[BANK_NOT_FOUND])
							if err != nil {
								log.Fatal(err)
							}

							bot.ReplyKeyboard.destroy()
						} else {
							err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
							if err != nil {
								log.Fatal(err)
							}

							processing.destroy(update.Message.Chat.ChatId)
						}
					} else {
						if process.Command == "/destroy_bank" {
							err = bank.destroy()
							if err != nil {
								log.Println(err)

								err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
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

							processing.destroy(update.Message.Chat.ChatId)
						}

						if process.Command == "/get_balance" {
							if err = bot.sendMessage(
								update.Message.Chat.ChatId,
								"Баланс копилки "+bank.Name+" составляет "+strconv.Itoa(bank.Balance)+" руб.",
							); err != nil {
								log.Fatal(err)
							}

							processing.destroy(update.Message.Chat.ChatId)
						}

						if process.Command == "/income" || process.Command == "/expense" {
							err = bot.sendMessage(update.Message.Chat.ChatId, "На какую сумму?")
							if err != nil {
								log.Fatal(err)
							}

							processing.create(
								update.Message.Chat.ChatId,
								"/set_operation_amount",
								Extra{
									Operation: Operation{
										Account:   update.Message.Chat.ChatId,
										Bank:      bank.Id,
										Operation: process.Command,
									},
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

						err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[INCORRECT_VALUE])
						if err != nil {
							log.Fatal(err)
						}
					} else {
						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Добавь комментарий к операции",
						); err != nil {
							log.Fatal(err)
						}

						processing.create(
							update.Message.Chat.ChatId,
							"/set_operation_comment",
							Extra{
								Operation: Operation{
									Account:   process.Extra.Operation.Account,
									Bank:      process.Extra.Bank.Id,
									Operation: process.Extra.Operation.Operation,
									Amount:    amount,
								},
							},
						)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_operation_comment" {
					// -------------------------------------- handle update in /set_operation_comment command processing
					processing.create(
						process.Chat,
						"/create_operation",
						Extra{
							Operation: Operation{
								Account:   process.Extra.Operation.Account,
								Bank:      process.Extra.Bank.Id,
								Operation: process.Extra.Operation.Operation,
								Amount:    process.Extra.Operation.Amount,
								Comment:   update.Message.Text,
							},
						},
					)
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/create_operation" {
					// ------------------------------------------- handle update in /create_operation command processing
					err = process.Extra.Operation.create()
					if err != nil {
						log.Println(err)

						err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
						if err != nil {
							log.Fatal(err)
						}
					} else {
						var bank Bank
						if err = db.getDocument(
							"banks",
							bson.M{
								"account": update.Message.Chat.ChatId,
								"id":      process.Extra.Operation.Bank,
							},
						).Decode(&bank); err != nil {
							log.Println(err)

							err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
							if err != nil {
								log.Fatal(err)
							}
						}

						var balance int

						if process.Extra.Operation.Operation == "/income" {
							balance = bank.Balance + process.Extra.Operation.Amount
						} else if process.Extra.Operation.Operation == "/expense" {
							balance = bank.Balance - process.Extra.Operation.Amount
						}

						bank.update(
							bson.M{"balance": balance},
						)

						if err = bot.sendMessage(
							update.Message.Chat.ChatId,
							"Баланс копилки был успешно изменен! Текущий баланс: "+
								strconv.Itoa(process.Extra.Bank.Balance)+" руб.",
						); err != nil {
							log.Fatal(err)
						}
					}

					processing.destroy(update.Message.Chat.ChatId)
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/create_debt" {
					// ------------------------------------------------ handle update in /create_debt command processing
					// debt := Debt{
					// 	Account: update.Message.Chat.ChatId,
					// 	Whose:   update.Message.Text,
					// }

					// if err = bot.sendMessage(
					// 	update.Message.Chat.ChatId,
					// 	"Какую сумму?",
					// ); err != nil {
					// 	log.Fatal(err)
					// }

					// processing.create(
					// 	update.Message.Chat.ChatId,
					// 	"/set_debt_amount",
					// 	Extra{
					// 		Debt: debt,
					// 	},
					// )
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_debt_amount" {
					// -------------------------------------------- handle update in /set_debt_amount command processing
					// amount, err := strconv.Atoi(update.Message.Text)
					// if err != nil {
					// 	log.Println(err)

					// 	err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[INCORRECT_VALUE])
					// 	if err != nil {
					// 		log.Fatal(err)
					// 	}
					// } else {
					// 	process.Extra.Debt.Amount = amount

					// 	if err = bot.sendMessage(
					// 		update.Message.Chat.ChatId,
					// 		"Добавь комментарий",
					// 	); err != nil {
					// 		log.Fatal(err)
					// 	}

					// 	processing.create(
					// 		update.Message.Chat.ChatId,
					// 		"/set_debt_comment",
					// 		Extra{
					// 			Debt: process.Extra.Debt,
					// 		},
					// 	)
					// }
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_debt_comment" {
					// ------------------------------------------- handle update in /set_debt_comment command processing
					// process.Extra.Debt.Comment = update.Message.Text

					// err = process.Extra.Debt.create()
					// if err != nil {
					// 	log.Println(err)

					// 	err = bot.sendMessage(update.Message.Chat.ChatId, bot.Errors[UNEXPECTED_ERROR])
					// 	if err != nil {
					// 		log.Fatal(err)
					// 	}
					// } else {
					// 	if err = bot.sendMessage(
					// 		update.Message.Chat.ChatId,
					// 		"Готово!",
					// 	); err != nil {
					// 		log.Fatal(err)
					// 	}

					// 	if err = bot.sendMessage(
					// 		update.Message.Chat.ChatId,
					// 		"Я буду переодически напоминать тебе о нём",
					// 	); err != nil {
					// 		log.Fatal(err)
					// 	}
					// }

					// processing.destroy(update.Message.Chat.ChatId)
					// -------------------------------------------------------------------------------------------------
				}
			}

			log.Println(update.Message.Chat.Username + " say: " + update.Message.Text)
			offset = update.UpdateId + 1
		}
	}
}
