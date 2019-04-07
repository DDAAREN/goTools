package common

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/config"
	"github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	mongo "gopkg.in/mgo.v2"

	"os"
	"runtime"
	"strconv"
	"time"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"milkman/models"
)

func init() {

	ConfigPath = "./conf"
	DownloadPath = "./download"
	println("ConfigPath:", ConfigPath)
	println("DownloadPath:", DownloadPath)

	//seelog
	var SeelogConfigPath string
	SeelogConfigPath = ConfigPath + "/seelog.xml"
	logger, err := seelog.LoggerFromConfigAsFile(SeelogConfigPath)
	if err != nil && (runtime.GOOS == OPS_LINUX) {
		print(os.Stderr, "err parsing config log file", err)
		os.Exit(-1)
	}
	seelog.ReplaceLogger(logger)
	defer seelog.Flush()
	println("init seelog done!")

	//mail acl
	aclConfig, err := config.NewConfig("ini", ConfigPath+"/acl.conf")
	if err != nil {
		panic("init acl.conf err:" + err.Error())
	}
	mailacl, _ := aclConfig.GetSection("mail")
	MaxAttchmentSize, _ = strconv.Atoi(mailacl["maxattchmentsize"])
	MaxInAttchmentSize, _ = strconv.Atoi(mailacl["maxinattchmentsize"])
	MaxAttchmentNum, _ = strconv.Atoi(mailacl["maxattchmentnum"])
	MaxBodySize, _ = strconv.Atoi(mailacl["maxbodysize"])
	MinIntervalTime, _ = strconv.Atoi(mailacl["minintervaltime"])
	MaxDailyMailNum, _ = strconv.Atoi(mailacl["maxdailymailnum"])

	//sms acl
	smsacl, _ := aclConfig.GetSection("sms")
	SmsMaxDailySendNum, _ = strconv.Atoi(smsacl["maxdailysmsnum"])

	//init db
	initMongo()
	initRedis()

	initAuthMap()
	initVar()

	println("init end!")
}

func initMongo() {
	//mongo-mail
	var err error
	mmhost := beego.AppConfig.String("mmhost")
	mmdb := beego.AppConfig.String("mmdb")
	mmuname := beego.AppConfig.String("mmuname")
	mmpwd := beego.AppConfig.String("mmpwd")
	mdialInfo := &mongo.DialInfo{
		Addrs:    []string{mmhost},
		Timeout:  time.Second * 60,
		Database: mmdb,
		Username: mmuname,
		Password: mmpwd,
	}
	MailMgoSession, err = mongo.DialWithInfo(mdialInfo)
	if nil != err {
		fmt.Printf("mongInfo:%+v", mdialInfo)
		panic("connect mail mongodb err:" + err.Error())
	}
	MailMgoDB = mmdb
	println("init mail mongodb success:", mmhost, "_", mmdb)

	//mongo-sms
	mshost := beego.AppConfig.String("mmhost")
	msdb := beego.AppConfig.String("mmdb")
	msuname := beego.AppConfig.String("mmuname")
	mspwd := beego.AppConfig.String("mmpwd")
	sdialInfo := &mongo.DialInfo{
		Addrs:    []string{mshost},
		Timeout:  time.Second * 60,
		Database: msdb,
		Username: msuname,
		Password: mspwd,
	}
	SmsMgoSession, err = mongo.DialWithInfo(sdialInfo)
	if nil != err {
		fmt.Printf("mongInfo:%+v", sdialInfo)
		panic("connect sms mongodb err:" + err.Error())
	}
	SmsMgoDB = msdb
	println("init sms mongodb success:", mshost, "_", msdb)
}

func initRedis() {
	rmhost := beego.AppConfig.String("rmhost")
	rmport := beego.AppConfig.String("rmport")
	rmdb, _ := beego.AppConfig.Int("rmdb")
	rmpwd := beego.AppConfig.String("rmpwd")
	redisAddr := rmhost + ":" + rmport
	MailRedisClient = &redis.Pool{
		MaxIdle:     1,
		MaxActive:   10,
		IdleTimeout: 600 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisAddr)
			if err != nil {
				beego.Error("redis pool", err)
				return nil, err
			}
			c.Do("AUTH", rmpwd)
			c.Do("SELECT", rmdb)
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if err != nil {
				beego.Error("mail redis pool err:", err)
			}
			return err
		},
	}
	println("init mail redis success:", rmhost, "_", rmdb)

	rshost := beego.AppConfig.String("rshost")
	rsport := beego.AppConfig.String("rsport")
	rsdb, _ := beego.AppConfig.Int("rsdb")
	rspwd := beego.AppConfig.String("rspwd")
	sredisAddr := rshost + ":" + rsport
	SmsRedisClient = &redis.Pool{
		MaxIdle:     1,
		MaxActive:   10,
		IdleTimeout: 600 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", sredisAddr)
			if err != nil {
				beego.Error("redis pool", err)
				return nil, err
			}
			c.Do("AUTH", rspwd)
			c.Do("SELECT", rsdb)
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if err != nil {
				beego.Error("sms redis pool err:", err)
			}
			return err
		},
	}
	println("init sms redis success:", rshost, "_", rsdb)
}

func initAuthMap() {
	path := beego.AppConfig.String("authmap")
	file, err := ioutil.ReadFile(path)
	if err != nil {
		beego.Error(err)
		panic("初始化白名单异常")
	}

	var authArr = make([]models.AuthItem, 0)
	err = json.Unmarshal(file, &authArr)
	if err != nil {
		beego.Error(err)
		panic("初始化白名单异常")
	}

	for _, v := range authArr {
		AuthMap[v.Token] = v
	}
	beego.Info("白名单:", AuthMap)
	beego.Info("初始化白名单成功！")
}

func initVar() {
	var err error
	MaxRetry, err = beego.AppConfig.Int("maxRetry")
	if err != nil {
		panic(err)
	}
	beego.Info("最大重试次数:", MaxRetry)
	beego.Info("初始化变量成功！")
}
