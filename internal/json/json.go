package json

import (
	"encoding/json"
	"fmt"
)

func UnmarshalTwice(v string, target interface{}) error {
	var intermediate string
	if err := UnmarshalString(v, &intermediate); err != nil {
		return err
	}

	if err := UnmarshalString(intermediate, target); err != nil {
		return err
	}

	return nil
}

func UnmarshalString(v string, target interface{}) error {
	if err := json.Unmarshal([]byte(v), &target); err != nil {
		return fmt.Errorf("unmarshal string: %w", err)
	}
	return nil
}

func MustMarshal(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("must marshal: %s", err))
	}

	return string(data)
}
