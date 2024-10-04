package gui

import (
	"game_main/common"

	"github.com/hajimehoshi/ebiten/v2"
)

var lastText []string = make([]string, 0, 5)

func ProcessUserLog(ecsmanager common.EntityManager, screen *ebiten.Image, msgUI *PlayerMessageUI) {

	tmpMessages := make([]string, 0, 5)
	anyMessages := false

	for _, m := range ecsmanager.World.Query(ecsmanager.WorldTags["messengers"]) {
		messages := m.Components[common.UserMsgComponent].(*common.UserMessage)
		if messages.AttackMessage != "" {
			tmpMessages = append(tmpMessages, messages.AttackMessage)
			anyMessages = true

			messages.AttackMessage = ""
		}

		if messages.StatusEffectMessage != "" {
			tmpMessages = append(tmpMessages, messages.StatusEffectMessage)
			anyMessages = true

			messages.StatusEffectMessage = ""
		}
	}

	if anyMessages {
		//This means there's a new message
		msgUI.SetTextWithArray(lastText)
		lastText = tmpMessages
	} else {

		//No messages changed

		msgUI.SetText("")
		msgUI.SetTextWithArray(lastText)

	}

}
