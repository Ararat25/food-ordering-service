package service

import "encoding/json"

// normalizeOptions - пустые опции превращает в [], чтобы в базу не писался NULL.
func normalizeOptions(o json.RawMessage) json.RawMessage {
	if len(o) == 0 {
		return json.RawMessage("[]")
	}
	return o
}
