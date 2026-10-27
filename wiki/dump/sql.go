package dump

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Column is one column of a table, as the CREATE TABLE statement gives it.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Value is one field of a row. A NULL field has Null set and an empty Raw.
// A quoted field has its escapes removed.
type Value struct {
	Raw    string
	Null   bool
	Quoted bool
}

// String returns the text of the value, or "" for NULL.
func (v Value) String() string { return v.Raw }

// Int returns the value as an integer.
func (v Value) Int() (int64, error) {
	if v.Null {
		return 0, errors.New("dump: NULL is not an integer")
	}
	return strconv.ParseInt(v.Raw, 10, 64)
}

// Row is one tuple of an INSERT statement, in the order of the columns.
type Row []Value

// SQLReader reads the rows of one table from a SQL dump file.
type SQLReader struct {
	br      *bufio.Reader
	closer  io.Closer
	table   string
	cols    []Column
	inList  bool
	lineNum int
}

// OpenSQL opens a SQL dump file. A name that ends in .gz is decompressed.
func OpenSQL(path string) (*SQLReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var r io.Reader = f
	var closer io.Closer = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(bufio.NewReaderSize(f, 1<<20))
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		r = gz
		closer = closeBoth{gz, f}
	}
	sr, err := NewSQLReader(r)
	if err != nil {
		_ = closer.Close()
		return nil, err
	}
	sr.closer = closer
	return sr, nil
}

type closeBoth struct{ a, b io.Closer }

func (c closeBoth) Close() error {
	err := c.a.Close()
	if err2 := c.b.Close(); err == nil {
		err = err2
	}
	return err
}

// NewSQLReader reads the head of a SQL dump up to the end of its CREATE TABLE
// statement. It returns an error when the stream has no CREATE TABLE.
func NewSQLReader(r io.Reader) (*SQLReader, error) {
	sr := &SQLReader{br: bufio.NewReaderSize(r, 1<<20)}
	if err := sr.readHead(); err != nil {
		return nil, err
	}
	return sr, nil
}

// Close closes the file that OpenSQL opened.
func (sr *SQLReader) Close() error {
	if sr.closer == nil {
		return nil
	}
	return sr.closer.Close()
}

// Table returns the name of the table.
func (sr *SQLReader) Table() string { return sr.table }

// Columns returns the columns of the table.
func (sr *SQLReader) Columns() []Column { return sr.cols }

// Index returns the positions of the named columns. It returns an error
// that names each column that the table does not have.
func (sr *SQLReader) Index(names ...string) ([]int, error) {
	out := make([]int, len(names))
	var missing []string
	for i, n := range names {
		out[i] = -1
		for j, c := range sr.cols {
			if c.Name == n {
				out[i] = j
				break
			}
		}
		if out[i] < 0 {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("dump: table %s has no column %s", sr.table, strings.Join(missing, ", "))
	}
	return out, nil
}

func (sr *SQLReader) readLine() (string, error) {
	var buf []byte
	for {
		chunk, err := sr.br.ReadSlice('\n')
		buf = append(buf, chunk...)
		if err == bufio.ErrBufferFull {
			continue
		}
		sr.lineNum++
		return string(buf), err
	}
}

func (sr *SQLReader) readHead() error {
	for {
		line, err := sr.readLine()
		if strings.HasPrefix(line, "CREATE TABLE ") {
			sr.table = quotedName(line[len("CREATE TABLE "):])
			return sr.readColumns()
		}
		if err == io.EOF {
			return errors.New("dump: no CREATE TABLE statement")
		}
		if err != nil {
			return err
		}
	}
}

func (sr *SQLReader) readColumns() error {
	for {
		line, err := sr.readLine()
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, ")") {
			if len(sr.cols) == 0 {
				return fmt.Errorf("dump: table %s has no columns", sr.table)
			}
			return nil
		}
		if strings.HasPrefix(t, "`") {
			name := quotedName(t)
			rest := strings.TrimSpace(t[len(name)+2:])
			typ, _, _ := strings.Cut(rest, " ")
			sr.cols = append(sr.cols, Column{Name: name, Type: strings.TrimSuffix(typ, ",")})
		}
		if err == io.EOF {
			return fmt.Errorf("dump: the CREATE TABLE of %s has no end", sr.table)
		}
		if err != nil {
			return err
		}
	}
}

// quotedName returns the text between the first pair of backticks of s.
func quotedName(s string) string {
	i := strings.IndexByte(s, '`')
	if i < 0 {
		return ""
	}
	j := strings.IndexByte(s[i+1:], '`')
	if j < 0 {
		return ""
	}
	return s[i+1 : i+1+j]
}

var insertPrefix = []byte("INSERT INTO ")

// Next returns the next row. It returns io.EOF after the last row. A row
// with a number of fields that is not the number of columns is an error.
func (sr *SQLReader) Next() (Row, error) {
	for {
		if !sr.inList {
			if err := sr.findInsert(); err != nil {
				return nil, err
			}
			sr.inList = true
		}
		c, err := sr.skipSpace()
		if err != nil {
			return nil, sr.eof(err)
		}
		switch c {
		case '(':
			row, err := sr.readTuple()
			if err != nil {
				return nil, err
			}
			if len(row) != len(sr.cols) {
				return nil, fmt.Errorf("dump: line %d: row has %d fields, table %s has %d columns", sr.lineNum+1, len(row), sr.table, len(sr.cols))
			}
			return row, nil
		case ',':
			continue
		case ';':
			sr.inList = false
		default:
			return nil, fmt.Errorf("dump: line %d: unexpected %q in the values of %s", sr.lineNum+1, c, sr.table)
		}
	}
}

func (sr *SQLReader) eof(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// findInsert moves to the first byte after the VALUES keyword of the next
// INSERT statement of the table.
func (sr *SQLReader) findInsert() error {
	for {
		head, err := sr.br.Peek(len(insertPrefix))
		if bytes.Equal(head, insertPrefix) {
			for {
				word, err := sr.readWord()
				if err != nil {
					return sr.eof(err)
				}
				if word == "VALUES" {
					return nil
				}
			}
		}
		if err == io.EOF && len(head) == 0 {
			return io.EOF
		}
		if _, err := sr.readLine(); err != nil {
			return err
		}
	}
}

func (sr *SQLReader) readWord() (string, error) {
	c, err := sr.skipSpace()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteByte(c)
	for {
		c, err := sr.br.ReadByte()
		if err != nil {
			return b.String(), err
		}
		if isSpace(c) {
			sr.count(c)
			return b.String(), nil
		}
		b.WriteByte(c)
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\n' || c == '\r' || c == '\t' }

func (sr *SQLReader) count(c byte) {
	if c == '\n' {
		sr.lineNum++
	}
}

func (sr *SQLReader) skipSpace() (byte, error) {
	for {
		c, err := sr.br.ReadByte()
		if err != nil {
			return 0, err
		}
		if !isSpace(c) {
			return c, nil
		}
		sr.count(c)
	}
}

// readTuple reads the fields of a tuple after its opening parenthesis.
func (sr *SQLReader) readTuple() (Row, error) {
	row := make(Row, 0, len(sr.cols))
	var buf []byte
	for {
		c, err := sr.skipSpace()
		if err != nil {
			return nil, sr.eof(err)
		}
		var v Value
		if c == '\'' {
			buf, err = sr.readQuoted(buf[:0])
			if err != nil {
				return nil, err
			}
			v = Value{Raw: string(buf), Quoted: true}
			c, err = sr.skipSpace()
			if err != nil {
				return nil, sr.eof(err)
			}
		} else {
			buf = buf[:0]
			for c != ',' && c != ')' {
				if !isSpace(c) {
					buf = append(buf, c)
				}
				if c, err = sr.br.ReadByte(); err != nil {
					return nil, sr.eof(err)
				}
			}
			if string(buf) == "NULL" {
				v = Value{Null: true}
			} else {
				v = Value{Raw: string(buf)}
			}
		}
		row = append(row, v)
		switch c {
		case ',':
		case ')':
			return row, nil
		default:
			return nil, fmt.Errorf("dump: line %d: unexpected %q after a field of %s", sr.lineNum+1, c, sr.table)
		}
	}
}

// readQuoted reads a quoted string after its opening quote and removes the
// MySQL escapes.
func (sr *SQLReader) readQuoted(buf []byte) ([]byte, error) {
	for {
		c, err := sr.br.ReadByte()
		if err != nil {
			return nil, sr.eof(err)
		}
		switch c {
		case '\\':
			e, err := sr.br.ReadByte()
			if err != nil {
				return nil, sr.eof(err)
			}
			buf = append(buf, unescape(e))
		case '\'':
			next, err := sr.br.Peek(1)
			if err == nil && next[0] == '\'' {
				_, _ = sr.br.ReadByte()
				buf = append(buf, '\'')
				continue
			}
			return buf, nil
		default:
			sr.count(c)
			buf = append(buf, c)
		}
	}
}

func unescape(e byte) byte {
	switch e {
	case '0':
		return 0
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'b':
		return '\b'
	case 'Z':
		return 0x1a
	default:
		return e
	}
}
