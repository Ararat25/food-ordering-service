package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"avito-kitchen/internal/domain"
)

type menuOption struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
	Default string   `json:"default,omitempty"`
}

// normalizeOptions - пустые опции превращает в [], чтобы в базу не писался NULL.
func normalizeOptions(o json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(o)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("[]")
	}
	return trimmed
}

func parseMenuOptions(raw json.RawMessage) ([]menuOption, error) {
	raw = normalizeOptions(raw)
	var definitions []menuOption
	if err := json.Unmarshal(raw, &definitions); err != nil {
		return nil, fmt.Errorf("%w: invalid options schema: %v", domain.ErrValidation, err)
	}
	if definitions == nil {
		return nil, fmt.Errorf("%w: options must be a JSON array", domain.ErrValidation)
	}
	seen := make(map[string]struct{}, len(definitions))
	for _, def := range definitions {
		if def.Name == "" || len(def.Options) == 0 {
			return nil, fmt.Errorf("%w: each option needs a name and at least one value", domain.ErrValidation)
		}
		if _, ok := seen[def.Name]; ok {
			return nil, fmt.Errorf("%w: duplicate option name %q", domain.ErrValidation, def.Name)
		}
		seen[def.Name] = struct{}{}
		values := make(map[string]struct{}, len(def.Options))
		for _, value := range def.Options {
			if value == "" {
				return nil, fmt.Errorf("%w: option values must not be empty", domain.ErrValidation)
			}
			if _, ok := values[value]; ok {
				return nil, fmt.Errorf("%w: duplicate option value %q", domain.ErrValidation, value)
			}
			values[value] = struct{}{}
		}
		if def.Default != "" {
			if _, ok := values[def.Default]; !ok {
				return nil, fmt.Errorf("%w: default must be one of the declared option values", domain.ErrValidation)
			}
		}
	}
	return definitions, nil
}

// validateOptionSelection принимает map {optionName: selectedValue}, заполняет объявленные значения по умолчанию
// и возвращает канонический объект JSON. Выбор опций в этом MVP носит только описательный характер; они
// не изменяют стоимость базового меню.
func validateOptionSelection(definitionsRaw, selectedRaw json.RawMessage) (json.RawMessage, error) {
	definitions, err := parseMenuOptions(definitionsRaw)
	if err != nil {
		return nil, err
	}
	selectedRaw = bytes.TrimSpace(selectedRaw)
	selected := map[string]string{}
	if len(selectedRaw) > 0 && !bytes.Equal(selectedRaw, []byte("null")) && !bytes.Equal(selectedRaw, []byte("[]")) {
		if err := json.Unmarshal(selectedRaw, &selected); err != nil || selected == nil {
			return nil, fmt.Errorf("%w: selected options must be a JSON object", domain.ErrValidation)
		}
	}
	allowed := make(map[string]map[string]struct{}, len(definitions))
	for _, def := range definitions {
		values := make(map[string]struct{}, len(def.Options))
		for _, value := range def.Options {
			values[value] = struct{}{}
		}
		allowed[def.Name] = values
		if _, ok := selected[def.Name]; !ok && def.Default != "" {
			selected[def.Name] = def.Default
		}
	}
	for name, value := range selected {
		values, ok := allowed[name]
		if !ok {
			return nil, fmt.Errorf("%w: unknown option %q", domain.ErrValidation, name)
		}
		if _, ok := values[value]; !ok {
			return nil, fmt.Errorf("%w: invalid value for option %q", domain.ErrValidation, name)
		}
	}
	if len(selected) == 0 {
		return json.RawMessage("[]"), nil
	}
	canonical, err := json.Marshal(selected)
	if err != nil {
		return nil, fmt.Errorf("encode selected options: %w", err)
	}
	return canonical, nil
}

// sameOptions сравнивает значения JSON независимо от порядка ключей объекта и пробелов.
func sameOptions(a, b json.RawMessage) bool {
	a, b = normalizeOptions(a), normalizeOptions(b)
	if bytes.Equal(a, b) {
		return true
	}
	var left, right any
	leftDecoder := json.NewDecoder(bytes.NewReader(a))
	leftDecoder.UseNumber()
	rightDecoder := json.NewDecoder(bytes.NewReader(b))
	rightDecoder.UseNumber()
	if leftDecoder.Decode(&left) != nil || rightDecoder.Decode(&right) != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}
