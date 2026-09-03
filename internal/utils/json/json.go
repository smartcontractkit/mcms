package json

import "maps"

import "encoding/json"

func Merge(json1, json2 []byte) ([]byte, error) {
	var map1, map2 map[string]any

	// Unmarshal both JSON objects into maps
	if err := json.Unmarshal(json1, &map1); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(json2, &map2); err != nil {
		return nil, err
	}

	// Merge map2 into map1
	maps.Copy(map1, map2)

	// Marshal the merged result back into JSON
	return json.Marshal(map1)
}
