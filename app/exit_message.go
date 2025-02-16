package app

import (
	"fmt"
	"strings"

	"github.com/olebedev/config"
)

// DisplayExitMessage displays the onscreen exit message when the app quits
func (wtfApp *WtfApp) DisplayExitMessage() {
	exitMessageIsDisplayable := readDisplayableConfig(wtfApp.config)

	wtfApp.displayExitMsg(exitMessageIsDisplayable)
}

/* -------------------- Unexported Functions -------------------- */

func (wtfApp *WtfApp) displayExitMsg(exitMessageIsDisplayable bool) string {
	msgs := []string{}

	displayMsg := strings.Join(msgs, "\n")

	fmt.Println(displayMsg)

	return displayMsg
}

// readDisplayableConfig figures out whether or not the exit message should be displayed
// per the user's wishes. It allows contributors and sponsors to opt out of the exit message
func readDisplayableConfig(cfg *config.Config) bool {
	displayExitMsg := cfg.UBool("wtf.exitMessage.display", true)
	return displayExitMsg
}
