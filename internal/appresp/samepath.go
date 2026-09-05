package appresp

// SamePath answers a Location that can only be this site: target when it is
// already a path here, "/" when it is anything else.
//
// A value that starts with a slash is not yet same-origin. "//evil.com" is a
// protocol-relative URL and some browsers normalise "/\evil.com" into one, so
// both are another origin written as a path. Both arrive that way from real
// sources: url.URL.RequestURI() of an absolute-form request line hands back
// "//evil.com", and so does the path of an absolute URL that genuinely names
// this host.
//
// One rule in one place, because this was got wrong twice by writing it twice:
// everything that builds a Location out of something a request carried goes
// through here, and nothing repeats the reasoning.
func SamePath(target string) string {
	if target == "" || target[0] != '/' {
		return "/"
	}

	if len(target) > 1 && (target[1] == '/' || target[1] == '\\') {
		return "/"
	}

	return target
}
