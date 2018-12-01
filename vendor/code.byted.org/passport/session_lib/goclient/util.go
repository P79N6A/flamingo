package goclient

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
)

func GetLocalIP() string {
	addressList, _ := net.InterfaceAddrs()
	for _, address := range addressList {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}

		}
	}
	return "127.4.0.4"
}

func EncryptUid(sessionKey string, uid int64) (string, error) {
	// 合法的session key长度应为32位
	if len(sessionKey) != 32 {
		return "", errors.New("Invalid session key")
	}

	// 取session key前16位作为key
	key := make([]byte, 16)
	copy(key, sessionKey)

	// 取uid文本
	uidStr := strconv.FormatInt(uid, 10)
	text := make([]byte, (len(uidStr)+15)&^15) // 长度补齐到16的整数倍
	copy(text, uidStr)

	// 加密
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, key)
	result := make([]byte, 16)
	mode.CryptBlocks(result, text)

	return hex.EncodeToString(result), nil
}

func DecryptUid(sessionKey string, uidCipherHex string) (uid int64, err error) {
	defer func() {
		if err != nil {
			tagKV := map[string]string{"from": "goclient"}
			EmitCounter(METRICS_SESSION_DECRYPT_ERROR, 1, tagKV)
		}
	}()
	// session key校验
	if len(sessionKey) != 32 {
		return 0, errors.New("Invalid session key")
	}

	// 构造key
	key := make([]byte, 16)
	copy(key, sessionKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// 待解密文本
	cipherText, _ := hex.DecodeString(uidCipherHex)
	if len(cipherText)%aes.BlockSize != 0 {
		return 0, errors.New("Cipher length invalid")
	}

	// 解密，session key作初始向量
	mode := cipher.NewCBCDecrypter(block, key)
	mode.CryptBlocks(cipherText, cipherText)

	// 去掉padding
	uidStr := string(bytes.TrimRight(cipherText, "\x00"))
	result, _ := strconv.ParseInt(uidStr, 10, 0)
	return result, nil
}
