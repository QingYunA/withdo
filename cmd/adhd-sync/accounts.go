package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

// runAccountCommand runs the one-off account commands, which are run by hand on the server:
//
//	adhd-sync pb-superuser   -pb-dir DIR -email EMAIL           password on stdin
//	adhd-sync pb-import-user -pb-dir DIR -email EMAIL -name NAME  password on stdin, prints the subject
//	adhd-sync move-subject   -database-url URL -from SUBJECT -to SUBJECT
//
// handled is false when name is not one of them. A password is read from stdin and is never
// printed; on a terminal it is read without echo.
func runAccountCommand(name string, args []string, stdin io.Reader, stdout io.Writer) (handled bool, err error) {
	switch name {
	case "pb-superuser":
		return true, pbSuperuser(args, stdin, stdout)
	case "pb-import-user":
		return true, pbImportUser(args, stdin, stdout)
	case "move-subject":
		return true, moveSubject(args, stdout)
	}
	return false, nil
}

func pbSuperuser(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("pb-superuser", flag.ContinueOnError)
	dir := fs.String("pb-dir", "", "PocketBase data directory")
	email := fs.String("email", "", "superuser email")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *email == "" {
		return errors.New("pb-superuser: -pb-dir and -email are required")
	}
	password, err := readPassword(stdin, stdout)
	if err != nil {
		return err
	}
	app, err := pbauth.Open(*dir)
	if err != nil {
		return err
	}
	defer func() { _ = pbauth.Close(app) }()
	if err := pbauth.UpsertSuperuser(app, *email, password); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "superuser %s saved\n", *email)
	return nil
}

func pbImportUser(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("pb-import-user", flag.ContinueOnError)
	dir := fs.String("pb-dir", "", "PocketBase data directory")
	email := fs.String("email", "", "the user's email, which is the sign-in name")
	name := fs.String("name", "", "the user's display name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *email == "" {
		return errors.New("pb-import-user: -pb-dir and -email are required")
	}
	password, err := readPassword(stdin, stdout)
	if err != nil {
		return err
	}
	app, err := pbauth.Open(*dir)
	if err != nil {
		return err
	}
	defer func() { _ = pbauth.Close(app) }()
	subject, err := pbauth.ImportUser(app, *email, *name, password)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, subject)
	return nil
}

func moveSubject(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("move-subject", flag.ContinueOnError)
	databaseURL := fs.String("database-url", "", "Postgres connection URL")
	from := fs.String("from", "", "the current subject (a Zitadel user id)")
	to := fs.String("to", "", "the new subject (pb:<record id>)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *databaseURL == "" || *from == "" || *to == "" {
		return errors.New("move-subject: -database-url, -from and -to are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// The server is opened only for its database; no request ever reaches this authenticator.
	server, err := syncserver.Open(ctx, *databaseURL, syncserver.StaticTokens{})
	if err != nil {
		return err
	}
	defer server.Close()
	if err := server.MoveSubject(ctx, *from, *to); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "moved %s to %s\n", *from, *to)
	return nil
}

// readPassword reads a password from stdin: without echo on a terminal, else the first line with
// its line ending dropped. The password is not trimmed.
func readPassword(stdin io.Reader, stdout io.Writer) (string, error) {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		_, _ = fmt.Fprint(os.Stderr, "Password: ")
		raw, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read the password: %w", err)
		}
		if len(raw) == 0 {
			return "", errors.New("the password is empty")
		}
		return string(raw), nil
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read the password: %w", err)
	}
	password := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if password == "" {
		return "", errors.New("the password is empty")
	}
	return password, nil
}
