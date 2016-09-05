package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/asiainfoLDP/datahub/utils/clog"
	"github.com/asiainfoLDP/datahub_alarm/ds"
	"github.com/asiainfoLDP/datahub_commons/mq"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

const (
	Platform_Local      = "local"
	Platform_DaoCloud   = "daocloud"
	Platform_DaoCloudUT = "daocloud_ut"
	Platform_DataOS     = "dataos"

	MQ_TOPIC_TO_ALARM = "to_alarm.json"
	MQ_LISTEN_OTHERS  = "others_to_alarm"

	TOUSER  = "hehl"
	TOPARTY = "6"
	MSGTYPE = "text"
	AGENTID = 2

	SERVICE_PORT = "0.0.0.0:8080"
)

var (
	corpid      string
	corpsecret  string
	accessToken string
	Platform    = Platform_DaoCloud

	Service_Name_Kafka           string
	DISCOVERY_CONSUL_SERVER_ADDR string
	DISCOVERY_CONSUL_SERVER_PORT string

	AlarmMQ mq.MessageQueue
)

func getAccessToken(corpid, corpsecret string) (string, error) {

	client := &http.Client{}
	url := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?" +
		"corpid=" + corpid + "&" +
		"corpsecret=" + corpsecret
	resp, err := client.Get(url)
	if err != nil {
		log.Error("Get accesstoken err:", err)
		return "", err
	}
	defer resp.Body.Close()

	var result = ds.GetTokenResult{}
	if resp.StatusCode == http.StatusOK {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("ioutil.ReadAll err:", err)
			return "", err
		}

		err = json.Unmarshal(respBody, &result)
		if err != nil {
			log.Error("Unmarshal err:", err)
			return "", err
		}
	}

	return result.Access_token, err
}

func refreshAccessToken() {
	ticker := time.Tick(2 * time.Hour)
	for _ = range ticker {
		newAccessToken, err := getAccessToken(corpid, corpsecret)
		if err != nil {
			log.Error("refresh accesstoken err:", err)
			return
		}
		if newAccessToken != "" {
			accessToken = newAccessToken
			log.Infof("refresh accessToken in ticker: %s", accessToken)
		} else {
			log.Warnf("blank new accessToken in ticker: %s", newAccessToken)
		}
	}
}

func sendAlarm() (bool, error) {

	text := ds.Text{}
	text.Content = fmt.Sprintf("Sender: %s\nAlarm: %s", msg.Sender, msg.Content)
	info := ds.SendInfo{Touser: TOUSER, Toparty: TOPARTY, Msgtype: MSGTYPE, Agentid: AGENTID, Text: text, Safe: 0}

	body, err := json.Marshal(&info)
	if err != nil {
		log.Error("Marshal err:", err)
		return false, err
	}

	client := &http.Client{}
	url := "https://qyapi.weixin.qq.com/cgi-bin/message/send?" +
		"access_token=" + accessToken
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Error("http.NewRequest err:", err)
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("client.Do err:", err)
		return false, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("ioutil.ReadAll err:", err)
		return false, err
	}
	log.Info(string(respBody))

	return true, err
}

func initMQ() {
	connectMQ()
	go updateMQ()
}

func connectMQ() {
	ip, port := getKafkaAddr()
	//kafka := fmt.Sprintf("%s:%s", os.Getenv("MQ_KAFKA_ADDR"), os.Getenv("MQ_KAFKA_PORT"))
	//ip, port := "10.1.235.98", "9092"
	kafka := fmt.Sprintf("%s:%s", ip, port)
	log.Infof("kafkas = %s", kafka)
	var err error
	AlarmMQ, err = mq.NewMQ([]string{kafka}) // ex. {"192.168.1.1:9092", "192.168.1.2:9092"}
	if err != nil {
		log.Errorf("initMQ error: %s", err.Error())
		return
	}

	// ...

	myListener := newMyMesssageListener(MQ_LISTEN_OTHERS)

	//AlarmMQ.SendSyncMessage(MQ_TOPIC_TO_ALARM, []byte(""), []byte("")) // force create the topic
	// 0 is the partition id
	err = AlarmMQ.SetMessageListener(MQ_TOPIC_TO_ALARM, 0, mq.Offset_Marked, myListener)
	if err != nil {
		log.Error("SetMessageListener (to_alarm.json) error: ", err)
	}

	atomic.StoreInt64(&nMqErrors, 0)
}

func updateMQ() {
	var err error
	ticker := time.Tick(5 * time.Second)
	for _ = range ticker {
		if AlarmMQ == nil {
			connectMQ()
		} else if err = pingMQ(); err != nil {
			AlarmMQ.Close()
			//theMQ = nil // draw snake feet
			connectMQ()
		}
	}
}

func pingMQ() error {
	n := atomic.LoadInt64(&nMqErrors)
	if n > 0 {
		return MqListenerAnyError
	}

	_, _, err := AlarmMQ.SendSyncMessage("__ping__", []byte(""), []byte(""))
	return err
}

func getKafkaAddr() (string, string) {
	switch Platform {
	case Platform_DaoCloud:
		entryList := dnsExchange(Service_Name_Kafka, DISCOVERY_CONSUL_SERVER_ADDR, DISCOVERY_CONSUL_SERVER_PORT)

		for _, v := range entryList {
			if v.port == "9092" {
				return v.ip, v.port
			}
		}
	case Platform_DataOS:
		return os.Getenv(os.Getenv("ENV_NAME_KAFKA_ADDR")), os.Getenv(os.Getenv("ENV_NAME_KAFKA_PORT"))
	case Platform_Local:
		return os.Getenv("MQ_KAFKA_ADDR"), os.Getenv("MQ_KAFKA_PORT")
	}
	return "", ""
}

func refreshKafka() {
	log.Debug("BEGIN")
	defer log.Debug("END")

	for {
		select {
		case <-time.After(time.Second * 5):
			if AlarmMQ == nil {
				initMQ()
			} else if _, _, err := AlarmMQ.SendSyncMessage("__ping__", []byte(""), []byte("")); err != nil {
				AlarmMQ.Close()
				initMQ()
			}
		}
	}
}

func sendMessage() {
	event := ds.AlarmEvent{Sender: "wangmeng", Content: "这只是个测试", Send_time: time.Now()}
	log.Debug("Begin send message")
	defer log.Debug("End send message")

	b, err := json.Marshal(&event)
	if err != nil {
		log.Error("Marshal err:", err)
		return
	}
	_, _, err = AlarmMQ.SendSyncMessage(MQ_TOPIC_TO_ALARM, []byte("alarm to weixin"), b)
	if err != nil {
		log.Errorf("SendAsyncMessage error: %s", err.Error())
		return
	}
	return
}

func main() {

	go refreshAccessToken()

	go refreshKafka()

	router := httprouter.New()
	router.GET("/alarm", rootHandler)
	router.POST("/alarm", sendMessageHandler)

	log.Info("listening on", SERVICE_PORT)
	err := http.ListenAndServe(SERVICE_PORT, router)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func init() {
	corpid = getEnv("CORPID", true)
	corpsecret = getEnv("CORPSECRET", true)

	Service_Name_Kafka = getEnv("kafka_service_name", false)
	DISCOVERY_CONSUL_SERVER_ADDR = getEnv("CONSUL_SERVER", false)
	DISCOVERY_CONSUL_SERVER_PORT = getEnv("CONSUL_DNS_PORT", false)

	initAccessToken, err := getAccessToken(corpid, corpsecret)
	if err != nil && initAccessToken == "" {
		log.Error("Init accessToken err:", err, initAccessToken)
	} else {
		accessToken = initAccessToken
		log.Infof("Init accessToken in ticker: %s", accessToken)
	}
	log.Info("Accesstoken is", accessToken)

	initMQ()
}
