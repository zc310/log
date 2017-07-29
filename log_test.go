package log

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"
)

func TestLog_Print(t *testing.T) {
	SetWriter(0, "", ioutil.Discard, ioutil.Discard)
	SetLevel(9)
	Print(1, 2, 3, 4, 5, 6)
	Print(6, 5, 4, 3, 2, 1, time.Now())
	Print(time.Now())
	Print(fmt.Errorf("Error"))
	Default.WithFormatter(&JSONFormatter{})
	Print(fmt.Errorf("Error2"))
	tmp := Default.NewWithPrefix("a.b.c.d")

	tmp.V(2).Print("abc")
	tmp.Error("abc")

}
func BenchmarkDummyLogger(b *testing.B) {
	SetWriter(0, "", ioutil.Discard, ioutil.Discard)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("https://github.com/notifications")
		}
	})
}

func BenchmarkDummyJSONLogger(b *testing.B) {
	Default.WithFormatter(&JSONFormatter{})
	SetWriter(0, "", ioutil.Discard, ioutil.Discard)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("https://github.com/notifications")
		}
	})
}
