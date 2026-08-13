package journal

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (r *ManagedArtifactRecord) UnmarshalJSON(data []byte) error {
	if err := rejectManagedArtifactNulls(data); err != nil {
		return err
	}
	type recordAlias ManagedArtifactRecord
	var decoded recordAlias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*r = ManagedArtifactRecord(decoded)
	return nil
}

func rejectManagedArtifactNulls(data []byte) error {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	return rejectManagedArtifactNullValue(value, "$")
}

func rejectManagedArtifactNullValue(value any, path string) error {
	switch typed := value.(type) {
	case nil:
		return fmt.Errorf("managed artifact execution record field %s must not be null", path)
	case map[string]any:
		for key, child := range typed {
			if err := rejectManagedArtifactNullValue(child, path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range typed {
			if err := rejectManagedArtifactNullValue(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
