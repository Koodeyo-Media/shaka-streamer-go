package streamer

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
