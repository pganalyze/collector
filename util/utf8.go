package util

// InvalidUTF8Replacement is the Unicode replacement character (U+FFFD). The collector
// substitutes it for runs of invalid bytes so a value that isn't valid UTF-8 can still
// be carried in a protobuf string field (proto3 strings must be valid UTF-8). Shared so
// every place that scrubs invalid bytes does so identically.
const InvalidUTF8Replacement = "\uFFFD"
