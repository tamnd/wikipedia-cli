package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func itoa(n int) string     { return strconv.Itoa(n) }
func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

// joinArgs joins positional arguments with spaces, so an unquoted multi-word
// query still works.
func joinArgs(args []string) string { return strings.Join(args, " ") }

// atoiArg parses a positional integer argument.
func atoiArg(s string) (int, error) { return strconv.Atoi(s) }

// Shared usage errors for the date/coordinate parsers.
var (
	errBadDate  = errors.New("dates must be YYYY-MM-DD")
	errBadCoord = errors.New("coordinate must be lat,lon (decimal degrees)")
)

// errStopStream ends a dump stream early once the limit is reached.
var errStopStream = errors.New("stop stream")

// readAllStdin reads all of stdin into memory.
func readAllStdin() ([]byte, error) { return io.ReadAll(os.Stdin) }

// toJSONString marshals v to a compact JSON string, ignoring errors.
func toJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// humanSize renders a byte count as a human-readable string.
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// downloadProgress returns a progress callback that prints a percentage to
// stderr, or a byte counter when the total size is unknown.
func downloadProgress(total int64) func(int64) {
	return func(have int64) {
		if total > 0 {
			fmt.Fprintf(os.Stderr, "\r  %s / %s (%.0f%%)", humanSize(have), humanSize(total), float64(have)/float64(total)*100)
		} else {
			fmt.Fprintf(os.Stderr, "\r  %s", humanSize(have))
		}
	}
}

func boolWord(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

// protectionSummary renders the protection levels as a compact "edit=sysop,
// move=autoconfirmed" string for the table view. The full list survives in JSON.
func protectionSummary(p []wiki.Protection) string {
	if len(p) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(p))
	for _, pr := range p {
		parts = append(parts, pr.Type+"="+pr.Level)
	}
	return strings.Join(parts, ",")
}

// resolveTarget parses a positional argument that may be a bare title or a
// pasted Wikipedia URL. When it is a URL on a different host, a client bound to
// that host is returned so the lookup hits the right wiki.
func (a *App) resolveTarget(arg string) (*wiki.Client, string, error) {
	title, host := wiki.ParseTarget(arg)
	if host == "" {
		c, err := a.Client()
		return c, title, err
	}
	cfg := a.Cfg
	cfg.SiteHost = host
	c, err := wiki.New(cfg, a.Cache)
	if err != nil {
		return nil, "", usageErr(err.Error())
	}
	return c, title, nil
}

// writeContent writes a body to stdout, optionally through a pager when stdout
// is a TTY and paging is not disabled.
func (a *App) writeContent(body string) error {
	if a.noPager || !isatty.IsTerminal(os.Stdout.Fd()) {
		_, err := fmt.Fprintln(cmdOut, body)
		return err
	}
	return page(body)
}

// page streams text through $PAGER (or less -R), falling back to stdout.
func page(body string) error {
	pager := os.Getenv("PAGER")
	args := []string{}
	if pager == "" {
		pager = "less"
		args = []string{"-R", "-F", "-X"}
	}
	cmd := exec.Command(pager, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_, e := fmt.Fprintln(cmdOut, body)
		return e
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_, e := fmt.Fprintln(cmdOut, body)
		return e
	}
	_, _ = io.WriteString(stdin, body)
	_ = stdin.Close()
	return cmd.Wait()
}

// argsOrStdin returns the positional args, or, when a single "-" is given,
// reads one item per line from stdin. JSONL on stdin is reduced to its "title"
// field so `search -o jsonl | get -` composes.
func argsOrStdin(args []string) ([]string, error) {
	if len(args) != 1 || args[0] != "-" {
		return args, nil
	}
	var out []string
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if t := jsonTitle(line); t != "" {
			out = append(out, t)
		} else {
			out = append(out, line)
		}
	}
	return out, sc.Err()
}

// jsonTitle pulls a "title" field out of a JSON object line, or returns "".
func jsonTitle(line string) string {
	if !strings.HasPrefix(line, "{") {
		return ""
	}
	var obj struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(line), &obj); err == nil {
		return obj.Title
	}
	return ""
}

// openBrowser opens a URL in the platform default browser.
func openBrowser(u string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		name = "xdg-open"
	}
	return exec.Command(name, append(args, u)...).Start()
}
