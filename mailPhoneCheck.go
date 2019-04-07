package common

import (
	"crypto/rand"
	"errors"
	"regexp"
	"sync"
	"time"
)

//生成token
func GenToken(size int) string {
	var strstr = []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	data := make([]byte, size)
	out := make([]byte, size)
	buffer := len(strstr)
	_, err := rand.Read(data)
	if err != nil {
		panic(err)
	}
	for id, key := range data {
		x := byte(int(key) % buffer)
		out[id] = strstr[x]
	}
	return string(out)
}

//程序等待
func Trigger(sec int) {
	if sec <= 0 {
		sec = 1
	}
	timer := time.NewTicker(time.Duration(sec) * time.Second)
	select {
	case <-timer.C:

	}
}

//生成uid
var (
	lastMills    = uint64(0)
	lastSequence = uint64(0)
	ulock        sync.Mutex
)

func GenUid() uint64 {
	nowMilli := uint64(int(time.Now().UnixNano() / time.Millisecond.Nanoseconds()))
	newMills := nowMilli<<SEQUENCE_CARRY + WORK_ID_CARRY
	newSequence, err := getSequence(nowMilli)
	if err != nil {
		panic(err)
	}
	uid := newMills + newSequence
	return uid
}
func getSequence(nowMills uint64) (uint64, error) {
	if lastMills == nowMills {
		defer ulock.Unlock()
		ulock.Lock()
		lastSequence++
		if lastSequence > (1<<SEQUENCE_CARRY - 1) {
			return 0, errors.New("序列生成超出范围")
		}
	} else {
		lastSequence = 1
		lastMills = nowMills
	}
	return lastSequence, nil
}

func GenFilePath() string {
	var strstr = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_-")
	data := make([]byte, 16)
	out := make([]byte, 16)
	buffer := len(strstr)
	_, err := rand.Read(data)
	if err != nil {
		panic(err)
	}
	for id, key := range data {
		x := byte(int(key) % buffer)
		out[id] = strstr[x]
	}
	return string(out)
}

//手机号格式验证
func PhoneNumVerify(phone string) bool {
	reg := `^1([34896][0-9]|4[57]|5[^4]|7[0135678])\d{8}$`
	rgx := regexp.MustCompile(reg)
	return rgx.MatchString(phone)
}

//email格式验证
func EmailVerify(mail string) bool {
	match, _ := regexp.MatchString(`^([a-zA-Z0-9_\.-])+@([a-zA-Z0-9_-])+(\.[a-zA-Z0-9_-]+)+$`, mail)
	return match
}

//根据短信内容长度计算真实短信数量
func SmsBillCount(msg string) int {
	msg = msg + "12345"
	if len(msg) == len([]rune(msg)) { //msg是纯英文
		if len(msg) <= 140 {
			return 1
		} else {
			x := len(msg) / 134
			y := len(msg) % 134
			if y > 0 {
				y = 1
			}
			return x + y
		}
	} else { //msg不是纯英文
		if len([]rune(msg)) <= 70 {
			return 1
		} else {
			x := len([]rune(msg)) / 67
			y := len([]rune(msg)) % 67
			if y > 0 {
				y = 1
			}
			return x + y
		}
	}
}
