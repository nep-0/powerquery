package query

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type Queryer interface {
	DoQuery(req QueryRequest) (QueryResponse, error)
}

type QueryRequest struct {
	Username string
	Password string
	Cookies  string
	RoomName string
}

type QueryResponse struct {
	Balance string
	Power   string
	Cookies string
}

type RodQueryer struct {
	browser *rod.Browser
	mu      sync.Mutex
}

func NewRodQueryer(url string) (*RodQueryer, error) {
	browser := rod.New().ControlURL(url).MustConnect()
	return &RodQueryer{
		browser: browser,
		mu:      sync.Mutex{},
	}, nil
}

func (rq *RodQueryer) DoQuery(req QueryRequest) (QueryResponse, error) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if req.Username == "" || req.Password == "" {
		if req.Cookies == "" {
			return QueryResponse{}, fmt.Errorf("missing authentication fields")
		}
	}

	if req.RoomName == "" {
		return QueryResponse{}, fmt.Errorf("missing room name")
	}

	if req.Cookies != "" {
		rq.setCookies(req.Cookies)
	}

	page := rq.browser.MustPage("https://eportal.uestc.edu.cn/qljfwapp/sys/lwUestcDormElecPrepaid/index.do#/record")
	defer page.Close()
	page.MustWaitDOMStable()

	if page.MustInfo().Title == "Unified identity authentication platform" {
		// username
		page.MustElement("#loginViewDiv > div:nth-child(1) > form:nth-child(1) > div:nth-child(1) > div:nth-child(1) > div:nth-child(1) > input:nth-child(3)").MustInput(req.Username)
		// password
		page.MustElement("#loginViewDiv > div:nth-child(1) > form:nth-child(1) > div:nth-child(1) > div:nth-child(1) > div:nth-child(2) > input:nth-child(3)").MustInput(req.Password)
		// remember me
		page.MustElement("#loginViewDiv > div:nth-child(1) > form:nth-child(1) > div:nth-child(1) > div:nth-child(1) > div:nth-child(4) > input:nth-child(1)").MustClick()
		// login
		page.MustElement("#loginViewDiv > div:nth-child(1) > form:nth-child(1) > div:nth-child(1) > div:nth-child(2) > div:nth-child(2) > a:nth-child(1)").MustClick()

		// Wait for navigation
		page.MustWaitNavigation()
	}

	page.MustWaitDOMStable()
	if page.MustInfo().Title != "清水河校区寝室电费充值" {
		page.MustScreenshot("debug.png")
		return QueryResponse{}, fmt.Errorf("failed to log in or navigate to the correct page: %s", page.MustInfo().Title)
	}

	roomId := "[{\"DORM_ID\":\"" + req.RoomName + "\"}]"
	resp := page.MustEval(`() => {
		return fetch('https://eportal.uestc.edu.cn/qljfwapp/sys/lwUestcDormElecPrepaid/dormElecPrepaidMan/queryRoomInfo.do', {
			method: 'POST',
			headers: {
					'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8'
				},
				body: 'roomIds=' + encodeURIComponent('` + roomId + `')
			}).then(response => response.json());
		}`)

	if resp.Get("0.code").Str() != "0" {
		return QueryResponse{}, fmt.Errorf("failed to query room info")
	}

	return QueryResponse{
		Balance: resp.Get("0.roomInfo.syje").Str(),
		Power:   resp.Get("0.roomInfo.sydl").Str(),
		Cookies: rq.getCookies(),
	}, nil
}

func (rq *RodQueryer) getCookies() string {
	cookies := rq.browser.MustGetCookies()
	cookiesJSON, _ := json.Marshal(cookies)
	return string(cookiesJSON)
}

func (rq *RodQueryer) setCookies(cookies string) {
	var networkCookies []*proto.NetworkCookie
	json.Unmarshal([]byte(cookies), &networkCookies)
	rq.browser.MustSetCookies(networkCookies...)
}
