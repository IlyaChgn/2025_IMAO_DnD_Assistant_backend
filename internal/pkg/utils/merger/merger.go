package merger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Merge сливает два JSON в соответствии со следующими правилами:
//
// 1. Существующие поля всегда сохраняются после слияния
//
// 2. Данные после слияния перезаписываются только если они были изменены
//
// Функция возвращает JSON, поля которого отсортированы в лексикографическом порядке
func Merge(dataBuf, patchBuf []byte) ([]byte, error) {
	var data, patch interface{}

	err := unmarshalJSON(dataBuf, &data)
	if err != nil {
		err = fmt.Errorf("something went wrong while unmarshalling data JSON: %v", err)

		return nil, err
	}

	err = unmarshalJSON(patchBuf, &patch)
	if err != nil {
		err = fmt.Errorf("something went wrong while unmarshalling patch JSON: %v", err)

		return nil, err
	}

	merged, err := mergeObjects(data, patch, nil)
	if err != nil {
		err = fmt.Errorf("something went wrong while merging JSON: %v", err)

		return nil, err
	}

	mergedBuf, err := json.Marshal(merged)
	if err != nil {
		err = fmt.Errorf("something went wrong while marshalling merged JSON: %v", err)

		return nil, err
	}

	return mergedBuf, nil
}

func mergeObjects(data, patch interface{}, path []string) (interface{}, error) {
	var err error

	if patchObject, ok := patch.(map[string]interface{}); ok {
		if dataArray, ok := data.([]interface{}); ok {
			ret := make([]interface{}, len(dataArray))

			for i, val := range dataArray {
				ret[i], err = mergeValue(path, patchObject, strconv.Itoa(i), val)
				if err != nil {
					return nil, err
				}
			}

			return ret, nil
		} else if dataObject, ok := data.(map[string]interface{}); ok {
			ret := make(map[string]interface{})
			visited := make([]string, 0)

			for k, v := range dataObject {
				ret[k], err = mergeValue(path, patchObject, k, v)
				if err != nil {
					return nil, err
				}

				visited = append(visited, k)
			}

			for k, v := range patchObject {
				if !slices.Contains(visited, k) {
					ret[k] = v
				}
			}

			return ret, nil
		}
	}

	return data, nil
}

func mergeValue(path []string, patch map[string]interface{}, key string,
	value interface{}) (interface{}, error) {
	patchValue, patchHasValue := patch[key]

	if !patchHasValue {
		return value, nil
	}

	_, patchValueIsObject := patchValue.(map[string]interface{})

	path = append(path, key)
	pathStr := strings.Join(path, ".")

	if _, ok := value.(map[string]interface{}); ok {
		if !patchValueIsObject {
			err := fmt.Errorf("patch value must be object for key \"%v\"", pathStr)

			return value, err
		}

		return mergeObjects(value, patchValue, path)
	}

	if _, ok := value.([]interface{}); ok && patchValueIsObject {
		return mergeObjects(value, patchValue, path)
	}

	return patchValue, nil
}

func unmarshalJSON(buff []byte, data interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(buff))
	decoder.UseNumber()

	return decoder.Decode(data)
}
