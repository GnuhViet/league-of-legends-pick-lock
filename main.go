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
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var username = "riot"
var authToken string
var appPort string
var champs []Champion
var selectedChampion Champion

var stopChan chan bool

var runningLable = widget.NewLabel("")

type Champion struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

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

func InitUI(championNames []string) {
	myApp := app.New()
	myWindow := myApp.NewWindow("Fyne SelectEntry Example")
	myWindow.Resize(fyne.NewSize(400, 300))
	myWindow.SetFixedSize(true)
	tienDoLable := widget.NewLabel("Tiến độ: ")

	startButton := widget.NewButton("Bắt đầu", func() {
		if stopChan == nil {
			stopChan = make(chan bool)
			go StartPickLock()
		}
	})

	stopButton := widget.NewButton("Stop", func() {
		if stopChan != nil {
			stopChan <- true // Gửi thông báo để kết thúc goroutine (nếu đang chạy).
			<-stopChan       // Đợi goroutine thực sự kết thúc.
			stopChan = nil

			runningLable.SetText("")
		}
	})

	startButton.Hidden = true
	stopButton.Hidden = true
	runningLable.Hidden = true
	tienDoLable.Hidden = true

	// Tạo một SelectEntry với dữ liệu từ mảng tên champion
	selectEntry := widget.NewSelectEntry(championNames)

	// Tạo một label ban đầu với nội dung trống
	resultLabel := widget.NewLabel("Chưa chọn champion")

	// Định nghĩa sự kiện khi người dùng chọn một champion
	selectEntry.OnChanged = func(championName string) {
		if len(championName) == 0 {
			selectEntry.SetOptions(championNames)
			selectedChampion.Name = ""
			selectedChampion.ID = -1
			return
		}

		var champsSlice []string

		for _, champs := range champs {
			if strings.HasPrefix(strings.ToLower(champs.Name), strings.ToLower(championName)) {
				champsSlice = append(champsSlice, champs.Name)
			}
		}

		selectEntry.SetOptions(champsSlice)

		// Tìm struct Champion tương ứng với tên champion được chọn
		for _, champ := range champs {
			if champ.Name == championName {
				selectedChampion = champ
				break
			}
		}
	}

	// Tạo một nút xác nhận
	confirmButton := widget.NewButton("Xác nhận", func() {
		if selectedChampion.ID == -1 || selectedChampion.Name == "" {
			resultLabel.SetText("Chưa chọn champion")
			startButton.Hidden = true
			stopButton.Hidden = true
			runningLable.Hidden = true
			tienDoLable.Hidden = true
			return
		}

		// Xử lý sự kiện tại đây, ví dụ: in ra thông tin của champion được chọn
		resultText := fmt.Sprintf("Champion được chọn - ID: %d, Tên: %s", selectedChampion.ID, selectedChampion.Name)
		resultLabel.SetText(resultText)
		startButton.Hidden = false
		stopButton.Hidden = false
		runningLable.Hidden = false
		tienDoLable.Hidden = false
	})

	// layout
	selectChampBox := container.NewWithoutLayout(
		selectEntry,
		confirmButton,
	)

	selectEntry.Move(fyne.NewPos(20, 1))
	selectEntry.Resize(fyne.NewSize(220, 36))

	confirmButton.Move(fyne.NewPos(270, 1))
	confirmButton.Resize(fyne.NewSize(80, 36))

	// Đặt SelectEntry vào một containter để hiển thị
	content := container.NewVBox(
		widget.NewLabel("Chọn một champion:"),
		selectChampBox,
		resultLabel,
		startButton,
		stopButton,
		widget.NewLabel(""),
		container.NewHBox(
			tienDoLable,
			runningLable,
		),
	)

	// Đặt nội dung vào cửa sổ
	myWindow.SetContent(content)

	// Hiển thị cửa sổ
	myWindow.ShowAndRun()
}

func StartPickLock() {
	defer func() {
		stopChan <- true // Gửi thông báo khi goroutine kết thúc.
		close(stopChan)
	}()

	// Tạo một channel để định thời gian gọi hàm
	tick := time.Tick(250 * time.Millisecond)

	// Vòng lặp vô hạn
	c := 0
	for {
		c = c + 1
		select {
		case <-stopChan:
			return // Kết thúc goroutine nếu nhận được thông báo từ channel.
		case <-tick:

			if !(CheckMatchFound() == true) {
				// Tạo một chuỗi mới với số lượng dấu chấm "." dựa trên giá trị của c
				dots := strings.Repeat(".", c)
				runningLable.SetText("waiting" + dots)
				//fmt.Printf("waiting%s", dots)
				if c == 4 {
					c = 0
				}

				continue
			}

			runningLable.SetText("MATCH FOUND!!")

			AcceptMatch()
			id := GetActionID()
			if id > -1 {
				sId := strconv.Itoa(id)
				PickChampion(sId, strconv.Itoa(233))
				LockChampion(sId)
				runningLable.SetText("PICK-LOCK SUCCESS!, ENJOY YOUR GAME!!!")
				break
			}
		}
	}
}

func CallApi(api string, method string, data []byte) []byte {
	// Tạo URL dựa trên appPort
	url := fmt.Sprintf("https://127.0.0.1:%s%s", appPort, api)

	// Tạo yêu cầu HTTP
	req, err := http.NewRequest(method, url, nil)

	if method == "PATCH" {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		fmt.Println("Lỗi khi tạo yêu cầu HTTP:", err)
		return nil
	}

	// Đặt tên đăng nhập và mật khẩu
	req.SetBasicAuth(username, authToken)
	req.Header.Set("User-Agent", "My-User-Agent")

	// Gửi yêu cầu HTTP
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: time.Millisecond * 500}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Lỗi khi gửi yêu cầu HTTP:", err)
		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil
	}

	// Đọc phản hồi từ máy chủ
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Lỗi khi đọc phản hồi từ máy chủ:", err)
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
		fmt.Println("Lỗi khi ánh xạ JSON:", err)
		return nil
	}

	sort.Slice(champions, func(i, j int) bool {
		return champions[i].Name < champions[j].Name
	})

	return champions
}

func CheckMatchFound() bool {
	body := CallApi("/lol-matchmaking/v1/ready-check", "GET", nil)

	if body == nil {
		return false
	}

	var data map[string]interface{} // Sử dụng map[string]interface{} để giải mã JSON
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

	// Truy cập giá trị của trường "localPlayerCellId"
	localPlayerCellID, ok := sessionData["localPlayerCellId"].(float64)
	if !ok {
		return -1
	}

	// Lặp qua danh sách "actions" trong sessionData
	actions, ok := sessionData["actions"].([]interface{})
	if !ok {
		return -1
	}

	for _, action := range actions {
		// Chuyển đổi action thành danh sách các action bên trong
		actionList, ok := action.([]interface{})
		if !ok {
			continue
		}

		// Lặp qua danh sách action bên trong
		for _, subAction := range actionList {
			actionData, ok := subAction.(map[string]interface{})
			if !ok {
				continue
			}

			// Lấy giá trị "actorCellId" từ actionData
			actorCellID, ok := actionData["actorCellId"].(float64)
			if !ok {
				continue
			}

			// Kiểm tra nếu actorCellId trùng với localPlayerCellID
			if actorCellID == localPlayerCellID {
				// Lấy giá trị "id" tương ứng
				id, ok := actionData["id"].(float64)
				if !ok {
					return -1
				}

				// Trả về giá trị "id" tương ứng
				return int(id)
			}
		}
	}

	// Trả về -1 nếu không tìm thấy
	return -1
}

func AcceptMatch() {
	CallApi("/lol-matchmaking/v1/ready-check/accept", "POST", nil)
}

func PickChampion(actionId string, championID string) {
	// Tạo dữ liệu JSON để gửi lên server
	data := map[string]string{"championId": championID}
	jsonData, _ := json.Marshal(data)

	CallApi("/lol-champ-select/v1/session/actions/"+actionId, "PATCH", jsonData)
}

func LockChampion(actionId string) {
	CallApi("/lol-champ-select/v1/session/actions/"+actionId+"/complete", "POST", nil)
}

func AssignAuthTokensAndAppPorts() {
	// Lệnh bạn muốn thực thi (ví dụ: "ipconfig /all")
	command := "WMIC PROCESS WHERE name='LeagueClientUx.exe' GET commandline"

	// Tạo một cửa sổ cmd.exe và chạy lệnh
	cmd := exec.Command("cmd", "/C", command)

	// Thực thi lệnh và nhận kết quả đầu ra
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Lỗi:", err)
		return
	}

	inputString := string(output)
	// Tạo biểu thức chính quy để tìm chuỗi "--remoting-auth-token=" và "--app-port="
	reAuthToken := regexp.MustCompile(`"--remoting-auth-token=([^"]+)"`)
	reAppPort := regexp.MustCompile(`"--app-port=([^"]+)"`)

	// Tìm chuỗi "--remoting-auth-token="
	authTokens := reAuthToken.FindAllStringSubmatch(inputString, -1)

	// Tìm chuỗi "--app-port="
	appPorts := reAppPort.FindAllStringSubmatch(inputString, -1)

	if !(len(authTokens) > 0 && len(appPorts) > 0) {
		fmt.Println("Không tìm thấy Auth Token hoặc App Port")
	}

	authToken = authTokens[0][1] // Lấy giá trị từ nhóm con [1] của kết quả
	appPort = appPorts[0][1]     // Lấy giá trị từ nhóm con [1] của kết quả
}
