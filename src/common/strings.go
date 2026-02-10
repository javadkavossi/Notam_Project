package common

import (
	"errors"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/hossein-repo/BaseProject/config"
)

var (
	lowerCharSet   = "abcdedfghijklmnopqrstuvwxyz"
	upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialCharSet = "!@#$%&*"
	numberSet      = "0123456789"
	allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// این تابع برای validator - فقط true/false برمی‌گرداند
func IsValidPassword(password string) bool {
	valid, _ := CheckPassword(password)
	return valid
}

// این همون تابع قبلی که پیام خطا میده - برای استفاده در handler
func CheckPassword(password string) (bool, error) {
	var (
		minLength    = 8
		lowerCharSet = `[a-z]`
		upperCharSet = `[A-Z]`
		numberSet    = `[0-9]`
		specialSet   = `[!@#~$%^&*()+|_.,<>?/{}\-]`
	)

	if len(password) < minLength {
		return false, errors.New("password must be at least 8 characters long")
	}
	if matched, _ := regexp.MatchString(lowerCharSet, password); !matched {
		return false, errors.New("password must contain at least one lowercase letter")
	}
	if matched, _ := regexp.MatchString(upperCharSet, password); !matched {
		return false, errors.New("password must contain at least one uppercase letter")
	}
	if matched, _ := regexp.MatchString(numberSet, password); !matched {
		return false, errors.New("password must contain at least one number")
	}
	if matched, _ := regexp.MatchString(specialSet, password); !matched {
		return false, errors.New("password must contain at least one special character")
	}

	return true, nil
}

func GeneratePassword() string {
	var password strings.Builder

	cfg := config.GetConfig()
	passwordLength := cfg.Password.MinLength + 2
	minSpecialChar := 2
	minNum := 3
	if !cfg.Password.IncludeDigits {
		minNum = 0
	}

	minUpperCase := 3
	if !cfg.Password.IncludeUppercase {
		minUpperCase = 0
	}

	minLowerCase := 3
	if !cfg.Password.IncludeLowercase {
		minLowerCase = 0
	}

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random := rand.Intn(len(specialCharSet))
		password.WriteString(string(specialCharSet[random]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	//Set lowercase
	for i := 0; i < minLowerCase; i++ {
		random := rand.Intn(len(lowerCharSet))
		password.WriteString(string(lowerCharSet[random]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}

func GenerateOtp() string {
	cfg := config.GetConfig()
	rand.Seed(time.Now().UnixNano())
	min := int(math.Pow(10, float64(cfg.Otp.Digits-1)))   // 10^d-1 100000
	max := int(math.Pow(10, float64(cfg.Otp.Digits)) - 1) // 999999 = 1000000 - 1 (10^d) -1

	var num = rand.Intn(max-min) + min
	return strconv.Itoa(num)
}

func HasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func HasLower(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func HasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func HasDigits(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// To snake case : CountryId -> country_id
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
