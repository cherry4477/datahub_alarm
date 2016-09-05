package ds

import "time"

const (
	ResultOK       = 0
	ErrorUnmarshal = iota + 6000
	ErrorMarshal
	ErrorSendAsyncMessage
	ErrorReadAll
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type GetTokenResult struct {
	Access_token string
	Expires_in   int
}

type SendInfo struct {
	Touser  string `json:"touser, omitempty"`
	Toparty string `json:"toparty, omitempty"`
	Totag   string `json:"totag, omitempty"`
	Msgtype string `json:"msgtype"`
	Agentid int    `json:"agentid"`
	Text    Text   `json:"text"`
	Safe    int    `json:"safe, omitempty"`
}

type Text struct {
	Content string `json:"content"`
}

type AlarmEvent struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Send_time time.Time `json:"sendTime"`
}
