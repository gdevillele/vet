// File header
const TEXT: &str = "https://example.com/*literal*/";
fn main() { // trailing
/* block
   /* nested */ comment */
/// Documentation
fn documented() {}
    /* inner */ println!(r##"// raw /* literal */"##);
}
//
const RAW_C: &std::ffi::CStr = cr##"embedded " // not a comment /* nor this */"##;
macro_rules! sample { () => { /* macro */ } }
