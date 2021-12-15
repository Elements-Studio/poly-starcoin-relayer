package config

import (
	"encoding/json"
	"os"
	"strings"
)

func replaceJsonEnvs(j []byte) []byte {
	m, err := unmarshalToMap(j)
	if err != nil {
		return j
	}
	replaceEnvs(m)
	r, err := json.Marshal(m)
	if err != nil {
		return j
	}
	return r
}

func replaceEnvs(obj map[string]interface{}) {
	for k, v := range obj {
		switch v.(type) {
		case string:
			ev, b := lookupEnv(v.(string))
			if b {
				obj[k] = ev
			}
		case map[string]interface{}:
			replaceEnvs(v.(map[string]interface{}))
		case []interface{}:
			for _, x := range v.([]interface{}) {
				switch x.(type) {
				case map[string]interface{}:
					replaceEnvs(x.(map[string]interface{}))
				}
			}
		}
	}
}

func lookupEnv(v string) (string, bool) {
	if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
		k := v[2 : len(v)-1]
		return os.LookupEnv(k)
	}
	return "", false
}

func unmarshalToMap(bs []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(bs, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
