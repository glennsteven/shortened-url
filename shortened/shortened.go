package shortened

import (
	"crypto/md5"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"shortened_url/consts"
	"shortened_url/entity"
	"strings"
	"time"
)

// save data with format [hash: link]
var (
	linkMap      = make(map[string]entity.URL)
	randomLookup = make(map[string]string)
	reg          = regexp.MustCompile("^(http|https)://")
)

func Shorten(in string) (string, error) {
	//Validation validUrl
	ok := ValidLink(in)
	if !ok {
		return "", errors.New("not found")
	}
	res, _ := TransformWWW(in)
	h := md5.New()
	h.Write([]byte(res))
	fmt.Println()
	hashOriginal := string(h.Sum(nil))
	//save hashOriginal
	original, ok := linkMap[hashOriginal]
	if ok {
		return original.Shortened, nil
	}

	found := true
	shortened := ""
	for found {
		shortened, _ = randomString(6)
		_, ok = randomLookup[shortened]
		if ok {
			continue
		}
		found = false
	}

	linkMap[hashOriginal] = entity.URL{
		Shortened: shortened,
		Original:  res,
	}
	randomLookup[shortened] = hashOriginal

	return shortened, nil
}

func Extend(shorten string) (string, error) {
	hash, ok := randomLookup[shorten]
	if !ok {
		return "", errors.New("not found")
	}

	v, ok := linkMap[hash]
	if !ok {
		return "", errors.New("not found")
	}

	return v.Original, nil
}

func randomString(n int) (string, error) {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = consts.LetterBytes[rand.Intn(len(consts.LetterBytes))]
	}

	return string(b), errors.New("exp")
}

func TransformWWW(u string) (string, error) {
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "www.")
	return fmt.Sprintf("https://%s", u), nil
}

func ValidLink(s string) bool {
	if strings.Contains(s, "://") {
		return reg.MatchString(s)
	}
	return true
}
