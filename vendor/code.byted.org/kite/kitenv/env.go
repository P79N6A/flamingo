/*

1: Parent Process Name: supervise or systemd
2: User: tiger

This package can run only on linux server

*/
package kitenv

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
)

const processNameFormat = "/proc/%d/comm"

type Cond func() bool

// Product return true if current service is running on product enviroment else false
func Product() bool {
	if os.Getenv("KITENV_DEV") != "" {
		return false
	}

	// please see: https://wiki.bytedance.net/pages/viewpage.action?pageId=63229064
	if os.Getenv("IS_PROD_RUNTIME") != "" {
		return true
	}

	u, err := User()
	if err != nil {
		return false
	}
	pn, err := ParentProcName()
	if err != nil {
		return false
	}
	if u == "tiger" && (pn == "supervise" || pn == "systemd") {
		return true
	}
	return false
}

// ParentProcName return the father of current process
func ParentProcName() (string, error) {
	ppid := os.Getppid()
	bs, err := ioutil.ReadFile(fmt.Sprintf(processNameFormat, ppid))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bs)), nil
}

// User return who start this process.
func User() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}
