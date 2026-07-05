package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/UnfriendlyMonkey/hsn/internal/config"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const progressEvery = 100_000

func main() {
	filePath := flag.String("file", "../database/people.v2.csv", "path to CSV file")
	password := flag.String("password", "password", "default password for imported users")
	flag.Parse()

	cfg := config.Load()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("password hash: %v", err)
	}

	f, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("open file: %v", err)
	}
	defer f.Close()

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connect: %v", err)
	}
	defer conn.Close(ctx)

	source := newCSVUserSource(f, string(hash))
	start := time.Now()

	n, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"users"},
		[]string{"first_name", "second_name", "birthdate", "city", "password_hash"},
		source,
	)
	if err != nil {
		log.Fatalf("copy from csv: %v (imported %d rows before failure)", err, source.imported)
	}

	if source.sanitized > 0 {
		log.Printf("sanitized invalid UTF-8 in %d rows", source.sanitized)
	}
	log.Printf("imported %d users in %s", n, time.Since(start).Round(time.Millisecond))
}

type csvUserSource struct {
	reader       *csv.Reader
	passwordHash string
	lineNum      int
	imported     int64
	sanitized    int64
	record       []string
	err          error
}

func newCSVUserSource(r io.Reader, passwordHash string) *csvUserSource {
	reader := csv.NewReader(r)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	return &csvUserSource{
		reader:       reader,
		passwordHash: passwordHash,
	}
}

func (s *csvUserSource) Next() bool {
	for {
		s.record, s.err = s.reader.Read()
		s.lineNum++
		if s.err == io.EOF {
			return false
		}
		if s.err != nil {
			return false
		}
		if len(s.record) == 0 || strings.TrimSpace(s.record[0]) == "" {
			continue
		}
		return true
	}
}

func (s *csvUserSource) Values() ([]any, error) {
	if len(s.record) < 3 {
		return nil, fmt.Errorf("line %d: expected at least 3 fields, got %d", s.lineNum, len(s.record))
	}

	name, nameFixed := sanitizeUTF8(strings.TrimSpace(s.record[0]))
	birthdate, birthdateFixed := sanitizeUTF8(strings.TrimSpace(s.record[1]))
	city, cityFixed := sanitizeUTF8(strings.TrimSpace(s.record[2]))
	if nameFixed || birthdateFixed || cityFixed {
		s.sanitized++
		log.Printf("line %d: sanitized invalid UTF-8 (name=%q, birthdate=%q, city=%q)", s.lineNum, name, birthdate, city)
	}

	secondName, firstName, err := splitName(name)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", s.lineNum, err)
	}

	var birthdateVal any
	if birthdate != "" {
		birthdateVal = birthdate
	}

	var cityVal any
	if city != "" {
		cityVal = city
	}

	s.imported++
	if s.imported%progressEvery == 0 {
		log.Printf("imported %d rows...", s.imported)
	}

	return []any{firstName, secondName, birthdateVal, cityVal, s.passwordHash}, nil
}

func (s *csvUserSource) Err() error {
	if s.err == io.EOF {
		return nil
	}
	return s.err
}

func splitName(full string) (secondName, firstName string, err error) {
	parts := strings.SplitN(strings.TrimSpace(full), " ", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid name %q", full)
	}
	return parts[0], parts[1], nil
}

func sanitizeUTF8(s string) (string, bool) {
	if utf8.ValidString(s) {
		return s, false
	}
	return strings.ToValidUTF8(s, ""), true
}
