package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var username = "riot"
var authToken string
var appPort string
var champs []Champion

type Champion struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	AssignAuthTokensAndAppPorts()
	//champions := GetChampList()
	//
	//fmt.Println("Danh sách các champion:")
	//for _, champion := range champions {
	//	fmt.Printf("ID: %d, Name: %s\n", champion.ID, champion.Name)
	//}

	//champs := GetChampList()

	// Tạo một channel để định thời gian gọi hàm
	tick := time.Tick(250 * time.Millisecond)

	// Vòng lặp vô hạn
	c := 0
	for {
		c = c + 1
		select {
		case <-tick:
			if CheckMatchFound() == true {
				fmt.Println("MATCH FOUND!!")
				AcceptMatch()
				id := GetActionID()
				if id > -1 {
					sId := strconv.Itoa(id)
					PickChampion(sId, strconv.Itoa(233))
					LockChampion(sId)

					os.Exit(0)
				}

			} else {
				fmt.Printf("waiting... %d", c)
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
