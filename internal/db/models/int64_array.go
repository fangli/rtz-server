package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Int64Array []int64

func (a *Int64Array) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}

	switch value := value.(type) {
	case int64:
		*a = Int64Array{value}
		return nil
	case int:
		*a = Int64Array{int64(value)}
		return nil
	case []byte:
		return a.scanString(string(value))
	case string:
		return a.scanString(value)
	default:
		return fmt.Errorf("unsupported Int64Array scan type %T", value)
	}
}

func (a Int64Array) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	values := make([]string, len(a))
	for i, value := range a {
		values[i] = strconv.FormatInt(value, 10)
	}
	return "{" + strings.Join(values, ",") + "}", nil
}

func (a *Int64Array) scanString(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		*a = Int64Array{}
		return nil
	}

	if strings.HasPrefix(value, "[") {
		var values []int64
		if err := json.Unmarshal([]byte(value), &values); err != nil {
			return err
		}
		*a = values
		return nil
	}

	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}")
	}
	if value == "" {
		*a = Int64Array{}
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, `"`))
		if part == "" || strings.EqualFold(part, "NULL") {
			continue
		}
		number, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return err
		}
		values = append(values, number)
	}
	*a = values
	return nil
}
