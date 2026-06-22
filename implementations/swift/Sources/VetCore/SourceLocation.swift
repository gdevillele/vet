import Foundation

public struct SourceLocation: Equatable {
    public let line: Int
    public let column: Int
}

struct LocationRequest {
    let source: String
    let offset: Int
}

enum SourceLocations {
    static func location(_ request: LocationRequest) -> SourceLocation {
        let characters = Array(request.source)
        let boundedOffset = max(0, min(request.offset, characters.count))
        var line = 1
        var column = 1

        for index in 0..<boundedOffset {
            if characters[index] == "\n" {
                line += 1
                column = 1
            } else {
                column += 1
            }
        }

        return SourceLocation(line: line, column: column)
    }
}
