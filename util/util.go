package util

import (
	"fmt"
	"io/ioutil"
	"mysql_slowlog_report/jsondata"
	"os"
	"os/exec"
	"strings"
)

// 序号从1开始
func Format_id(args ...interface{}) int {
	id, _ := args[0].(int)
	return id+1
}

// 去掉时间字符串里的T字符,例如将2019-07-18T22:01:07变为2019-07-18 22:01:07，免得看着别扭
func Format_date(args ...interface{}) string {
	ok := false
	var s string
	if len(args) == 1 {
		s, ok = args[0].(string)
	}
	if !ok {
		s = fmt.Sprint(args...)
	}

	substrs := strings.Split(s, "T")
	if len(substrs) != 2 {
		return s
	}
	return (substrs[0] + " " + substrs[1])
}

// 解析json文件
func Parse_json(file_path string) (jsondata.SLOWLOG_JSON,error) {
	file_data := Read_data(file_path)
	json := jsondata.SLOWLOG_JSON{}
	err := json.UnmarshalJSON([]byte(file_data))
	if err != nil {
		return jsondata.SLOWLOG_JSON{},err
	}
	return json,nil
}

// 读取待解析的文件内容
func Read_data(file_path string) string {
	file, err := os.Open(file_path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	fd, err := ioutil.ReadAll(file)
	return string(fd)
}

// 判断文件或目录是否存在
func FileOrDirIfExists(binfile string) bool {
	_, err := os.Stat(binfile)
	if err != nil {
		return false
	}
	return true
}

// 执行shell命令
func Exec_shell_cmd(shellcmd string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", shellcmd)
	out,err := cmd.CombinedOutput()
	return string(out), err
}
