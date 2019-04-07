package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"
)

var sendAlarm = true
var (
	newClient *http.Client
)

func init() {
	transport := http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, time.Second*3)
		},
		DisableKeepAlives: true,
	}
	newClient = &http.Client{
		Transport: &transport,
	}
}

type payloadMetric struct {
	Mtype     string      `json:"mtype"`       // metric type(host,switch,docker)
	Host      string      `json:"host"`        // 机器名
	Metric    string      `json:"metric"`      // 监控数据项,点号分割,第一段为Mtype
	Value     string      `json:"value"`       // 监控数据值
	Step      int64       `json:"step"`        // 数据间隔
	Type      string      `json:"counterType"` // 统计方式
	Status    string      `json:"status"`
	Conf      interface{} `json:"conf"`
	Func      string      `json:"func"`
	Clusters  string      `json:"clusters"`
	Timestamp int64       `json:"timestamp"`
}

func alarmToServer(metric, msg string) {
	if !sendAlarm {
		return
	}
	data := payloadMetric{
		Mtype:     "host",
		Host:      "10.1.1.1",
		Metric:    metric,
		Step:      60,
		Type:      "ALARM",
		Clusters:  "network_hardware_all",
		Timestamp: time.Now().Unix(),
	}
	if msg != "" {
		data.Status = "err"
		data.Value = msg
		data.Func = "FireInTheHole"
	} else {
		data.Status = "ok"
	}
	jsonStr, _ := json.Marshal([]payloadMetric{data})
	req, err := http.NewRequest("POST", "http://10.1.1.1:8091/api/receiver", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println(err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := newClient.Do(req)
	if err != nil {
		log.Println(err.Error())
		return
	}
	resp.Body.Close()
}

func recordToServer(metric, value string) {
	data := payloadMetric{
		Mtype:     "host",
		Host:      "10.1.1.1",
		Metric:    metric,
		Step:      60,
		Type:      "GAUGE",
		Clusters:  "aaa",
		Timestamp: time.Now().Unix(),
	}
	data.Value = value
	jsonStr, _ := json.Marshal([]payloadMetric{data})
	log.Printf("Sent metric data: %s\n", string(jsonStr))
	req, err := http.NewRequest("POST", "http://10.1.1.1:8091/api/receiver", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println(err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := newClient.Do(req)
	if err != nil {
		log.Println(err.Error())
		return
	}
	resp.Body.Close()
}
