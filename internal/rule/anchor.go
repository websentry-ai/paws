package rule

// CommandStart is the regex the catalog's command triggers begin with: a command starting the line, after ; & | or a
// newline, inside $( ), a subshell, or the script of bash -c "...". The git events use it too.
const CommandStart = gitAnchor
