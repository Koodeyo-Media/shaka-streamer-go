package streamer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Streamer Version
var Version = "0.5.1"

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

// ParseInt parses a string into an integer or returns 0 if it fails
func ParseInt(s string) int {
	n, err := fmt.Sscanf(s, "%d")
	if err != nil || n != 1 {
		return 0
	}

	return n
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// checkCommandVersion checks the version of a given command
func CheckCommandVersion(name string, command []string, minimumVersion []int) error {
	// minimumVersionString := fmt.Sprintf("%d.%d.%d", minimumVersion[0], minimumVersion[1], minimumVersion[2])
	versionString := strings.Trim(fmt.Sprint(minimumVersion), "[]")
	minimumVersionString := strings.ReplaceAll(versionString, " ", ".")
	output, err := exec.Command(command[0], command[1:]...).Output()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Fprintln(os.Stderr, string(exitError.Stderr))
		}

		return &VersionError{Name: name, Problem: "not found", RequiredVersion: minimumVersionString}
	}

	// Matches two or more numbers (one or more digits each) separated by dots.
	// For example: 4.1.3 or 7.2 or 216.999.8675309
	versionRegex := regexp.MustCompile(`[0-9]+(?:\.[0-9]+)+`)
	versionMatch := versionRegex.FindString(string(output))

	if versionMatch != "" {
		versionString := versionMatch
		versionSlice := make([]int, 0)

		for _, piece := range regexp.MustCompile(`\.`).Split(versionString, -1) {
			num, _ := strconv.Atoi(piece)
			versionSlice = append(versionSlice, num)
		}

		version := versionSlice

		if version[0] < minimumVersion[0] || (version[0] == minimumVersion[0] && version[1] < minimumVersion[1]) || (version[0] == minimumVersion[0] && version[1] == minimumVersion[1] && version[2] < minimumVersion[2]) {
			return &VersionError{Name: name, Problem: "out of date", RequiredVersion: minimumVersionString}
		}
	} else {
		return &VersionError{Name: name, Problem: "version could not be parsed", RequiredVersion: minimumVersionString}
	}

	return nil
}

func StartsWithAny(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}

	return false
}

func IsURL(outputLocation string) bool {
	return strings.HasPrefix(outputLocation, "http:") || strings.HasPrefix(outputLocation, "https:")
}

func RemoveIfExists(outputLocation string) error {
	if _, err := os.Stat(outputLocation); err == nil {
		// Output location exists, so remove it and all its contents
		if err := os.RemoveAll(outputLocation); err != nil {
			return err
		}
	}

	return nil
}

func ManifestFormatListToStringList(manifestFormatList []ManifestFormat) []string {
	stringList := make([]string, len(manifestFormatList))
	for i, manifestFormat := range manifestFormatList {
		stringList[i] = string(manifestFormat)
	}
	return stringList
}
