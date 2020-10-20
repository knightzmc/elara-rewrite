package lexer

import (
	"strings"
	"testing"
)

func BenchmarkEverySymbol(b *testing.B) {
	code := strings.Repeat(`
		let a = 30
		let a = 3.5
		let a = "Hello"
		let a = true
		let a = () => {}
		let hello-world => print "Hello World"
		hello-world()
		()[]{}<>

		+ - * / % && || ^ == != > >= < <= !
		, : _
		let	mut	struct if else match while
		`, 1_000)
	for n := 0; n < b.N; n++ {
		lex(nil, code)
	}
}
