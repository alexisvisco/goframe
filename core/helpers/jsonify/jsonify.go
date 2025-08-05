package jsonify

import "encoding/json"

func JSON(data any, indent ...bool) string {
	indented := true
	if len(indent) > 0 {
		indented = indent[0]
	}

	if indented {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "{}"
		}
		return string(jsonData)
	} else {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return "{}"
		}
		return string(jsonData)
	}
}
