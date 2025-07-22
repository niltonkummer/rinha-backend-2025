package log

type Log interface {
	WithTags(tag string) Log
	Write(fmt string, args ...any)
}
