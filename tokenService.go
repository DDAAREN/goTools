package service

import (
	"milkman/common"
	"milkman/models"

	"github.com/cihub/seelog"
	"gopkg.in/mgo.v2/bson"
	mgo "gopkg.in/mgo.v2"
	"github.com/pkg/errors"

	"time"
)

type TokenService struct {
}

var TokenSvc TokenService

func ProductToken(tokeninfo models.TokenInfo) (string, error) {
	mgoclient := common.MailMgoSession.Clone()
	defer mgoclient.Close()
	tokenCollection := mgoclient.DB(common.MailMgoDB).C("tokeninfo")

	//生成16位随机字符串,数据库中是否含有该key，如果存在重新生成
	token := common.GenToken(16)
	for {
		n, err := tokenCollection.Find(bson.M{"key": token}).Count()
		if err != nil {
			seelog.Error("Find Repeat Token in Mongo Err:", err)
			return "", err
		}
		if n > 0 {
			token = common.GenToken(16)
		} else {
			break
		}
	}
	seelog.Info("token:", token)

	//存入数据库-mongo
	tokeninfo.Key = token
	seelog.Info("tokeninfo:%+v", tokeninfo)

	err := tokenCollection.Insert(tokeninfo)
	if err != nil {
		seelog.Info("Insert Token Info to Mongo Err:", err)
		return "", err
	}
	return token, nil
}

func SetTokenAcl(token string, acl models.TokenAcl) error {
	mgoclient := common.MailMgoSession.Clone()
	defer mgoclient.Close()
	tokenCollection := mgoclient.DB(common.MailMgoDB).C("tokenacl")

	//配置token对应邮件ACL
	acl.Token = token
	err := tokenCollection.Insert(&acl)
	if err != nil {
		seelog.Info("Insert Token ACL to Mongo Err:", err)
		return err
	}
	return nil
}

func (this TokenService) Insert(token models.Token) (string, error) {
	var key string
	timeout := time.After(time.Second * 1)
genToken:
	for {
		select {
		case <-timeout:
			return key, errors.New("生成token超时")
		default:
			key = common.GenToken(16)
			has, _, err := this.Get(key, "")
			if err != nil {
				seelog.Error(err)
				return key, err
			} else if !has {
				token.Key = key
				break genToken
			}
		}
	}

	//初始化mgo连接
	mgoclient := common.MailMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.MailMgoDB)
	err := m.C("token").Insert(token)
	if err != nil {
		seelog.Error(err)
		return key, err
	}
	return key, nil
}

func (TokenService) Get(key string, typ string) (bool, models.Token, error) {
	var token models.Token
	var query = make(bson.M)

	//初始化mgo连接
	mgoclient := common.MailMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.MailMgoDB)

	//校验token
	if key == "" && typ == "" {
		return false, token, errors.New("查询条件为空")
	}
	if key != "" {
		query["key"] = key
	}
	if typ != "" {
		query["type"] = typ
	}

	err := m.C("token").Find(query).One(&token)
	if err == mgo.ErrNotFound {
		return false, token, nil
	} else if err != nil {
		seelog.Error(err)
		return false, token, err
	}

	return true, token, nil
}

func (TokenService) Update(token models.Token) error {
	//初始化mgo连接
	mgoclient := common.MailMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.MailMgoDB)

	//校验token
	if token.Key == "" || token.Type == "" {
		return errors.New("token或类型为空")
	}
	var update = bson.M{}
	if token.DailyLimit != 0 {
		update["dailyLimit"] = token.DailyLimit
	}
	if len(token.AuthIp) != 0 {
		update["authIP"] = token.AuthIp
	}
	if token.Interval != 0 {
		update["interval"] = token.Interval
	}

	err := m.C("token").Update(bson.M{"key": token.Key, "type": token.Type}, bson.M{"$set": update})
	if err == mgo.ErrNotFound {
		err = seelog.Error("token不存在")
		return err
	} else if err != nil {
		seelog.Error(err)
		return err
	}
	return nil
}

func (TokenService) Delete(key string, typ string) error {
	//初始化mgo连接
	mgoclient := common.MailMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.MailMgoDB)

	//校验token
	if key == "" || typ == "" {
		return errors.New("token 或类型为空")
	}
	err := m.C("token").Remove(bson.M{"key": key, "type": typ})
	if err != nil {
		seelog.Error(err)
		return err
	}
	return nil
}
func (TokenService) All() ([]models.TokenInfo, error) {
	//初始化mgo连接
	mgoclient := common.MailMgoSession.Clone()
	defer mgoclient.Close()
	m := mgoclient.DB(common.MailMgoDB)

	//校验token
	var result = make([]models.TokenInfo, 0)
	err := m.C("tokeninfo").Find(bson.M{}).All(&result)
	if err != nil {
		seelog.Error(err)
		return nil, err
	}
	return result, err
}
