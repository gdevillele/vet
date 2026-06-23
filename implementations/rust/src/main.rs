use std::io::{stderr, stdout};

fn main() {
    let args = std::env::args().skip(1);
    let mut stdout = stdout();
    let mut stderr = stderr();
    std::process::exit(vet::run(args, &mut stdout, &mut stderr));
}
