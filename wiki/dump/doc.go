// Package dump reads the SQL table files and the multistream page dumps of
// the Wikimedia dumps, and checks dump files against the sha1sums file of a
// run.
//
// The SQL files are MariaDB dumps of one table each. A reader takes the
// column names from the CREATE TABLE statement at the head of the file and
// then gives the rows of the INSERT statements one at a time. It does not
// need a database server. The table schemas change between MediaWiki
// versions, so a caller asks for its columns by name, and a missing column is
// an error.
//
// A multistream page dump is a series of bz2 streams, and its index gives
// the stream of each page. ReadPages reads only the streams that hold the
// pages that the caller asks for, with several workers.
package dump
