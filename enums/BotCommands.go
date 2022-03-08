package enums

type BotCommand int

const (
	START BotCommand = iota
	CANCEL
	CREATE_BANK
	DESTROY_BANK
	GET_BALANCE
	INCOME
	EXPENSE
)

var BotCommands = map[BotCommand]string{
	START:        "/start",
	CANCEL:       "/cancel",
	CREATE_BANK:  "/create_bank",
	DESTROY_BANK: "/destroy_bank",
	GET_BALANCE:  "/get_balance",
	INCOME:       "/income",
	EXPENSE:      "/expense",
}
