package streamer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

func ContainsInputType(arr []InputType, val InputType) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func ContainsString(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

// Redundant code
func (bc *BitrateConfig) SortedVideoResolutionValues() []*VideoResolution {
	values := make([]*VideoResolution, 0, len(bc.VideoResolutions))

	for _, v := range bc.VideoResolutions {
		values = append(values, v)
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i].MaxWidth < values[j].MaxWidth
	})

	return values
}

// Get className and field type
func GetStructName(s interface{}) string {
	// Get the reflect.Value of the interface value
	rv := reflect.ValueOf(s)

	// Check if the reflect.Value is a struct
	if rv.Kind() == reflect.Struct {
		return reflect.TypeOf(s).Name()
	} else {
		return ""
	}
}

func GetStructFieldType(s interface{}, fieldName string) string {
	// Get the reflect.Value of the interface value
	rv := reflect.ValueOf(s)

	// Check if the reflect.Value is a struct
	if rv.Kind() == reflect.Struct {
		// Print the value of the Name and Age fields
		nf := rv.FieldByName(fieldName)

		return nf.Type().Name()
	} else {
		return ""
	}
}

// get value from struct, provided key/field
func GetStructFieldValue(s interface{}, fieldName string) string {
	return fmt.Sprintf("%v", reflect.ValueOf(s).FieldByName(fieldName).Interface())
}

func StructFieldHasValue(s interface{}, fieldName string) bool {
	fv := GetStructFieldValue(s, fieldName)

	return len(fv) > 0
}

// redundant code
type HexString string

func (h HexString) Name() string {
	return "hexadecimal string"
}

func (h HexString) Encode() string {
	return hex.EncodeToString([]byte(h))
}

func (h HexString) Decode() ([]byte, error) {
	if err := h.Validate(); err != nil {
		return nil, err
	}
	return hex.DecodeString(string(h))
}

func (h HexString) Validate() error {
	if len(h)%2 != 0 {
		return errors.New("hex string must have even number of characters")
	}

	if _, err := hex.DecodeString(string(h)); err != nil {
		return errors.New("hex string contains invalid characters")
	}

	return nil
}

func MergeMaps(dst map[string]interface{}, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

func RootDir() (string, error) {
	rootDir, err := os.Getwd() // get the current working directory
	if err != nil {
		return "", err
	}

	for {
		// check if a "go.mod" file exists in the current directory
		if _, err := os.Stat(filepath.Join(rootDir, "go.mod")); err == nil {
			break
		}

		// move up one directory
		rootDir = filepath.Dir(rootDir)

		// check if we've reached the root directory
		if rootDir == filepath.Dir(rootDir) {
			return "", fmt.Errorf("could not find root directory")
		}
	}

	return rootDir, nil
}
