package libs

import "fmt"

type Map map[string]interface{}
type Arr []interface{}

func (r Map) GetString(key string) string {
	return fmt.Sprintf("%v", r[key])
}

func (r Map) GetMap(key string) Map {
	return Map(r[key].(map[string]interface{}))
}

func (r Map) GetArr(key string) Arr {
	return Arr(r[key].([]interface{}))
}

func (r Arr) ToArrMap() []Map {
	arrMap := make([]Map, len(r))
	for k, v := range r {
		arrMap[k] = Map(v.(map[string]interface{}))
	}
	return arrMap
}

//func (r Map) GetArr(key string) []Map {
//	return []Map((r[key].([]map[string]interface{})))
//}
