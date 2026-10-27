package pbauth

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRenameToWithdoRewritesAnOldDatabaseAndIsIdempotent(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = Close(app) })

	// A database the earlier migrations set up under the old name.
	settings := app.Settings()
	settings.Meta.AppName = legacyAppName
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		t.Fatal(err)
	}
	users.OTP.EmailTemplate = core.EmailTemplate{
		Subject: strings.ReplaceAll(codeSubject, appName, legacyAppName),
		Body:    strings.ReplaceAll(codeBody, appName, legacyAppName),
	}
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}

	for run := 1; run <= 2; run++ {
		if err := renameToWithdo(app); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		users, err = app.FindCollectionByNameOrId(UsersCollection)
		if err != nil {
			t.Fatal(err)
		}
		tpl := users.OTP.EmailTemplate
		if tpl.Subject != codeSubject || tpl.Body != codeBody {
			t.Errorf("run %d: template = %q / %q, want the Withdo one", run, tpl.Subject, tpl.Body)
		}
		if got := app.Settings().Meta.AppName; got != "Withdo" {
			t.Errorf("run %d: app name = %q", run, got)
		}
	}
	if strings.Contains(users.OTP.EmailTemplate.Subject+users.OTP.EmailTemplate.Body, legacyAppName) {
		t.Error("the old name is still in the template")
	}
}

func TestRenameToWithdoKeepsAnEditedTemplate(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = Close(app) })
	users, err := app.FindCollectionByNameOrId(UsersCollection)
	if err != nil {
		t.Fatal(err)
	}
	users.OTP.EmailTemplate = core.EmailTemplate{Subject: "Your code: {OTP}", Body: "<p>{OTP}</p>"}
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}
	if err := renameToWithdo(app); err != nil {
		t.Fatal(err)
	}
	users, _ = app.FindCollectionByNameOrId(UsersCollection)
	if tpl := users.OTP.EmailTemplate; tpl.Subject != "Your code: {OTP}" || tpl.Body != "<p>{OTP}</p>" {
		t.Errorf("an edited template changed: %q / %q", tpl.Subject, tpl.Body)
	}
}
