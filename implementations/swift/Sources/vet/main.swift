import Foundation
import VetCore

let code = CLI.run(CLIInvocation(
    arguments: Array(CommandLine.arguments.dropFirst()),
    stdout: { text in
        FileHandle.standardOutput.write(Data(text.utf8))
    },
    stderr: { text in
        FileHandle.standardError.write(Data(text.utf8))
    }
))

Foundation.exit(Int32(code))
