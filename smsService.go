package service

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2/bson"

	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"milkman/common"
	"milkman/models"
	"gopkg.in/mgo.v2"
)

//将一个请求按batchsize拆成多个task
func SmsTaskAssign(sreq models.SmsRequest) []models.SmsTask {
	var result []models.SmsTask
	bs, _ := beego.AppConfig.Int("smsbatchsize")
	d := sreq.SmsData

	for i := 0; i < len(d.Phone); {
		var phoneList []string
		if i+bs > len(d.Phone) {
			phoneList = d.Phone[i:len(d.Phone)]
		} else {
			phoneList = d.Phone[i: i+bs]
		}
		i += bs

		tasksms := models.SmsTask{}
		tasksms.Id = common.GenUid()
		tasksms.Token = sreq.Token
		tasksms.Sender = sreq.User
		tasksms.Data = models.SmsData{
			Phone: phoneList,
			Msg:   d.Msg,
		}
		tasksms.Priority = sreq.Priority
		tasksms.Status = models.SMS_TASK_CREATED
		tasksms.CreatedAt = time.Now()
		tasksms.BillNum = common.SmsBillCount(d.Msg) * len(phoneList)
		result = append(result, tasksms)
	}
	return result
}

//验证smsrequest是否符合ACL可被执行
func SmsAclCheck(sreq models.SmsRequest) error {
	mgoclient := common.SmsMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.SmsMgoDB)
	year, month, day := time.Now().Date()
	todayStr := fmt.Sprintf("%d%02d%02d", year, month, day)
	cTokenAcl := m.C("tokenacl")
	cTokenInfo := m.C("tokeninfo")
	// token有效Check
	n, err := cTokenAcl.Find(bson.M{"token": sreq.Token,}).Count()
	if err != nil {
		return fmt.Errorf("Mongo Err:获取Token信息异常")
	}
	if n < 1 {
		return fmt.Errorf("Token Invalid")
	}
	var tokenAclResult models.TokenAcl
	err = cTokenAcl.Find(bson.M{"token": sreq.Token}).One(&tokenAclResult)
	if err != nil {
		return fmt.Errorf("Mongo Err:获取TokenAcl信息异常")
	}

	var tokenInfoResult models.TokenInfo
	err = cTokenInfo.Find(bson.M{"key": sreq.Token}).One(&tokenInfoResult)
	if err != nil {
		return fmt.Errorf("Mongo Err:获取TokenInfo信息异常")
	}
	// 是否在锁定状态Check
	if tokenInfoResult.OnLock {
		return fmt.Errorf("Token 锁定中")
	}
	// 是否有短信发送权限Check
	//if tokenInfoResult.SmsRight != 1 {
	//	return fmt.Errorf("Token 无权发送短信")
	//}
	// 每天最大发送量Check
	var max int // 配置文件中的配置为default配置，两者取最低
	if tokenAclResult.DailySMSLimit < common.SmsMaxDailySendNum {
		max = tokenAclResult.DailySMSLimit
	} else {
		max = common.SmsMaxDailySendNum
	}
	reqBillCount := common.SmsBillCount(sreq.SmsData.Msg) * len(sreq.SmsData.Phone)
	var currentCount uint
	cBill := m.C("sms_bill")
	n, err = cBill.Find(bson.M{"token": sreq.Token, "date": todayStr}).Count()
	if err != nil {
		return fmt.Errorf("Mongo Err:获取bill信息异常")
	}
	if n < 1 {
		if reqBillCount > max {
			return fmt.Errorf("Sms Acl:超出每日最大发送量")
		}
		return nil
	}
	result := models.SmsBill{}
	err = cBill.Find(bson.M{"token": sreq.Token, "date": todayStr}).One(&result)
	if err != nil {
		return fmt.Errorf("Mongo Err:获取bill信息异常")
	}
	currentCount = result.Count
	if currentCount+uint(reqBillCount) > uint(max) {
		return fmt.Errorf("Sms Acl:超出每日最大发送量")
	}

	return nil
}

//task压入redis队列
func SmsPushTaskToQueue(task models.SmsTask) error {
	c := common.SmsRedisClient.Get()
	defer c.Close()
	taskJson, _ := json.Marshal(task)
	_, err := c.Do("LPUSH", common.SmsPriorityQueue[task.Priority], taskJson)
	return err
}

//task计入MongoDB,有则更新,无则新建 & 短信记账
func SmsUpdateTaskToDB(task models.SmsTask) error {
	mgoclient := common.SmsMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.SmsMgoDB)
	year, month, day := time.Now().Date()
	todayStr := fmt.Sprintf("%d%02d%02d", year, month, day)
	cTask := m.C("sms_tasks")
	cBill := m.C("sms_bill")
	selector := bson.M{"id": task.Id}

	//保留更新前状态
	var preUpdate models.SmsTask
	cTask.Find(selector).One(&preUpdate) // 如果find没有结果，preUpdate是零值

	//更新消息记录
	info, err := cTask.Upsert(selector, task)
	infoJson, _ := json.Marshal(info)
	seelog.Debug(string(infoJson))
	if err != nil {
		return err
	}

	//记账 如果task信息更新不涉及发送状态变化，跳过记账
	if task.Status == models.SMS_TASK_DELIVERING && task.Status != preUpdate.Status {
		selector = bson.M{"token": task.Token, "date": todayStr}
		n, err := cBill.Find(selector).Count()
		if err != nil {
			return fmt.Errorf("Mongo Err:获取bill信息异常")
		}
		if n < 1 {
			err = cBill.Insert(
				models.SmsBill{
					Token: task.Token,
					Date:  todayStr,
					Count: uint(task.BillNum),
				},
			)
			if err != nil {
				seelog.Error("Mongo Err: ", err)
				return fmt.Errorf("Mongo Err:更新bill信息异常")
			}
		} else {
			var currentBill models.SmsBill
			err = cBill.Find(selector).One(&currentBill)
			if err != nil {
				return fmt.Errorf("Mongo Err:获取bill信息异常")
			}
			err = cBill.Update(
				selector,
				models.SmsBill{
					Token: task.Token,
					Date:  todayStr,
					Count: uint(task.BillNum) + currentBill.Count,
				},
			)
			if err != nil {
				seelog.Error("Mongo Err: ", err)
				return fmt.Errorf("Mongo Err:更新bill信息异常")
			}
		}
	} else if task.Status == models.SMS_TASK_FAIL && task.Retry == common.MaxRetry { //当发送失败时 且重试次数到最大值时 记账减少
		selector = bson.M{"token": task.Token, "date": todayStr}
		currentBill := models.SmsBill{}
		err = cBill.Find(selector).One(&currentBill)
		if err != nil {
			return fmt.Errorf("Mongo Err:获取bill信息异常")
		}
		newCount := currentBill.Count - uint(task.BillNum)
		if newCount < 0 {
			return fmt.Errorf("Mongo Err:记账信息异常为负数")
		}
		err = cBill.Update(selector, models.SmsBill{
			Token: task.Token,
			Date:  todayStr,
			Count: newCount,
		})
		if err != nil {
			seelog.Error(err)
			return err
		}
	}

	return nil
}

//取redis任务队列数据执行
func SmsWork(workid int) {
	seelog.Infof("worker_%v start", workid)
	c := common.SmsRedisClient.Get()
	defer c.Close()
	for {
		cmdArgs := []interface{}{}
		for i := 1; i <= len(common.SmsPriorityQueue); i++ {
			cmdArgs = append(cmdArgs, interface{}(common.SmsPriorityQueue[i]))
		}
		cmdArgs = append(cmdArgs, interface{}(0)) // 增加block时间设置
		values, err := redis.Values(c.Do("BRPOP", cmdArgs...))
		if err != nil {
			seelog.Error("Work_", workid, "Pull Task From Redis Err:", err)
		} else {
			currKey := values[0]
			taskStr := values[1]
			if taskJson, ok := taskStr.([]byte); ok {
				task := models.SmsTask{}
				err := json.Unmarshal(taskJson, &task)
				if err != nil {
					seelog.Error("Worker_", workid, "Redis:Parse Task Err:", err)
				} else {
					seelog.Debugf("Worker_%v Queue:%v-Get Task to Deliver:%+v", workid, string(currKey.([]byte)), task)
					//task.Timestamp = time.Now().Unix()
					//err := SmsUpdateTaskToDB(task)
					//if err != nil {
					//	seelog.Error("Worker_", workid, "Mongo:Update Task Err:", err)
					//	continue
					//}
					ExecSmsTask(task)
				}
			} else {
				seelog.Error("Worker_", workid, "Redis:Res type Err:", string(taskStr.([]byte)))
			}
		}
	}
}

//执行sms任务 任务失败时记录失败记录 重新入redis队列 超过最大重试次数放弃继续执行
func ExecSmsTask(task models.SmsTask) {
	var result map[string]interface{}

	smsHttpUrl := beego.AppConfig.String("smsurl")
	smsId := beego.AppConfig.String("smsid")
	smsKey := beego.AppConfig.String("smskey")
	nowtime := time.Now().Unix()
	token := strconv.Itoa(int(nowtime)) + smsId + smsKey
	h := sha1.New()
	h.Write([]byte(token))
	tokenstring := fmt.Sprintf("%x", h.Sum(nil))
	seelog.Info("tokenstring:", tokenstring)

	//send http request
	phoneListStr := strings.Join(task.Data.Phone, ",")
	data := []map[string]string{
		{
			"msg":   task.Data.Msg,
			"phone": phoneListStr,
		},
	}
	datajson, _ := json.Marshal(data)
	seelog.Debug(string(datajson))

	httpreq := httplib.Post(smsHttpUrl)
	httpreq.Param("key", tokenstring)
	httpreq.Param("id", smsId)
	httpreq.Param("unix_time", strconv.Itoa(int(nowtime)))
	httpreq.Param("data", string(datajson))

	//get http response
	resp, err := httpreq.Response()
	if err != nil {
		seelog.Error("Http Request Err:", err)
		task.Status = models.SMS_TASK_FAIL
		goto end
	}
	defer resp.Body.Close()

	if body, err := ioutil.ReadAll(resp.Body); err != nil {
		seelog.Error("Read Response Err:", err, ":Resp.Body:", resp.Body)
		task.Status = models.SMS_TASK_FAIL
		goto end
	} else {
		seelog.Info("Http Response:", string(body))
		err = json.Unmarshal(body, &result)
		if err != nil {
			seelog.Error("请求结果解析错误")
			task.Status = models.SMS_TASK_FAIL
			goto end
		}

		if resp.StatusCode != 200 {
			seelog.Error("Call Serivce to Send Message Err:", resp.StatusCode)
			task.Status = models.SMS_TASK_FAIL
			goto end
		} else if _, ok := result["error"]; ok {
			task.Status = models.SMS_TASK_FAIL
			goto end
		} else {
			task.Status = models.SMS_TASK_DONE
			task.Result = string(body)
		}
	}

end: //update task status &  push fail task to redis again
	err = SmsUpdateTaskToDB(task)
	if err != nil {
		panic(err)
	}
	if task.Status == models.SMS_TASK_FAIL {
		//查询错误计数
		//var otask models.SmsTask
		//mgoclient := common.SmsMgoSession.Clone()
		//defer mgoclient.Close()
		//m := mgoclient.DB(common.SmsMgoDB)
		//err = m.C("sms_tasks").Find(bson.M{"id": task.Id}).One(&otask)
		//if err != nil {
		//	seelog.Error("查询短信任务异常:", task.Id)
		//	return
		//}
		//未达到最大重试次数重入队列
		//if otask.Retry < common.MaxRetry {

		if task.Retry < common.MaxRetry {
			task.Retry = task.Retry + 1
			err = SmsPushTaskToQueue(task)
			if err != nil {
				seelog.Error("失败任务重入Redis失败:", err)
			}
			seelog.Info("sleep 5s")
			time.Sleep(1)
		}
	}
}

type SmsService struct{}

var SmsSvc SmsService

func (SmsService) DayCount(token string) (int, error) {
	mgoclient := common.SmsMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.SmsMgoDB)
	year, month, day := time.Now().Date()
	todayStr := fmt.Sprintf("%d%02d%02d", year, month, day)

	//每天最大发送量Check
	var smsBill models.SmsBill
	err := m.C("sms_bill").Find(bson.M{"token": token, "date": todayStr}).One(&smsBill)
	if err == mgo.ErrNotFound {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		return int(smsBill.Count), nil
	}
}
