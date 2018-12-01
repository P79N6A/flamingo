package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	Enabled  = 1
	Disabled = 2
)

const localFile string = "users.dat"

type User struct {
	Id         int64
	Name       string
	Ak         string
	Sk         string
	Creator    string
	Owner      string
	Status     int32
	IpList     map[string]bool
	CreateTime int
	ModifyTime int
	Privilege  string
}

type AuthReq struct {
	Ak      string
	Stamp   int
	Expire  int
	SignRes string
}

func LoadUsers() (map[string]*User, error) {
	users, err := LoadUsersFromDB()
	if err != nil {
		users, err = LoadUsersFromFile()
		if err != nil {
			return nil, err
		}
	}
	return users, nil
}

func LoadUsersFromDB() (map[string]*User, error) {
	dusers, err := GetUsersFromDB()
	if err != nil {
		return nil, err
	}
	// 保存最新的数据到本地文件
	SaveUsersToFile(dusers)
	return BuildUserMap(dusers), nil
}

func LoadUsersFromFile() (map[string]*User, error) {
	dusers, err := GetUsersFromFile()
	if err != nil {
		return nil, err
	}
	return BuildUserMap(dusers), nil
}

func BuildUserMap(dusers []PushUser) map[string]*User {
	users := make(map[string]*User)
	for _, duser := range dusers {
		ips := strings.Split(duser.IpList, ",")
		ipList := make(map[string]bool)
		for _, ip := range ips {
			if ip != "" {
				ipList[ip] = true
			}
		}
		users[duser.Ak] = &User{
			Id:        duser.Id,
			Name:      duser.Name,
			Ak:        duser.Ak,
			Sk:        duser.Sk,
			Status:    duser.Status,
			IpList:    ipList,
			Privilege: duser.Privilege,
		}
	}
	return users
}

func SaveUsersToFile(users []PushUser) error {
	tmpFile := localFile + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	encoder := gob.NewEncoder(file)
	encoder.Encode(users)
	file.Close()
	os.Remove(localFile)
	os.Rename(tmpFile, localFile)
	return nil
}

func GetUsersFromFile() ([]PushUser, error) {
	file, err := os.Open(localFile)
	if err != nil {
		return nil, err
	}
	decoder := gob.NewDecoder(file)
	var users []PushUser
	err = decoder.Decode(&users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func parseAuth(authStr string) (*AuthReq, error) {
	// 'auth-v1/ak/stamp/expire/signRes'
	tokens := strings.Split(authStr, "/")
	if len(tokens) != 5 || tokens[0] != "auth-v1" {
		return nil, fmt.Errorf("invalid auth header")
	}

	// check ak
	ak := tokens[1]
	if len(ak) != 32 {
		return nil, fmt.Errorf("auth ak invalid")
	}

	// check stamp
	stamp, err := strconv.Atoi(tokens[2])
	if err != nil {
		return nil, fmt.Errorf("auth stamp conv invalid")
	}

	// check expire
	expire, err := strconv.Atoi(tokens[3])
	if err != nil {
		return nil, fmt.Errorf("auth expire invalid")
	}

	// check signRes
	signRes := tokens[4]
	if len(signRes) != 64 {
		return nil, fmt.Errorf("auth signRes len invalid")
	}

	// check expire
	now := int(time.Now().Unix())
	if now < stamp-300 || now > stamp+expire+300 {
		return nil, fmt.Errorf("auth time invalid")
	}

	return &AuthReq{
		Ak:      ak,
		Stamp:   stamp,
		Expire:  expire,
		SignRes: signRes,
	}, nil
}

func GetUser(authStr string, body []byte, Users map[string]*User) (*User, error) {
	auth, err := parseAuth(authStr)
	if err != nil {
		return nil, err
	}
	user, ok := Users[auth.Ak]
	if !ok {
		return nil, fmt.Errorf("no such user")
	}

	signRes := sign(user, auth, body)
	if auth.SignRes != signRes {
		return nil, fmt.Errorf("auth failed")
	}
	if user.Status == Disabled {
		return nil, fmt.Errorf("user blocked")
	}
	return user, nil
}

func GetUserWithSrcAddrCheck(authStr string, body []byte, psm string, srcAddr string, Users map[string]*User) (*User, error) {
	user, err := GetUser(authStr, body, Users)
	if err != nil {
		return nil, err
	}
	//先判断下psm
	if psm != "" {
		_, psm_exist := user.IpList[psm]
		if psm_exist {
			return user, nil
		}
	}

	_, wildcard := user.IpList["*"] //如果设置了通配,则不做ip检查.  TODO 这里可以用正则做的更正规一点
	if wildcard {
		return user, nil
	}

	srcAddr = strings.Split(srcAddr, ":")[0]
	_, ok := user.IpList[srcAddr]
	if !ok {
		return nil, fmt.Errorf("src addr invalid")
	}
	return user, nil
}

func GetUserPrivilegeStr(authStr string, body []byte, Users map[string]*User) (string, error) {
	user, err := GetUser(authStr, body, Users)
	if err != nil {
		return "", err
	}
	return user.Privilege, nil
}

func GetUserPrivilegeStrWithSrcAddrCheck(authStr string, body []byte, srcAddr string, Users map[string]*User) (string, error) {
	user, err := GetUserWithSrcAddrCheck(authStr, body, "", srcAddr, Users)
	if err != nil {
		return "", err
	}
	return user.Privilege, nil
}

func HMAC(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return []byte(fmt.Sprintf("%x", mac.Sum(nil)))
}

func sign(user *User, auth *AuthReq, body []byte) string {
	signKeyInfo := fmt.Sprintf("auth-v1/%s/%d/%d", user.Ak, auth.Stamp, auth.Expire)
	signKey := HMAC([]byte(signKeyInfo), []byte(user.Sk))
	signResult := HMAC(body, signKey)
	return string(signResult)
}

func SimpleSign(ak string, sk string, body []byte) string {
	// 'auth-v1/ak/stamp/expire/signRes'
	signKeyInfo := fmt.Sprintf("auth-v1/%s/%d/%d", ak, time.Now().Unix(), 1800)
	signKey := HMAC([]byte(signKeyInfo), []byte(sk))
	signResult := HMAC(body, signKey)
	return fmt.Sprintf("%v/%v", signKeyInfo, string(signResult))
}
