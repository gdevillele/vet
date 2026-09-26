// File header
let text = "https://example.com/*literal*/"
func sample() { // trailing
/* block
   /* nested */ comment */
/// Documentation
func documented() {}
    /* inner */ print(#"// raw /* literal */"#)
}
//
let interpolated = "value \(/* expression */ 1)"
let regex = #/https?://example.com/\*literal\*/#
