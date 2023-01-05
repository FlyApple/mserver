package util

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

//
type TAny interface{}
type TMapAny map[string]TAny
type TA TAny
type TMA TMapAny

// TIME
const (
	TIME_SECOND   = 1.0
	TIME_MINUTE   = 1.0 * 60.0
	TIME_HOUR     = 1.0 * 60.0 * 60.0
	TIME_DAY      = 1.0 * 60.0 * 60.0 * 24
	TIME_7DAY     = 1.0 * 60.0 * 60.0 * 24 * 7
	TIME_30DAY    = 1.0 * 60.0 * 60.0 * 24 * 30
	TIME_KEEP     = -1
	TIME_KEEPN    = TIME_DAY * 365 * 100
	TIME_EXPIREDN = -1.0
)

// ERROR
const (
	RESULT_ERROR_NOT_EXIST = -7
	RESULT_ERROR_NOT_FOUND = -6
	RESULT_ERROR_INTERNAL  = -3
	RESULT_ERROR_INVALID   = -2
	RESULT_ERROR_UNKNOW    = -1
	RESULT_FAILED          = -1
	RESULT_SUCCESSED       = 0
	RESULT_OK              = 0

	STATUS_ERROR_NOT_EXIST = "ERROR_NOT_EXIST"
	STATUS_ERROR_NOT_FOUND = "ERROR_NOT_FOUND"
	STATUS_ERROR_INTERNAL  = "ERROR_INTERNAL"
	STATUS_ERROR_INVALID   = "ERROR_INVALID"
	STATUS_ERROR_UNKNOW    = "ERROR"
	STATUS_FAILED          = "FAILED"
	STATUS_OK              = "OK"
	STATUS_SUCCESSED       = "SUCCESSED"
)

// RANDOM
var random_seed uint64 = 0

// TIMESTAMP
func GetTimeStamp() uint32 {
	return uint32(time.Now().Unix())
}

func GetTimeStamp64() uint64 {
	return uint64(math.Round(float64(time.Now().UnixMicro()) * 0.001))
}

func GetTimeStamp64M() uint64 {
	return uint64(time.Now().UnixMicro())
}

func CheckTimestamp64(prev uint64, next uint64) int32 {
	if prev == 0 {
		return 0
	}
	if int64(prev) == -1 {
		return TIME_KEEP
	}
	return int32(next - prev)
}

func ExpiredTimestamp64(timestamp uint64, expired float32) float32 {
	if timestamp == 0 {
		return TIME_EXPIREDN
	}
	if int64(timestamp) == -1 {
		return TIME_KEEPN
	}

	t := timestamp + uint64(math.Round(float64(expired*1000.0)))
	n := CheckTimestamp64(GetTimeStamp64(), t)
	if n == 0 {
		return TIME_EXPIREDN
	}
	if n == -1 {
		return TIME_KEEPN
	}
	v := float32(n) * 0.001
	return v
}

// DATE
func DateFormat(date time.Time, level int) string {

	//
	if level == 0 {
		return date.Format("2006-01-02 15:04:05")
	} else if level == 1 {
		return date.Format("2006-01-02")
	} else if level == 2 {
		return date.Format("15:04:05")
	} else if level == 3 {
		return date.Format("2006-01-02 15:04:05.000")
	} else if level == 8 {
		return date.Format("2006-01-02 15:04:05.000 -0700 MST")
	} else if level == 9 {
		return date.Format("2006-01-02 15:04:05.000 Mon")
	}

	//yyyy-MM-dd HH:mm:ss.ms ZONE MST WEEK
	return date.Format("2006-01-02 15:04:05.000 -0700 MST Mon")
}

// RANDOM Number (1-8)
func RandomInit() uint32 {
	if random_seed == 0 {
		random_seed = GetTimeStamp64()
		rand.Seed(int64(random_seed))
	}
	var value = random_seed + uint64(rand.Intn(0xFFFF)) + (uint64(rand.Intn(0xFFFF)) << 0x16)
	return uint32(value & 0xFFFFFFFF)
}

func RandomNumber() uint32 {
	var value = RandomInit()
	var a = uint32(rand.Intn(0x0FFF))
	var b = uint32(rand.Intn(0x0FFF))
	var c = uint32(rand.Intn(0x0FFF))
	var d = uint32(rand.Intn(0x0FFF))
	var r = a<<0x00 | b<<0x08 | c<<0x16 | d<<0x24
	return (r ^ value) & 0xFFFFFFFF
}

func RandomRange(min int32, max int32) uint32 {
	var value = RandomNumber()
	if max > 0 {
		value = value % uint32(max)
	}
	if min > 0 && value+uint32(min) < uint32(max) {
		value = value + uint32(min)
	}
	return value
}

//
func RandomChars(max int32, level int32) string {
	var chars1 = "0123456789"
	var chars2 = "abcdefghijklmnopqrstuvwxyz"
	var chars3 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var chars4 = "abcdefhkmnprstvwxy"
	var chars5 = "ABCDEFHKMNPRSTVWXY"
	var chars = chars1 + chars2 + chars3
	// lowercase characters + number
	if level == 0 {
		chars = chars1 + chars2
		// uppercase characters + number
	} else if level == 1 {
		chars = chars1 + chars3
	} else if level == 2 {
		chars = chars1 + chars4
	} else if level == 3 {
		chars = chars1 + chars5
	} else if level == 9 {
		chars = chars1 + chars4 + chars5
	}
	// default all characters
	var count = len(chars)
	var value = int(RandomRange(1000, 9999))
	var text = ""
	for n := 0; n < int(max); n++ {
		i := value % count
		var c = chars[i]
		text = text + string(c)
		value = value / count
		if value == 0 {
			value = int(RandomRange(1000, 9999))
		}
	}
	return text
}

func GenerateAuthCode(level int32) string {
	// Random 6 number
	var value = RandomRange(100000, 999999)
	if level == 0 {
		return fmt.Sprintf("%d", value)
		// Random 8 number
	} else if level == 1 {
		value = RandomRange(10000000, 99999999)
		return fmt.Sprintf("%d", value)
		// Random 6 characters lowercase, uppercase and number
	} else if level == 2 {
		return RandomChars(6, 9)
		// Random 8 characters uppercase and number
	} else if level == 3 {
		return RandomChars(8, 3)
		// Random 8 characters lowercase and number
	} else if level == 4 {
		return RandomChars(8, 2)
	}

	return ""
}

// default : 8 number
// level 1: 10 number
// level 2: 12 number
func GenerateIDX(level int) int64 {
	var date = time.Now()
	year := date.Year()%1000 + 1000
	a := int(date.Month())*10 + date.Day()
	b := date.Hour()
	c := date.Minute()
	x := int64(RandomRange(1000, 9999))
	var value int64 = int64(year + a)
	if level == 1 {
		value = value*100 + int64(b+c)
	} else if level == 2 {
		y := int64(RandomRange(1000, 9999))
		value = value*100 + y
	} else if level == 3 || level == 4 {
		value = value + int64(b)
	}

	value = value*1000 + x

	var cc = []int{1, 2, 3, 4, 5, 6, 7}
	var n = value
	i := 0
	v := 0
	for n > 0 {
		var m = int(n % 10)
		v = v + m*cc[i%len(cc)]
		n = n / 10
		i++
	}

	value = value*10 + int64(v%10)
	var cx = []int{30, 31, 32, 33, 35, 36, 38, 39}
	var cv = int64(cx[v%len(cx)])
	if level == 3 || level == 4 {
		value = cv*10000*10000 + value
	}
	if level == 4 {
		value = 100*10000*10000 + value
	}
	return value
}

//MAP Concat
func MapConcat[T TMapAny | TMA](maps ...T) T {
	var result T = nil
	for _, m := range maps {
		if m == nil {
			continue
		}

		if result == nil {
			result = make(T)
		}

		for i, v := range m {
			_, ok := result[i]
			// Overwrite the value of the same key
			if ok {
				result[i] = v
				// Newwrite
			} else {
				result[i] = v
			}
		}
	}
	return result
}

func MapConcatPtr[T TMapAny | TMA](maps ...*T) *T {
	var result T = nil
	for _, m := range maps {
		if m == nil || *m == nil {
			continue
		}

		mv := *m
		if result == nil {
			result = make(T)
		}

		for i, v := range mv {
			_, ok := result[i]
			// Overwrite the value of the same key
			if ok {
				result[i] = v
				// Newwrite
			} else {
				result[i] = v
			}
		}
	}
	return &result
}
