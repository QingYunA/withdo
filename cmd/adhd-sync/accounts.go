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
//	adhd-sync move-subject   -database-url URL -from SUBJECT -to SUBJECT [-keep-alias=true]
//	adhd-sync add-alias      -database-url URL -subject SUBJECT -target SUBJECT
//	adhd-sync drop-alias     -database-url URL -subject SUBJECT
//	adhd-sync list-subjects  -database-url URL
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
	case "add-alias":
		return true, addAlias(args, stdout)
	case "drop-alias":
		return true, dropAlias(args, stdout)
	case "list-subjects":
		return true, listSubjects(args, stdout)
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
	return dbCommand("move-subject", args, func(fs *flag.FlagSet) func(context.Context, *syncserver.Server) error {
		from := fs.String("from", "", "the current subject (a Zitadel user id)")
		to := fs.String("to", "", "the new subject (pb:<record id>)")
		keepAlias := fs.Bool("keep-alias", true, "keep the old subject as an alias, so devices still signed in with it reach the same workspace")
		return func(ctx context.Context, server *syncserver.Server) error {
			if *from == "" || *to == "" {
				return errors.New("move-subject: -from and -to are required")
			}
			if err := server.MoveSubject(ctx, *from, *to, *keepAlias); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(stdout, "moved %s to %s (alias kept: %t)\n", *from, *to, *keepAlias)
			return nil
		}
	})
}

func addAlias(args []string, stdout io.Writer) error {
	return dbCommand("add-alias", args, func(fs *flag.FlagSet) func(context.Context, *syncserver.Server) error {
		subject := fs.String("subject", "", "the alias: an old subject that should reach the target's user")
		target := fs.String("target", "", "the subject of the user the alias reaches")
		return func(ctx context.Context, server *syncserver.Server) error {
			if *subject == "" || *target == "" {
				return errors.New("add-alias: -subject and -target are required")
			}
			if err := server.AddAlias(ctx, *subject, *target); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(stdout, "%s is now an alias of %s\n", *subject, *target)
			return nil
		}
	})
}

func dropAlias(args []string, stdout io.Writer) error {
	return dbCommand("drop-alias", args, func(fs *flag.FlagSet) func(context.Context, *syncserver.Server) error {
		subject := fs.String("subject", "", "the alias to remove")
		return func(ctx context.Context, server *syncserver.Server) error {
			if *subject == "" {
				return errors.New("drop-alias: -subject is required")
			}
			if err := server.DropAlias(ctx, *subject); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(stdout, "dropped alias %s\n", *subject)
			return nil
		}
	})
}

func listSubjects(args []string, stdout io.Writer) error {
	return dbCommand("list-subjects", args, func(*flag.FlagSet) func(context.Context, *syncserver.Server) error {
		return func(ctx context.Context, server *syncserver.Server) error {
			infos, err := server.ListSubjects(ctx)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, "user_id subject kind devices rows")
			for _, i := range infos {
				kind := "primary"
				if i.Alias {
					kind = "alias"
				}
				_, _ = fmt.Fprintf(stdout, "%d %s %s %d %d\n", i.UserID, i.Subject, kind, i.Devices, i.Rows)
			}
			return nil
		}
	})
}

// dbCommand runs a one-off command against the database: setup declares the command's own flags
// and returns its body, which runs once -database-url and the flags are parsed. The server is
// opened only for its database; no request ever reaches its authenticator.
func dbCommand(name string, args []string, setup func(*flag.FlagSet) func(context.Context, *syncserver.Server) error) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	databaseURL := fs.String("database-url", "", "Postgres connection URL")
	body := setup(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *databaseURL == "" {
		return fmt.Errorf("%s: -database-url is required", name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	server, err := syncserver.Open(ctx, *databaseURL, syncserver.StaticTokens{})
	if err != nil {
		return err
	}
	defer server.Close()
	return body(ctx, server)
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
