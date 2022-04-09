# ![](./assets/logo-24.png) BIEAS v1.1   
Базовая система учета доходов и расходов BIEAS реализована в качестве [Telegram-бота](https://t.me/BIEAS_bot).

## Прицнип использования
Использование системы предпологает следующий подход:
1. Пользователь создает копилку, которая по своей сути является одной из его статей расходов;
2. После дохода пользователь распределяет полученные деньги по копилкам;
3. После расхода пользователь уменьшает баланс соответствующей копилки.

Так же у пользователя есть возможность в любой момент посмотреть баланс копилки для того, чтобы запланировать свои дальнейшие расходы.

## Команды
На данный момент доступны следующие команды:  
`/create_bank` - создать копилку  
`/destroy_bank` - удалить копилку  
`/income` - увеличить баланс копилки  
`/expense` - уменьшить баланс копилки  
`/get_balance` - узанть баланс копилки  
`/create_transfer` - создать перевод
