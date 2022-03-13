package models

import "BIEAS_bot/enums"

// ---------------------------------------------------------------------------
// --------------------------------------------------------- PROCESSING MODELS
type Processing struct {
	Processes []Process
}

func (processing *Processing) Create(chat int, command Command, extra Extra) {
	processing.Destroy(chat)

	processing.Processes = append(processing.Processes, Process{
		Chat:    chat,
		Command: command,
		Extra:   extra,
	})
}

func (processing *Processing) Destroy(chat int) {
	for index, command := range processing.Processes {
		if command.Chat == chat {
			processing.Processes[index] = processing.Processes[len(processing.Processes)-1]
			processing.Processes = processing.Processes[:len(processing.Processes)-1]

			break
		}
	}
}

// Process Models ------------------------------------------------------------
type Process struct {
	Chat    int
	Command Command
	Extra   Extra
}

type Command struct {
	Name enums.BotCommand
	Step int
}

type Extra struct {
	Bank      *Bank
	Operation Operation
	Keyboard  []string
}
