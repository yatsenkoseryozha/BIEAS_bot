package main

import (
	"BIEAS_bot/enums"
	"BIEAS_bot/models"
	"BIEAS_bot/utils"
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
	db.Collections["banks"] = client.Database("develop").Collection("banks")
	db.Collections["operations"] = client.Database("develop").Collection("operations")

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
			if update.Message.Text == enums.BotCommands[enums.START] {
				// ---------------------------------------------------------------------------------- handle /start command
				processing.Destroy(update.Message.Chat.ChatId)

				banks, err := db.GetDocuments(
					ctx,
					"banks",
					bson.M{
						"account": update.Message.Chat.ChatId,
					},
				)
				defer banks.Close(ctx)

				if err != nil {
					log.Println(err)

					err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.UNEXPECTED_ERROR])
					if err != nil {
						log.Fatal(err)
					}
				} else {
					if banks.RemainingBatchLength() > 0 {
						if err = bot.SendMessage(
							update.Message.Chat.ChatId,
							"Для работы с ботом используй одну из следующих команд:%0A"+
								"/create_bank - создать копилку%0A"+
								"/Destroy_bank - удалить копилку%0A"+
								"/income - увеличить баланс копилки%0A"+
								"/expense - уменьшить баланс копилки%0A"+
								"/get_balance - узнать баланс копилки",
						); err != nil {
							log.Fatal(err)
						}
					} else {
						if err = bot.SendMessage(
							update.Message.Chat.ChatId,
							"Привет! Давай создадим для тебя копилку. Какое название дадим ей?",
						); err != nil {
							log.Fatal(err)
						}

						processing.Create(update.Message.Chat.ChatId, enums.BotCommands[enums.CREATE_BANK], models.Extra{})
					}
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == enums.BotCommands[enums.CANCEL] {
				// --------------------------------------------------------------------------------- handle /cancel command
				processing.Destroy(update.Message.Chat.ChatId)

				if err = bot.SendMessage(
					update.Message.Chat.ChatId,
					"Что-нибудь ещё?",
				); err != nil {
					log.Fatal(err)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == enums.BotCommands[enums.CREATE_BANK] {
				// ---------------------------------------------------------------------------- handle /create_bank command
				processing.Destroy(update.Message.Chat.ChatId)

				if err = bot.SendMessage(
					update.Message.Chat.ChatId,
					"Как хочешь назвать новую копилку? Напиши /cancel, если передумал",
				); err != nil {
					log.Fatal(err)
				}

				processing.Create(update.Message.Chat.ChatId, update.Message.Text, models.Extra{})
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == enums.BotCommands[enums.DESTROY_BANK] {
				// --------------------------------------------------------------------------- handle /Destroy_bank command
				processing.Destroy(update.Message.Chat.ChatId)

				if keyboardButtons, err := utils.CreateKeyboardButtons(
					ctx,
					&db,
					"banks",
					bson.M{
						"account": update.Message.Chat.ChatId,
					},
				); err != nil {
					bot.SendMessage(update.Message.Chat.ChatId, err.Error())
				} else {
					bot.ReplyKeyboard.Create(keyboardButtons)

					if err = bot.SendMessage(
						update.Message.Chat.ChatId,
						"Какую копилку ты хочешь удалить? Напиши /cancel, если передумал",
					); err != nil {
						log.Fatal(err)
					}

					bot.ReplyKeyboard.Destroy()
					processing.Create(
						update.Message.Chat.ChatId,
						update.Message.Text,
						models.Extra{
							Keyboard: keyboardButtons,
						},
					)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == enums.BotCommands[enums.GET_BALANCE] {
				// ---------------------------------------------------------------------------- handle /get_balance command
				if keyboardButtons, err := utils.CreateKeyboardButtons(
					ctx,
					&db,
					"banks",
					bson.M{
						"account": update.Message.Chat.ChatId,
					},
				); err != nil {
					bot.SendMessage(update.Message.Chat.ChatId, err.Error())
				} else {
					if err = bot.SendMessage(
						update.Message.Chat.ChatId,
						"Баланс какой копилки ты хочешь узнать? Напиши /cancel, если передумал",
					); err != nil {
						log.Fatal(err)
					}

					bot.ReplyKeyboard.Destroy()
					processing.Create(
						update.Message.Chat.ChatId,
						update.Message.Text,
						models.Extra{
							Keyboard: keyboardButtons,
						},
					)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == enums.BotCommands[enums.INCOME] {
				// --------------------------------------------------------------------------------- handle /income command
				if keyboardButtons, err := utils.CreateKeyboardButtons(
					ctx,
					&db,
					"banks",
					bson.M{
						"account": update.Message.Chat.ChatId,
					},
				); err != nil {
					bot.SendMessage(update.Message.Chat.ChatId, err.Error())
				} else {
					if err = bot.SendMessage(
						update.Message.Chat.ChatId,
						"Баланс какой копилки будем изменять? Напиши /cancel, если передумал",
					); err != nil {
						log.Fatal(err)
					}

					bot.ReplyKeyboard.Destroy()
					processing.Create(
						update.Message.Chat.ChatId,
						update.Message.Text,
						models.Extra{
							Keyboard: keyboardButtons,
						},
					)
				}
				// --------------------------------------------------------------------------------------------------------
			} else if update.Message.Text == enums.BotCommands[enums.EXPENSE] {
				// -------------------------------------------------------------------------------- handle /expense command
				if keyboardButtons, err := utils.CreateKeyboardButtons(
					ctx,
					&db,
					"banks",
					bson.M{
						"account": update.Message.Chat.ChatId,
					},
				); err != nil {
					bot.SendMessage(update.Message.Chat.ChatId, err.Error())
				} else {
					if err = bot.SendMessage(
						update.Message.Chat.ChatId,
						"Баланс какой копилки будем изменять? Напиши /cancel, если передумал",
					); err != nil {
						log.Fatal(err)
					}

					bot.ReplyKeyboard.Destroy()
					processing.Create(
						update.Message.Chat.ChatId,
						update.Message.Text,
						models.Extra{
							Keyboard: keyboardButtons,
						},
					)
				}
				// --------------------------------------------------------------------------------------------------------
			} else {
				var process models.Process

				for _, proc := range processing.Processes {
					if proc.Chat == update.Message.Chat.ChatId {
						process = proc
					}
				}

				if process.Command == "" {
					// -------------------------------------------------------------------------- handle unexpected message
					if err = bot.SendMessage(
						update.Message.Chat.ChatId,
						"Для работы с ботом используй одну из следующих команд:%0A"+
							"/create_bank - создать копилку%0A"+
							"/Destroy_bank - удалить копилку%0A"+
							"/income - увеличить баланс копилки%0A"+
							"/expense - уменьшить баланс копилки%0A"+
							"/get_balance - узнать баланс копилки%0A",
					); err != nil {
						log.Fatal(err)
					}
					// -----------------------------------------------------------------------------------------------------
				} else if process.Command == enums.BotCommands[enums.CREATE_BANK] {
					// ------------------------------------------------ handle update in /create_bank command processing
					var bank models.Bank

					err = db.GetDocument(
						ctx,
						"banks",
						bson.M{
							"account": update.Message.Chat.ChatId,
							"name":    update.Message.Text,
						},
					).Decode(&bank)
					if err != nil && err.Error() != "mongo: no documents in result" {
						log.Println(err)

						err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.UNEXPECTED_ERROR])
						if err != nil {
							log.Fatal(err)
						}

						processing.Destroy(update.Message.Chat.ChatId)
					} else if bank.Name != "" {
						log.Println(err)

						err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.BANK_NAME_IS_EXIST])
						if err != nil {
							log.Fatal(err)
						}
					} else {
						bank.Account = update.Message.Chat.ChatId
						bank.Name = update.Message.Text

						err = bank.Create(ctx, &db)
						if err != nil {
							log.Println(err)

							err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.UNEXPECTED_ERROR])
							if err != nil {
								log.Fatal(err)
							}
						} else {
							if err = bot.SendMessage(
								update.Message.Chat.ChatId,
								"Копилка успешно создана!",
							); err != nil {
								log.Fatal(err)
							}
						}

						processing.Destroy(update.Message.Chat.ChatId)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == enums.BotCommands[enums.DESTROY_BANK] {
					// ----------------------------------------------- handle update in /destroy_bank command processing
					if bank, err := utils.GetBank(ctx, &db, bson.M{
						"account": update.Message.Chat.ChatId,
						"bank":    update.Message.Text,
					}); err != nil {
						log.Println(err)

						if err.Error() == enums.UserErrors[enums.BANK_NOT_FOUND] {
							bot.ReplyKeyboard.Create(process.Extra.Keyboard)

							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							bot.ReplyKeyboard.Destroy()
						} else {
							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							processing.Destroy(update.Message.Chat.ChatId)
						}
					} else {
						err = bank.Destroy(ctx, &db)
						if err != nil {
							log.Println(err)

							err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.UNEXPECTED_ERROR])
							if err != nil {
								log.Fatal(err)
							}
						} else {
							if err = bot.SendMessage(
								update.Message.Chat.ChatId,
								"Копилка успешно удалена!",
							); err != nil {
								log.Fatal(err)
							}
						}

						processing.Destroy(update.Message.Chat.ChatId)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == enums.BotCommands[enums.GET_BALANCE] {
					// ------------------------------------------------ handle update in /get_balance command processing
					if bank, err := utils.GetBank(ctx, &db, bson.M{
						"account": update.Message.Chat.ChatId,
						"bank":    update.Message.Text,
					}); err != nil {
						log.Println(err)

						if err.Error() == enums.UserErrors[enums.BANK_NOT_FOUND] {
							bot.ReplyKeyboard.Create(process.Extra.Keyboard)

							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							bot.ReplyKeyboard.Destroy()
						} else {
							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							processing.Destroy(update.Message.Chat.ChatId)
						}
					} else {
						if err = bot.SendMessage(
							update.Message.Chat.ChatId,
							"Баланс копилки "+bank.Name+" составляет "+strconv.Itoa(bank.Balance)+" руб.",
						); err != nil {
							log.Fatal(err)
						}

						processing.Destroy(update.Message.Chat.ChatId)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == enums.BotCommands[enums.INCOME] {
					// ----------------------------------------------------- handle update in /income command processing
					if bank, err := utils.GetBank(ctx, &db, bson.M{
						"account": update.Message.Chat.ChatId,
						"bank":    update.Message.Text,
					}); err != nil {
						log.Println(err)

						if err.Error() == enums.UserErrors[enums.BANK_NOT_FOUND] {
							bot.ReplyKeyboard.Create(process.Extra.Keyboard)

							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							bot.ReplyKeyboard.Destroy()
						} else {
							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							processing.Destroy(update.Message.Chat.ChatId)
						}
					} else {
						err = bot.SendMessage(update.Message.Chat.ChatId, "На какую сумму?")
						if err != nil {
							log.Fatal(err)
						}

						processing.Create(
							update.Message.Chat.ChatId,
							"/set_operation_amount",
							models.Extra{
								Operation: models.Operation{
									Account:   update.Message.Chat.ChatId,
									Bank:      bank.Id,
									Operation: process.Command,
								},
							},
						)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == enums.BotCommands[enums.EXPENSE] {
					// ---------------------------------------------------- handle update in /expense command processing
					if bank, err := utils.GetBank(ctx, &db, bson.M{
						"account": update.Message.Chat.ChatId,
						"bank":    update.Message.Text,
					}); err != nil {
						log.Println(err)

						if err.Error() == enums.UserErrors[enums.BANK_NOT_FOUND] {
							bot.ReplyKeyboard.Create(process.Extra.Keyboard)

							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							bot.ReplyKeyboard.Destroy()
						} else {
							err = bot.SendMessage(update.Message.Chat.ChatId, err.Error())
							if err != nil {
								log.Fatal(err)
							}

							processing.Destroy(update.Message.Chat.ChatId)
						}
					} else {
						err = bot.SendMessage(update.Message.Chat.ChatId, "На какую сумму?")
						if err != nil {
							log.Fatal(err)
						}

						processing.Create(
							update.Message.Chat.ChatId,
							"/set_operation_amount",
							models.Extra{
								Operation: models.Operation{
									Account:   update.Message.Chat.ChatId,
									Bank:      bank.Id,
									Operation: process.Command,
								},
							},
						)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_operation_amount" {
					// --------------------------------------- handle update in /set_operation_amount command processing
					amount, err := strconv.Atoi(update.Message.Text)
					if err != nil {
						log.Println(err)

						err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.INCORRECT_VALUE])
						if err != nil {
							log.Fatal(err)
						}
					} else {
						if err = bot.SendMessage(
							update.Message.Chat.ChatId,
							"Добавь комментарий к операции",
						); err != nil {
							log.Fatal(err)
						}

						processing.Create(
							update.Message.Chat.ChatId,
							"/set_operation_comment",
							models.Extra{
								Operation: models.Operation{
									Account:   process.Extra.Operation.Account,
									Bank:      process.Extra.Operation.Bank,
									Operation: process.Extra.Operation.Operation,
									Amount:    amount,
								},
							},
						)
					}
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/set_operation_comment" {
					// -------------------------------------- handle update in /set_operation_comment command processing
					processing.Create(
						process.Chat,
						"/create_operation",
						models.Extra{
							Operation: models.Operation{
								Account:   process.Extra.Operation.Account,
								Bank:      process.Extra.Operation.Bank,
								Operation: process.Extra.Operation.Operation,
								Amount:    process.Extra.Operation.Amount,
								Comment:   update.Message.Text,
							},
						},
					)

					continue
					// -------------------------------------------------------------------------------------------------
				} else if process.Command == "/create_operation" {
					// ------------------------------------------- handle update in /create_operation command processing
					err = process.Extra.Operation.Create(ctx, &db)
					if err != nil {
						log.Println(err)

						err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.UNEXPECTED_ERROR])
						if err != nil {
							log.Fatal(err)
						}
					} else {
						var bank models.Bank
						if err = db.GetDocument(
							ctx,
							"banks",
							bson.M{
								"account": update.Message.Chat.ChatId,
								"id":      process.Extra.Operation.Bank,
							},
						).Decode(&bank); err != nil {
							log.Println(err)

							err = bot.SendMessage(update.Message.Chat.ChatId, enums.UserErrors[enums.UNEXPECTED_ERROR])
							if err != nil {
								log.Fatal(err)
							}
						} else {
							var balance int

							if process.Extra.Operation.Operation == enums.BotCommands[enums.INCOME] {
								balance = bank.Balance + process.Extra.Operation.Amount
							} else if process.Extra.Operation.Operation == enums.BotCommands[enums.EXPENSE] {
								balance = bank.Balance - process.Extra.Operation.Amount
							}

							bank.Update(ctx, &db, bson.M{"balance": balance})

							if err = bot.SendMessage(
								update.Message.Chat.ChatId,
								"Баланс копилки был успешно изменен! Текущий баланс: "+
									strconv.Itoa(bank.Balance)+" руб.",
							); err != nil {
								log.Fatal(err)
							}
						}
					}

					processing.Destroy(update.Message.Chat.ChatId)
					// -------------------------------------------------------------------------------------------------
				}
			}

			log.Println(update.Message.Chat.Username + " say: " + update.Message.Text)
			offset = update.UpdateId + 1
		}
	}
}
