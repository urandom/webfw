package context

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urandom/webfw/util"
)

var (
	ErrExpired        = errors.New("Session expired")
	ErrNotExist       = errors.New("Session does not exist")
	ErrCookieNotExist = errors.New("Session cookie does not exist")
)

/*
The Session represents persistent user data. Its Name field is how it is
identified.  The secret is used to salt the hmac, sent along the data
to the client as a cookie. The data is a base64 encoded string, containing
the session name and date showing when it was last used. The actual session
data is stored in the filesystem, in a directory specified by the Path
field. The data is serialized using encoding/gob, therefore any custom
data type should be registered with it. The MaxAge field specifies a duration,
after which an unused session will get cleared of its data and marked as
expired. It is also used as the max-age and expires fields of the session
cookie.
*/
type Session interface {
	Read(*http.Request, Context) error
	Write(http.ResponseWriter) error
	Name() string
	SetName(string)
	MaxAge() time.Duration
	SetMaxAge(time.Duration)
	CookieName() string
	SetCookieName(string)

	Set(interface{}, interface{})
	Get(interface{}) (interface{}, bool)
	GetAll() SessionValues
	DeleteAll()
	Delete(interface{})
	Flash(interface{}) (interface{}, bool)
	SetFlash(interface{}, interface{})
}

type SessionGenerator func(secret, cipher []byte, path string) Session

type SessionValues map[interface{}]interface{}
type FlashValues map[interface{}]interface{}

type session struct {
	Path string

	name       string
	maxAge     time.Duration
	values     SessionValues
	secret     []byte
	block      cipher.Block
	cookieName string
	mutex      sync.RWMutex
}

type fileData struct {
	Name       string
	MaxAge     time.Duration
	Values     SessionValues
	CookieName string
}

type contextKey string

var fsMutex sync.RWMutex

// NewSession creates a new session object.
func NewSession(secret, cipher []byte, path string) Session {
	s := &session{
		Path:       path,
		maxAge:     time.Hour,
		values:     SessionValues{},
		secret:     secret,
		cookieName: "session",
	}

	if cipher != nil && len(cipher) > 0 {
		if b, err := aes.NewCipher(cipher); err == nil {
			s.block = b
		} else {
			panic(err)
		}
	}

	return s
}

// CleanupSessions is a helper function for clearing all session data
// from the filesystem older than a given age. If the age is 0, all
// session data is removed.
func CleanupSessions(path string, age time.Duration) error {
	fsMutex.Lock()
	defer fsMutex.Unlock()

	files, err := ioutil.ReadDir(path)

	if err != nil {
		return err
	}

	for _, fi := range files {
		if age > 0 {
			min := time.Now().Add(-age).UnixNano()
			if fi.ModTime().UnixNano() >= min {
				continue
			}
		}

		if err := os.Remove(filepath.Join(path, fi.Name())); err != nil {
			return err
		}
	}

	return nil
}

// Read fetches the session from the cookie, and loads the session data from
// the filesystem. It may return a generic error due to the various read
// operations, or one of the following:
//  - ErrExpired - if its older than the set max-age. The session data
//    is removed
//  - ErrNotExist - if session data hasn't been found for this session
//  - ErrCookieNotExist - if a session cookie doesn't exist
func (s *session) Read(r *http.Request, c Context) error {
	d := http.Dir(s.Path)

	if cookie, err := r.Cookie(s.cookieName); err == nil {
		name, date, err := s.decodeName(cookie.Value)

		if err != nil {
			return err
		}

		var data *fileData
		var ok bool

		if data, ok = getSessionData(s.name, r, c); ok {
			s.fromData(data)
		} else {
			fsMutex.RLock()
			defer fsMutex.RUnlock()

			if file, err := d.Open(name); err == nil {
				defer file.Close()

				data = &fileData{}

				dec := gob.NewDecoder(file)

				if err := dec.Decode(data); err == nil {
					s.fromData(data)
				} else {
					return err
				}
			}
		}

		if data == nil {
			s.name = name

			return ErrNotExist
		} else {
			if s.maxAge != 0 {
				maxAge := time.Now().Add(-s.maxAge).Unix()

				if date < maxAge {
					s.DeleteAll()
					return ErrExpired
				}
			}
		}

		if c != nil {
			c.Set(r, contextKey(name), s.toData())
		}

		return nil
	} else {
		return ErrCookieNotExist
	}
}

// Write stores the session name in the session cookie along with the current
// date, and writes the session data to the filesystem, under the path which
// is stored in the session itself.
func (s *session) Write(w http.ResponseWriter) error {
	buf := util.BufferPool.GetBuffer()
	defer util.BufferPool.Put(buf)

	enc := gob.NewEncoder(buf)

	if err := enc.Encode(s.toData()); err != nil {
		return err
	}

	if filepath.Separator != '/' && strings.IndexRune(s.name, filepath.Separator) >= 0 ||
		strings.Contains(s.name, "\x00") {
		return errors.New("http: invalid character in file path")
	}

	val, date, err := s.encodeName()

	if err != nil {
		return err
	}

	var cookie *http.Cookie

	if s.maxAge > 0 {
		t := time.Unix(date, 0).Add(s.maxAge)
		cookie = &http.Cookie{Name: s.cookieName, Value: val, Path: "/", MaxAge: int(s.maxAge.Seconds()), Expires: t}
	} else {
		cookie = &http.Cookie{Name: s.cookieName, Value: val, Path: "/"}
	}
	http.SetCookie(w, cookie)

	fsMutex.Lock()
	defer fsMutex.Unlock()

	if err := os.MkdirAll(s.Path, os.FileMode(0700)); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(s.Path, filepath.FromSlash(path.Clean("/"+s.name))),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return err
	}

	f.Write(buf.Bytes())
	f.Close()

	return nil
}

// Name returns the name of the session
func (s *session) Name() string {
	return s.name
}

// SetName sets a new name for the session
func (s *session) SetName(name string) {
	s.name = name
}

// MaxAge returns the max-age of the session
func (s *session) MaxAge() time.Duration {
	return s.maxAge
}

// SetMaxAge sets a new max age for the session
func (s *session) SetMaxAge(maxAge time.Duration) {
	s.maxAge = maxAge
}

// CookieName returns the cookie name, used to store the session name in
// the client
func (s *session) CookieName() string {
	return s.cookieName
}

// SetCookieName sets a new cookie name for the session
func (s *session) SetCookieName(name string) {
	s.cookieName = name
}

// Set stores a key-value pair in the session.
func (s *session) Set(key interface{}, val interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.values[key] = val
}

// Get fetches a value for a given key.
func (s *session) Get(key interface{}) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if val, ok := s.values[key]; ok {
		return val, ok
	}
	return nil, false
}

// GetAll returns all values stored in the session
func (s *session) GetAll() SessionValues {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.values
}

// DeleteAll removes all key-value pairs from the session.
func (s *session) DeleteAll() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.values = SessionValues{}
}

// Delete removes a value for a given key.
func (s *session) Delete(key interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.values, key)
}

// Flash gets a flash value for a given key from the session.  Flash values
// are temporary values that are removed when they are fetched.
func (s *session) Flash(key interface{}) (interface{}, bool) {
	var val interface{}

	if flash, ok := s.Get(contextKey("flashValues")); ok {
		flashValues := flash.(FlashValues)

		if val, ok = flashValues[key]; ok {
			delete(flashValues, key)

			return val, ok
		}
	}

	return val, false
}

// SetFlash stores a flash value under a given key.
func (s *session) SetFlash(key interface{}, value interface{}) {
	var flashValues FlashValues

	if val, ok := s.Get(contextKey("flashValues")); ok {
		flashValues = val.(FlashValues)
	} else {
		flashValues = FlashValues{}
	}

	flashValues[key] = value

	s.Set(contextKey("flashValues"), flashValues)
}

func (s *session) toData() *fileData {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &fileData{
		Name:       s.name,
		MaxAge:     s.maxAge,
		Values:     s.values,
		CookieName: s.cookieName,
	}
}

func (s *session) fromData(data *fileData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.name = data.Name
	s.maxAge = data.MaxAge
	s.values = data.Values
	s.cookieName = data.CookieName
}

func (s *session) decodeName(data string) (string, int64, error) {
	decoded, err := base64.URLEncoding.DecodeString(data)

	if err != nil {
		return "", 0, err
	}

	parts := bytes.SplitN(decoded, []byte("|"), 3)
	if len(parts) != 3 {
		return "", 0, errors.New("Not enough cookie parts")
	}

	t1, err := strconv.ParseInt(string(parts[1]), 10, 64)

	if err != nil {
		return "", 0, err
	}

	sig := parts[2]

	if s.block != nil {
		size := s.block.BlockSize()
		if len(sig) > size {
			key := sig[:size]
			sig = sig[size:]

			ctr := cipher.NewCTR(s.block, key)
			ctr.XORKeyStream(sig, sig)
		} else {
			return "", 0, errors.New("Invalid cookie encryption part")
		}
	}

	if !s.checkSignature(sig, parts[0], t1) {
		return "", 0, errors.New("Signatures don't match")
	}

	return string(parts[0]), t1, nil
}

func (s *session) encodeName() (string, int64, error) {
	now := time.Now().Unix()

	sig, err := createSignature(s.cookieName, []byte(s.name), s.secret, now)

	if err != nil {
		return "", 0, err
	}

	if s.block != nil {
		if key, err := randomData(s.block.BlockSize()); err == nil {
			ctr := cipher.NewCTR(s.block, key)
			ctr.XORKeyStream(sig, sig)

			sig = append(key, sig...)
		} else {
			return "", 0, err
		}
	}

	message := []byte(fmt.Sprintf("%s|%d|%s", s.name, now, sig))

	encoded := base64.URLEncoding.EncodeToString(message)

	return string(encoded), now, nil
}

func (s *session) checkSignature(signature, name []byte, date int64) bool {
	expected, err := createSignature(s.cookieName, name, s.secret, date)
	if err != nil {
		return false
	}

	return hmac.Equal(signature, expected)
}

func createSignature(cookieName string, name, secret []byte, date int64) ([]byte, error) {
	hm := hmac.New(sha256.New, secret)

	message := []byte(fmt.Sprintf("%s|%s|%d", cookieName, name, date))

	if _, err := hm.Write(message); err != nil {
		return nil, err
	}

	mac := hm.Sum(nil)

	return mac, nil
}

func getSessionData(name string, r *http.Request, c Context) (*fileData, bool) {
	if c != nil {
		if sd, ok := c.Get(r, contextKey(name)); ok {
			if s, ok := sd.(*fileData); ok {
				return s, true
			}
		}
	}

	return nil, false
}

func randomData(size int) ([]byte, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}
	return data, nil
}
