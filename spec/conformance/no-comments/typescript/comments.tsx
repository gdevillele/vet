// File header
const text = "https://example.com/*literal*/";
function sample() { // trailing
/* block
   comment */
/** Documentation */
function documented() {}
    /* inner */ const raw = `// raw ${/* expression */ 1} /* literal */`;
}
//
const regex = /[/*]/;
const element = <div>// JSX text /* literal */{/* JSX comment */}</div>;
