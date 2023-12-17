package collect

// Temp 临时数据缓存
type Temp struct {
	data map[string]interface{}
}

// Get 返回临时缓存数据
func (t *Temp) Get(key string) interface{} {
	return t.data[key]
}

func (t *Temp) Set(key string, value interface{}) error {
	if t.data == nil {
		t.data = make(map[string]interface{})
	}
	t.data[key] = value
	return nil
}
