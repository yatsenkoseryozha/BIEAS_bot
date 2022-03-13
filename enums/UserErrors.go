package enums

import "os"

type UserError int

const (
	NO_BANKS UserError = iota
	BANK_NAME_IS_EXIST
	BANK_NOT_FOUND
	INCORRECT_VALUE
	UNEXPECTED_ERROR
)

var developer = os.Getenv("DEVELOPER")
var UserErrors = map[UserError]string{
	NO_BANKS:           "На твоем аккаунте нет ни одной копилки!",
	BANK_NAME_IS_EXIST: "Копилка с таким названием уже существует. Попробуй снова",
	BANK_NOT_FOUND:     "Копилка с таким названием не найдена. Попробуй снова",
	INCORRECT_VALUE:    "Некорректное значение. Попробуй снова",
	UNEXPECTED_ERROR:   "Произошла непредвиденная ошибка. Пожалуйста, напиши об этом разработчику @" + developer,
}
