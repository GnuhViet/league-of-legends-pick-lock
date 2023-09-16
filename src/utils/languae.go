package utils

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

type UIText struct {
	AppNameText        string
	SelectChampText    string
	SelectedChampText  string
	ConfirmButtonText  string
	StartButtonText    string
	StopButtonText     string
	CheckboxLockText   string
	PickLockButtonText string
	ResultLabelText    string
	StatusLabelText    string
}

type AlertText struct {
	Notification     string
	WaitClientText   string
	NotInMatchMaking string
	NotSelectChamp   string
}

type MessageText struct {
	NotStarted      string
	PickLockCancel  string
	NotSelectChamp  string
	MatchCancelled  string
	ReadyPickLock   string
	PickLockSuccess string
}

func ReadEnv(language string) (UIText, AlertText, MessageText) {
	if language == "en" {
		err := godotenv.Load("en.env")
		if err != nil {
			fmt.Println("error loading env")
			return UIText{}, AlertText{}, MessageText{}
		}
	} else if language == "vi" {
		err := godotenv.Load("vi.env")
		if err != nil {
			fmt.Println("error loading env")
			return UIText{}, AlertText{}, MessageText{}
		}
	}

	var uiText UIText
	var alertText AlertText
	var messageText MessageText

	uiText.AppNameText = os.Getenv("AppNameText")
	uiText.SelectChampText = os.Getenv("SelectChampText")
	uiText.SelectedChampText = os.Getenv("SelectedChampText")
	uiText.ConfirmButtonText = os.Getenv("ConfirmButtonText")
	uiText.StartButtonText = os.Getenv("StartButtonText")
	uiText.StopButtonText = os.Getenv("StopButtonText")
	uiText.CheckboxLockText = os.Getenv("CheckboxLockText")
	uiText.PickLockButtonText = os.Getenv("PickLockButtonText")
	uiText.ResultLabelText = os.Getenv("ResultLabelText")
	uiText.StatusLabelText = os.Getenv("StatusLabelText")

	alertText.Notification = os.Getenv("Notification")
	alertText.WaitClientText = os.Getenv("WaitClientText")
	alertText.NotInMatchMaking = os.Getenv("NotInMatchMaking")
	alertText.NotSelectChamp = os.Getenv("NotSelectChampAlert")

	messageText.NotStarted = os.Getenv("NotStarted")
	messageText.PickLockCancel = os.Getenv("PickLockCancel")
	messageText.NotSelectChamp = os.Getenv("NotSelectChampMess")
	messageText.MatchCancelled = os.Getenv("MatchCancelled")
	messageText.ReadyPickLock = os.Getenv("ReadyPickLock")
	messageText.PickLockSuccess = os.Getenv("PickLockSuccess")

	os.Clearenv()
	return uiText, alertText, messageText
}
