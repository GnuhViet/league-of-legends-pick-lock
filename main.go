package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gen2brain/beeep"
	"hello/src/utils"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	username         = "riot"
	authToken        string
	appPort          string
	champs           []Champion
	selectedChampion Champion
	stopChan         chan bool
	pickLocking      = false
)

var (
	selectLanguage = widget.NewSelect([]string{"en", "vi"}, nil)
	selectEntry    = widget.NewSelectEntry(nil)
	confirmButton  = widget.NewButton("", nil)
	startButton    = widget.NewButton("", nil)
	stopButton     = widget.NewButton("", nil)
	checkboxLock   = widget.NewCheck("", nil)
	pickLockButton = widget.NewButton("", nil)
	statusLabel    = widget.NewLabel("")
	resultLabel    = widget.NewLabel("")
	runningLabel   = widget.NewLabel("")
	titleLabel     = widget.NewLabel("")
)

var (
	uiText      utils.UIText
	alertText   utils.AlertText
	messageText utils.MessageText
	appConfig   utils.Config
)

type Champion struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

////////////
// -------------------------------------------- main -------------------------------------
///////////

func main() {
	AssignAuthTokensAndAppPorts()
	champs = GetChampList()
	InitUI(GetStringChampName(champs))
}

func GetStringChampName(champs []Champion) []string {
	championNames := make([]string, len(champs))
	for i, champ := range champs {
		championNames[i] = champ.Name
	}
	return championNames
}

/////////////
// --------------------------------------------- ui --------------------------------------
///////////

func ReloadUIText() {
	titleLabel.SetText(uiText.SelectChampText)
	confirmButton.SetText(uiText.ConfirmButtonText)
	startButton.SetText(uiText.StartButtonText)
	stopButton.SetText(uiText.StopButtonText)
	checkboxLock.SetText(uiText.CheckboxLockText)
	pickLockButton.SetText(uiText.PickLockButtonText)
	statusLabel.SetText(uiText.StatusLabelText)

	if selectedChampion.ID == -1 || selectedChampion.Name == "" {
		resultLabel.SetText(uiText.ResultLabelText)
	} else {
		resultText := fmt.Sprintf(uiText.SelectedChampText+" "+selectedChampion.Name+"-- ID: %d", selectedChampion.ID)
		resultLabel.SetText(resultText)
	}

	if pickLocking == true {
		runningLabel.SetText(messageText.ReadyPickLock)
	} else {
		runningLabel.SetText(messageText.NotStarted)
	}
}

func InitUI(championNames []string) {
	appConfig = utils.ReadIniFile()
	uiText, alertText, messageText = utils.ReadEnv(appConfig.Language)

	myApp := app.New()
	myWindow := myApp.NewWindow(uiText.AppNameText)
	myWindow.Resize(fyne.NewSize(400, 300))
	myWindow.SetFixedSize(true)

	titleLabel.SetText(uiText.SelectChampText)
	confirmButton.SetText(uiText.ConfirmButtonText)
	startButton.SetText(uiText.StartButtonText)
	stopButton.SetText(uiText.StopButtonText)
	checkboxLock.SetText(uiText.CheckboxLockText)
	pickLockButton.SetText(uiText.PickLockButtonText)
	resultLabel.SetText(uiText.ResultLabelText)
	statusLabel.SetText(uiText.StatusLabelText)
	runningLabel.SetText(messageText.NotStarted)
	selectLanguage.Selected = appConfig.Language

	selectEntry.SetOptions(championNames)

	checkboxLock.Checked = appConfig.AutoLock
	checkboxLock.Refresh()
	startButton.Disable()
	stopButton.Disable()
	pickLockButton.Disable()

	/*	startButton.OnTapped = func() {
			beeep.Alert("Cảnh báo", "Bắt đầu pick-lock", "pick-lock.ico")

			startButton.Disable()
			stopButton.Enable()
			confirmButton.Disable()
			selectEntry.Disable()

			stopChan = make(chan bool)
			go StartAcceptMatchPickLock(selectedChampion.ID)
		}

		stopButton.OnTapped = func() {
			stopChan <- true // Gửi thông báo để kết thúc goroutine (nếu đang chạy).
			<-stopChan       // Đợi goroutine thực sự kết thúc.

			stopButton.Disable()
			startButton.Enable()
			confirmButton.Enable()
			selectEntry.Enable()
			runningLabel.SetText("Chưa bắt đầu")
		}*/

	selectLanguage.OnChanged = func(vi string) {
		uiText, alertText, messageText = utils.ReadEnv(selectLanguage.Selected)
		ReloadUIText()
	}

	var spam = true

	pickLockButton.OnTapped = func() {
		if isInMatchMaking() == false {
			if spam {
				spam = !spam
				runningLabel.SetText(messageText.NotFoundMatch)
			} else {
				spam = !spam
				runningLabel.SetText(messageText.PleaseWait)
			}
			//fmt.Println("button not accepted")
			return
		}

		stopButton.Enable()
		pickLockButton.Disable()
		confirmButton.Disable()
		selectEntry.Disable()

		stopChan = make(chan bool)
		go StartPickLock(selectedChampion.ID)
	}

	stopButton.OnTapped = func() {
		stopChan <- true // Send notification to end goroutine (if running).
		<-stopChan       // Wait for the goroutine to actually finish.

		stopButton.Disable()
		pickLockButton.Enable()
		confirmButton.Enable()
		selectEntry.Enable()
		runningLabel.SetText(messageText.PickLockCancel)
	}

	// select entry event
	// if empty text then set option to all
	// search and fill the option by input text
	selectEntry.OnChanged = func(championName string) {
		if len(championName) == 0 {
			selectEntry.SetOptions(championNames)
			selectedChampion.Name = ""
			selectedChampion.ID = -1

			resultLabel.SetText(messageText.NotSelectChamp)
			startButton.Disable()
			stopButton.Disable()
			pickLockButton.Disable()
			return
		}

		var champsSlice []string

		for _, champs := range champs {
			if strings.HasPrefix(strings.ToLower(champs.Name), strings.ToLower(championName)) {
				champsSlice = append(champsSlice, champs.Name)
			}
		}

		selectEntry.SetOptions(champsSlice)

		// search and fill the option by input text
		for _, champ := range champs {
			if champ.Name == championName {
				selectedChampion = champ
				break
			}
		}
	}

	confirmButton.OnTapped = func() {
		if selectedChampion.ID == -1 || selectedChampion.Name == "" {
			resultLabel.SetText(messageText.NotSelectChamp)
			beeep.Alert(alertText.Notification, alertText.NotSelectChamp, "pick-lock.ico")
			return
		}

		resultText := fmt.Sprintf(uiText.SelectedChampText+" "+selectedChampion.Name+" -- ID: %d", selectedChampion.ID)
		resultLabel.SetText(resultText)
		startButton.Enable()
		pickLockButton.Enable()
	}

	//------------ layout----------
	tileBox := container.NewWithoutLayout(
		titleLabel,
		selectLanguage,
	)

	selectLanguage.Move(fyne.NewPos(280, 8))
	selectLanguage.Resize(fyne.NewSize(58, 20))

	selectChampBox := container.NewWithoutLayout(
		selectEntry,
		confirmButton,
	)

	selectEntry.Move(fyne.NewPos(20, 1))
	selectEntry.Resize(fyne.NewSize(220, 36))

	confirmButton.Move(fyne.NewPos(270, 1))
	confirmButton.Resize(fyne.NewSize(80, 36))

	content := container.NewVBox(
		tileBox,
		selectChampBox,
		resultLabel,
		//startButton,
		checkboxLock,
		pickLockButton,
		stopButton,
		container.NewHBox(
			statusLabel,
			runningLabel,
		),
	)

	myWindow.SetMaster()
	myWindow.SetOnClosed(func() {
		appConfig.Language = selectLanguage.Selected
		appConfig.AutoLock = checkboxLock.Checked
		utils.WriteIniFile(appConfig)
	})

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

// ------------------------------------ pick lock logic --------------------------

// StartPickLock if not in matchmaking then stop the process
// spam pick & lock request
// default is 250ms polling rate
func StartPickLock(championID int) {
	defer func() {
		//fmt.Println("stopping goroutine")
		pickLocking = false
		stopButton.Disable()
		startButton.Enable()
		confirmButton.Enable()
		pickLockButton.Enable()
		selectEntry.Enable()

		// do any cleanup/logic above this line!!!
		stopChan <- true // Send notification when goroutine finishes.
		close(stopChan)
	}()

	AcceptMatch()

	// polling with 250ms time
	tick := time.Tick(250 * time.Millisecond)
	var done = false
	pickLocking = true

	var matchAccepted = false

	c := 0
	for !done {
		c++
		select {
		case <-stopChan:
			return // End the goroutine if a tapped the stop button
		case <-tick:
			if isMatchAccepted() {
				matchAccepted = true
			}

			if matchAccepted {
				if !isMatchAccepted() {
					runningLabel.SetText(messageText.MatchCancelled)
					done = true
					break
				}
			}

			dots := strings.Repeat(".", c)
			runningLabel.SetText(messageText.ReadyPickLock + dots)
			if c == 6 {
				c = 0
			}

			id := GetActionID()
			if id > -1 {
				sId := strconv.Itoa(id)
				PickChampion(sId, strconv.Itoa(championID))
				if checkboxLock.Checked == true {
					LockChampion(sId)
				}
				runningLabel.SetText(messageText.PickLockSuccess)
				done = true
			}
		}
	}
}

////
// --------------------------------------------- API ----------------------------------------
////

func CallApi(api string, method string, data []byte) []byte {

	url := fmt.Sprintf("https://127.0.0.1:%s%s", appPort, api)

	req, err := http.NewRequest(method, url, nil)

	if method == "PATCH" {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		//fmt.Println("Lỗi khi tạo yêu cầu HTTP:", err)
		return nil
	}

	req.SetBasicAuth(username, authToken)
	req.Header.Set("User-Agent", "My-User-Agent")

	// disable dls check
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: time.Millisecond * 500}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode == 404 {
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return body
}

func GetChampList() []Champion {
	body := CallApi("/lol-champions/v1/owned-champions-minimal", "GET", nil)

	if body == nil {
		return nil
	}

	var champions []Champion
	err := json.Unmarshal(body, &champions)
	if err != nil {
		return nil
	}

	sort.Slice(champions, func(i, j int) bool {
		return champions[i].Name < champions[j].Name
	})

	return champions
}

func isMatchAccepted() bool {
	body := CallApi("/lol-matchmaking/v1/search", "GET", nil)

	if body == nil {
		return false
	}

	var data map[string]interface{}
	err := json.Unmarshal([]byte(body), &data)
	if err != nil {
		return false
	}

	readyCheck, ok := data["readyCheck"].(map[string]interface{})
	if !ok {
		return false
	}

	fmt.Println(readyCheck)

	playerResponse, ok := readyCheck["playerResponse"].(string)
	if !ok {
		return false
	}

	fmt.Println(playerResponse)

	return playerResponse == "Accepted"
}

func isInMatchMaking() bool {
	body := CallApi("/lol-matchmaking/v1/ready-check", "GET", nil)

	if body == nil {
		return false
	}

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return false
	}

	state, ok := data["state"].(string)
	if !ok {
		return false
	}

	return state == "InProgress"
}

func GetActionID() int {
	body := CallApi("/lol-champ-select/v1/session", "GET", nil)

	if body == nil {
		return -1
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal(body, &sessionData); err != nil {
		return -1
	}

	localPlayerCellID, ok := sessionData["localPlayerCellId"].(float64)
	if !ok {
		return -1
	}

	actions, ok := sessionData["actions"].([]interface{})
	if !ok {
		return -1
	}

	for _, action := range actions {
		actionList, ok := action.([]interface{})
		if !ok {
			continue
		}

		for _, subAction := range actionList {
			actionData, ok := subAction.(map[string]interface{})
			if !ok {
				continue
			}

			actorCellID, ok := actionData["actorCellId"].(float64)
			if !ok {
				continue
			}

			if actorCellID == localPlayerCellID {
				id, ok := actionData["id"].(float64)
				if !ok {
					return -1
				}

				return int(id)
			}
		}
	}

	return -1
}

func AcceptMatch() {
	CallApi("/lol-matchmaking/v1/ready-check/accept", "POST", nil)
}

func PickChampion(actionId string, championID string) {
	data := map[string]string{"championId": championID}
	jsonData, _ := json.Marshal(data)

	CallApi("/lol-champ-select/v1/session/actions/"+actionId, "PATCH", jsonData)
}

func LockChampion(actionId string) {
	CallApi("/lol-champ-select/v1/session/actions/"+actionId+"/complete", "POST", nil)
}

func AssignAuthTokensAndAppPorts() {
	command := "WMIC PROCESS WHERE name='LeagueClientUx.exe' GET commandline"

	cmd := exec.Command("cmd", "/C", command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		//fmt.Println("Lỗi:", err)
		return
	}

	inputString := string(output)
	reAuthToken := regexp.MustCompile(`"--remoting-auth-token=([^"]+)"`)
	reAppPort := regexp.MustCompile(`"--app-port=([^"]+)"`)

	authTokens := reAuthToken.FindAllStringSubmatch(inputString, -1)

	appPorts := reAppPort.FindAllStringSubmatch(inputString, -1)

	if !(len(authTokens) > 0 && len(appPorts) > 0) {
		err := beeep.Alert("CLIENT NOT FOUND!!!", "Please run the game first", "pick-lock.ico")
		if err != nil {
			return
		}
		os.Exit(0)
	}

	authToken = authTokens[0][1]
	appPort = appPorts[0][1]
}
