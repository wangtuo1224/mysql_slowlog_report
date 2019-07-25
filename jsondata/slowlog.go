package jsondata

//easyjson:json
type SLOWLOG_JSON struct {
	Classes []Class_arr `json:"classes"`
	Global Global_obj `json:"global"`
}

//easyjson:json
type Global_obj struct {
	Query_count int `json:"query_count"`
	Unique_query_count int `json:"unique_query_count"`
}

//easyjson:json
type Class_arr struct {
	Attribute string `json:"attribute"`
	Checksum string `json:"checksum"`
	//Distillate string `json:"distillate"`
	//Fingerprint string `json:"fingerprint"`
	Examples Example_obj `json:"example"`
	Query_count int `json:"query_count"`
	Ts_max string `json:"ts_max"`
	Ts_min string `json:"ts_min"`
	Metrics Metric_obj `json:"metrics"`
}

//easyjson:json
type Example_obj struct {
	Query_time string `json:"Query_time"`
	Query string `json:"query"`
	Ts string `json:"ts"`
}

//easyjson:json
type Metric_obj struct {
	Dbs Db_obj `json:"db"`
	Hosts Host_obj `json:"host"`
	Users User_obj `json:"user"`
	Query_time Query_time_obj `json:"Query_time"`
}
//easyjson:json
type Db_obj struct {
	Value string `json:"value"`
}
//easyjson:json
type Host_obj struct {
	Value string `json:"value"`
}
//easyjson:json
type User_obj struct {
	Value string `json:"value"`
}

//easyjson:json
type Query_time_obj struct {
	Avg string `json:"avg"`
	Max string `json:"max"`
	Min string `json:"min"`
	Pct_95 string `json:"pct_95"`
}
