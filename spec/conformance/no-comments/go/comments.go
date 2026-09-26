// File header
package sample
var text = "https://example.com/*literal*/" // trailing
/* block
   comment */
// Documented describes the function.
func Documented() {
    /* inner */ println(`// raw /* literal */`)
}
//
//line virtual.go:400
var _ = `/* still a literal */`
