package main

import (
	"encoding/json"
	log "github.com/asiainfoLDP/datahub/utils/clog"
	"sync/atomic"
	"time"
	"errors"
)

var (
	msg       alarmEvent
	nMqErrors int64
	MqListenerAnyError = errors.New("error when receiving messages")
)

type alarmEvent struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Send_time time.Time `json:"sendTime"`
}

type MyMesssageListener struct {
	name string
}

func newMyMesssageListener(name string) *MyMesssageListener {
	return &MyMesssageListener{name: name}
}

func (listener *MyMesssageListener) OnMessage(topic string, partition int32, offset int64, key, value []byte) bool {
	log.Debugf("%s received: (%d) message: %s", listener.name, offset, string(value))
	if len(value) > 0 {
		if err := json.Unmarshal(value, &msg); err != nil {
			log.Error("Unmarshal error:", err)
			return false
		}
		if isSend, err := sendAlarm(); err == nil && isSend == true {
			log.Info("Send alarm succeed!")
			return true
		}
	}
	return true // to save offset on server
}

func (listener *MyMesssageListener) OnError(err error) bool {
	atomic.AddInt64(&nMqErrors, 1)

	log.Debugf("Alarmlistener OnError: %s", err.Error())
	return false // will not stop listenning
}
