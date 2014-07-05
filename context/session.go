package context

import (
	"bytes"
	"crypto/hmac"
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
	ErrExpired = errors.New("Session expired")
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
type Session struct {
	Name   string
	Values SessionValues
	MaxAge time.Duration
	Path   string

	secret []byte
	mutex  sync.RWMutex
}

type SessionValues map[interface{}]interface{}
type FlashValues map[interface{}]interface{}

type contextKey string

var cookieName = "session"

var fsMutex sync.RWMutex

// NewSession creates a new session object.
func NewSession(name string, secret []byte, path string) *Session {
	return &Session{
		Name:   name,
		Values: SessionValues{},
		MaxAge: time.Hour,
		Path:   path,
		secret: secret,
	}
}

// GetSession fetches the named session from the context.
func GetSession(name string, r *http.Request, c *Context) (*Session, bool) {
	if c != nil {
		if sd, ok := c.Get(r, contextKey(name)); ok {
			if s, ok := sd.(*Session); ok {
				return s, true
			}
		}
	}

	return nil, false
}

// ReadSession reads the session cookie from the request, and then fetches
// the session data from the filesystem.
func ReadSession(secret []byte, path string, r *http.Request, c *Context) (*Session, error) {
	d := http.Dir(path)

	if cookie, err := r.Cookie(cookieName); err == nil {
		name, date, err := decodeName(cookie.Value, secret)

		if err != nil {
			return nil, err
		}

		var s *Session
		var ok bool

		if s, ok = GetSession(name, r, c); !ok {
			fsMutex.RLock()
			defer fsMutex.RUnlock()

			if file, err := d.Open(name); err == nil {
				defer file.Close()

				s = &Session{}

				dec := gob.NewDecoder(file)

				if err := dec.Decode(s); err == nil {
					s.secret = secret
				} else {
					return nil, err
				}
			}
		}

		if s == nil {
			s = NewSession(name, secret, path)
		} else {
			if s.MaxAge != 0 {
				maxAge := time.Now().Add(-s.MaxAge).Unix()

				if date < maxAge {
					s.DeleteAll()
					s.MaxAge = time.Hour
					return s, ErrExpired
				}
			}
		}

		if c != nil {
			c.Set(r, contextKey(name), s)
		}

		return s, nil
	} else {
		return nil, nil
	}

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

// Write stores the session name in the session cookie along with the current
// date, and writes the session data to the filesystem, under the path which
// is stored in the session itself.
func (s *Session) Write(w http.ResponseWriter) error {
	buf := util.BufferPool.GetBuffer()
	defer util.BufferPool.Put(buf)

	enc := gob.NewEncoder(buf)

	if err := enc.Encode(s); err != nil {
		return err
	}

	if filepath.Separator != '/' && strings.IndexRune(s.Name, filepath.Separator) >= 0 ||
		strings.Contains(s.Name, "\x00") {
		return errors.New("http: invalid character in file path")
	}

	val, date, err := encodeName([]byte(s.Name), s.secret)

	if err != nil {
		return err
	}

	var cookie *http.Cookie

	if s.MaxAge > 0 {
		t := time.Unix(date, 0).Add(s.MaxAge)
		cookie = &http.Cookie{Name: cookieName, Value: val, Path: "/", MaxAge: int(s.MaxAge.Seconds()), Expires: t}
	} else {
		cookie = &http.Cookie{Name: cookieName, Value: val, Path: "/"}
	}
	http.SetCookie(w, cookie)

	fsMutex.Lock()
	defer fsMutex.Unlock()

	if err := os.MkdirAll(s.Path, os.FileMode(0700)); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(s.Path, filepath.FromSlash(path.Clean("/"+s.Name))),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return err
	}

	f.Write(buf.Bytes())
	f.Close()

	return nil
}

// Set stores a key-value pair in the session.
func (s *Session) Set(key interface{}, val interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Values[key] = val
}

// Get fetches a value for a given key.
func (s *Session) Get(key interface{}) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if val, ok := s.Values[key]; ok {
		return val, ok
	}
	return nil, false
}

// DeleteAll removes all key-value pairs from the session.
func (s *Session) DeleteAll() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Values = SessionValues{}
}

// Delete removes a value for a given key.
func (s *Session) Delete(key interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.Values, key)
}

// Flash gets a flash value for a given key from the session.  Flash values
// are temporary values that are removed when they are fetched.
func (s *Session) Flash(key interface{}) interface{} {
	var val interface{}

	if flash, ok := s.Get(contextKey("flashValues")); ok {
		flashValues := flash.(FlashValues)

		if val, ok = flashValues[key]; ok {
			delete(flashValues, key)

			return val
		}
	}

	return val
}

// SetFlash stores a flash value under a given key.
func (s *Session) SetFlash(key interface{}, value interface{}) {
	var flashValues FlashValues

	if val, ok := s.Get(contextKey("flashValues")); ok {
		flashValues = val.(FlashValues)
	} else {
		flashValues = FlashValues{}
	}

	flashValues[key] = value

	s.Set(contextKey("flashValues"), flashValues)
}

func decodeName(data string, secret []byte) (string, int64, error) {
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

	if !checkSignature(parts[2], parts[0], secret, t1) {
		return "", 0, errors.New("Signatures don't match")
	}

	return string(parts[0]), t1, nil
}

func encodeName(name, secret []byte) (string, int64, error) {
	now := time.Now().Unix()

	sig, err := createSignature(name, secret, now)

	if err != nil {
		return "", 0, err
	}

	message := []byte(fmt.Sprintf("%s|%d|%s", name, now, sig))

	encoded := base64.URLEncoding.EncodeToString(message)

	return string(encoded), now, nil
}

func checkSignature(signature, name, secret []byte, date int64) bool {
	expected, err := createSignature(name, secret, date)
	if err != nil {
		return false
	}

	return hmac.Equal(signature, expected)
}

func createSignature(name, secret []byte, date int64) ([]byte, error) {
	hm := hmac.New(sha256.New, secret)

	message := []byte(fmt.Sprintf("%s|%s|%d", cookieName, name, date))

	if _, err := hm.Write(message); err != nil {
		return nil, err
	}

	mac := hm.Sum(nil)

	return mac, nil
}
