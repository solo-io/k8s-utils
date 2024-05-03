package kubeutils

import (
	"crypto/md5"
	"fmt"
	"testing"
)

// This function is a copy of the old version of this function.
// It is used to ensure parity with the old implementation.
func shortenName(name string) string {
	if len(name) > 63 {
		hash := md5.Sum([]byte(name))
		name = fmt.Sprintf("%s-%x", name[:31], hash)
		name = name[:63]
	}
	return name
}

func BenchmarkShortenEqual(b *testing.B) {
	b.Run("shorten name old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			shortenName("jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf")
		}
	})

	b.Run("shortened equals--worst case", func(b *testing.B) {
		shortened := "jfdklanfkljasfhjhldacaslkhdfkjs-f1e0028d0fbfe9afbd1a8bb9b53848d"
		standard := "jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf"
		for i := 0; i < b.N; i++ {
			ShortenedEquals(shortened, standard)
		}
	})

	b.Run("shortened equals--different prefix", func(b *testing.B) {
		shortened := "jfdklanfkljasfhjhlxacaslkhdfkjs-f1e0028d0fbfe9afbd1a8bb9b53848d"
		standard := "jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf"
		for i := 0; i < b.N; i++ {
			ShortenedEquals(shortened, standard)
		}
	})

	b.Run("shortened equals--less than 63 characters", func(b *testing.B) {
		shortened := "hello"
		standard := "hello"
		for i := 0; i < b.N; i++ {
			ShortenedEquals(shortened, standard)
		}
	})
}

func FuzzShortNameParity(f *testing.F) {
	// Random string < 63
	f.Add("hello")
	// Random string > 63
	f.Add("jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf")
	f.Fuzz(func(t *testing.T, a string) {
		oldName := shortenName(a)
		newName := ShortenName(a)
		if oldName != newName {
			t.Fatalf("shortenName(%s) = %s, ShortenName(%s) = %s", a, oldName, a, newName)
		}

		equal := ShortenedEquals(newName, a)
		if !equal {
			t.Fatalf("ShortenedEquals(%s, %s) = %t", newName, a, equal)
		}
	})
}

func TestShortenName(t *testing.T) {
	t.Run("shorten name < 63", func(t *testing.T) {
		name := "hello"
		shortened := ShortenName(name)
		if shortened != name {
			t.Fatalf("ShortenName(%s) = %s", name, shortened)
		}
	})

	t.Run("shorten name > 63", func(t *testing.T) {
		name := "jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf"
		shortened := ShortenName(name)
		if len(shortened) != 63 {
			t.Fatalf("ShortenName(%s) = %s", name, shortened)
		}

		if shortened != "jfdklanfkljasfhjhldacaslkhdfkjs-f1e0028d0fbfe9afbd1a8bb9b53848d" {
			t.Fatalf("ShortenName(%s) = %s", name, shortened)
		}
	})
}

func TestShortenedEquals(t *testing.T) {

	testCases := []struct {
		name      string
		shortened string
		equal     bool
	}{
		{
			name:      "hello",
			shortened: "hello",
			equal:     true,
		},
		{
			name:      "jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf",
			shortened: "jfdklanfkljasfhjhldacaslkhdfkjs-f1e0028d0fbfe9afbd1a8bb9b53848d",
			equal:     true,
		},
		{
			name:      "jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf",
			shortened: "jfdklanfkljasfhjhldacaslkhdfkjs-f1e0028d0fbfe9afbd1a8bb9b53848",
			equal:     false,
		},
		{
			name:      "jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf",
			shortened: "jfdklanfkljasfhjhldacaslkhdfkjsf1e0028d0fbfe9afbd1a8bb9b53848ds",
			equal:     false,
		},
		{
			name:      "jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf",
			shortened: "jfdklanfkjasfhjhldacaslkhdfkjs-f1e0028d0fbfe9afbd1a8bb9b53848ds",
			equal:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal := ShortenedEquals(tc.shortened, tc.name)
			if equal != tc.equal {
				t.Fatalf("ShortenedEquals(%s, %s) = %t", tc.shortened, tc.name, equal)
			}
		})

	}

}
