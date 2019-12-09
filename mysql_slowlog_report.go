package main

import (
	"bytes"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"html/template"
	"log"
	"mysql_slowlog_report/util"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 定义命令行接收的参数
var (
	limitRecords = kingpin.Flag(
		"limit",
		"Limit the number of display records.",
	).Default("500").Int()
	ptBin = kingpin.Flag(
		"ptbin.path",
		"Where is pt-query-digest.",
	).Default("/usr/bin/pt-query-digest").String()
	slowlogPath = kingpin.Flag(
		"slowlog.path",
		"Where is mysql slowlog.",
	).Default("/tmp/slowquery.log").String()
	tag = kingpin.Flag(
		"tag",
		"Business tag.",
		).Default("AWS Master").String()
	emailRecivers = kingpin.Flag("email.recivers", "Send email to users.").Default("").String()
	emailServerHost = kingpin.Flag("email.serverHost", "Email server host.").Default("").String()
	emailServerPort = kingpin.Flag("email.serverPort", "Email server port.").Default("0").Int()
	fromEmail = kingpin.Flag("email.from", "Email from user.").Default("").String()
	fromPassword = kingpin.Flag("email.password", "Email from user's password.").Default("").String()
	excludeUsers = kingpin.Flag("exclude.users", "Exclude mysql users comma separated.").Default("").String()
	analyzeDay = kingpin.Flag("analyze.day","-1 means yestarday, -2 means the day before yestarday.").Default("-1").Int()
	)

// 从JSON解析出数据后，重新组装的Slowlog结构体，可根据实际情况增删
type Slowlog struct {
	Ts_min string // 第一次出现的时间
	Ts_max string // 最后一次出现的时间
	Query_count int // 出现次数
	Query string // 语句示例
	Pct_95 float64 // 平均耗时(秒)
	User string // 用户
	Host string // 主机
	Db string // 数据库
}

type SlowlogSlice []Slowlog

func (a SlowlogSlice) Len() int {
	return len(a)
}

func (a SlowlogSlice) Swap(i,j int) {
	a[i],a[j] = a[j],a[i]
}

func (a SlowlogSlice) Less(i,j int) bool {
	return a[j].Query_count < a[i].Query_count
}

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Version("0.2")
	kingpin.Parse()

	// 初始化发邮件相关参数, 覆盖默认参数
	u := util.InitNewUser()
	if u.ServerHost == "" {
		u.ServerHost = *emailServerHost
	}
	if u.ServerPort == 0 {
		u.ServerPort = *emailServerPort
	}
	if u.FromEmail == "" {
		u.FromEmail = *fromEmail
	}
	if u.FromPassword == "" {
		u.FromPassword = *fromPassword
	}
	if u.Toers == "" {
		u.Toers = *emailRecivers
	}

	util.InitEmail(u)

	// 判断pt工具是否存在
	if !util.FileOrDirIfExists(*ptBin) {
		log.Panicf("can not find pt-query-digest on %s!", *ptBin)
	}

	// 判断指定的slowlog是否存在
	if !util.FileOrDirIfExists(*slowlogPath) {
		log.Panicf("can not find slowlog on %s!", *slowlogPath)
	}

	currentDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	startTime := currentDate.AddDate(0, 0, *analyzeDay).Format("2006-01-02 15:04:05")
	endTime := currentDate.AddDate(0, 0, 0).Add(-time.Second).Format("2006-01-02 15:04:05")

	// 拼凑分析命令 --limit default 95%:20
	pt_cmd := fmt.Sprintf("%s %s --output json --since \"%s\" --until \"%s\" --limit 10000 > mysql_slowlog.json && sed -i /^$/d mysql_slowlog.json && sed -i /^#/d mysql_slowlog.json", *ptBin, *slowlogPath, startTime, endTime)
	_, err := util.Exec_shell_cmd(pt_cmd)
	if err != nil {
		log.Panicf("execute %s have errors->%s", pt_cmd, err)
	}

	fileInfo, err := os.Stat("mysql_slowlog.json")
	if err != nil {
		log.Panicf("get mysql_slowlog.json have errors->%s", err)
	}

	rend_result := new(bytes.Buffer)
	var total_records int
	// 如果slowlog.json文件大小为0，即在此时间段没有慢查询, 则直接跳过后面解析步骤
	if fileInfo.Size() == 0 {
		no_slowquery_tpl := template.New("no_slowquery.html")
		// 直接解析模板文件
		// no_slowquery_tpl, err = no_slowquery_tpl.ParseFiles("templates/no_slowquery.html")
		no_slowquery_tpl,err = no_slowquery_tpl.Parse(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
	<style type="text/css">
		p {
			font-size:5em;
			margin:5px;
			padding:20px;
			display: inline-block;
		}

		.p1 {
			background:black;
			text-align:left;
			text-shadow: 0 -5px 4px #FF3,2px -10px 6px #fd3,-2px -15px 11px #f80,2px -25px 18px #f20;
			color:red;
		}
</style>
</head>
<body>
<div align="center">
<p class="p1">恭喜，没有慢查询，感谢大家的共同努力 \(^o^)/ </p>
</div>
<footer >
	<p style="text-align: center;font-size: 14px;width: 100%;position: absolute;"><a href="https://github.com/wangtuo1224/mysql_slowlog_report">GitHub</a> <a href="https://aikbuk.com">个人技术博客</a></p>
</footer>
</body>
</html>`)
		if err != nil {
			log.Panicf("parse html file have errors->%s", err)
		}

		if err := no_slowquery_tpl.Execute(rend_result, nil); err !=nil {
			log.Panicf("render to html have errors->%s", err)
		}
	} else {
		data, err := util.Parse_json("mysql_slowlog.json")
		if err != nil {
			log.Panicf("parse mysql_slowlog.json have errors->%s",err)
		}
		// 求出过滤用户后的记录总数，从而给slince分配合适的len->这里应该还可以优化，因为多循环了一遍
		for _, v := range data.Classes {
			if strings.Index(*excludeUsers, v.Metrics.Users.Value) != -1 {
				continue
			}
			total_records++
		}
		slowlogs := make([]Slowlog,total_records)
		// 防止超出索引范围
		var incr int
		// 将解析后的json数据重新组装
		for _, v := range data.Classes {
			// 过滤指定用户
			if strings.Index(*excludeUsers, v.Metrics.Users.Value) != -1 {
				continue
			}
			slowlogs[incr].Ts_min = v.Ts_min
			slowlogs[incr].Ts_max = v.Ts_max
			slowlogs[incr].Query_count = v.Query_count
			slowlogs[incr].Query = v.Examples.Query
			// 从JSON解析出来的Pct_95为字符串，将字符串类型转换为float64类型
			float_Pct_95,err := strconv.ParseFloat(v.Metrics.Query_time.Pct_95,10)
			if err != nil {
				log.Panicf("parse pct_95 from str to float have errors->%s", err)
			}
			slowlogs[incr].Pct_95 = float_Pct_95
			slowlogs[incr].User = v.Metrics.Users.Value
			slowlogs[incr].Host = v.Metrics.Hosts.Value
			slowlogs[incr].Db = v.Metrics.Dbs.Value
			incr++
		}

		// 过滤用户后没有慢查询
		if len(slowlogs) == 0 {
			no_slowquery_tpl := template.New("no_slowquery.html")
			no_slowquery_tpl,err = no_slowquery_tpl.Parse(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
	<style type="text/css">
		p {
			font-size:5em;
			margin:5px;
			padding:20px;
			display: inline-block;
		}

		.p1 {
			background:black;
			text-align:left;
			text-shadow: 0 -5px 4px #FF3,2px -10px 6px #fd3,-2px -15px 11px #f80,2px -25px 18px #f20;
			color:red;
		}
</style>
</head>
<body>
<div align="center">
<p class="p1">恭喜，没有慢查询，感谢大家的共同努力 \(^o^)/ </p>
</div>
<footer >
	<p style="text-align: center;font-size: 14px;width: 100%;position: absolute;"><a href="https://github.com/wangtuo1224/mysql_slowlog_report">GitHub</a> <a href="https://aikbuk.com">个人技术博客</a></p>
</footer>
</body>
</html>`)
			if err != nil {
				log.Panicf("parse html file have errors->%s", err)
			}

			if err := no_slowquery_tpl.Execute(rend_result, nil); err !=nil {
				log.Panicf("render to html have errors->%s", err)
			}
		} else {
			// 按照出现次数降序
			sort.Sort(SlowlogSlice(slowlogs))
			// 按照出现次数升序
			//sort.Sort(sort.Reverse(SlowlogSlice(slowlogs)))

			// 这里的名字必须和ParseFiles的参数一致，否则template: "XXX" is an incomplete or empty template
			slowquery_tpl := template.New("mysql_slowquery.html")
			slowquery_tpl = slowquery_tpl.Funcs(template.FuncMap{"format_date": util.Format_date,"format_id": util.Format_id})
			// 直接解析模板文件
			// slowquery_tpl, err = slowquery_tpl.ParseFiles("templates/mysql_slowquery.html")
			slowquery_tpl, err = slowquery_tpl.Parse(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
	<style type="text/css">
		#hor-minimalist-b
		{
			font-family: "Lucida Sans Unicode", "Lucida Grande", Sans-Serif;
			font-size: 14px;
			background: #fff;
			margin: 10px;
			width: auto;
			border-collapse: collapse;
			text-align: center;
		}
		#hor-minimalist-b th
		{
			font-size: 14px;
			font-weight: normal;
			color: #039;
			padding: 10px 8px;
			border-bottom: 2px solid #6678b1;
		}
		#hor-minimalist-b td
		{
			border-bottom: 1px solid #ccc;
			color: #669;
			padding: 6px 8px;
		}
		#hor-minimalist-b tbody tr:hover td
		{
			color: #009;
		}
	</style>
</head>
<body>
<table id="hor-minimalist-b" style="table-layout:fixed;word-break:break-all;">
	<thead>
	<tr>
		<th width="3%">序号</th>
		<th width="8.5%">数据库</th>
		<th width="10%">查询用户</th>
		<th width="49.5%">语句示例</th>
		<th width="9%">第一次出现的时间</th>
		<th width="9%">最后一次出现的时间</th>
		<th width="4.5%">出现次数</th>
		<th width="5.5%">平均耗时(秒)</th>
	</tr>
	</thead>
	<tbody>
	{{ range $i,$v := . }}
		<tr>
			<td title="序号" width="3%">{{ $i | format_id }}</td>
			<td title="数据库" width="8.5%">{{ .Db }}</td>
			<td title="查询用户" width="10%">{{ .User }}@{{ .Host }}</td>
			<td title="语句示例" style="text-align: left;width: 49.5%">{{ .Query }}</td>
			<td title="第一次出现的时间" width="9.5%">{{ .Ts_min | format_date }}</td>
			<td title="最后一次出现的时间" width="9.5%">{{ .Ts_max | format_date }}</td>
			<td title="出现次数" width="4.5%">{{ .Query_count }}</td>
			<td title="平均耗时(秒)" width="5.5%">{{ .Pct_95 | printf "%.1f"}}</td>
		</tr>
	{{ end }}
	</tbody>
</table>
<footer>
	<p style="text-align: center"><a href="https://github.com/wangtuo1224/mysql_slowlog_report">GitHub</a> <a href="https://aikbuk.com">个人技术博客</a></p>
</footer>
</body>
</html>
`)
			if err != nil {
				log.Panicf("parse html file have errors->%s", err)
			}

			// 如果配置的limit小于记录总数, 则取0-limitRecords，否则取0-记录总数
			if *limitRecords <= total_records {
				if err := slowquery_tpl.Execute(rend_result, slowlogs[0:*limitRecords]); err != nil {
					log.Panicf("render to html have errors->%s", err)
				}
			} else {
				if err := slowquery_tpl.Execute(rend_result, slowlogs[0:total_records]); err !=nil {
					log.Panicf("render to html have errors->%s", err)
				}
			}
		}
	}

	if err := util.SendEmail("请及时处理("+*tag+") MySQL慢查询"+"("+startTime+"-"+ endTime+")", rend_result.String(),""); err != nil {
		log.Panicf("sendmail have errors->%s", err)
	}
}