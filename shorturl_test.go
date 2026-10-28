package shorturl

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func service(t *testing.T) *Service {
	t.Helper()
	s, err := New(&MemoryStore{}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestURLValidation(t *testing.T) {
	s := service(t)
	valid := []string{
		"https://example.com", "https://www.Example.com/a%2Fb?sig=A%2Bb&z=2&a=1#part%20one",
		"https://127.0.0.1:443/", "https://[::1]:8080/", "https://[::ffff:192.0.2.1]/",
		"https://localhost/", "https://xn--bcher-kva.example/", "https://example.com:65535/",
		"https://example.com/日本語", "https://example.com/?q=a+b",
	}
	for _, in := range valid {
		t.Run(in, func(t *testing.T) {
			if err := s.ValidateURL(in); err != nil {
				t.Fatal(err)
			}
			l, err := s.Shorten(context.Background(), in, 0)
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.Resolve(context.Background(), l.Code)
			if err != nil || got != in {
				t.Fatal("destination was rewritten or rejected", err)
			}
		})
	}
	invalid := []string{
		"", "example.com", "//example.com", "/a", "https:example.com", "http://example.com",
		"ftp://example.com", "javascript:alert(1)", "https:///path", "https://",
		"https://user:password@example.com", "https://@example.com", " https://example.com", "https://example.com\n",
		"https://example.com/\t", "https://example.com/\\evil", "https://example.com/%", "https://example.com/?x=%GG",
		"https://example.com/#%q0", "https://example.com/?x=%0d%0a", "https://example.com/%00", "https://example.com/%5cfoo",
		"https://example.com.", "https://bücher.example/", "https://-example.com", "https://example-.com", "https://a..com",
		"https://example_com", "https://example.com:", "https://example.com:0", "https://example.com:65536",
		"https://example.com:0443", "https://example.com:+443", "https://example.com:no",
		"https://[example.com]", "https://[127.0.0.1]", "https://[fe80::1%25eth0]", "https://::1",
		"https://127.1", "https://2130706433", "https://0177.0.0.1", "https://0x7f000001", "https://0x7f.0.0.1",
		"https://a.123", "https://example.com/\x7f", "https://example.com/\xff",
		"https://" + strings.Repeat("a", 64) + ".com", "https://example.com/" + strings.Repeat("a", 8192),
	}
	for _, in := range invalid {
		t.Run(in, func(t *testing.T) {
			if err := s.ValidateURL(in); !errors.Is(err, ErrInvalidURL) {
				t.Fatal("invalid URL accepted")
			}
		})
	}
	httpService, _ := New(&MemoryStore{}, Config{AllowHTTP: true})
	if err := httpService.ValidateURL("http://example.com"); err != nil {
		t.Fatal(err)
	}
	small, _ := New(&MemoryStore{}, Config{MaxURLBytes: 20})
	if err := small.ValidateURL("https://example.com/"); err != nil {
		t.Fatal(err)
	}
	if err := small.ValidateURL("https://example.com/a"); !errors.Is(err, ErrInvalidURL) {
		t.Fatal(err)
	}
}

func TestConfigAndCodes(t *testing.T) {
	var nilStore *MemoryStore
	for _, st := range []Store{nil, nilStore} {
		if _, err := New(st, Config{}); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
	if _, err := New(&MemoryStore{}, Config{MaxURLBytes: -1}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	code := base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	if err := ValidateCode(code); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "abcdef", code + "=", strings.Repeat("!", 22), code[:21] + "B", strings.Repeat("A", 21), code[:10] + "\n" + code[11:]} {
		if !errors.Is(ValidateCode(bad), ErrInvalidCode) {
			t.Fatalf("accepted malformed code %q", bad)
		}
	}
}

func TestLifecycle(t *testing.T) {
	s := service(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	ctx := context.Background()
	if _, err := s.Shorten(ctx, "https://example.com", -1); !errors.Is(err, ErrInvalidTTL) {
		t.Fatal(err)
	}
	link, err := s.Shorten(ctx, "https://www.example.com/a%2Fb?b=2&a=1#frag", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !link.CreatedAt.Equal(now) || !link.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatal("incorrect timestamps")
	}
	now = link.ExpiresAt.Add(-time.Nanosecond)
	if _, err := s.Resolve(ctx, link.Code); err != nil {
		t.Fatal(err)
	}
	now = link.ExpiresAt
	if _, err := s.Resolve(ctx, link.Code); !errors.Is(err, ErrExpired) {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := s.Revoke(ctx, link.Code); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Resolve(ctx, link.Code); !errors.Is(err, ErrRevoked) {
		t.Fatal(err)
	}
	permanent, err := s.Shorten(ctx, "https://example.com", 0)
	if err != nil || !permanent.ExpiresAt.IsZero() {
		t.Fatal("permanent link", err)
	}
	again, err := s.Shorten(ctx, "https://example.com", 0)
	if err != nil || again.Code == permanent.Code {
		t.Fatal("unexpected deduplication", err)
	}
	missing := base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	if _, err := s.Resolve(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.Revoke(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := s.Resolve(ctx, "legacy"); !errors.Is(err, ErrInvalidCode) {
		t.Fatal(err)
	}
	if err := s.Revoke(ctx, "legacy"); !errors.Is(err, ErrInvalidCode) {
		t.Fatal(err)
	}
}

type fakeStore struct {
	insert func(context.Context, Link) error
	get    func(context.Context, string) (Link, error)
	revoke func(context.Context, string) error
}

func (f fakeStore) Insert(ctx context.Context, l Link) error        { return f.insert(ctx, l) }
func (f fakeStore) Get(ctx context.Context, c string) (Link, error) { return f.get(ctx, c) }
func (f fakeStore) Revoke(ctx context.Context, c string) error      { return f.revoke(ctx, c) }

func TestCollisionHandling(t *testing.T) {
	ctx := context.Background()
	s := service(t)
	zeros := make([]byte, 16)
	first := base64.RawURLEncoding.EncodeToString(zeros)
	original := Link{Code: first, Destination: "https://original.example"}
	if err := s.store.Insert(ctx, original); err != nil {
		t.Fatal(err)
	}
	s.random = bytes.NewReader(append(zeros, bytes.Repeat([]byte{1}, 16)...))
	l, err := s.Shorten(ctx, "https://new.example", 0)
	if err != nil || l.Code == first {
		t.Fatal("collision retry failed", err)
	}
	stored, _ := s.store.Get(ctx, first)
	if stored != original {
		t.Fatal("collision overwrote original")
	}
	s.random = bytes.NewReader(make([]byte, 16*insertAttempts))
	if _, err := s.Shorten(ctx, "https://new.example", 0); !errors.Is(err, ErrCodeExhausted) {
		t.Fatal(err)
	}
	if _, err := s.Shorten(ctx, "https://new.example", 0); !errors.Is(err, ErrRandom) {
		t.Fatal(err)
	}
}

func TestStorageFailuresAndCancellation(t *testing.T) {
	backend := errors.New("backend contains private credentials")
	for _, cause := range []error{backend, context.Canceled, context.DeadlineExceeded} {
		calls := 0
		st := fakeStore{
			insert: func(context.Context, Link) error { calls++; return cause },
			get:    func(context.Context, string) (Link, error) { return Link{}, cause },
			revoke: func(context.Context, string) error { return cause },
		}
		s, _ := New(st, Config{})
		code := base64.RawURLEncoding.EncodeToString(make([]byte, 16))
		_, a := s.Shorten(context.Background(), "https://example.com", 0)
		_, b := s.Resolve(context.Background(), code)
		c := s.Revoke(context.Background(), code)
		for _, err := range []error{a, b, c} {
			if !errors.Is(err, ErrStorage) || !errors.Is(err, cause) || strings.Contains(err.Error(), "credentials") {
				t.Fatal("lost classification or leaked error", err)
			}
		}
		if calls != 1 {
			t.Fatal("retried operational error")
		}
	}
	s := service(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, a := s.Shorten(ctx, "https://example.com", 0)
	_, b := s.Resolve(ctx, strings.Repeat("A", 22))
	c := s.Revoke(ctx, strings.Repeat("A", 22))
	for _, err := range []error{a, b, c} {
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	// Cancellation during a backend call reaches the caller even if that backend returns success.
	ctx, cancel = context.WithCancel(context.Background())
	st := fakeStore{insert: func(received context.Context, _ Link) error {
		if received != ctx {
			t.Fatal("context not propagated")
		}
		cancel()
		return nil
	}}
	s, _ = New(st, Config{})
	if _, err := s.Shorten(ctx, "https://example.com", 0); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestResolveValidatesStoredData(t *testing.T) {
	code := strings.Repeat("A", 22)
	for _, link := range []Link{
		{Code: code, Destination: "javascript:alert(1)"},
		{Code: code, Destination: "http://example.com"},
		{Code: "different", Destination: "https://example.com"},
	} {
		s, _ := New(fakeStore{get: func(context.Context, string) (Link, error) { return link, nil }}, Config{})
		if got, err := s.Resolve(context.Background(), code); err == nil || got != "" {
			t.Fatal("unsafe stored data returned")
		}
	}
}

func TestMemoryStoreConcurrency(t *testing.T) {
	var m MemoryStore
	ctx := context.Background()
	original := Link{Code: strings.Repeat("A", 22), Destination: "https://example.com"}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var winners atomic.Int32
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := m.Insert(ctx, original)
			if err == nil {
				winners.Add(1)
			} else if !errors.Is(err, ErrConflict) {
				t.Error(err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatal("insert was not atomic")
	}
	copy, err := m.Get(ctx, original.Code)
	if err != nil {
		t.Fatal(err)
	}
	copy.Destination = "https://changed.example"
	if got, _ := m.Get(ctx, original.Code); got != original {
		t.Fatal("mutable storage alias")
	}
	start = make(chan struct{})
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := m.Revoke(ctx, original.Code); err != nil {
				t.Error(err)
			}
			got, err := m.Get(ctx, original.Code)
			if err != nil || !got.Revoked {
				t.Error("post-revocation read", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	a := m.Insert(ctx, Link{Code: "other"})
	_, b := m.Get(ctx, original.Code)
	c := m.Revoke(ctx, original.Code)
	for _, err := range []error{a, b, c} {
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
}

func TestConcurrentService(t *testing.T) {
	s := service(t)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			l, err := s.Shorten(ctx, "https://example.com", 0)
			if err != nil {
				t.Error(err)
				return
			}
			if _, err := s.Resolve(ctx, l.Code); err != nil {
				t.Error(err)
			}
			if err := s.Revoke(ctx, l.Code); err != nil {
				t.Error(err)
			}
			if _, err := s.Resolve(ctx, l.Code); !errors.Is(err, ErrRevoked) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestRandomFailure(t *testing.T) {
	s := service(t)
	s.random = io.LimitReader(bytes.NewReader(make([]byte, 16)), 15)
	if _, err := s.Shorten(context.Background(), "https://example.com", 0); !errors.Is(err, ErrRandom) {
		t.Fatal(err)
	}
}

func FuzzValidateURL(f *testing.F) {
	for _, s := range []string{"https://example.com/a%2Fb?b=2&a=1#x", "https://127.1", "https://[::1]", "https://example.com/?x=%0d", "\\", "https://example.com/%+a"} {
		f.Add(s)
	}
	s, _ := New(&MemoryStore{}, Config{})
	f.Fuzz(func(t *testing.T, input string) {
		err := s.ValidateURL(input)
		if err != nil && !errors.Is(err, ErrInvalidURL) {
			t.Fatal(err)
		}
		if err == nil {
			// Successful validation must not require URL rewriting to survive a round trip.
			st, _ := New(&MemoryStore{}, Config{})
			l, err := st.Shorten(context.Background(), input, 0)
			if err != nil {
				t.Fatal(err)
			}
			got, err := st.Resolve(context.Background(), l.Code)
			if err != nil || got != input {
				t.Fatal("round trip failed", err)
			}
		}
	})
}

func FuzzValidateCode(f *testing.F) {
	for _, c := range []string{strings.Repeat("A", 22), strings.Repeat("_", 22), "abcdef", "", strings.Repeat("A", 21) + "B"} {
		f.Add(c)
	}
	f.Fuzz(func(t *testing.T, code string) {
		if ValidateCode(code) == nil {
			b, err := base64.RawURLEncoding.Strict().DecodeString(code)
			if err != nil || len(b) != 16 || base64.RawURLEncoding.EncodeToString(b) != code {
				t.Fatal("noncanonical code")
			}
		}
	})
}
