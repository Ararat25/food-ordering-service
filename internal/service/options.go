package service

import "encoding/json"

// normalizeOptions - пустые опции превращает в [], чтобы в базу не писался NULL.
// NULL из JSONB-колонки не сканируется в json.RawMessage и роняет чтение.
func normalizeOptions(o json.RawMessage) json.RawMessage {
	if len(o) == 0 {
		return json.RawMessage("[]")
	}
	return o
}
